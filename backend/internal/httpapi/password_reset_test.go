package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordResetThroughAPI(t *testing.T) {
	dsn := os.Getenv("PASSWORD_RESET_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PASSWORD_RESET_TEST_DATABASE_URL to a migrated test database")
	}
	t.Setenv("AUTH_SECRET", "test-only-auth-secret-for-password-reset")
	mail := captureResetMail(t)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("password-reset-%d@example.test", time.Now().UnixNano())
	var userID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO users(email,password_hash,nickname,status,email_verified_at)
		VALUES($1,$2,'test','alumni_readonly',now()) RETURNING id`, email, string(hash)).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := pool.Exec(ctx, "DELETE FROM users WHERE id=$1", userID); err != nil {
			t.Error(err)
		}
	}()
	handler := New(pool)
	remote := fmt.Sprintf("reset-test-%d:1234", time.Now().UnixNano())
	call := func(method, path string, body any, token string, status int) *httptest.ResponseRecorder {
		t.Helper()
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, "/api/v1/auth/"+path, bytes.NewReader(payload))
		req.RemoteAddr = remote
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != status {
			t.Fatalf("%s returned %d, want %d (%s)", path, rec.Code, status, rec.Body.String())
		}
		return rec
	}
	login := func(password string, status int) *httptest.ResponseRecorder {
		t.Helper()
		return call(http.MethodPost, "login", map[string]string{"email": email, "password": password}, "", status)
	}
	oldSession := login("old-password", 200)
	var old struct {
		Access  string `json:"access_token"`
		Refresh string `json:"refresh_token"`
	}
	if err := json.Unmarshal(oldSession.Body.Bytes(), &old); err != nil || old.Access == "" || old.Refresh == "" {
		t.Fatal("login response did not contain both tokens")
	}
	call(http.MethodGet, "me", nil, old.Access, 200)
	request := func(address string) string {
		t.Helper()
		return call(http.MethodPost, "forgot-password", passwordResetRequest{Email: address}, "", 202).Body.String()
	}
	if request("  "+strings.ToUpper(email)+"  ") != request("missing-"+email) {
		t.Fatal("known and unknown email requests returned different messages")
	}
	readCode := func() string {
		t.Helper()
		select {
		case code := <-mail:
			return code
		case <-time.After(5 * time.Second):
			t.Fatal("password reset email did not arrive")
			return ""
		}
	}
	code := readCode()
	request(email) // A request during cooldown must not send a replacement.
	reset := func(code string, status int) {
		t.Helper()
		call(http.MethodPost, "reset-password", passwordResetConfirmation{Email: email, Code: code, Password: "new-password"}, "", status)
	}
	wrong := "000000"
	if code == wrong {
		wrong = "999999"
	}
	for range 5 {
		reset(wrong, 400)
	}
	reset(code, 400)
	login("old-password", 200)

	if _, err := pool.Exec(ctx, `UPDATE password_resets SET sent_at=now()-interval '2 minutes' WHERE user_id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	request(email)
	code = readCode()
	if _, err := pool.Exec(ctx, `UPDATE password_resets SET expires_at=now()-interval '1 second', sent_at=now()-interval '2 minutes' WHERE user_id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	reset(code, 400)
	request(email)
	code = readCode()
	reset(code, 200)
	reset(code, 400)
	login("old-password", 401)
	call(http.MethodGet, "me", nil, old.Access, 401)
	call(http.MethodPost, "refresh", map[string]string{"refresh_token": old.Refresh}, "", 401)
	session := login("new-password", 200)
	var current = old
	if err := json.Unmarshal(session.Body.Bytes(), &current); err != nil {
		t.Fatal(err)
	}
	call(http.MethodGet, "me", nil, current.Access, 200)
	call(http.MethodPost, "refresh", map[string]string{"refresh_token": current.Refresh}, "", 200)
	if !strings.Contains(session.Body.String(), "alumni_readonly") {
		t.Fatal("password reset changed the account status")
	}
}

// A local SMTP sink exercises the real mail transport without delivering mail externally.
func captureResetMail(t *testing.T) <-chan string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SMTP_HOST", host)
	t.Setenv("SMTP_PORT", port)
	t.Setenv("SMTP_FROM", "sender@example.test")
	t.Setenv("SMTP_TLS", "")
	t.Setenv("SMTP_USER", "")
	codes := make(chan string, 5)
	codePattern := regexp.MustCompile(`验证码是 ([0-9]{6})`)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
			smtp := textproto.NewConn(conn)
			_ = smtp.PrintfLine("220 test SMTP")
			for {
				line, err := smtp.ReadLine()
				if err != nil {
					break
				}
				switch line {
				case "DATA":
					_ = smtp.PrintfLine("354 End with a dot")
					data, _ := smtp.ReadDotBytes()
					if match := codePattern.FindSubmatch(data); len(match) == 2 {
						codes <- string(match[1])
					}
				case "QUIT":
					_ = smtp.PrintfLine("221 Goodbye")
					_ = smtp.Close()
					continue
				}
				_ = smtp.PrintfLine("250 OK")
			}
			_ = smtp.Close()
		}
	}()
	return codes
}
