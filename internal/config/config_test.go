package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != DefaultPort {
		t.Fatalf("port = %d, want %d", cfg.Server.Port, DefaultPort)
	}
	if cfg.Pigo.BaseURL != "http://127.0.0.1:4096" {
		t.Fatalf("baseURL = %q", cfg.Pigo.BaseURL)
	}
}

func TestLoadFileAndEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
[server]
port = 3000

[pigo]
base_url = "http://localhost:5000"

[filesystem]
allowed_roots = ["C:/work"]
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PI_WEB_GO_PORT", "3100")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 3100 {
		t.Fatalf("port = %d, want env override 3100", cfg.Server.Port)
	}
	if cfg.Pigo.BaseURL != "http://localhost:5000" {
		t.Fatalf("baseURL = %q", cfg.Pigo.BaseURL)
	}
	if len(cfg.Filesystem.AllowedRoots) != 1 || cfg.Filesystem.AllowedRoots[0] != "C:/work" {
		t.Fatalf("allowed roots = %#v", cfg.Filesystem.AllowedRoots)
	}
}
