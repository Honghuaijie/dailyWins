package data

import (
	"context"
	"net"
	"time"

	"dailyWins-service/internal/biz"
	"dailyWins-service/internal/conf"
	httpErrors "dailyWins-service/internal/errors"

	"github.com/go-kratos/kratos/v2/log"
	driver "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewDB, NewTransaction, NewUserRepo, NewTaskRepo, NewAccountRepo)

type Data struct {
	log *log.Helper
	db  *gorm.DB
}

func NewData(c *conf.Data, logger log.Logger, db *gorm.DB) (*Data, func(), error) {
	helper := log.NewHelper(logger)
	cleanup := func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		helper.Info("closing data resources")
	}
	return &Data{
		log: helper,
		db:  db,
	}, cleanup, nil
}

func NewDB(c *conf.Data) (*gorm.DB, error) {
	if c == nil || c.Database == nil {
		return nil, httpErrors.EWithMessage(httpErrors.ErrConfigInvalid, "数据库配置不能为空")
	}

	location, err := time.LoadLocation(c.Database.Loc)
	if err != nil {
		return nil, httpErrors.EWithMessage(httpErrors.ErrConfigInvalid, "数据库时区配置无效")
	}
	// 使用已有 MySQL 驱动编码 DSN，正确处理密码/库名中的特殊字符。
	cfg := driver.NewConfig()
	cfg.User = c.Database.Username
	cfg.Passwd = c.Database.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(c.Database.Host, c.Database.Port)
	cfg.DBName = c.Database.Database
	cfg.ParseTime = c.Database.ParseTime
	cfg.Loc = location
	cfg.Params = map[string]string{"charset": c.Database.Charset}
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 15 * time.Second
	cfg.WriteTimeout = 15 * time.Second

	// SQL 日志可能包含账号哈希和会话标识，统一由仓储输出脱敏错误日志。
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: gormlog.Default.LogMode(gormlog.Silent), NowFunc: func() time.Time { return time.Now().UTC() }})
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

type contextTxKey struct{}

func NewTransaction(d *Data) biz.Transaction {
	return d
}

func (d *Data) ExecTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, contextTxKey{}, tx)
		return fn(txCtx)
	})
}

func (d *Data) DB(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB)
	if ok {
		return tx
	}
	return d.db
}
