package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONConfigRunOnStartDefault(t *testing.T) {
	cfg := defaultCLIConfig()
	loadTestJSONConfig(t, "{}", &cfg)

	if !cfg.runOnStart {
		t.Fatalf("run_on_start=%v, want true", cfg.runOnStart)
	}
}

func TestLoadJSONConfigRunOnStartOverride(t *testing.T) {
	cfg := defaultCLIConfig()
	loadTestJSONConfig(t, `{"run_on_start": false}`, &cfg)

	if cfg.runOnStart {
		t.Fatalf("run_on_start=%v, want false", cfg.runOnStart)
	}
}

func loadTestJSONConfig(t *testing.T, content string, cfg *cliConfig) {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := loadJSONConfig(configPath, cfg); err != nil {
		t.Fatalf("loadJSONConfig error: %v", err)
	}
}
