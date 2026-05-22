package auth

import "testing"

func TestHashPINAndVerifyPIN(t *testing.T) {
	hash, err := HashPIN("123456")
	if err != nil {
		t.Fatalf("HashPIN returned error: %v", err)
	}
	if hash == "123456" {
		t.Fatal("HashPIN returned plaintext PIN")
	}
	if !VerifyPIN(hash, "123456") {
		t.Fatal("VerifyPIN returned false for correct PIN")
	}
	if VerifyPIN(hash, "000000") {
		t.Fatal("VerifyPIN returned true for incorrect PIN")
	}
}

func TestHashPINRejectsShortPIN(t *testing.T) {
	_, err := HashPIN("123")
	if err == nil {
		t.Fatal("HashPIN returned nil error for short PIN")
	}
}
