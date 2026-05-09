package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create a temporary directory for config files
	err := os.MkdirAll("config", 0755)
	if err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	defer os.RemoveAll("config")

	t.Run("success", func(t *testing.T) {
		content := `{
			"server": {
				"port": 9000,
				"read_timeout_seconds": 10,
				"write_timeout_seconds": 10,
				"idle_timeout_seconds": 120
			}
		}`
		err := os.WriteFile("config/config.test.json", []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}

		cfg, err := Load("test")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if cfg.Server.Port != 9000 {
			t.Errorf("expected port 9000, got %d", cfg.Server.Port)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := Load("non-existent")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		err := os.WriteFile("config/config.invalid.json", []byte("{ invalid"), 0644)
		if err != nil {
			t.Fatalf("failed to create invalid config file: %v", err)
		}

		_, err = Load("invalid")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
