package main

import (
	"os"
	"path/filepath"
	"testing"

	"dailyWins-service/internal/conf"
)

func TestLoadConfigResolvesProcessEnvironment(t *testing.T) {
	t.Setenv("DB_PASSWORD", "from-process")
	configDir := t.TempDir()
	content := []byte("data:\n  database:\n    password: \"${DB_PASSWORD}\"\n")
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), content, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := loadConfig(configDir, "")
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	defer cfg.Close()

	var bootstrap conf.Bootstrap
	if err := cfg.Scan(&bootstrap); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if got := bootstrap.Data.Database.Password; got != "from-process" {
		t.Fatalf("database password = %q, want process environment value", got)
	}
}
