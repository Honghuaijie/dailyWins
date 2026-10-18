package auth

import (
	"strings"
	"testing"
)

func TestPasswordLengthBoundaries(t *testing.T) {
	for _, password := range []string{"123456", "六个中文字符", strings.Repeat("a", 72)} {
		hash, err := HashPassword(password)
		if err != nil {
			t.Fatalf("valid password length rejected: %v", err)
		}
		if !CheckPassword(hash, password) {
			t.Fatal("password did not verify")
		}
	}
	for _, password := range []string{"", "12345", "五个中文字", strings.Repeat("a", 73), strings.Repeat("字", 25)} {
		if _, err := HashPassword(password); err == nil {
			t.Fatal("invalid password length accepted")
		}
	}
}

func TestPasswordHash(t *testing.T) {
	h, err := HashPassword("a-long-test-password")
	if err != nil {
		t.Fatal(err)
	}
	if h == "a-long-test-password" || !CheckPassword(h, "a-long-test-password") || CheckPassword(h, "wrong-password") {
		t.Fatal("unsafe password verification")
	}
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short password accepted")
	}
}

func TestSessionJWT(t *testing.T) {
	token, err := GenerateJWT("test-secret", JWTClaims{UserID: 1, SessionID: "random-session"}, 1000000000)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseJWT("test-secret", token)
	if err != nil || claims.SessionID != "random-session" {
		t.Fatal("session identity lost", err)
	}
	if _, err := ParseJWT("different-secret", token); err == nil {
		t.Fatal("tampered signature accepted")
	}
}
