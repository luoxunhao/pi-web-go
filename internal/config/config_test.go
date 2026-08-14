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

func TestLoadAutoFindsConfigToml(t *testing.T) {
	dir := t.TempDir()
	content := `
[server]
port = 3100
hostname = "localhost"

[web]
allowed_hosts = ["127.0.0.1"]
`
	cfgPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	// Change into dir and call Load("") — it should find config.toml automatically
	oldPwd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldPwd)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 3100 {
		t.Fatalf("port = %d, want 3100", cfg.Server.Port)
	}
	if cfg.Server.Hostname != "localhost" {
		t.Fatalf("hostname = %q, want localhost", cfg.Server.Hostname)
	}
	if len(cfg.Web.AllowedHosts) != 1 || cfg.Web.AllowedHosts[0] != "127.0.0.1" {
		t.Fatalf("allowed_hosts = %#v", cfg.Web.AllowedHosts)
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

func TestPigoPortEnvUpdatesBaseURL(t *testing.T) {
	t.Setenv("PIGO_PORT", "14096")
	t.Setenv("PIGO_HOST", "127.0.0.1")
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Pigo.BaseURL != "http://127.0.0.1:14096" {
		t.Fatalf("baseURL = %q, want http://127.0.0.1:14096", cfg.Pigo.BaseURL)
	}
}
