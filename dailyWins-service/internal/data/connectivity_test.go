package data

import (
	"context"
	"database/sql"
	"net"
	"os"
	"testing"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// 只读连接探测：不迁移、不插入、不输出连接信息或密码。
func TestConfiguredDatabaseConnectivity(t *testing.T) {
	if os.Getenv("DAILYWINS_CHECK_DB") != "1" {
		t.Skip("显式开启才读取本地真实配置")
	}
	vars, err := godotenv.Read("../../configs/.env")
	if err != nil {
		t.Fatal("本地配置不可读取")
	}
	cfg := driver.NewConfig()
	cfg.User = vars["DB_USER"]
	cfg.Passwd = vars["DB_PASSWORD"]
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(vars["DB_HOST"], vars["DB_PORT"])
	cfg.DBName = vars["DB_NAME"]
	cfg.Timeout = 5 * time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal("无法建立连接")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal("现有配置数据库连接失败（详情已隐藏，避免暴露连接信息）")
	}
	t.Log("现有配置数据库只读连接检查通过")
}
