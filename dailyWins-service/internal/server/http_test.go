package server

import (
	"bytes"
	"context"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	v1 "dailyWins-service/api/all-pb-go/v1"
	"dailyWins-service/internal/biz"
	"dailyWins-service/internal/handler"
	"dailyWins-service/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	kh "github.com/go-kratos/kratos/v2/transport/http"
)

type fakeUserRepo struct {
	createFn func(context.Context, *biz.User) (*biz.User, error)
}

func (r fakeUserRepo) Create(ctx context.Context, user *biz.User) (*biz.User, error) {
	if r.createFn != nil {
		return r.createFn(ctx, user)
	}
	return nil, nil
}

func (fakeUserRepo) GetByID(context.Context, int32) (*biz.User, error) {
	return nil, nil
}

func (fakeUserRepo) List(context.Context) ([]*biz.User, error) {
	return nil, nil
}

func (fakeUserRepo) Update(context.Context, *biz.User) (*biz.User, error) {
	return nil, nil
}

func (fakeUserRepo) Delete(context.Context, int32) error {
	return nil
}

type fakeTx struct{}

func (fakeTx) ExecTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func newTestHTTPServer(repo fakeUserRepo) *httptest.Server {
	logger := log.NewStdLogger(io.Discard)
	uc := biz.NewUserUsecase(repo, fakeTx{}, logger)
	userSvc := service.NewUserService(uc)
	// 模板接口仅在该隔离测试夹具注册，生产服务不暴露示例 CRUD。
	srv := kh.NewServer(kh.ErrorEncoder(errorEncoder), kh.Middleware(requestLogMiddleware(logger)))
	v1.RegisterUserHTTPServer(srv, userSvc)
	srv.Route("").GET("/health", handler.HealthCheckHandler)
	srv.Route("").POST("/v1/users/avatarUpload", userSvc.AvatarUpload)
	return httptest.NewServer(srv)
}

func TestHTTPCreateUser_MissingNameOrEmail(t *testing.T) {
	ts := newTestHTTPServer(fakeUserRepo{})
	defer ts.Close()

	body := []byte(`{"name":"","email":"","phone":"123"}`)
	resp, err := stdhttp.Post(ts.URL+"/v1/users/create", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != stdhttp.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, stdhttp.StatusBadRequest)
	}
	if !strings.Contains(string(respBody), `"code":40001`) {
		t.Fatalf("expected error code 40001, got body %s", string(respBody))
	}
	if !strings.Contains(string(respBody), "name 和 email 不能为空") {
		t.Fatalf("expected message in body, got %s", string(respBody))
	}
}

func TestHTTPCreateUser_Success(t *testing.T) {
	now := time.Now()
	ts := newTestHTTPServer(fakeUserRepo{
		createFn: func(ctx context.Context, user *biz.User) (*biz.User, error) {
			return &biz.User{
				ID:        1,
				Name:      user.Name,
				Email:     user.Email,
				Phone:     user.Phone,
				CreatedAt: now,
				UpdatedAt: now,
			}, nil
		},
	})
	defer ts.Close()

	body := []byte(`{"name":"Alice","email":"alice@example.com","phone":"123"}`)
	resp, err := stdhttp.Post(ts.URL+"/v1/users/create", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, stdhttp.StatusOK)
	}
	if !strings.Contains(string(respBody), `"code":0`) {
		t.Fatalf("expected success code in body, got %s", string(respBody))
	}
	if !strings.Contains(string(respBody), `"message":"CreateUser success"`) {
		t.Fatalf("expected success message in body, got %s", string(respBody))
	}
	if !strings.Contains(string(respBody), `"name":"Alice"`) {
		t.Fatalf("expected response body to contain user name, got %s", string(respBody))
	}
}

func TestHTTPHealth_Success(t *testing.T) {
	ts := newTestHTTPServer(fakeUserRepo{})
	defer ts.Close()

	resp, err := stdhttp.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if resp.StatusCode != stdhttp.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", resp.StatusCode, stdhttp.StatusOK)
	}
	if !strings.Contains(string(respBody), `"status":"ok"`) {
		t.Fatalf("expected health response body, got %s", string(respBody))
	}
}
