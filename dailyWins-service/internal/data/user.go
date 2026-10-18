package data

import (
	"context"
	"errors"

	"dailyWins-service/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

type UserRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &UserRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *UserRepo) Create(ctx context.Context, user *biz.User) (*biz.User, error) {
	model := &User{
		Name:  user.Name,
		Email: user.Email,
		Phone: user.Phone,
	}
	if err := r.data.DB(ctx).WithContext(ctx).Create(model).Error; err != nil {
		return nil, err
	}
	return toBizUser(model), nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int32) (*biz.User, error) {
	var model User
	err := r.data.DB(ctx).WithContext(ctx).First(&model, uint64(id)).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toBizUser(&model), nil
}

func (r *UserRepo) List(ctx context.Context) ([]*biz.User, error) {
	var models []*User
	if err := r.data.DB(ctx).WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]*biz.User, 0, len(models))
	for _, model := range models {
		items = append(items, toBizUser(model))
	}
	return items, nil
}

func (r *UserRepo) Update(ctx context.Context, user *biz.User) (*biz.User, error) {
	db := r.data.DB(ctx).WithContext(ctx)
	var model User
	err := db.First(&model, user.ID).Error
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	model.Name = user.Name
	model.Email = user.Email
	model.Phone = user.Phone
	if err := db.Save(&model).Error; err != nil {
		return nil, err
	}
	return toBizUser(&model), nil
}

func (r *UserRepo) Delete(ctx context.Context, id int32) error {
	return r.data.DB(ctx).WithContext(ctx).Delete(&User{}, uint64(id)).Error
}

func toBizUser(model *User) *biz.User {
	if model == nil {
		return nil
	}
	return &biz.User{
		ID:        int32(model.ID),
		Name:      model.Name,
		Email:     model.Email,
		Phone:     model.Phone,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
