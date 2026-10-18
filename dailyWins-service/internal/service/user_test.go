package service

import (
	"context"
	"io"
	"testing"
	"time"

	v1 "dailyWins-service/api/all-pb-go/v1"
	"dailyWins-service/internal/biz"
	httpErrors "dailyWins-service/internal/errors"

	"github.com/go-kratos/kratos/v2/log"
)

type fakeUserRepo struct {
	createFn func(context.Context, *biz.User) (*biz.User, error)
	listFn   func(context.Context) ([]*biz.User, error)
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

func (r fakeUserRepo) List(ctx context.Context) ([]*biz.User, error) {
	if r.listFn != nil {
		return r.listFn(ctx)
	}
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

func newTestUserService(repo fakeUserRepo) *UserService {
	logger := log.NewStdLogger(io.Discard)
	uc := biz.NewUserUsecase(repo, fakeTx{}, logger)
	return NewUserService(uc)
}

func TestUserServiceCreateUser_ReturnsBizError(t *testing.T) {
	svc := newTestUserService(fakeUserRepo{
		createFn: func(context.Context, *biz.User) (*biz.User, error) {
			return nil, httpErrors.E(httpErrors.ErrDBOperation)
		},
	})

	reply, err := svc.CreateUser(context.Background(), &v1.CreateUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Phone: "123",
	})

	if reply != nil {
		t.Fatalf("expected nil reply, got %+v", reply)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	se := httpErrors.FromError(err)
	if se == nil {
		t.Fatal("expected structured error, got nil")
	}
	if se.Code != httpErrors.ErrDBOperation.Code {
		t.Fatalf("unexpected code: got %d want %d", se.Code, httpErrors.ErrDBOperation.Code)
	}
}

func TestUserServiceCreateUser_InvalidRequest(t *testing.T) {
	svc := newTestUserService(fakeUserRepo{})

	reply, err := svc.CreateUser(context.Background(), &v1.CreateUserRequest{
		Name:  "",
		Email: "",
		Phone: "123",
	})

	if reply != nil {
		t.Fatalf("expected nil reply, got %+v", reply)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	se := httpErrors.FromError(err)
	if se == nil {
		t.Fatal("expected structured error, got nil")
	}
	if se.Code != httpErrors.ErrMissingRequiredField.Code {
		t.Fatalf("unexpected code: got %d want %d", se.Code, httpErrors.ErrMissingRequiredField.Code)
	}
}

func TestUserServiceCreateUser_Success(t *testing.T) {
	now := time.Now()
	svc := newTestUserService(fakeUserRepo{
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

	reply, err := svc.CreateUser(context.Background(), &v1.CreateUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Phone: "123",
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if reply == nil || reply.Data == nil || reply.Data.User == nil {
		t.Fatalf("expected user reply, got %+v", reply)
	}
	if reply.Code != 0 {
		t.Fatalf("unexpected code: got %d want %d", reply.Code, 0)
	}
	if reply.Message != successMessage("CreateUser") {
		t.Fatalf("unexpected message: got %q want %q", reply.Message, successMessage("CreateUser"))
	}
	if reply.Data.User.Name != "Alice" {
		t.Fatalf("unexpected name: got %q want %q", reply.Data.User.Name, "Alice")
	}
	if reply.Data.User.Email != "alice@example.com" {
		t.Fatalf("unexpected email: got %q want %q", reply.Data.User.Email, "alice@example.com")
	}
	if reply.Data.User.Phone != "123" {
		t.Fatalf("unexpected phone: got %q want %q", reply.Data.User.Phone, "123")
	}
}

func TestUserServiceListUsers_Success(t *testing.T) {
	now := time.Now()
	svc := newTestUserService(fakeUserRepo{
		listFn: func(context.Context) ([]*biz.User, error) {
			return []*biz.User{
				{ID: 1, Name: "Alice", Email: "alice@example.com", Phone: "123", CreatedAt: now, UpdatedAt: now},
				{ID: 2, Name: "Bob", Email: "bob@example.com", Phone: "456", CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	})

	reply, err := svc.ListUsers(context.Background(), &v1.ListUsersRequest{})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if reply == nil {
		t.Fatal("expected reply, got nil")
	}
	if reply.Code != 0 {
		t.Fatalf("unexpected code: got %d want %d", reply.Code, 0)
	}
	if reply.Message != successMessage("ListUsers") {
		t.Fatalf("unexpected message: got %q want %q", reply.Message, successMessage("ListUsers"))
	}
	if reply.Data == nil {
		t.Fatal("expected data, got nil")
	}
	if reply.Data.Total != 2 {
		t.Fatalf("unexpected total: got %d want %d", reply.Data.Total, 2)
	}
	if len(reply.Data.Items) != 2 {
		t.Fatalf("unexpected items length: got %d want %d", len(reply.Data.Items), 2)
	}
	if reply.Data.Items[0].Name != "Alice" {
		t.Fatalf("unexpected first name: got %q want %q", reply.Data.Items[0].Name, "Alice")
	}
	if reply.Data.Items[1].Name != "Bob" {
		t.Fatalf("unexpected second name: got %q want %q", reply.Data.Items[1].Name, "Bob")
	}
}
