package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	DefaultPort     = 30141
	DefaultHostname = "127.0.0.1"
	DefaultPigoPort = 4096
	DefaultPigoHost = "127.0.0.1"
)

// Config is the pi-web-go server configuration. Agent-related settings are
// intentionally not stored here: they are owned by pigo and read through its
// HTTP API.
type Config struct {
	Server     ServerConfig     `toml:"server"`
	Pigo       PigoConfig       `toml:"pigo"`
	Filesystem FilesystemConfig `toml:"filesystem"`
	Web        WebConfig        `toml:"web"`
}

type ServerConfig struct {
	Port     int    `toml:"port"`
	Hostname string `toml:"hostname"`
}

type PigoConfig struct {
	Command   string   `toml:"command"`
	Args      []string `toml:"args"`
	BaseURL   string   `toml:"base_url"`
	Host      string   `toml:"host"`
	Port      int      `toml:"port"`
	Password  string   `toml:"password"`
	AutoStart bool     `toml:"auto_start"`
}

type FilesystemConfig struct {
	AllowedRoots []string `toml:"allowed_roots"`
}

type WebConfig struct {
	Password     string   `toml:"password"`
	AllowedHosts []string `toml:"allowed_hosts"`
	FrontendDir  string   `toml:"frontend_dir"`
}

// Default returns the built-in defaults. Environment variables are layered on
// top by Load.
func Default() Config {
	return Config{
		Server: ServerConfig{
			Port:     DefaultPort,
			Hostname: DefaultHostname,
		},
		Pigo: PigoConfig{
			Command:   "pigo",
			Args:      []string{"serve"},
			BaseURL:   fmt.Sprintf("http://%s:%d", DefaultPigoHost, DefaultPigoPort),
			Host:      DefaultPigoHost,
			Port:      DefaultPigoPort,
			AutoStart: true,
		},
		Web: WebConfig{
			FrontendDir: "frontend/dist",
		},
	}
}

// Load reads config.toml when present, applies environment overrides, and
// validates the result. When path is empty, it looks for config.toml in the
// current working directory before falling back to built-in defaults.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		if _, err := os.Stat("config.toml"); err == nil {
			path = "config.toml"
		}
	}
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if _, err := toml.DecodeFile(path, &cfg); err != nil {
				return nil, fmt.Errorf("parse config %s: %w", path, err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat config %s: %w", path, err)
		}
	}
	applyEnv(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("PI_WEB_GO_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = n
		}
	}
	if v := os.Getenv("PI_WEB_GO_HOSTNAME"); v != "" {
		cfg.Server.Hostname = v
	}
	if v := os.Getenv("PIGO_BASE_URL"); v != "" {
		cfg.Pigo.BaseURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("PIGO_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Pigo.Port = n
			cfg.Pigo.BaseURL = fmt.Sprintf("http://%s:%d", cfg.Pigo.Host, cfg.Pigo.Port)
		}
	}
	if v := os.Getenv("PIGO_HOST"); v != "" {
		cfg.Pigo.Host = v
		cfg.Pigo.BaseURL = fmt.Sprintf("http://%s:%d", cfg.Pigo.Host, cfg.Pigo.Port)
	}
	if v := os.Getenv("PIGO_PASSWORD"); v != "" {
		cfg.Pigo.Password = v
	}
	if v := os.Getenv("PIGO_COMMAND"); v != "" {
		cfg.Pigo.Command = v
	}
	if v := os.Getenv("PI_WEB_GO_PASSWORD"); v != "" {
		cfg.Web.Password = v
	}
	if v := os.Getenv("PI_WEB_GO_ALLOWED_HOSTS"); v != "" {
		cfg.Web.AllowedHosts = strings.Split(v, ",")
	}
	if v := os.Getenv("PI_WEB_GO_FRONTEND_DIR"); v != "" {
		cfg.Web.FrontendDir = v
	}
}

func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server.port: %d", cfg.Server.Port)
	}
	if cfg.Pigo.Port <= 0 || cfg.Pigo.Port > 65535 {
		return fmt.Errorf("invalid pigo.port: %d", cfg.Pigo.Port)
	}
	if cfg.Pigo.BaseURL == "" {
		cfg.Pigo.BaseURL = fmt.Sprintf("http://%s:%d", cfg.Pigo.Host, cfg.Pigo.Port)
	}
	if cfg.Web.FrontendDir == "" {
		cfg.Web.FrontendDir = "frontend/dist"
	}
	if !filepath.IsAbs(cfg.Web.FrontendDir) {
		abs, err := filepath.Abs(cfg.Web.FrontendDir)
		if err != nil {
			return fmt.Errorf("resolve frontend_dir: %w", err)
		}
		cfg.Web.FrontendDir = abs
	}
	return nil
}
