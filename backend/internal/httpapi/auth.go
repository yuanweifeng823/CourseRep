package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"mime"
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
	UserID         int64 `json:"sub"`
	Expires        int64 `json:"exp"`
	SessionVersion int64 `json:"sv,omitempty"`
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
	registerPasswordResetRoutes(mux, pool)
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
			log.Print("registration email delivery failed")
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
		role := "user"
		if adminEmails()[req.Email] {
			role = "admin"
		}
		err = tx.QueryRow(r.Context(), `INSERT INTO users(email,password_hash,nickname,status,email_verified_at,role) VALUES($1,$2,$3,'verified',now(),$4) RETURNING id`, req.Email, passwordHash, nickname, role).Scan(&userID)
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
		var version int64
		if err := pool.QueryRow(r.Context(), `SELECT id,status,role,password_hash,session_version FROM users WHERE email=$1`, email).Scan(&u.ID, &u.Status, &u.Role, &passwordHash, &version); err != nil || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil || u.Status == "disabled" {
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
		_, err = pool.Exec(r.Context(), `INSERT INTO refresh_sessions(user_id,token_hash,expires_at,session_version) VALUES($1,$2,now()+interval '30 days',$3)`, u.ID, hex.EncodeToString(refreshHash[:]), version)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法创建登录会话"})
			return
		}
		access, err := signAccess(u.ID, version, secret, time.Now().Add(15*time.Minute))
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
		var uid, version int64
		var status, role string
		err = tx.QueryRow(r.Context(), `UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now() RETURNING user_id,session_version`, hex.EncodeToString(h[:])).Scan(&uid, &version)
		if err == nil {
			err = tx.QueryRow(r.Context(), `SELECT status,role FROM users WHERE id=$1 AND session_version=$2`, uid, version).Scan(&status, &role)
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
		_, err = tx.Exec(r.Context(), `INSERT INTO refresh_sessions(user_id,token_hash,expires_at,session_version) VALUES($1,$2,now()+interval '30 days',$3)`, uid, hex.EncodeToString(nh[:]), version)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "无法刷新会话"})
			return
		}
		access, e := signAccess(uid, version, secret, time.Now().Add(15*time.Minute))
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
func signAccess(uid, version int64, secret []byte, expires time.Time) (string, error) {
	payload, e := json.Marshal(accessClaims{UserID: uid, Expires: expires.Unix(), SessionVersion: version})
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
		if pool.QueryRow(r.Context(), `SELECT id,status,role FROM users WHERE id=$1 AND session_version=$2`, c.UserID, c.SessionVersion).Scan(&u.ID, &u.Status, &u.Role) != nil || u.Status == "disabled" {
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
	n := strings.TrimSpace(req.Nickname)
	return validAccountEmail(req.Email) && emailDomainAllowed(req.Email) && validPassword(req.Password) && len([]rune(n)) >= 1 && len([]rune(n)) <= 40
}

func validAccountEmail(email string) bool {
	e := strings.TrimSpace(email)
	addr, err := mail.ParseAddress(e)
	return err == nil && addr.Address == e && len(e) <= 254 && !strings.ContainsAny(e, "\r\n")
}

func validPassword(password string) bool {
	return len(password) >= 10 && len(password) <= 72
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
	return sendAccountMail(to, "CourseRep 邮箱验证码", "你的验证码是 "+code+"，10 分钟内有效。若非本人操作，请忽略此邮件。")
}

func sendAccountMail(to, title, message string) error {
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
	subject := mime.QEncoding.Encode("utf-8", title)
	body := []byte("To: " + to + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + message + "\r\n")
	deadline := time.Now().Add(12 * time.Second)
	dialer := net.Dialer{Deadline: deadline}
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	implicit := os.Getenv("SMTP_TLS") == "implicit" || port == "465" || port == "994"
	if implicit {
		conn = tls.Client(conn, tlsConfig)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if supported, _ := client.Extension("STARTTLS"); !implicit && supported {
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// adminEmails 返回通过邮箱自动获得运营权限的账号。
// 默认包含项目约定的初始管理员账号，可用逗号分隔的 ADMIN_EMAILS 覆盖。
func adminEmails() map[string]bool {
	raw := strings.TrimSpace(os.Getenv("ADMIN_EMAILS"))
	if raw == "" {
		raw = "weifeyuan2-c@my.cityu.edu.hk"
	}
	emails := make(map[string]bool)
	for _, item := range strings.Split(raw, ",") {
		email := strings.ToLower(strings.TrimSpace(item))
		if email != "" {
			emails[email] = true
		}
	}
	return emails
}

// syncAdminEmails 把 ADMIN_EMAILS 中已注册的账号提升为管理员。
// 只提升、不降级，避免改坏运营人员手工授予的权限。
func syncAdminEmails(ctx context.Context, pool *pgxpool.Pool) error {
	emails := adminEmails()
	if len(emails) == 0 {
		return nil
	}
	list := make([]string, 0, len(emails))
	for email := range emails {
		list = append(list, email)
	}
	_, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE lower(email)=ANY($1) AND role<>'admin'`, list)
	return err
}
