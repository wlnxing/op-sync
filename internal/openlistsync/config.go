package openlistsync

import (
	"fmt"
	"strings"
	"time"
)

const (
	DefaultPerPage = 0
	defaultTimeout = 30 * time.Second
)

type Config struct {
	BaseURL string
	Token   string
	SrcDir  string
	DstDir  string
	PerPage int
	Timeout time.Duration
	DryRun  bool
	Logger  *Logger
}

func normalizeConfig(cfg Config) (Config, error) {
	cfg.Token = strings.TrimSpace(cfg.Token)
	if cfg.Token == "" {
		return Config{}, fmt.Errorf("token is empty")
	}

	cfg.SrcDir = strings.TrimSpace(cfg.SrcDir)
	cfg.DstDir = strings.TrimSpace(cfg.DstDir)
	if cfg.SrcDir == "" || cfg.DstDir == "" {
		return Config{}, fmt.Errorf("both src and dst are required")
	}

	if cfg.PerPage < 0 {
		return Config{}, fmt.Errorf("per_page must be >= 0")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.Logger == nil {
		cfg.Logger = NewLogger(nil, LogLevelError)
	}

	cfg.BaseURL = normalizeBaseURL(cfg.BaseURL)
	cfg.SrcDir = normalizeOLPath(cfg.SrcDir)
	cfg.DstDir = normalizeOLPath(cfg.DstDir)
	return cfg, nil
}
