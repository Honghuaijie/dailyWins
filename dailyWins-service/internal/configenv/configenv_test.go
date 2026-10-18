package configenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dailyWins-service/internal/conf"
)

func TestLoadLocalDoesNotOverrideExistingEnvironment(t *testing.T) {
	t.Setenv("DB_PASSWORD", "from-process")
	configDir := t.TempDir()
	envPath := filepath.Join(configDir, ".env")
	if err := os.WriteFile(envPath, []byte("DB_PASSWORD=from-file\nDB_NAME=from-file\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	if err := LoadLocal(configDir); err != nil {
		t.Fatalf("LoadLocal() error = %v", err)
	}
	if got := os.Getenv("DB_PASSWORD"); got != "from-process" {
		t.Fatalf("DB_PASSWORD = %q, want process value", got)
	}
	if got := os.Getenv("DB_NAME"); got != "from-file" {
		t.Fatalf("DB_NAME = %q, want dotenv value", got)
	}
}

func TestLoadLocalAllowsMissingFile(t *testing.T) {
	if err := LoadLocal(t.TempDir()); err != nil {
		t.Fatalf("LoadLocal() error = %v, want nil", err)
	}
}

func TestValidateDatabaseReportsNamesWithoutValues(t *testing.T) {
	const sentinel = "must-not-appear-in-error"
	err := ValidateDatabase(&conf.Data{Database: &conf.Data_Database{
		Host:     sentinel,
		Port:     "3306",
		Username: "",
		Password: "",
		Database: "app_db",
	}})
	if err == nil {
		t.Fatal("ValidateDatabase() error = nil, want missing variable error")
	}
	message := err.Error()
	for _, name := range []string{"DB_USER", "DB_PASSWORD"} {
		if !strings.Contains(message, name) {
			t.Fatalf("error %q does not contain %s", message, name)
		}
	}
	if strings.Contains(message, sentinel) {
		t.Fatalf("error leaked configuration value: %q", message)
	}
}

func TestValidateDatabaseAcceptsCompleteConfig(t *testing.T) {
	err := ValidateDatabase(&conf.Data{Database: &conf.Data_Database{
		Host:     "mysql",
		Port:     "3306",
		Username: "app_user",
		Password: "secret",
		Database: "app_db",
	}})
	if err != nil {
		t.Fatalf("ValidateDatabase() error = %v, want nil", err)
	}
}
