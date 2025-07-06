package config

import (
	"github.com/ifuryst/ripple/pkg/logger"
	"time"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Logger    logger.Config   `yaml:"logger"`
	Notion    NotionConfig    `yaml:"notion"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Publisher PublisherConfig `yaml:"publisher"`
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
	SyncInterval time.Duration `yaml:"sync_interval"`
	Enabled      bool          `yaml:"enabled"`
}

type PublisherConfig struct {
	AlFolio        AlFolioConfig        `yaml:"al_folio"`
	WeChatOfficial WeChatOfficialConfig `yaml:"wechat_official"`
}

type AlFolioConfig struct {
	Enabled       bool   `yaml:"enabled"`
	RepoURL       string `yaml:"repo_url"`
	Branch        string `yaml:"branch"`
	WorkspaceDir  string `yaml:"workspace_dir"`
	BaseURL       string `yaml:"base_url"`
	CommitMessage string `yaml:"commit_message"`
	AutoPublish   bool   `yaml:"auto_publish"`
}

type WeChatOfficialConfig struct {
	Enabled            bool   `yaml:"enabled"`
	AppID              string `yaml:"app_id"`
	AppSecret          string `yaml:"app_secret"`
	AutoPublish        bool   `yaml:"auto_publish"`
	NeedOpenComment    int    `yaml:"need_open_comment"`
	OnlyFansCanComment int    `yaml:"only_fans_can_comment"`
	DefaultThumbMediaID string `yaml:"default_thumb_media_id"`
}
