package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrMissingSecret = errors.New("jwt secret is required")
	ErrInvalidToken  = errors.New("invalid jwt token")
	ErrExpiredToken  = errors.New("expired jwt token")
)

type JWTClaims struct {
	UserID    int32  `json:"user_id"`
	ExpiresAt int64  `json:"exp"`
	SessionID string `json:"sid"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func GenerateJWT(secret string, claims JWTClaims, expireDuration time.Duration) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrMissingSecret
	}
	if claims.ExpiresAt == 0 {
		if expireDuration <= 0 {
			return "", fmt.Errorf("%w: expire duration must be positive", ErrInvalidToken)
		}
		claims.ExpiresAt = time.Now().Add(expireDuration).Unix()
	}

	headerBytes, err := json.Marshal(jwtHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	headerPart := base64.RawURLEncoding.EncodeToString(headerBytes)
	claimsPart := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := headerPart + "." + claimsPart
	signature := signHS256(secret, signingInput)
	return signingInput + "." + signature, nil
}

func ParseJWT(secret string, token string) (*JWTClaims, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrMissingSecret
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSignature := signHS256(secret, signingInput)
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrInvalidToken
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return nil, ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var claims JWTClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.ExpiresAt <= 0 {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() >= claims.ExpiresAt {
		return nil, ErrExpiredToken
	}
	return &claims, nil
}

func signHS256(secret string, signingInput string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
