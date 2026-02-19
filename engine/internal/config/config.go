package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/three-body-engine/internal/domain"
)

// ProviderConfig defines how to launch a code agent provider process.
type ProviderConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// Config holds the engine's runtime configuration.
type Config struct {
	DBPath               string                    `json:"db_path"`
	Workspace            string                    `json:"workspace"`
	BudgetCapUSD         float64                   `json:"budget_cap_usd"`
	Providers            map[string]ProviderConfig `json:"providers"`
	CheckIntervalSec     int                       `json:"check_interval_sec"`
	HeartbeatMaxAge      int                       `json:"heartbeat_max_age"`
	MaxConcurrentWorkers int                       `json:"max_concurrent_workers"`
	ListenAddr           string                    `json:"listen_addr"`
	MaxRounds            int                       `json:"max_rounds"`
	RateLimitPerMinute   int                       `json:"rate_limit_per_minute"`
}

// Load reads a JSON config file, applies defaults, and validates.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config JSON: %w", err)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.CheckIntervalSec == 0 {
		c.CheckIntervalSec = 10
	}
	if c.MaxConcurrentWorkers == 0 {
		c.MaxConcurrentWorkers = 5
	}
	if c.ListenAddr == "" {
		c.ListenAddr = ":9800"
	}
	if c.MaxRounds == 0 {
		c.MaxRounds = 3
	}
	if c.RateLimitPerMinute == 0 {
		c.RateLimitPerMinute = 60
	}
	if c.HeartbeatMaxAge == 0 {
		c.HeartbeatMaxAge = 30
	}
}

func (c *Config) validate() error {
	var problems []string

	if c.DBPath == "" {
		problems = append(problems, "db_path is required")
	}
	if c.Workspace == "" {
		problems = append(problems, "workspace is required")
	}
	if c.BudgetCapUSD <= 0 {
		problems = append(problems, "budget_cap_usd must be positive")
	}
	if len(c.Providers) == 0 {
		problems = append(problems, "at least one provider is required")
	}

	if len(problems) > 0 {
		return &domain.EngineError{
			Code:    domain.ErrConfigInvalid.Code,
			Message: fmt.Sprintf("%s: %v", domain.ErrConfigInvalid.Message, problems),
		}
	}
	return nil
}
