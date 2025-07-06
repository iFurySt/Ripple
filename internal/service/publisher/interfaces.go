package publisher

import (
	"context"
	"strings"
	"time"

	"github.com/ifuryst/ripple/internal/models"
)

// PublishContent represents the content to be published
type PublishContent struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	Summary     string            `json:"summary"`
	Tags        []string          `json:"tags"`
	Author      string            `json:"author"`
	PublishDate *time.Time        `json:"publish_date"`
	Metadata    map[string]string `json:"metadata"`
	Resources   []Resource        `json:"resources"`
}

// Resource represents a media resource (image, video, etc.)
type Resource struct {
	ID        string            `json:"id"`
	Type      ResourceType      `json:"type"`
	URL       string            `json:"url"`
	LocalPath string            `json:"local_path"`
	Metadata  map[string]string `json:"metadata"`
}

// ResourceType defines the type of resource
type ResourceType string

const (
	ResourceTypeImage ResourceType = "image"
	ResourceTypeVideo ResourceType = "video"
	ResourceTypeFile  ResourceType = "file"
)

// PublishResult represents the result of a publish operation
type PublishResult struct {
	Success     bool              `json:"success"`
	PublishID   string            `json:"publish_id,omitempty"`
	URL         string            `json:"url,omitempty"`
	Error       error             `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	PublishedAt time.Time         `json:"published_at"`
}

// PublishConfig represents platform-specific configuration
type PublishConfig struct {
	PlatformName string            `json:"platform_name"`
	Enabled      bool              `json:"enabled"`
	Config       map[string]string `json:"config"`
}

// Publisher is the unified interface for all platform operations
type Publisher interface {
	GetPlatformName() string

	Initialize(ctx context.Context, config PublishConfig) error
	ValidateConfig(config PublishConfig) error

	TransformContent(ctx context.Context, content PublishContent) (*PublishContent, error)
	ProcessResources(ctx context.Context, content *PublishContent, config PublishConfig) error

	SaveToDraft(ctx context.Context, content PublishContent, config PublishConfig) (*PublishResult, error)
	Publish(ctx context.Context, draftID string, config PublishConfig) (*PublishResult, error)
	PublishDirect(ctx context.Context, content PublishContent, config PublishConfig) (*PublishResult, error)

	GetPublishStatus(ctx context.Context, publishID string, config PublishConfig) (*PublishResult, error)
	Cleanup(ctx context.Context, publishID string, config PublishConfig) error
}

// Utility functions for content conversion

// FromNotionPage converts a NotionPage to PublishContent
func FromNotionPage(page *models.NotionPage) *PublishContent {
	// Convert StringArray directly to []string
	tags := []string(page.Tags)
	platforms := []string(page.Platforms)
	contentTypes := []string(page.ContentType)

	metadata := map[string]string{
		"notion_id": page.NotionID,
		"status":    page.Status,
	}

	// Join arrays as comma-separated strings for metadata compatibility
	if len(platforms) > 0 {
		metadata["platforms"] = strings.Join(platforms, ",")
	}
	if len(contentTypes) > 0 {
		metadata["content_type"] = strings.Join(contentTypes, ",")
	}

	if page.ENTitle != "" {
		metadata["en_title"] = page.ENTitle
	}

	return &PublishContent{
		ID:          page.NotionID,
		Title:       page.Title,
		Content:     page.Content,
		Summary:     page.Summary,
		Tags:        tags,
		Author:      page.Owner,
		PublishDate: page.PostDate,
		Metadata:    metadata,
		Resources:   []Resource{}, // Will be populated during processing
	}
}
