package models

import (
	"time"

	"gorm.io/gorm"
)

type NotionPage struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	NotionID     string         `gorm:"uniqueIndex;not null;size:255" json:"notion_id"`
	Title        string         `gorm:"not null;size:500" json:"title"`
	ENTitle      string         `gorm:"size:500" json:"en_title"`
	Content      string         `gorm:"type:text" json:"content"`
	Summary      string         `gorm:"type:text" json:"summary"`
	Tags         string         `gorm:"type:text" json:"tags"`
	Status       string         `gorm:"size:50;default:'draft'" json:"status"`
	PostDate     *time.Time     `json:"post_date"`
	Owner        string         `gorm:"size:500" json:"owner"`
	Platforms    string         `gorm:"type:text" json:"platforms"`
	ContentType  string         `gorm:"type:text" json:"content_type"`
	Properties   string         `gorm:"type:jsonb" json:"properties"`
	LastModified time.Time      `json:"last_modified"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type DistributionJob struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	PageID     uint           `gorm:"not null;index" json:"page_id"`
	PlatformID uint           `gorm:"not null;index" json:"platform_id"`
	Status     string         `gorm:"size:50;default:'pending'" json:"status"`
	Content    string         `gorm:"type:text" json:"content"`
	Error      string         `gorm:"type:text" json:"error"`
	PublishedAt *time.Time    `json:"published_at"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Page     NotionPage `gorm:"foreignKey:PageID" json:"page"`
	Platform Platform   `gorm:"foreignKey:PlatformID" json:"platform"`
}

type Platform struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null;size:100" json:"name"`
	DisplayName string         `gorm:"not null;size:100" json:"display_name"`
	Config      string         `gorm:"type:jsonb" json:"config"`
	Enabled     bool           `gorm:"default:true" json:"enabled"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}