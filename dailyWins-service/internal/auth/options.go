package auth

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	Secret  string
	TTL     time.Duration
	Secure  bool
	Origins []string
}

func NewOptions() (*Options, error) {
	secret := os.Getenv("AUTH_SECRET")
	if len(secret) < 32 || strings.Contains(strings.ToUpper(secret), "CHANGE_ME") {
		return nil, fmt.Errorf("AUTH_SECRET 必须为至少 32 字节的随机密钥")
	}
	ttl := 7 * 24 * time.Hour
	if raw := os.Getenv("AUTH_TTL_HOURS"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 720 {
			return nil, fmt.Errorf("AUTH_TTL_HOURS 须为 1–720")
		}
		ttl = time.Duration(n) * time.Hour
	}
	secure := true
	if raw := os.Getenv("COOKIE_SECURE"); raw != "" {
		var err error
		secure, err = strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("COOKIE_SECURE 必须为 true 或 false")
		}
	}
	var origins []string
	for _, o := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		u, err := url.Parse(o)
		if err != nil || u.Host == "" || strings.Contains(u.Host, "*") || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return nil, fmt.Errorf("ALLOWED_ORIGINS 必须为明确的 http(s) 来源，不含路径或通配符")
		}
		origins = append(origins, o)
	}
	return &Options{Secret: secret, TTL: ttl, Secure: secure, Origins: origins}, nil
}
