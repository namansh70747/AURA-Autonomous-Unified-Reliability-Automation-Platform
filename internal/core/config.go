// Package core provides configuration management for AURA
package core

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all AURA configuration with validation
type Config struct {
	App struct {
		Name     string `yaml:"name"`
		Version  string `yaml:"version"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"app"`

	Database struct {
		Host           string `yaml:"host"`
		Port           int    `yaml:"port"`
		User           string `yaml:"user"`
		Password       string `yaml:"password"`
		DBName         string `yaml:"dbname"`
		MaxConnections int    `yaml:"max_connections"`
	} `yaml:"database"`

	Prometheus struct {
		URL            string `yaml:"url"`
		ScrapeInterval string `yaml:"scrape_interval"`
	} `yaml:"prometheus"`

	Kubernetes struct {
		Enabled         bool   `yaml:"enabled"`
		Namespace       string `yaml:"namespace"`
		MetricsInterval string `yaml:"metrics_interval"`
	} `yaml:"kubernetes"`

	Observer struct {
		MetricsInterval string `yaml:"metrics_interval"`
		RetentionPeriod string `yaml:"retention_period"`
	} `yaml:"observer"`

	Analyzer struct {
		CPUThreshold       float64 `yaml:"cpu_threshold"`
		MemoryThreshold    float64 `yaml:"memory_threshold"`
		ErrorRateThreshold float64 `yaml:"error_rate_threshold"`
		LatencyThreshold   float64 `yaml:"latency_threshold"`
	} `yaml:"analyzer"`

	Decision struct {
		ConfidenceThreshold float64 `yaml:"confidence_threshold"`
		DryRun              bool    `yaml:"dry_run"`
	} `yaml:"decision"`
}

// LoadConfig reads and validates configuration from YAML file
func LoadConfig(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	config.ApplyEnvOverrides()
	return &config, nil
}

// Validate checks if configuration values are valid
func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name cannot be empty")
	}
	if c.App.Version == "" {
		return fmt.Errorf("app.version cannot be empty")
	}
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.App.LogLevel] {
		return fmt.Errorf("app.log_level must be one of: debug, info, warn, error")
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database.host cannot be empty")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("database.port must be between 1 and 65535")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database.user cannot be empty")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database.dbname cannot be empty")
	}
	if c.Database.MaxConnections <= 0 {
		return fmt.Errorf("database.max_connections must be positive")
	}

	if c.Prometheus.URL == "" {
		return fmt.Errorf("prometheus.url cannot be empty")
	}
	if !strings.HasPrefix(c.Prometheus.URL, "http://") && !strings.HasPrefix(c.Prometheus.URL, "https://") {
		return fmt.Errorf("prometheus.url must start with http:// or https://")
	}

	if c.Analyzer.CPUThreshold <= 0 || c.Analyzer.CPUThreshold > 100 {
		return fmt.Errorf("analyzer.cpu_threshold must be between 0 and 100")
	}
	if c.Analyzer.MemoryThreshold <= 0 || c.Analyzer.MemoryThreshold > 100 {
		return fmt.Errorf("analyzer.memory_threshold must be between 0 and 100")
	}
	if c.Analyzer.ErrorRateThreshold < 0 {
		return fmt.Errorf("analyzer.error_rate_threshold must be non-negative")
	}
	if c.Analyzer.LatencyThreshold < 0 {
		return fmt.Errorf("analyzer.latency_threshold must be non-negative")
	}

	if c.Decision.ConfidenceThreshold < 0 || c.Decision.ConfidenceThreshold > 100 {
		return fmt.Errorf("decision.confidence_threshold must be between 0 and 100")
	}

	return nil
}

// ApplyEnvOverrides applies environment variable overrides
func (c *Config) ApplyEnvOverrides() {
	if host := os.Getenv("AURA_DB_HOST"); host != "" {
		c.Database.Host = host
	}
	if user := os.Getenv("AURA_DB_USER"); user != "" {
		c.Database.User = user
	}
	if password := os.Getenv("AURA_DB_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if dbname := os.Getenv("AURA_DB_NAME"); dbname != "" {
		c.Database.DBName = dbname
	}
	if url := os.Getenv("AURA_PROMETHEUS_URL"); url != "" {
		c.Prometheus.URL = url
	}
	if logLevel := os.Getenv("AURA_LOG_LEVEL"); logLevel != "" {
		c.App.LogLevel = logLevel
	}
}

// GetDatabaseURL returns PostgreSQL connection string
func (c *Config) GetDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&pool_max_conns=%d",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
		c.Database.MaxConnections,
	)
}
