package service

import (
	"context"
	"io"
	"net/http"
	"strings"

	v1 "dailyWins-service/api/all-pb-go/v1"
	"dailyWins-service/internal/biz"
	httpErrors "dailyWins-service/internal/errors"

	httpCtx "github.com/go-kratos/kratos/v2/transport/http"
)

type AvatarUploadReply struct {
	Code    int32                      `json:"code"`
	Message string                     `json:"message"`
	Data    *biz.AvatarUploadReplyData `json:"data,omitempty"`
}

type UserService struct {
	v1.UnimplementedUserServer

	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

func (s *UserService) AvatarUpload(ctx httpCtx.Context) error {
	input, err := validateAvatarUploadRequest(ctx)
	if err != nil {
		return err
	}
	data, err := s.uc.AvatarUpload(ctx.Request().Context(), input)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, &AvatarUploadReply{
		Code:    0,
		Message: successMessage("AvatarUpload"),
		Data:    data,
	})
}

func (s *UserService) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.CreateUserReply, error) {
	input, err := validateCreateUserRequest(req)
	if err != nil {
		return nil, err
	}
	data, err := s.uc.CreateUser(ctx, input)
	if err != nil {
		return nil, err
	}
	return &v1.CreateUserReply{Code: 0, Message: successMessage("CreateUser"), Data: data}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserReply, error) {
	id, err := validateUserID(req.GetId())
	if err != nil {
		return nil, err
	}
	data, err := s.uc.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetUserReply{Code: 0, Message: successMessage("GetUser"), Data: data}, nil
}

func (s *UserService) ListUsers(ctx context.Context, _ *v1.ListUsersRequest) (*v1.ListUsersReply, error) {
	data, err := s.uc.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListUsersReply{
		Code:    0,
		Message: successMessage("ListUsers"),
		Data:    data,
	}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.UpdateUserReply, error) {
	input, err := validateUpdateUserRequest(req)
	if err != nil {
		return nil, err
	}
	data, err := s.uc.UpdateUser(ctx, input)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateUserReply{Code: 0, Message: successMessage("UpdateUser"), Data: data}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*v1.DeleteUserReply, error) {
	id, err := validateUserID(req.GetId())
	if err != nil {
		return nil, err
	}
	data, err := s.uc.DeleteUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v1.DeleteUserReply{
		Code:    0,
		Message: successMessage("DeleteUser"),
		Data:    data,
	}, nil
}

func successMessage(method string) string {
	return method + " success"
}

func validateCreateUserRequest(req *v1.CreateUserRequest) (*biz.CreateUserInput, error) {
	if req == nil {
		return nil, httpErrors.E(httpErrors.ErrInvalidRequestBody)
	}
	name := strings.TrimSpace(req.GetName())
	email := strings.TrimSpace(req.GetEmail())
	phone := strings.TrimSpace(req.GetPhone())
	if name == "" || email == "" {
		return nil, httpErrors.EWithMessage(httpErrors.ErrMissingRequiredField, "name 和 email 不能为空")
	}
	return &biz.CreateUserInput{
		Name:  name,
		Email: email,
		Phone: phone,
	}, nil
}

func validateUpdateUserRequest(req *v1.UpdateUserRequest) (*biz.UpdateUserInput, error) {
	if req == nil {
		return nil, httpErrors.E(httpErrors.ErrInvalidRequestBody)
	}
	id, err := validateUserID(req.GetId())
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.GetName())
	email := strings.TrimSpace(req.GetEmail())
	phone := strings.TrimSpace(req.GetPhone())
	if name == "" || email == "" {
		return nil, httpErrors.EWithMessage(httpErrors.ErrMissingRequiredField, "name 和 email 不能为空")
	}
	return &biz.UpdateUserInput{
		ID:    id,
		Name:  name,
		Email: email,
		Phone: phone,
	}, nil
}

func validateUserID(id int32) (int32, error) {
	if id <= 0 {
		return 0, httpErrors.E(httpErrors.ErrInvalidID)
	}
	return id, nil
}

func validateAvatarUploadRequest(ctx httpCtx.Context) (*biz.AvatarUploadInput, error) {
	file, header, err := ctx.Request().FormFile("photo")
	if err != nil {
		return nil, httpErrors.EWithMessage(httpErrors.ErrMissingRequiredField, "photo 文件不能为空")
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, httpErrors.EWithMessage(httpErrors.ErrInvalidRequestBody, "读取 photo 文件失败")
	}
	if len(fileBytes) == 0 {
		return nil, httpErrors.EWithMessage(httpErrors.ErrMissingRequiredField, "photo 文件不能为空")
	}
	if header == nil || strings.TrimSpace(header.Filename) == "" {
		return nil, httpErrors.EWithMessage(httpErrors.ErrInvalidParam, "文件名不能为空")
	}

	return &biz.AvatarUploadInput{
		FileBytes: fileBytes,
		FileName:  strings.TrimSpace(header.Filename),
	}, nil
}
