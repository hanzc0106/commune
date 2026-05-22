package http

import (
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	handler := NewHandler(Options{})
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "ok\n" {
		t.Fatalf("body = %q, want ok newline", rec.Body.String())
	}
}

func TestMissingStaticFallbackReturnsNotFound(t *testing.T) {
	handler := NewHandler(Options{})
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
