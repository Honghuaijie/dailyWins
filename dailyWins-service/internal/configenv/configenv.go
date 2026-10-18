package configenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dailyWins-service/internal/conf"

	"github.com/joho/godotenv"
)

// LoadLocal loads optional local variables without overriding values already
// supplied by the operating system or container runtime.
func LoadLocal(basePath string) error {
	err := godotenv.Load(filepath.Join(basePath, ".env"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// ValidateDatabase reports missing environment-backed database fields without
// including any configuration values in the error.
func ValidateDatabase(data *conf.Data) error {
	missing := make([]string, 0, 5)
	if data == nil || data.Database == nil {
		missing = append(missing, "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME")
	} else {
		database := data.Database
		fields := []struct {
			name  string
			value string
		}{
			{name: "DB_HOST", value: database.Host},
			{name: "DB_PORT", value: database.Port},
			{name: "DB_USER", value: database.Username},
			{name: "DB_PASSWORD", value: database.Password},
			{name: "DB_NAME", value: database.Database},
		}
		for _, field := range fields {
			if strings.TrimSpace(field.value) == "" {
				missing = append(missing, field.name)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}
