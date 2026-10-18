package data

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"dailyWins-service/internal/auth"
	"gorm.io/gorm"
)

// 沿用项目 AutoMigrate；可重复执行，不删除表、列或历史数据。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Account{}, &LoginSession{}, &DailyTask{}, &TaskRequest{})
}

// 固定 ID 确保只能初始化一个个人账号；存在即报错，绝不覆盖密码。
func InitializeAccount(ctx context.Context, db *gorm.DB, username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || utf8.RuneCountInString(username) > 64 {
		return fmt.Errorf("账号名须为 1–64 字")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if err := db.WithContext(ctx).Create(&Account{ID: 1, Username: username, PasswordHash: hash}).Error; err != nil {
		return fmt.Errorf("账号初始化失败：可能已存在；不会覆盖已有账号")
	}
	return nil
}
