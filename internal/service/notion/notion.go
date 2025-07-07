package notion

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ifuryst/ripple/internal/config"
	"github.com/ifuryst/ripple/internal/models"
)

type (
	DatabaseResponse struct {
		Results    []PageResponse `json:"results"`
		NextCursor string         `json:"next_cursor"`
		HasMore    bool           `json:"has_more"`
	}

	PageResponse struct {
		ID             string         `json:"id"`
		CreatedTime    string         `json:"created_time"`
		LastEditedTime string         `json:"last_edited_time"`
		Properties     map[string]any `json:"properties"`
		Children       []Block        `json:"children,omitempty"`
	}

	Block struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Content any    `json:"content"`
	}
)

type Service struct {
	config *config.NotionConfig
	db     *gorm.DB
	logger *zap.Logger
	client *http.Client
}

func NewService(config *config.NotionConfig, db *gorm.DB, logger *zap.Logger) *Service {
	tr := &http.Transport{
		IdleConnTimeout:       120 * time.Second,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   10,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
	}
	return &Service{
		config: config,
		db:     db,
		logger: logger,
		client: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
	}
}

func (s *Service) SyncPages() error {
	s.logger.Info("Starting Notion pages sync")

	cursor := ""
	for {
		response, err := s.queryDatabase(cursor)
		if err != nil {
			return fmt.Errorf("failed to query database: %w", err)
		}

		for _, page := range response.Results {
			if err := s.processPage(page); err != nil {
				s.logger.Error("Failed to process page", zap.String("page_id", page.ID), zap.Error(err))
				continue
			}
		}

		if !response.HasMore {
			break
		}
		cursor = response.NextCursor
	}

	s.logger.Info("Notion pages sync completed")
	return nil
}

func (s *Service) processPage(page PageResponse) error {
	// Parse timestamps
	lastModified, err := time.Parse(time.RFC3339, page.LastEditedTime)
	if err != nil {
		return fmt.Errorf("failed to parse last_edited_time: %w", err)
	}

	// Extract all properties
	title := s.extractTitle(page.Properties)
	enTitle := s.extractENTitle(page.Properties)
	tags := s.extractTags(page.Properties)
	status := s.extractStatus(page.Properties)
	postDate := s.extractPostDate(page.Properties)
	owner := s.extractOwner(page.Properties)
	platforms := s.extractPlatforms(page.Properties)
	contentType := s.extractContentType(page.Properties)

	// Serialize properties
	propertiesJSON, err := json.Marshal(page.Properties)
	if err != nil {
		return fmt.Errorf("failed to marshal properties: %w", err)
	}

	// Get page content
	content, err := s.getPageContent(page.ID)
	if err != nil {
		s.logger.Warn("Failed to get page content", zap.String("page_id", page.ID), zap.Error(err))
		content = ""
	}

	// Check if page exists
	var existingPage models.NotionPage
	result := s.db.Where("notion_id = ?", page.ID).First(&existingPage)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to query existing page: %w", result.Error)
	}

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// Create new page
		newPage := models.NotionPage{
			NotionID:     page.ID,
			Title:        title,
			ENTitle:      enTitle,
			Content:      content,
			Tags:         tags,
			Status:       status,
			PostDate:     postDate,
			Owner:        owner,
			Platforms:    platforms,
			ContentType:  contentType,
			Properties:   string(propertiesJSON),
			LastModified: lastModified,
		}

		if err := s.db.Create(&newPage).Error; err != nil {
			return fmt.Errorf("failed to create page: %w", err)
		}

		s.logger.Info("Created new page", zap.String("page_id", page.ID), zap.String("title", title))
	} else {
		// Check if we need to force refresh content (for image link expiration)
		needsContentRefresh := s.shouldRefreshContent(existingPage)
		
		// Update existing page if modified or needs content refresh
		if existingPage.LastModified.Before(lastModified) || needsContentRefresh {
			existingPage.Title = title
			existingPage.ENTitle = enTitle
			existingPage.Content = content
			existingPage.Tags = tags
			existingPage.Status = status
			existingPage.PostDate = postDate
			existingPage.Owner = owner
			existingPage.Platforms = platforms
			existingPage.ContentType = contentType
			existingPage.Properties = string(propertiesJSON)
			existingPage.LastModified = lastModified

			if err := s.db.Save(&existingPage).Error; err != nil {
				return fmt.Errorf("failed to update page: %w", err)
			}

			if needsContentRefresh {
				s.logger.Info("Force refreshed page content", zap.String("page_id", page.ID), zap.String("title", title), zap.String("reason", "content_refresh"))
			} else {
				s.logger.Info("Updated existing page", zap.String("page_id", page.ID), zap.String("title", title))
			}
		}
	}

	return nil
}

func (s *Service) shouldRefreshContent(existingPage models.NotionPage) bool {
	// Force refresh if content is older than 4 hours (image links typically expire in 1-24 hours)
	refreshThreshold := time.Now().Add(-4 * time.Hour)
	
	// Check if page was last updated more than 4 hours ago
	if existingPage.UpdatedAt.Before(refreshThreshold) {
		// Check if content contains AWS image URLs that might expire
		if s.containsAWSImageURLs(existingPage.Content) {
			return true
		}
	}
	
	return false
}

func (s *Service) containsAWSImageURLs(content string) bool {
	// Check for AWS S3 URLs in content that are commonly used by Notion
	awsPatterns := []string{
		"s3.us-west-2.amazonaws.com",
		"prod-files-secure.s3.us-west-2.amazonaws.com",
		"amazonaws.com",
		"?X-Amz-Algorithm=",
		"?X-Amz-Credential=",
	}
	
	for _, pattern := range awsPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	
	return false
}

func (s *Service) getPageContent(pageID string) (string, error) {
	allBlocks, err := s.getAllBlocksRecursively(pageID)
	if err != nil {
		return "", fmt.Errorf("failed to get page blocks recursively: %w", err)
	}

	// Store raw blocks JSON instead of converting to markdown
	blocksJSON, err := json.Marshal(allBlocks)
	if err != nil {
		return "", fmt.Errorf("failed to marshal blocks: %w", err)
	}

	return string(blocksJSON), nil
}

func (s *Service) GetAllPages() ([]models.NotionPage, error) {
	var pages []models.NotionPage
	if err := s.db.Find(&pages).Error; err != nil {
		return nil, fmt.Errorf("failed to get pages: %w", err)
	}
	return pages, nil
}
