package data

import (
	"context"
	"encoding/json"
	"time"

	"dailyWins-service/internal/biz"
	e "dailyWins-service/internal/errors"
	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm/clause"
)

// 独立业务表不会修改模板 users 数据，迁移只增加本版所需表与索引。
type Account struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement:false"`
	Username     string `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:128;not null"`
	CreatedAt    time.Time
}

func (Account) TableName() string { return "dw_accounts" }

type LoginSession struct {
	ID        string    `gorm:"type:char(64);primaryKey"`
	UserID    uint64    `gorm:"index;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}

func (LoginSession) TableName() string { return "dw_sessions" }

type DailyTask struct {
	ID     string `gorm:"type:char(36);primaryKey"`
	UserID uint64 `gorm:"not null;index:idx_dw_owner_date,priority:1"`
	// CHAR 而非 time.Time：业务日期不参与任何时区转换。
	Date                 string `gorm:"type:char(10);not null;index:idx_dw_owner_date,priority:2"`
	Name                 string `gorm:"size:200;not null"`
	Note                 string `gorm:"type:text;not null"`
	Done                 bool   `gorm:"not null"`
	Deleted              bool   `gorm:"not null;default:false"`
	Version              int32  `gorm:"not null"`
	CreatedAt, UpdatedAt time.Time
}

func (DailyTask) TableName() string { return "dw_tasks" }

type TaskRequest struct {
	UserID      uint64 `gorm:"primaryKey;autoIncrement:false"`
	RequestID   string `gorm:"type:char(36);primaryKey"`
	Fingerprint string `gorm:"type:char(64);not null"`
	Result      string `gorm:"type:text;not null"`
	CreatedAt   time.Time
}

func (TaskRequest) TableName() string { return "dw_task_requests" }

type dailyRepo struct {
	data *Data
	log  *log.Helper
}

func NewTaskRepo(d *Data, logger log.Logger) biz.TaskRepo {
	return &dailyRepo{d, log.NewHelper(logger)}
}
func NewAccountRepo(d *Data, logger log.Logger) biz.AccountRepo {
	return &dailyRepo{d, log.NewHelper(logger)}
}

func (r *dailyRepo) LockOwner(ctx context.Context, u int32) error {
	var a Account
	err := r.data.DB(ctx).WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, uint64(u)).Error
	if isNotFound(err) {
		return e.E(e.ErrUnauthorized)
	}
	return r.dbError(err)
}
func (r *dailyRepo) GetTask(ctx context.Context, u int32, id string) (*biz.Task, error) {
	var t DailyTask
	err := r.data.DB(ctx).WithContext(ctx).Where("user_id = ? AND id = ?", uint64(u), id).First(&t).Error
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, r.dbError(err)
	}
	return taskDomain(t), nil
}
func (r *dailyRepo) ListTasks(ctx context.Context, u int32, from, to string) ([]*biz.Task, error) {
	var rows []DailyTask
	err := r.data.DB(ctx).WithContext(ctx).Where("user_id = ? AND date >= ? AND date <= ? AND deleted = ?", uint64(u), from, to, false).Order("date, created_at, id").Find(&rows).Error
	if err != nil {
		return nil, r.dbError(err)
	}
	result := make([]*biz.Task, 0, len(rows))
	for _, t := range rows {
		result = append(result, taskDomain(t))
	}
	return result, nil
}
func taskDomain(t DailyTask) *biz.Task {
	return &biz.Task{ID: t.ID, UserID: int32(t.UserID), Date: t.Date, Name: t.Name, Note: t.Note, Done: t.Done, Deleted: t.Deleted, Version: t.Version, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}
func (r *dailyRepo) SaveTask(ctx context.Context, t *biz.Task) error {
	db := r.data.DB(ctx).WithContext(ctx)
	model := DailyTask{ID: t.ID, UserID: uint64(t.UserID), Date: t.Date, Name: t.Name, Note: t.Note, Done: t.Done, Deleted: t.Deleted, Version: t.Version, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
	if t.Version == 1 {
		return r.dbError(db.Create(&model).Error)
	}
	// 所有 UPDATE 再次带 owner 和 version 条件，避免任何越权或意外覆盖。
	result := db.Model(&DailyTask{}).Where("id = ? AND user_id = ? AND version = ?", t.ID, uint64(t.UserID), t.Version-1).Select("Date", "Name", "Note", "Done", "Deleted", "Version", "UpdatedAt").Updates(&model)
	if result.Error != nil {
		return r.dbError(result.Error)
	}
	if result.RowsAffected != 1 {
		return e.E(e.ErrConflict)
	}
	return nil
}
func (r *dailyRepo) GetOperation(ctx context.Context, u int32, id string) (*biz.TaskOperation, error) {
	var row TaskRequest
	err := r.data.DB(ctx).WithContext(ctx).Where("user_id = ? AND request_id = ?", uint64(u), id).First(&row).Error
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, r.dbError(err)
	}
	var task biz.Task
	if err := json.Unmarshal([]byte(row.Result), &task); err != nil {
		return nil, e.E(e.ErrSerialization)
	}
	return &biz.TaskOperation{Fingerprint: row.Fingerprint, Task: &task}, nil
}
func (r *dailyRepo) SaveOperation(ctx context.Context, u int32, id string, op *biz.TaskOperation) error {
	raw, err := json.Marshal(op.Task)
	if err != nil {
		return e.E(e.ErrSerialization)
	}
	return r.dbError(r.data.DB(ctx).WithContext(ctx).Create(&TaskRequest{UserID: uint64(u), RequestID: id, Fingerprint: op.Fingerprint, Result: string(raw)}).Error)
}
func (r *dailyRepo) FindAccount(ctx context.Context, name string) (*biz.Account, error) {
	var a Account
	err := r.data.DB(ctx).WithContext(ctx).Where("username = ?", name).First(&a).Error
	return accountDomain(a, err)
}
func (r *dailyRepo) GetAccount(ctx context.Context, id int32) (*biz.Account, error) {
	var a Account
	err := r.data.DB(ctx).WithContext(ctx).First(&a, uint64(id)).Error
	return accountDomain(a, err)
}
func accountDomain(a Account, err error) (*biz.Account, error) {
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, e.E(e.ErrDBOperation)
	}
	return &biz.Account{ID: int32(a.ID), Username: a.Username, PasswordHash: a.PasswordHash}, nil
}
func (r *dailyRepo) CreateSession(ctx context.Context, s *biz.Session) error {
	return r.dbError(r.data.DB(ctx).WithContext(ctx).Create(&LoginSession{ID: s.ID, UserID: uint64(s.UserID), ExpiresAt: s.ExpiresAt}).Error)
}
func (r *dailyRepo) FindSession(ctx context.Context, id string) (*biz.Session, error) {
	var s LoginSession
	err := r.data.DB(ctx).WithContext(ctx).Where("id = ? AND expires_at > ?", id, time.Now().UTC()).First(&s).Error
	if isNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, r.dbError(err)
	}
	return &biz.Session{ID: s.ID, UserID: int32(s.UserID), ExpiresAt: s.ExpiresAt}, nil
}
func (r *dailyRepo) RevokeSession(ctx context.Context, id string) error {
	return r.dbError(r.data.DB(ctx).WithContext(ctx).Where("id = ?", id).Delete(&LoginSession{}).Error)
}
func (r *dailyRepo) dbError(err error) error {
	if err == nil {
		return nil
	}
	r.log.Errorw("component", "dailywins_repo", "message", "database operation failed")
	return e.E(e.ErrDBOperation)
}
