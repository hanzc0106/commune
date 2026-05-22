package auth

import (
	"net/http"
	"testing"
	"time"
)

func TestSessionCookie(t *testing.T) {
	cookie := SessionCookie("token", time.Unix(100, 0))
	if cookie.Name != SessionCookieName {
		t.Fatalf("Name = %q", cookie.Name)
	}
	if cookie.Value != "token" {
		t.Fatalf("Value = %q", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Fatalf("Path = %q", cookie.Path)
	}
	if !cookie.HttpOnly {
		t.Fatal("HttpOnly = false")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v", cookie.SameSite)
	}
}

func TestClearSessionCookie(t *testing.T) {
	cookie := ClearSessionCookie()
	if cookie.Name != SessionCookieName {
		t.Fatalf("Name = %q", cookie.Name)
	}
	if cookie.Value != "" {
		t.Fatalf("Value = %q", cookie.Value)
	}
	if cookie.MaxAge != -1 {
		t.Fatalf("MaxAge = %d, want -1", cookie.MaxAge)
	}
}
