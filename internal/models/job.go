package models

import (
	"gorm.io/gorm"
	"time"
)

type DistributionJob struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	PageID      uint           `gorm:"not null;index" json:"page_id"`
	PlatformID  uint           `gorm:"not null;index" json:"platform_id"`
	Status      string         `gorm:"size:50;default:'pending'" json:"status"`
	Content     string         `gorm:"type:text" json:"content"`
	Error       string         `gorm:"type:text" json:"error"`
	PublishedAt *time.Time     `json:"published_at"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Page     NotionPage `gorm:"foreignKey:PageID" json:"page"`
	Platform Platform   `gorm:"foreignKey:PlatformID" json:"platform"`
}
