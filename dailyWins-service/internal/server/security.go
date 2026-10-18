package server

import (
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"dailyWins-service/internal/auth"
	e "dailyWins-service/internal/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func decodeJSON(r *http.Request, target interface{}) error {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return e.E(e.ErrInvalidRequestBody)
	}
	m, ok := target.(proto.Message)
	if !ok {
		return e.E(e.ErrInvalidRequestBody)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, m); err != nil {
		return e.EWithMessage(e.ErrInvalidRequestBody, "请求 JSON 无效，包含未知字段或字段类型错误")
	}
	return nil
}

func securityFilter(options *auth.Options) func(http.Handler) http.Handler {
	type bucket struct {
		count int
		until time.Time
	}
	var mu sync.Mutex
	buckets := map[string]bucket{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			origin := r.Header.Get("Origin")
			allowed := origin == ""
			if origin != "" {
				u, err := url.Parse(origin)
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				allowed = err == nil && u.Scheme == scheme && u.Host == r.Host && u.Path == "" && u.User == nil
				for _, o := range options.Origins {
					if o == origin {
						allowed = true
					}
				}
				if !allowed {
					e.EncodeHTTPError(w, r, e.E(e.ErrForbidden))
					return
				}
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Daily-Wins")
				w.WriteHeader(204)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/v1/") {
				// 不信任客户端伪造的 X-Forwarded-For；代理部署时在网关追加限流。
				ip, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					ip = r.RemoteAddr
				}
				now := time.Now()
				mu.Lock()
				for k, b := range buckets {
					if now.After(b.until) {
						delete(buckets, k)
					}
				}
				b := buckets[ip]
				limited := b.count >= 300 || (b.count == 0 && len(buckets) >= 10000)
				if !limited {
					if b.count == 0 {
						b.until = now.Add(time.Minute)
					}
					b.count++
					buckets[ip] = b
				}
				mu.Unlock()
				if limited {
					w.Header().Set("Retry-After", "60")
					e.EncodeHTTPError(w, r, e.E(e.ErrTooManyRequests))
					return
				}
			}
			if r.Method == http.MethodPost {
				// 自定义头让跨站表单无法携带有效写入请求；跨域 JS 仍须通过上方 allowlist。
				if r.Header.Get("X-Daily-Wins") != "1" {
					e.EncodeHTTPError(w, r, e.EWithMessage(e.ErrForbidden, "缺少请求保护头"))
					return
				}
				kind, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if err != nil || kind != "application/json" || r.URL.RawQuery != "" {
					e.EncodeHTTPError(w, r, e.E(e.ErrInvalidRequestBody))
					return
				}
				r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
			}
			next.ServeHTTP(w, r)
		})
	}
}
