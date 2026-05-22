package auth

import "testing"

func TestNewSessionTokenAndHash(t *testing.T) {
	token, err := NewSessionToken()
	if err != nil {
		t.Fatalf("NewSessionToken returned error: %v", err)
	}
	if len(token) < 32 {
		t.Fatalf("token length = %d, want at least 32", len(token))
	}
	hash := HashSessionToken(token)
	if hash == token {
		t.Fatal("HashSessionToken returned raw token")
	}
	if HashSessionToken(token) != hash {
		t.Fatal("HashSessionToken is not deterministic")
	}
}
