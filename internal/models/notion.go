package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// StringArray represents a PostgreSQL text[] type
type StringArray []string

// Scan implements the sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = StringArray{}
		return nil
	}

	switch v := value.(type) {
	case string:
		// Handle PostgreSQL array format: {value1,value2,value3}
		if v == "{}" || v == "" {
			*s = StringArray{}
			return nil
		}

		// Remove outer braces and split by comma
		trimmed := strings.Trim(v, "{}")
		if trimmed == "" {
			*s = StringArray{}
			return nil
		}

		parts := strings.Split(trimmed, ",")
		result := make([]string, len(parts))
		for i, part := range parts {
			// Remove quotes if present
			result[i] = strings.Trim(strings.TrimSpace(part), "\"")
		}
		*s = result
		return nil
	case []byte:
		// Try to parse as JSON first
		var arr []string
		if err := json.Unmarshal(v, &arr); err == nil {
			*s = arr
			return nil
		}
		// Fallback to string parsing
		return s.Scan(string(v))
	default:
		return errors.New(fmt.Sprintf("cannot scan %T into StringArray", value))
	}
}

// Value implements the driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "{}", nil
	}

	// Format as PostgreSQL array: {value1,value2,value3}
	quoted := make([]string, len(s))
	for i, v := range s {
		// Escape quotes and wrap in quotes
		escaped := strings.ReplaceAll(v, "\"", "\\\"")
		quoted[i] = fmt.Sprintf("\"%s\"", escaped)
	}

	return fmt.Sprintf("{%s}", strings.Join(quoted, ",")), nil
}

type NotionPage struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	NotionID     string         `gorm:"uniqueIndex;not null;size:255" json:"notion_id"`
	Title        string         `gorm:"not null;size:500" json:"title"`
	ENTitle      string         `gorm:"size:500" json:"en_title"`
	Content      string         `gorm:"type:text" json:"content"`
	Summary      string         `gorm:"type:text" json:"summary"`
	Tags         StringArray    `gorm:"type:text[]" json:"tags"`
	Status       string         `gorm:"size:50;default:'draft'" json:"status"`
	PostDate     *time.Time     `json:"post_date"`
	Owner        string         `gorm:"size:500" json:"owner"`
	Platforms    StringArray    `gorm:"type:text[]" json:"platforms"`
	ContentType  StringArray    `gorm:"type:text[]" json:"content_type"`
	Properties   string         `gorm:"type:jsonb" json:"properties"`
	LastModified time.Time      `json:"last_modified"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}
