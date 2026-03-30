package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all runtime configuration for the flashclaw-im-channel service.
type Config struct {
	// Active environment: dev, test, prod (default: dev)
	Env string

	// HTTP server bind address, default "127.0.0.1"
	Bind string
	// HTTP server port, default 18790
	Port int
	// API key required in X-Api-Key header (required)
	APIKey string

	// Path to openclaw.json config file.
	// Default: $OPENCLAW_STATE_DIR/openclaw.json or ~/.openclaw/openclaw.json
	OpenclawConfigPath string
}

// fileConfig is the JSON structure for environment-specific config files
// located at configs/config-{env}.json (relative to the working directory,
// or under $FLASHCLAW_CONFIG_DIR).
//
// All fields are optional. Environment variables always take precedence.
type fileConfig struct {
	Bind               string `json:"bind"`
	Port               int    `json:"port"`
	APIKey             string `json:"apiKey"`
	OpenclawConfigPath string `json:"openclawConfigPath"`
}

// Load reads configuration with the following precedence (highest → lowest):
//  1. Environment variables
//  2. configs/config-{FLASHCLAW_ENV}.json (or $FLASHCLAW_CONFIG_DIR/config-{env}.json)
//  3. Built-in defaults
func Load() (*Config, error) {
	env := envOr("FLASHCLAW_ENV", "dev")

	cfg := &Config{
		Env:  env,
		Bind: "127.0.0.1",
		Port: 18790,
	}

	// Layer 1: config file (optional)
	if err := loadFromFile(cfg, env); err != nil {
		return nil, fmt.Errorf("config file error: %w", err)
	}

	// Layer 2: environment variables override file values
	if v := strings.TrimSpace(os.Getenv("FLASHCLAW_BIND")); v != "" {
		cfg.Bind = v
	}
	if raw := os.Getenv("FLASHCLAW_PORT"); raw != "" {
		p, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || p <= 0 {
			return nil, errors.New("FLASHCLAW_PORT must be a positive integer")
		}
		cfg.Port = p
	}
	if v := strings.TrimSpace(os.Getenv("FLASHCLAW_API_KEY")); v != "" {
		cfg.APIKey = v
	}
	if cfg.APIKey == "" {
		return nil, errors.New("FLASHCLAW_API_KEY is required (set via env var or config file apiKey field)")
	}

	if v := strings.TrimSpace(os.Getenv("OPENCLAW_CONFIG_PATH")); v != "" {
		cfg.OpenclawConfigPath = v
	}
	if cfg.OpenclawConfigPath == "" {
		cfg.OpenclawConfigPath = resolveOpenclawConfigPath()
	}

	return cfg, nil
}

// loadFromFile reads configs/config-{env}.json and applies non-zero values to cfg.
// The file is optional — if it does not exist, no error is returned.
func loadFromFile(cfg *Config, env string) error {
	configDir := envOr("FLASHCLAW_CONFIG_DIR", "configs")
	path := filepath.Join(configDir, fmt.Sprintf("config-%s.json", env))

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if fc.Bind != "" {
		cfg.Bind = fc.Bind
	}
	if fc.Port > 0 {
		cfg.Port = fc.Port
	}
	if fc.APIKey != "" {
		cfg.APIKey = fc.APIKey
	}
	if fc.OpenclawConfigPath != "" {
		cfg.OpenclawConfigPath = fc.OpenclawConfigPath
	}

	return nil
}

// resolveOpenclawConfigPath determines the path to openclaw.json.
// Priority: OPENCLAW_STATE_DIR > CLAWDBOT_STATE_DIR > ~/.openclaw
func resolveOpenclawConfigPath() string {
	if v := strings.TrimSpace(os.Getenv("OPENCLAW_STATE_DIR")); v != "" {
		return filepath.Join(v, "openclaw.json")
	}
	if v := strings.TrimSpace(os.Getenv("CLAWDBOT_STATE_DIR")); v != "" {
		return filepath.Join(v, "openclaw.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "openclaw.json")
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
