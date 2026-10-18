package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	v1 "dailyWins-service/api/all-pb-go/v1"
	"dailyWins-service/internal/auth"
	e "dailyWins-service/internal/errors"
)

type Account struct {
	ID                     int32
	Username, PasswordHash string
}
type Session struct {
	ID        string
	UserID    int32
	ExpiresAt time.Time
}
type AccountRepo interface {
	FindAccount(context.Context, string) (*Account, error)
	GetAccount(context.Context, int32) (*Account, error)
	CreateSession(context.Context, *Session) error
	FindSession(context.Context, string) (*Session, error)
	RevokeSession(context.Context, string) error
}
type attempt struct {
	count int
	until time.Time
}
type AccountUsecase struct {
	repo      AccountRepo
	options   *auth.Options
	mu        sync.Mutex
	attempts  map[string]attempt
	dummyHash string
}

func NewAccountUsecase(repo AccountRepo, options *auth.Options) (*AccountUsecase, error) {
	dummy, err := auth.HashPassword("dummy-account-password-only")
	if err != nil {
		return nil, err
	}
	return &AccountUsecase{repo: repo, options: options, attempts: map[string]attempt{}, dummyHash: dummy}, nil
}

// 以来源地址为限流键，未知账号也执行相同成本的哈希验证。单实例适用；多实例应增加网关限流。
func (uc *AccountUsecase) allow(key string) bool {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	now := time.Now()
	for k, a := range uc.attempts {
		if now.After(a.until) {
			delete(uc.attempts, k)
		}
	}
	a := uc.attempts[key]
	if a.count >= 5 {
		return false
	}
	if a.count == 0 {
		if len(uc.attempts) >= 10000 {
			return false
		}
		a.until = now.Add(15 * time.Minute)
	}
	a.count++
	uc.attempts[key] = a
	return true
}
func (uc *AccountUsecase) Login(ctx context.Context, username, password, source string) (*v1.LoginReply_Data, error) {
	if !uc.allow(source) {
		return nil, e.EWithMessage(e.ErrTooManyRequests, "登录尝试过多，请 15 分钟后重试")
	}
	account, err := uc.repo.FindAccount(ctx, username)
	if err != nil {
		return nil, err
	}
	hash := uc.dummyHash
	if account != nil {
		hash = account.PasswordHash
	}
	valid := auth.CheckPassword(hash, password)
	if !valid || account == nil {
		return nil, e.EWithMessage(e.ErrUnauthorized, "账号或密码不正确")
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return nil, err
	}
	sid := hex.EncodeToString(random)
	expires := time.Now().UTC().Add(uc.options.TTL).Truncate(time.Second)
	token, err := auth.GenerateJWT(uc.options.Secret, auth.JWTClaims{UserID: account.ID, SessionID: sid, ExpiresAt: expires.Unix()}, uc.options.TTL)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.CreateSession(ctx, &Session{ID: sessionKey(sid), UserID: account.ID, ExpiresAt: expires}); err != nil {
		return nil, err
	}
	uc.mu.Lock()
	delete(uc.attempts, source)
	uc.mu.Unlock()
	return &v1.LoginReply_Data{Account: &v1.AccountInfo{Id: account.ID, Username: account.Username}, Token: token, ExpiresAt: expires.Format(time.RFC3339)}, nil
}
func sessionKey(sid string) string { s := sha256.Sum256([]byte(sid)); return hex.EncodeToString(s[:]) }
func (uc *AccountUsecase) Authenticate(ctx context.Context, token string) (*Session, error) {
	if strings.TrimSpace(token) == "" {
		return nil, e.E(e.ErrMissingToken)
	}
	claims, err := auth.ParseJWT(uc.options.Secret, token)
	if errors.Is(err, auth.ErrExpiredToken) {
		return nil, e.E(e.ErrExpiredToken)
	}
	if err != nil || claims.UserID <= 0 || claims.SessionID == "" {
		return nil, e.E(e.ErrInvalidToken)
	}
	session, err := uc.repo.FindSession(ctx, sessionKey(claims.SessionID))
	if err != nil {
		return nil, err
	}
	if session == nil || session.UserID != claims.UserID || !time.Now().Before(session.ExpiresAt) {
		return nil, e.E(e.ErrExpiredToken)
	}
	return session, nil
}
func (uc *AccountUsecase) Me(ctx context.Context, s *Session) (*v1.MeReply_Data, error) {
	a, err := uc.repo.GetAccount(ctx, s.UserID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, e.E(e.ErrUnauthorized)
	}
	return &v1.MeReply_Data{Account: &v1.AccountInfo{Id: a.ID, Username: a.Username}}, nil
}
func (uc *AccountUsecase) Logout(ctx context.Context, s *Session) (*v1.LogoutReply_Data, error) {
	if err := uc.repo.RevokeSession(ctx, s.ID); err != nil {
		return nil, err
	}
	return &v1.LogoutReply_Data{Revoked: true}, nil
}
