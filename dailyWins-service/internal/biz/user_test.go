package biz

import (
	"context"
	"io"
	"testing"

	httpErrors "dailyWins-service/internal/errors"

	"github.com/go-kratos/kratos/v2/log"
)

type fakeUserRepo struct {
	createFn  func(context.Context, *User) (*User, error)
	getByIDFn func(context.Context, int32) (*User, error)
	listFn    func(context.Context) ([]*User, error)
}

func (r fakeUserRepo) Create(ctx context.Context, user *User) (*User, error) {
	if r.createFn != nil {
		return r.createFn(ctx, user)
	}
	return nil, nil
}

func (r fakeUserRepo) GetByID(ctx context.Context, id int32) (*User, error) {
	if r.getByIDFn != nil {
		return r.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (r fakeUserRepo) List(ctx context.Context) ([]*User, error) {
	if r.listFn != nil {
		return r.listFn(ctx)
	}
	return nil, nil
}

func (fakeUserRepo) Update(context.Context, *User) (*User, error) {
	return nil, nil
}

func (fakeUserRepo) Delete(context.Context, int32) error {
	return nil
}

type fakeTx struct{}

func (fakeTx) ExecTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestUserUsecaseCreate_Success(t *testing.T) {
	repoCalled := false
	repo := fakeUserRepo{
		createFn: func(ctx context.Context, user *User) (*User, error) {
			repoCalled = true
			if user.Name != "Alice" {
				t.Fatalf("unexpected name passed to repo: got %q", user.Name)
			}
			if user.Email != "alice@example.com" {
				t.Fatalf("unexpected email passed to repo: got %q", user.Email)
			}
			if user.Phone != "123" {
				t.Fatalf("unexpected phone passed to repo: got %q", user.Phone)
			}
			return &User{
				ID:    1,
				Name:  user.Name,
				Email: user.Email,
				Phone: user.Phone,
			}, nil
		},
	}
	tx := fakeTx{}
	logger := log.NewStdLogger(io.Discard)
	uc := NewUserUsecase(repo, tx, logger)

	data, err := uc.CreateUser(context.Background(), &CreateUserInput{
		Name:  "Alice",
		Email: "alice@example.com",
		Phone: "123",
	})

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repoCalled {
		t.Fatal("expected repo.Create to be called")
	}
	if data == nil || data.User == nil {
		t.Fatalf("expected user data, got %+v", data)
	}
	if data.User.Id != 1 {
		t.Fatalf("unexpected id: got %d want %d", data.User.Id, 1)
	}
	if data.User.Name != "Alice" {
		t.Fatalf("unexpected name: got %q want %q", data.User.Name, "Alice")
	}
	if data.User.Email != "alice@example.com" {
		t.Fatalf("unexpected email: got %q want %q", data.User.Email, "alice@example.com")
	}
	if data.User.Phone != "123" {
		t.Fatalf("unexpected phone: got %q want %q", data.User.Phone, "123")
	}
}

func TestUserUsecaseGet_UserNotFound(t *testing.T) {
	repo := fakeUserRepo{
		getByIDFn: func(context.Context, int32) (*User, error) {
			return nil, nil
		},
	}
	tx := fakeTx{}
	logger := log.NewStdLogger(io.Discard)
	uc := NewUserUsecase(repo, tx, logger)

	data, err := uc.GetUser(context.Background(), 1)

	if data != nil {
		t.Fatalf("expected nil data, got %+v", data)
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	se := httpErrors.FromError(err)
	if se == nil {
		t.Fatal("expected structured error, got nil")
	}
	if se.Code != httpErrors.ErrUserNotFound.Code {
		t.Fatalf("unexpected code: got %d want %d", se.Code, httpErrors.ErrUserNotFound.Code)
	}
	if se.HttpCode != httpErrors.ErrUserNotFound.HTTPCode {
		t.Fatalf("unexpected http code: got %d want %d", se.HttpCode, httpErrors.ErrUserNotFound.HTTPCode)
	}
}
