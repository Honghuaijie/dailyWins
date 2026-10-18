package data

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"dailyWins-service/internal/auth"
	"dailyWins-service/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gl "gorm.io/gorm/logger"
	"io"
)

// 只能显式指定测试库；所有测试数据在外层事务回滚，不修改个人账号。
func TestMySQLPersistence(t *testing.T) {
	dsn := os.Getenv("DAILYWINS_TEST_DSN")
	if dsn == "" {
		t.Skip("未设置 DAILYWINS_TEST_DSN，未验证真实 MySQL")
	}
	if !strings.Contains(dsn, "/dailywins_test_") {
		t.Fatal("只允许 dailywins_test_ 前缀的独立测试库")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: gl.Default.LogMode(gl.Silent)})
	if err != nil {
		t.Fatal("测试数据库连接失败")
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal("迁移不可重复", err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	logger := log.NewStdLogger(io.Discard)
	d, _, _ := NewData(nil, logger, tx)
	if err := InitializeAccount(context.Background(), tx, "test-owner", "test-password-only"); err != nil {
		t.Fatal(err)
	}
	uc := biz.NewTaskUsecase(NewTaskRepo(d, logger), d)
	ctx := context.Background()
	m := biz.TaskMutation{ID: uuid.NewString(), RequestID: uuid.NewString(), Name: "真实数据库", Date: "2024-02-29", Note: "持久化检查"}
	created, err := uc.CreateTask(ctx, 1, m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateTask(ctx, 1, m); err != nil {
		t.Fatal(err)
	}
	rows, err := uc.ListTasks(ctx, 1, "2024-02-01", "2024-02-29")
	if err != nil || len(rows.Items) != 1 || rows.Items[0].Date != "2024-02-29" {
		t.Fatal("日期或幂等失败", err)
	}
	if _, err := uc.UpdateTask(ctx, 2, biz.TaskMutation{ID: m.ID, RequestID: uuid.NewString(), Version: 1, Name: "越权"}); err == nil {
		t.Fatal("越权写入")
	}
	moved, err := uc.MoveTask(ctx, 1, biz.TaskMutation{ID: m.ID, RequestID: uuid.NewString(), Version: created.Task.Version, Date: "2027-01-01"})
	if err != nil || moved.Task.Note != "持久化检查" {
		t.Fatal(err)
	}
	if _, err := uc.DeleteTask(ctx, 1, biz.TaskMutation{ID: m.ID, RequestID: uuid.NewString(), Version: 2}); err != nil {
		t.Fatal(err)
	}
	restored, err := uc.RestoreTask(ctx, 1, biz.TaskMutation{ID: m.ID, RequestID: uuid.NewString(), Version: 3})
	if err != nil || restored.Task.Id != m.ID {
		t.Fatal(err)
	}
	account, err := NewAccountRepo(d, logger).FindAccount(ctx, "test-owner")
	if err != nil || !auth.CheckPassword(account.PasswordHash, "test-password-only") {
		t.Fatal("账号哈希失败")
	}
	sessions := NewAccountRepo(d, logger)
	s := &biz.Session{ID: strings.Repeat("a", 64), UserID: 1, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	if err := sessions.CreateSession(ctx, s); err != nil {
		t.Fatal(err)
	}
	if got, err := sessions.FindSession(ctx, s.ID); err != nil || got == nil {
		t.Fatal("会话持久化失败")
	}
	if err := sessions.RevokeSession(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := sessions.FindSession(ctx, s.ID); got != nil {
		t.Fatal("退出未失效")
	}
}
