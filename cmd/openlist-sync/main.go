package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"op-sync/internal/openlistsync"
)

type cliConfig struct {
	configPath  string
	baseURL     string
	tokenFile   string
	srcDir      string
	dstDir      string
	excludes    []string
	minSizeDiff int64
	logLevelStr string
	logLevel    openlistsync.LogLevel
	perPage     int
	timeout     time.Duration
	dryRun      bool
}

const bytesPerKiB int64 = 1024

type jsonConfig struct {
	BaseURL           *string   `json:"base_url"`
	TokenFile         *string   `json:"token_file"`
	SrcDir            *string   `json:"src"`
	DstDir            *string   `json:"dst"`
	Blacklist         *[]string `json:"blacklist"`
	MinSizeDiff       *int64    `json:"min_size_diff"`
	SizeDiffThreshold *int64    `json:"size_diff_threshold"` // backward compatible (bytes)
	LogLevel          *string   `json:"log_level"`
	PerPage           *int      `json:"per_page"`
	Timeout           *string   `json:"timeout"`
	DryRun            *bool     `json:"dry_run"`
}

func defaultCLIConfig() cliConfig {
	return cliConfig{
		configPath:  "config.json",
		baseURL:     "http://localhost:35244",
		tokenFile:   "token.txt",
		logLevelStr: "info",
		perPage:     openlistsync.DefaultPerPage,
		timeout:     30 * time.Second,
	}
}

func main() {
	cfg, err := parseFlags()
	if err != nil {
		exitWithErr(2, err)
	}

	token, err := readToken(cfg.tokenFile)
	if err != nil {
		exitWithErr(2, fmt.Errorf("read token failed: %w", err))
	}

	runCfg := openlistsync.Config{
		BaseURL:     cfg.baseURL,
		Token:       token,
		SrcDir:      cfg.srcDir,
		DstDir:      cfg.dstDir,
		Blacklist:   cfg.excludes,
		MinSizeDiff: cfg.minSizeDiff,
		PerPage:     cfg.perPage,
		Timeout:     cfg.timeout,
		DryRun:      cfg.dryRun,
		Logger:      openlistsync.NewLogger(os.Stdout, cfg.logLevel),
	}
	if err := openlistsync.Run(context.Background(), runCfg); err != nil {
		exitWithErr(1, err)
	}
}

func parseFlags() (cliConfig, error) {
	cfg := defaultCLIConfig()
	detectedConfigPath, err := detectConfigPath(os.Args[1:], cfg.configPath)
	if err != nil {
		return cliConfig{}, err
	}
	cfg.configPath = detectedConfigPath

	if err := loadJSONConfig(cfg.configPath, &cfg); err != nil {
		if !hasHelpFlag(os.Args[1:]) {
			return cliConfig{}, err
		}
	}

	flag.StringVar(&cfg.configPath, "config", cfg.configPath, "path to JSON config file")
	flag.StringVar(&cfg.baseURL, "base-url", cfg.baseURL, "OpenList base URL")
	flag.StringVar(&cfg.tokenFile, "token-file", cfg.tokenFile, "path to token file")
	flag.StringVar(&cfg.srcDir, "src", cfg.srcDir, "source directory path in OpenList")
	flag.StringVar(&cfg.dstDir, "dst", cfg.dstDir, "destination directory path in OpenList")
	flag.Func("exclude", "blacklist wildcard pattern, repeatable or comma-separated", func(v string) error {
		cfg.excludes = append(cfg.excludes, splitPatterns(v)...)
		return nil
	})
	flag.StringVar(&cfg.logLevelStr, "log-level", cfg.logLevelStr, "log level: debug, info, error")
	flag.IntVar(&cfg.perPage, "per-page", cfg.perPage, "list API page size")
	flag.Int64Var(&cfg.minSizeDiff, "min-size-diff", cfg.minSizeDiff, "copy only when src-dst size diff is >= this value (KiB)")
	flag.DurationVar(&cfg.timeout, "timeout", cfg.timeout, "HTTP timeout")
	flag.BoolVar(&cfg.dryRun, "dry-run", cfg.dryRun, "plan only, do not submit copy")
	flag.Parse()

	cfg.srcDir = strings.TrimSpace(cfg.srcDir)
	cfg.dstDir = strings.TrimSpace(cfg.dstDir)
	if cfg.srcDir == "" || cfg.dstDir == "" {
		return cliConfig{}, fmt.Errorf("both -src and -dst are required")
	}
	if cfg.perPage < 0 {
		return cliConfig{}, fmt.Errorf("-per-page must be >= 0")
	}
	if cfg.minSizeDiff < 0 {
		return cliConfig{}, fmt.Errorf("-min-size-diff must be >= 0")
	}
	lv, err := openlistsync.ParseLogLevel(cfg.logLevelStr)
	if err != nil {
		return cliConfig{}, err
	}
	cfg.logLevel = lv
	return cfg, nil
}

func detectConfigPath(args []string, defaultPath string) (string, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--config=") {
			path := strings.TrimSpace(strings.TrimPrefix(arg, "--config="))
			if path == "" {
				return "", fmt.Errorf("--config cannot be empty")
			}
			return path, nil
		}
		if strings.HasPrefix(arg, "-config=") {
			path := strings.TrimSpace(strings.TrimPrefix(arg, "-config="))
			if path == "" {
				return "", fmt.Errorf("-config cannot be empty")
			}
			return path, nil
		}
		if arg == "--config" || arg == "-config" {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", arg)
			}
			path := strings.TrimSpace(args[i+1])
			if path == "" {
				return "", fmt.Errorf("%s cannot be empty", arg)
			}
			return path, nil
		}
	}
	return defaultPath, nil
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func splitPatterns(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func loadJSONConfig(configPath string, cfg *cliConfig) error {
	b, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config file failed (%s): %w", configPath, err)
	}

	var jc jsonConfig
	if err := json.Unmarshal(b, &jc); err != nil {
		return fmt.Errorf("parse config file failed (%s): %w", configPath, err)
	}

	if jc.BaseURL != nil {
		cfg.baseURL = *jc.BaseURL
	}
	if jc.TokenFile != nil {
		cfg.tokenFile = *jc.TokenFile
	}
	if jc.SrcDir != nil {
		cfg.srcDir = *jc.SrcDir
	}
	if jc.DstDir != nil {
		cfg.dstDir = *jc.DstDir
	}
	if jc.Blacklist != nil {
		cfg.excludes = append([]string(nil), *jc.Blacklist...)
	}
	if jc.MinSizeDiff != nil {
		cfg.minSizeDiff = *jc.MinSizeDiff
	} else if jc.SizeDiffThreshold != nil {
		cfg.minSizeDiff = bytesToKiBCeil(*jc.SizeDiffThreshold)
	}
	if jc.LogLevel != nil {
		cfg.logLevelStr = *jc.LogLevel
	}
	if jc.PerPage != nil {
		cfg.perPage = *jc.PerPage
	}
	if jc.Timeout != nil {
		d, err := time.ParseDuration(strings.TrimSpace(*jc.Timeout))
		if err != nil {
			return fmt.Errorf("invalid timeout in config file (%s): %w", configPath, err)
		}
		cfg.timeout = d
	}
	if jc.DryRun != nil {
		cfg.dryRun = *jc.DryRun
	}
	return nil
}

func bytesToKiBCeil(v int64) int64 {
	if v <= 0 {
		return 0
	}
	kib := v / bytesPerKiB
	if v%bytesPerKiB != 0 {
		kib++
	}
	return kib
}

func readToken(tokenFile string) (string, error) {
	b, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(b))
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}
	return token, nil
}

func exitWithErr(code int, err error) {
	openlistsync.NewLogger(os.Stderr, openlistsync.LogLevelError).Errorf("%v", err)
	os.Exit(code)
}
