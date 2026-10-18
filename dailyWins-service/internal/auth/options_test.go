package auth

import (
	"strings"
	"testing"
	"time"
)

func TestAuthConfiguration(t *testing.T) {
	t.Setenv("AUTH_SECRET", strings.Repeat("x", 32))
	t.Setenv("AUTH_TTL_HOURS", "")
	t.Setenv("COOKIE_SECURE", "")
	t.Setenv("ALLOWED_ORIGINS", "")
	o, err := NewOptions()
	if err != nil || !o.Secure || o.TTL != 168*time.Hour {
		t.Fatal("unsafe defaults", err)
	}
	for _, origin := range []string{"*", "https://*.example.com", "https://example.com/path", "https://user:pass@example.com"} {
		t.Setenv("ALLOWED_ORIGINS", origin)
		if _, err := NewOptions(); err == nil {
			t.Fatalf("unsafe origin accepted: %s", origin)
		}
	}
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("AUTH_SECRET", "CHANGE_ME_TO_A_RANDOM_SECRET_AT_LEAST_32_BYTES")
	if _, err := NewOptions(); err == nil {
		t.Fatal("placeholder secret accepted")
	}
}

func TestExpiredAndWrongAlgorithmJWT(t *testing.T) {
	secret := strings.Repeat("s", 32)
	expired, _ := GenerateJWT(secret, JWTClaims{UserID: 1, SessionID: "sid", ExpiresAt: time.Now().Unix() - 1}, time.Hour)
	if _, err := ParseJWT(secret, expired); err != ErrExpiredToken {
		t.Fatal("expired token accepted")
	}
	const wrongHeader = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"
	valid, _ := GenerateJWT(secret, JWTClaims{UserID: 1, SessionID: "sid"}, time.Hour)
	parts := strings.Split(valid, ".")
	input := wrongHeader + "." + parts[1]
	if _, err := ParseJWT(secret, input+"."+signHS256(secret, input)); err == nil {
		t.Fatal("wrong algorithm accepted")
	}
}
