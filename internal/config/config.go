package config

import (
	yamlenv "github.com/ifuryst/go-yaml-env"
	"github.com/ifuryst/ripple/pkg/logger"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Logger    logger.Config   `yaml:"logger"`
	Notion    NotionConfig    `yaml:"notion"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
}

type ServerConfig struct {
	Port     int    `yaml:"port"`
	Host     string `yaml:"host"`
	Mode     string `yaml:"mode"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type DatabaseConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"ssl_mode"`
	TimeZone string `yaml:"timezone"`
}

type NotionConfig struct {
	Token      string `yaml:"token"`
	DatabaseID string `yaml:"database_id"`
	APIVersion string `yaml:"api_version"`
}

type SchedulerConfig struct {
	SyncInterval string `yaml:"sync_interval"`
	Enabled      bool   `yaml:"enabled"`
}

func LoadConfig(configPath string) (*Config, error) {
	cfg, err := yamlenv.LoadConfig[Config](configPath)
	if err != nil {
		return nil, err
	}

	// Set default values
	if cfg.Server.Host == "" {
		cfg.Server.Host = "localhost"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 5334
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "debug"
	}
	if cfg.Database.Type == "" {
		cfg.Database.Type = "postgres"
	}
	if cfg.Database.Host == "" {
		cfg.Database.Host = "localhost"
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	if cfg.Database.TimeZone == "" {
		cfg.Database.TimeZone = "UTC"
	}
	if cfg.Notion.APIVersion == "" {
		cfg.Notion.APIVersion = "2022-06-28"
	}
	if cfg.Scheduler.SyncInterval == "" {
		cfg.Scheduler.SyncInterval = "30m"
	}
	if !cfg.Scheduler.Enabled {
		cfg.Scheduler.Enabled = true
	}

	return cfg, nil
}
