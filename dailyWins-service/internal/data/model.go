package data

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type ModelOpt struct {
	CreatedAt time.Time             `json:"createdAt" gorm:"not null;autoCreateTime"`
	CreatedBy string                `json:"createdBy" gorm:"not null;size:64;default:''"`
	UpdatedAt time.Time             `json:"updatedAt" gorm:"not null;autoUpdateTime"`
	UpdatedBy string                `json:"updatedBy" gorm:"not null;size:64;default:''"`
	Deleted   soft_delete.DeletedAt `json:"deleted" gorm:"softDelete:flag;not null;default:0"`
}

type User struct {
	ID uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	ModelOpt
	Name  string `json:"name" gorm:"size:128;not null"`
	Email string `json:"email" gorm:"size:128;not null;index"`
	Phone string `json:"phone" gorm:"size:32;default:''"`
}

func (User) TableName() string {
	return "users"
}
