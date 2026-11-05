package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type registration struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type accessClaims struct {
	UserID  int64 `json:"sub"`
	Expires int64 `json:"exp"`
}
type authUser struct {
	ID     int64
	Status string
	Role   string
}
type authUserKey struct{}

func userFromContext(ctx context.Context) authUser {
	u, _ := ctx.Value(authUserKey{}).(authUser)
	return u
}

// registerRoutes implements the first step of email verification. SMTP settings
// are mandatory; verification codes and credentials are never returned by API.
func registerRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if !allowRate(w, r, "register", 5, time.Hour) {
			return
		}
		var req registration
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil || !validRegistration(req) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请检查邮箱、密码和昵称格式"})
			return
		}
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		if !emailDomainAllowed(req.Email) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱不在允许的学生邮箱域名范围内"})
			return
		}
		var exists bool
		if err := pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`, req.Email).Scan(&exists); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "暂时无法处理注册"})
			return
		}
		if exists {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "该邮箱已注册"})
			return
		}
		var recent int
		if err := pool.QueryRow(r.Context(), `SELECT count(*) FROM email_verifications WHERE email=$1 AND created_at > now()-interval '1 hour'`, req.Email).Scan(&recent); err != nil || recent >= 5 {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "验证码请求次数过多，请稍后再试"})
			return
		}
		codeBytes := make([]byte, 4)
		if _, err := rand.Read(codeBytes); err != nil {
			writeJSON(w, 500, map[string]string{"error": "暂时无法处理注册"})
			return
		}
		code := fmt.Sprintf("%06d", (uint32(codeBytes[0])<<24|uint32(codeBytes[1])<<16|uint32(codeBytes[2])<<8|uint32(codeBytes[3]))%1_000_000)
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "暂时无法处理注册"})
			return
		}
		codeHash := sha256.Sum256([]byte(code))
		_, err = pool.Exec(r.Context(), `INSERT INTO email_verifications(email,code_hash,password_hash,nickname,expires_at) VALUES($1,$2,$3,$4,now()+interval '10 minutes')`, req.Email, hex.EncodeToString(codeHash[:]), string(hash), strings.TrimSpace(req.Nickname))
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "暂时无法处理注册"})
			return
		}
		if err := sendVerification(req.Email, code); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "验证邮件暂时无法发送，请稍后重试"})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"message": "验证码已发送，请在 10 分钟内完成验证"})
	})
	mux.HandleFunc("POST /api/v1/auth/verify", func(w http.ResponseWriter, r *http.Request) {
		if !allowRate(w, r, "verify", 20, time.Hour) {
			return
		}
		var req struct {
			Email string `json:"email"`
			Code  string `json:"code"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&req); err != nil || len(req.Code) != 6 {
			writeJSON(w, 400, map[string]string{"error": "验证码无效或已过期"})
			return
		}
		req.Email = strings.ToLower(strings.TrimSpace(req.Email))
		codeHash := sha256.Sum256([]byte(req.Code))
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "暂时无法完成验证"})
			return
		}
		defer tx.Rollback(r.Context())
		var challengeID int64
		var passwordHash, nickname string
		err = tx.QueryRow(r.Context(), `SELECT id,password_hash,nickname FROM email_verifications WHERE email=$1 AND code_hash=$2 AND consumed_at IS NULL AND expires_at>now() AND attempts<5 ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, req.Email, hex.EncodeToString(codeHash[:])).Scan(&challengeID, &passwordHash, &nickname)
		if err != nil {
			_, _ = tx.Exec(r.Context(), `UPDATE email_verifications SET attempts=attempts+1 WHERE email=$1 AND consumed_at IS NULL AND expires_at>now()`, req.Email)
			writeJSON(w, 400, map[string]string{"error": "验证码无效或已过期"})
			return
		}
		var userID int64
		err = tx.QueryRow(r.Context(), `INSERT INTO users(email,password_hash,nickname,status,email_verified_at) VALUES($1,$2,$3,'verified',now()) RETURNING id`, req.Email, passwordHash, nickname).Scan(&userID)
		if err == nil {
			_, err = tx.Exec(r.Context(), `UPDATE email_verifications SET consumed_at=now() WHERE email=$1 AND consumed_at IS NULL`, req.Email)
		}
		if err != nil {
			writeJSON(w, 409, map[string]string{"error": "该邮箱已注册或无法完成验证"})
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			writeJSON(w, 500, map[string]string{"error": "暂时无法完成验证"})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"user_id": userID, "status": "verified", "message": "邮箱验证完成，请登录"})
	})
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if !allowRate(w, r, "login", 10, time.Minute) {
			return
		}
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&req) != nil || len(req.Password) > 72 {
			writeJSON(w, 400, map[string]string{"error": "邮箱或密码无效"})
			return
		}
		email := strings.ToLower(strings.TrimSpace(req.Email))
		var u authUser
		var passwordHash string
		if err := pool.QueryRow(r.Context(), `SELECT id,status,role,password_hash FROM users WHERE email=$1`, email).Scan(&u.ID, &u.Status, &u.Role, &passwordHash); err != nil || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil || u.Status == "disabled" {
			writeJSON(w, 401, map[string]string{"error": "邮箱或密码无效"})
			return
		}
		secret := []byte(os.Getenv("AUTH_SECRET"))
		if len(secret) < 32 {
			writeJSON(w, 503, map[string]string{"error": "认证服务尚未配置"})
			return
		}
		refresh, err := opaqueToken()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法创建登录会话"})
			return
		}
		refreshHash := sha256.Sum256([]byte(refresh))
		_, err = pool.Exec(r.Context(), `INSERT INTO refresh_sessions(user_id,token_hash,expires_at) VALUES($1,$2,now()+interval '30 days')`, u.ID, hex.EncodeToString(refreshHash[:]))
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法创建登录会话"})
			return
		}
		access, err := signAccess(u.ID, secret, time.Now().Add(15*time.Minute))
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法创建登录会话"})
			return
		}
		writeJSON(w, 200, map[string]any{"access_token": access, "expires_in": 900, "refresh_token": refresh, "user": map[string]any{"id": u.ID, "status": u.Status, "role": u.Role}})
	})
	mux.HandleFunc("POST /api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if !allowRate(w, r, "refresh", 30, time.Minute) {
			return
		}
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&req) != nil || req.RefreshToken == "" {
			writeJSON(w, 400, map[string]string{"error": "刷新令牌无效"})
			return
		}
		secret := []byte(os.Getenv("AUTH_SECRET"))
		if len(secret) < 32 {
			writeJSON(w, 503, map[string]string{"error": "认证服务尚未配置"})
			return
		}
		h := sha256.Sum256([]byte(req.RefreshToken))
		tx, err := pool.Begin(r.Context())
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法刷新会话"})
			return
		}
		defer tx.Rollback(r.Context())
		var uid int64
		var status, role string
		err = tx.QueryRow(r.Context(), `UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now() RETURNING user_id`, hex.EncodeToString(h[:])).Scan(&uid)
		if err == nil {
			err = tx.QueryRow(r.Context(), `SELECT status,role FROM users WHERE id=$1`, uid).Scan(&status, &role)
		}
		if err != nil || status == "disabled" {
			writeJSON(w, 401, map[string]string{"error": "刷新令牌无效或账号不可用"})
			return
		}
		newRefresh, e := opaqueToken()
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "无法刷新会话"})
			return
		}
		nh := sha256.Sum256([]byte(newRefresh))
		_, err = tx.Exec(r.Context(), `INSERT INTO refresh_sessions(user_id,token_hash,expires_at) VALUES($1,$2,now()+interval '30 days')`, uid, hex.EncodeToString(nh[:]))
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法刷新会话"})
			return
		}
		access, e := signAccess(uid, secret, time.Now().Add(15*time.Minute))
		if e != nil {
			writeJSON(w, 500, map[string]string{"error": "无法刷新会话"})
			return
		}
		if tx.Commit(r.Context()) != nil {
			writeJSON(w, 500, map[string]string{"error": "无法刷新会话"})
			return
		}
		writeJSON(w, 200, map[string]any{"access_token": access, "expires_in": 900, "refresh_token": newRefresh, "user": map[string]any{"id": uid, "status": status, "role": role}})
	})
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&req) != nil {
			writeJSON(w, 400, map[string]string{"error": "请求无效"})
			return
		}
		h := sha256.Sum256([]byte(req.RefreshToken))
		_, err := pool.Exec(r.Context(), `UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, hex.EncodeToString(h[:]))
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法结束会话"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("GET /api/v1/auth/me", authenticate(pool, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		writeJSON(w, 200, map[string]any{"id": u.ID, "status": u.Status, "role": u.Role})
	})))
}

type rateEntry struct {
	count int
	reset time.Time
}

var rateMu sync.Mutex
var rateEntries = map[string]rateEntry{}

func allowRate(w http.ResponseWriter, r *http.Request, endpoint string, limit int, window time.Duration) bool {
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	key := endpoint + "\x00" + ip
	now := time.Now()
	rateMu.Lock()
	defer rateMu.Unlock()
	entry := rateEntries[key]
	if now.After(entry.reset) {
		entry = rateEntry{reset: now.Add(window)}
	}
	if entry.count >= limit {
		w.Header().Set("Retry-After", strconv.Itoa(max(1, int(time.Until(entry.reset).Seconds()))))
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "请求过于频繁，请稍后再试"})
		return false
	}
	entry.count++
	rateEntries[key] = entry
	if len(rateEntries) > 10000 {
		for k, v := range rateEntries {
			if now.After(v.reset) {
				delete(rateEntries, k)
			}
		}
	}
	return true
}

func opaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func signAccess(uid int64, secret []byte, expires time.Time) (string, error) {
	payload, e := json.Marshal(accessClaims{uid, expires.Unix()})
	if e != nil {
		return "", e
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func authenticate(pool *pgxpool.Pool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")), ".")
		secret := []byte(os.Getenv("AUTH_SECRET"))
		if len(parts) != 2 || len(secret) < 32 {
			writeJSON(w, 401, map[string]string{"error": "需要登录"})
			return
		}
		mac := hmac.New(sha256.New, secret)
		_, _ = mac.Write([]byte(parts[0]))
		sig, e := base64.RawURLEncoding.DecodeString(parts[1])
		if e != nil || !hmac.Equal(sig, mac.Sum(nil)) {
			writeJSON(w, 401, map[string]string{"error": "登录凭据无效"})
			return
		}
		data, e := base64.RawURLEncoding.DecodeString(parts[0])
		var c accessClaims
		if e != nil || json.Unmarshal(data, &c) != nil || c.UserID < 1 || c.Expires < time.Now().Unix() {
			writeJSON(w, 401, map[string]string{"error": "登录凭据已过期"})
			return
		}
		var u authUser
		if pool.QueryRow(r.Context(), `SELECT id,status,role FROM users WHERE id=$1`, c.UserID).Scan(&u.ID, &u.Status, &u.Role) != nil || u.Status == "disabled" {
			writeJSON(w, 401, map[string]string{"error": "账号不可用"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authUserKey{}, u)))
	})
}

func requireVerified(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		if u.Status != "verified" {
			writeJSON(w, 403, map[string]string{"error": "当前账号状态不允许此操作"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
func requireModerator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		if u.Role != "moderator" && u.Role != "admin" {
			writeJSON(w, 403, map[string]string{"error": "需要运营权限"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := userFromContext(r.Context())
		if u.Role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "需要管理员权限"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validRegistration(req registration) bool {
	e := strings.TrimSpace(req.Email)
	n := strings.TrimSpace(req.Nickname)
	addr, err := mail.ParseAddress(e)
	return err == nil && addr.Address == e && len(e) <= 254 && emailDomainAllowed(e) && !strings.ContainsAny(e, "\r\n") && len(req.Password) >= 10 && len(req.Password) <= 72 && len([]rune(n)) >= 1 && len([]rune(n)) <= 40
}

func emailDomainAllowed(email string) bool {
	configured := strings.TrimSpace(os.Getenv("STUDENT_EMAIL_DOMAINS"))
	if configured == "" {
		configured = "my.cityu.edu.hk"
	}
	parts := strings.Split(strings.ToLower(strings.TrimSpace(email)), "@")
	if len(parts) != 2 || parts[0] == "" {
		return false
	}
	domain := parts[1]
	for _, candidate := range strings.Split(configured, ",") {
		candidate = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(candidate)), "@")
		if candidate != "" && domain == candidate {
			return true
		}
	}
	return false
}

func sendVerification(to, code string) error {
	host := os.Getenv("SMTP_HOST")
	from := os.Getenv("SMTP_FROM")
	if host == "" || from == "" {
		return fmt.Errorf("SMTP_HOST and SMTP_FROM must be configured")
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587"
	}
	address := net.JoinHostPort(host, port)
	user, pass := os.Getenv("SMTP_USER"), os.Getenv("SMTP_PASSWORD")
	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}
	body := []byte("To: " + to + "\r\nSubject: CourseRep 邮箱验证码\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n你的验证码是 " + code + "，10 分钟内有效。若非本人操作，请忽略此邮件。\r\n")
	return smtp.SendMail(address, auth, from, []string{to}, body)
}
