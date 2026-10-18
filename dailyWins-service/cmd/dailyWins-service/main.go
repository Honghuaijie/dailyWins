package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dailyWins-service/internal/conf"
	"dailyWins-service/internal/configenv"
	"dailyWins-service/internal/data"
	"dailyWins-service/internal/logger"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	envconfig "github.com/go-kratos/kratos/v2/config/env"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

var (
	Name     = "dailyWins-service"
	Version  = "v0.1.0"
	flagconf string
	curEnv   string
	command  string
	id, _    = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path")
	flag.StringVar(&curEnv, "env", "local", "config env, eg: local")
	flag.StringVar(&command, "command", "serve", "serve, migrate, or init-account")
}

func main() {
	if err := setTimeZone("Asia/Shanghai"); err != nil {
		log.Fatalf("failed to set time zone: %v", err)
	}

	flag.Parse()
	if err := configenv.LoadLocal(flagconf); err != nil {
		log.Fatalf("failed to load local environment: %v", err)
	}
	cfg, err := loadConfig(flagconf, curEnv)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	defer cfg.Close()

	var bc conf.Bootstrap
	if err := cfg.Scan(&bc); err != nil {
		log.Fatalf("failed to scan config: %v", err)
	}
	if err := configenv.ValidateDatabase(bc.Data); err != nil {
		log.Fatalf("invalid database config: %v", err)
	}
	if command == "migrate" || command == "init-account" {
		db, err := data.NewDB(bc.Data)
		if err != nil {
			log.Fatal("数据库连接或迁移失败，请检查配置与数据库权限")
			return
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()
		if command == "init-account" {
			if err := data.InitializeAccount(context.Background(), db, os.Getenv("INIT_USERNAME"), os.Getenv("INIT_PASSWORD")); err != nil {
				log.Fatal(err)
				return
			}
		}
		log.Info("command completed: ", command)
		return
	}
	if command != "serve" {
		log.Fatal("unknown command")
		return
	}
	if addr := os.Getenv("HTTP_ADDR"); addr != "" {
		bc.Server.Http.Addr = addr
	}
	if addr := os.Getenv("GRPC_ADDR"); addr != "" {
		bc.Server.Grpc.Addr = addr
	}

	appLogger := logger.NewLogger(Name, bc.Logging)
	app, cleanup, err := wireApp(bc.Server, bc.Data, appLogger)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	defer cleanup()
	if err := app.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}
}

func setTimeZone(tz string) error {
	location, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}
	time.Local = location
	return nil
}

func loadConfig(basePath, env string) (config.Config, error) {
	sources := []config.Source{
		file.NewSource(filepath.Join(basePath, "config.yaml")),
	}
	if env != "" {
		sources = append(sources, file.NewSource(filepath.Join(basePath, fmt.Sprintf("config-%s.yaml", env))))
	}
	sources = append(sources, envconfig.NewSource())

	c := config.New(config.WithSource(sources...))
	if err := c.Load(); err != nil {
		return nil, err
	}
	return c, nil
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(gs, hs),
	)
}
