package biz

import (
	"context"
	"time"

	v1 "dailyWins-service/api/all-pb-go/v1"
	httpErrors "dailyWins-service/internal/errors"

	"github.com/go-kratos/kratos/v2/log"
)

type CreateUserInput struct {
	Name  string
	Email string
	Phone string
}

type UpdateUserInput struct {
	ID    int32
	Name  string
	Email string
	Phone string
}

type AvatarUploadInput struct {
	FileBytes []byte
	FileName  string
}

type User struct {
	ID        int32
	Name      string
	Email     string
	Phone     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepo interface {
	Create(context.Context, *User) (*User, error)
	GetByID(context.Context, int32) (*User, error)
	List(context.Context) ([]*User, error)
	Update(context.Context, *User) (*User, error)
	Delete(context.Context, int32) error
}

type UserUsecase struct {
	repo UserRepo
	tx   Transaction
	log  *log.Helper
}

func NewUserUsecase(repo UserRepo, tx Transaction, logger log.Logger) *UserUsecase {
	return &UserUsecase{
		repo: repo,
		tx:   tx,
		log:  log.NewHelper(logger),
	}
}

type AvatarUploadReplyData struct {
	Url string
}

func (uc *UserUsecase) AvatarUpload(ctx context.Context, input *AvatarUploadInput) (*AvatarUploadReplyData, error) {
	_ = ctx
	_ = input
	// 示例上传接口只演示标准分层写法，具体上传实现由实际业务补充。
	return &AvatarUploadReplyData{
		Url: "",
	}, nil
}

func (uc *UserUsecase) CreateUser(ctx context.Context, input *CreateUserInput) (*v1.CreateUserReply_Data, error) {
	user, err := uc.repo.Create(ctx, &User{
		Name:  input.Name,
		Email: input.Email,
		Phone: input.Phone,
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreateUserReply_Data{User: toUserInfo(user)}, nil
}

func (uc *UserUsecase) GetUser(ctx context.Context, id int32) (*v1.GetUserReply_Data, error) {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, httpErrors.E(httpErrors.ErrUserNotFound)
	}
	return &v1.GetUserReply_Data{User: toUserInfo(user)}, nil
}

func (uc *UserUsecase) ListUsers(ctx context.Context) (*v1.ListUsersReply_Data, error) {
	users, err := uc.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.UserInfo, 0, len(users))
	for _, user := range users {
		items = append(items, toUserInfo(user))
	}
	return &v1.ListUsersReply_Data{
		Items: items,
		Total: int32(len(items)),
	}, nil
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, input *UpdateUserInput) (*v1.UpdateUserReply_Data, error) {
	var updated *v1.UpdateUserReply_Data
	err := uc.tx.ExecTx(ctx, func(txCtx context.Context) error {
		existing, err := uc.repo.GetByID(txCtx, input.ID)
		if err != nil {
			return err
		}
		if existing == nil {
			return httpErrors.E(httpErrors.ErrUserNotFound)
		}
		existing.Name = input.Name
		existing.Email = input.Email
		existing.Phone = input.Phone
		user, err := uc.repo.Update(txCtx, existing)
		if err != nil {
			return err
		}
		updated = &v1.UpdateUserReply_Data{User: toUserInfo(user)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (uc *UserUsecase) DeleteUser(ctx context.Context, id int32) (*v1.DeleteUserReply_Data, error) {
	var data *v1.DeleteUserReply_Data
	err := uc.tx.ExecTx(ctx, func(txCtx context.Context) error {
		existing, err := uc.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if existing == nil {
			return httpErrors.E(httpErrors.ErrUserNotFound)
		}
		if err := uc.repo.Delete(txCtx, id); err != nil {
			return err
		}
		data = &v1.DeleteUserReply_Data{Id: id}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func toUserInfo(user *User) *v1.UserInfo {
	if user == nil {
		return nil
	}
	return &v1.UserInfo{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Phone:     user.Phone,
		CreatedAt: formatTime(user.CreatedAt),
		UpdatedAt: formatTime(user.UpdatedAt),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(time.Local).Format(time.RFC3339)
}
