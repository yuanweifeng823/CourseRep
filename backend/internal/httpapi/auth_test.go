package httpapi

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthRateLimitUsesRemoteIP(t *testing.T) {
	path := "test-rate-limit-" + time.Now().Format("150405.000000000")
	first := httptest.NewRequest("POST", "/", nil)
	first.RemoteAddr = "192.0.2.10:4000"
	if !allowRate(httptest.NewRecorder(), first, path, 1, time.Minute) {
		t.Fatal("first request should pass")
	}
	second := httptest.NewRequest("POST", "/", nil)
	second.RemoteAddr = "192.0.2.10:4001"
	w := httptest.NewRecorder()
	if allowRate(w, second, path, 1, time.Minute) || w.Code != 429 {
		t.Fatalf("second request from the same IP should be limited, got %d", w.Code)
	}
	third := httptest.NewRequest("POST", "/", nil)
	third.RemoteAddr = "192.0.2.11:4000"
	if !allowRate(httptest.NewRecorder(), third, path, 1, time.Minute) {
		t.Fatal("a different IP should have a separate limit")
	}
}

func TestValidRegistrationBounds(t *testing.T) {
	if !validRegistration(registration{Email: "student@my.cityu.edu.hk", Password: "correct-horse", Nickname: "学生"}) {
		t.Fatal("valid registration should pass")
	}
	for _, req := range []registration{
		{Email: "student@outside.example", Password: "correct-horse", Nickname: "A"},
		{Email: "student@my.cityu.edu.hk\r\nBcc: x@example.com", Password: "correct-horse", Nickname: "A"},
		{Email: "student@my.cityu.edu.hk", Password: "short", Nickname: "A"},
	} {
		if validRegistration(req) {
			t.Fatalf("invalid registration accepted: %#v", req)
		}
	}
}
