package httpapi

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type passwordResetRequest struct {
	Email string `json:"email" maxLength:"254"`
}

type passwordResetConfirmation struct {
	Email    string `json:"email" maxLength:"254"`
	Code     string `json:"code" pattern:"^[0-9]{6}$"`
	Password string `json:"password" minLength:"10" maxLength:"72"`
}

func registerPasswordResetRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mailSlots := make(chan struct{}, 8)
	mux.HandleFunc("POST /api/v1/auth/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		if !allowRate(w, r, "forgot-password", 10, time.Hour) {
			return
		}
		var req passwordResetRequest
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&req) != nil || !validAccountEmail(req.Email) {
			writeJSON(w, 400, map[string]string{"error": "请输入有效的注册邮箱"})
			return
		}
		secret := []byte(os.Getenv("AUTH_SECRET"))
		if len(secret) < 32 || os.Getenv("SMTP_HOST") == "" || os.Getenv("SMTP_FROM") == "" {
			writeJSON(w, 503, map[string]string{"error": "密码重置邮件服务暂不可用"})
			return
		}
		select {
		case mailSlots <- struct{}{}:
		default:
			writeJSON(w, 503, map[string]string{"error": "密码重置邮件服务繁忙，请稍后再试"})
			return
		}
		handedOff := false
		defer func() {
			if !handedOff {
				<-mailSlots
			}
		}()
		email := strings.ToLower(strings.TrimSpace(req.Email))
		number, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
		if err != nil {
			passwordResetError(w)
			return
		}
		code := fmt.Sprintf("%06d", number.Int64())
		result, err := pool.Exec(r.Context(), `
			INSERT INTO password_resets(user_id, code_hash, sent_at, expires_at)
			SELECT id, $2, now(), now()+interval '10 minutes'
			FROM users WHERE email=$1 AND email_verified_at IS NOT NULL AND status<>'disabled'
			ON CONFLICT (user_id) DO UPDATE
			SET code_hash=excluded.code_hash, sent_at=excluded.sent_at,
			    expires_at=excluded.expires_at, attempts=0
			WHERE password_resets.sent_at <= now()-interval '1 minute'`, email, passwordResetHash(secret, email, code))
		if err != nil {
			passwordResetError(w)
			return
		}
		if result.RowsAffected() == 1 {
			handedOff = true
			go func() {
				defer func() { <-mailSlots }()
				if err := sendAccountMail(email, "CourseRep 密码重置验证码", "你的密码重置验证码是 "+code+"，10 分钟内有效。若非本人操作，请忽略此邮件。"); err != nil {
					log.Print("password reset email delivery failed")
				}
			}()
		}
		writeJSON(w, 202, map[string]string{"message": "如果该邮箱可用于重置密码，验证码将发送到该邮箱"})
	})
	mux.HandleFunc("POST /api/v1/auth/reset-password", func(w http.ResponseWriter, r *http.Request) {
		if !allowRate(w, r, "reset-password", 20, time.Hour) {
			return
		}
		var req passwordResetConfirmation
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&req) != nil || !validAccountEmail(req.Email) {
			writeJSON(w, 400, map[string]string{"error": "请输入有效的注册邮箱"})
			return
		}
		if !validPassword(req.Password) {
			writeJSON(w, 400, map[string]string{"error": "新密码须为 10 至 72 字节"})
			return
		}
		if len(req.Code) != 6 || strings.Trim(req.Code, "0123456789") != "" {
			writeJSON(w, 400, map[string]string{"error": "请输入 6 位数字验证码"})
			return
		}
		secret := []byte(os.Getenv("AUTH_SECRET"))
		if len(secret) < 32 {
			writeJSON(w, 503, map[string]string{"error": "密码重置服务暂不可用"})
			return
		}
		email := strings.ToLower(strings.TrimSpace(req.Email))
		ctx := r.Context()
		tx, err := pool.Begin(ctx)
		if err != nil {
			passwordResetError(w)
			return
		}
		defer tx.Rollback(ctx)
		var userID int64
		var codeHash []byte
		err = tx.QueryRow(ctx, `
			UPDATE password_resets AS p SET attempts=p.attempts+1
			FROM users AS u
			WHERE p.user_id=u.id AND u.email=$1 AND u.email_verified_at IS NOT NULL AND u.status<>'disabled'
			  AND p.expires_at>clock_timestamp() AND p.attempts<5
			RETURNING p.user_id, p.code_hash`, email).Scan(&userID, &codeHash)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, 400, map[string]string{"error": "验证码错误、已过期或已用完，请检查后重试"})
			return
		}
		if err != nil {
			passwordResetError(w)
			return
		}
		if !hmac.Equal(codeHash, passwordResetHash(secret, email, req.Code)) {
			if err := tx.Commit(ctx); err != nil {
				passwordResetError(w)
				return
			}
			writeJSON(w, 400, map[string]string{"error": "验证码错误、已过期或已用完，请检查后重试"})
			return
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			passwordResetError(w)
			return
		}
		result, err := tx.Exec(ctx, `
			UPDATE users SET password_hash=$2, session_version=session_version+1, updated_at=now()
			WHERE id=$1 AND email=$3 AND email_verified_at IS NOT NULL AND status<>'disabled'`, userID, string(passwordHash), email)
		if err != nil {
			passwordResetError(w)
			return
		}
		if result.RowsAffected() == 0 {
			writeJSON(w, 400, map[string]string{"error": "验证码错误、已过期或已用完，请检查后重试"})
			return
		}
		if _, err := tx.Exec(ctx, `UPDATE password_resets SET expires_at=now() WHERE user_id=$1`, userID); err != nil {
			passwordResetError(w)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			passwordResetError(w)
			return
		}
		writeJSON(w, 200, map[string]string{"message": "密码已重置，请使用新密码登录"})
	})
}

func passwordResetError(w http.ResponseWriter) {
	writeJSON(w, 500, map[string]string{"error": "暂时无法处理密码重置，请稍后再试"})
}

func passwordResetHash(secret []byte, email, code string) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte("password-reset\x00" + email + "\x00" + code))
	return mac.Sum(nil)
}
