package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ifuryst/ripple/internal/config"
	"github.com/ifuryst/ripple/internal/models"
)

type NotionService struct {
	config *config.NotionConfig
	db     *gorm.DB
	logger *zap.Logger
	client *http.Client
}

type NotionDatabaseResponse struct {
	Results    []NotionPageResponse `json:"results"`
	NextCursor string               `json:"next_cursor"`
	HasMore    bool                 `json:"has_more"`
}

type NotionPageResponse struct {
	ID             string                 `json:"id"`
	CreatedTime    string                 `json:"created_time"`
	LastEditedTime string                 `json:"last_edited_time"`
	Properties     map[string]interface{} `json:"properties"`
	Children       []NotionBlock          `json:"children,omitempty"`
}

type NotionBlock struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Content interface{} `json:"content"`
}

func NewNotionService(config *config.NotionConfig, db *gorm.DB, logger *zap.Logger) *NotionService {
	return &NotionService{
		config: config,
		db:     db,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *NotionService) SyncPages() error {
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

func (s *NotionService) queryDatabase(cursor string) (*NotionDatabaseResponse, error) {
	url := fmt.Sprintf("https://api.notion.com/v1/databases/%s/query", s.config.DatabaseID)

	body := map[string]interface{}{
		"page_size": 100,
		"filter": map[string]interface{}{
			"property": "Status",
			"status": map[string]interface{}{
				"equals": "Done",
			},
		},
	}
	if cursor != "" {
		body["start_cursor"] = cursor
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", s.config.APIVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("notion API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response NotionDatabaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (s *NotionService) processPage(page NotionPageResponse) error {
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

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to query existing page: %w", result.Error)
	}

	if result.Error == gorm.ErrRecordNotFound {
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
		// Update existing page if modified
		if existingPage.LastModified.Before(lastModified) {
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

			s.logger.Info("Updated existing page", zap.String("page_id", page.ID), zap.String("title", title))
		}
	}

	return nil
}

func (s *NotionService) getPageContent(pageID string) (string, error) {
	url := fmt.Sprintf("https://api.notion.com/v1/blocks/%s/children", pageID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.config.Token)
	req.Header.Set("Notion-Version", s.config.APIVersion)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("notion API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Results []map[string]interface{} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Simple text extraction - this could be enhanced
	var content []string
	for _, block := range response.Results {
		if text := s.extractTextFromBlock(block); text != "" {
			content = append(content, text)
		}
	}

	return fmt.Sprintf("%s", content), nil
}

func (s *NotionService) extractTitle(properties map[string]interface{}) string {
	// Look for title property
	for _, prop := range properties {
		if propMap, ok := prop.(map[string]interface{}); ok {
			if propMap["type"] == "title" {
				if title, ok := propMap["title"].([]interface{}); ok && len(title) > 0 {
					if titleObj, ok := title[0].(map[string]interface{}); ok {
						if plainText, ok := titleObj["plain_text"].(string); ok {
							return plainText
						}
					}
				}
			}
		}
	}
	return "Untitled"
}

func (s *NotionService) extractTags(properties map[string]interface{}) string {
	// Look for tags/multi_select property
	for _, prop := range properties {
		if propMap, ok := prop.(map[string]interface{}); ok {
			if propMap["type"] == "multi_select" {
				if tags, ok := propMap["multi_select"].([]interface{}); ok {
					var tagNames []string
					for _, tag := range tags {
						if tagMap, ok := tag.(map[string]interface{}); ok {
							if name, ok := tagMap["name"].(string); ok {
								tagNames = append(tagNames, name)
							}
						}
					}
					return fmt.Sprintf("%v", tagNames)
				}
			}
		}
	}
	return ""
}

func (s *NotionService) extractStatus(properties map[string]interface{}) string {
	// Look for status property
	for _, prop := range properties {
		if propMap, ok := prop.(map[string]interface{}); ok {
			if propMap["type"] == "status" {
				if statusObj, ok := propMap["status"].(map[string]interface{}); ok {
					if name, ok := statusObj["name"].(string); ok {
						return name
					}
				}
			}
		}
	}
	return "draft"
}

func (s *NotionService) extractENTitle(properties map[string]interface{}) string {
	// Look for EN Title rich_text property
	for propName, prop := range properties {
		if propName == "EN Title" {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propMap["type"] == "rich_text" {
					if richText, ok := propMap["rich_text"].([]interface{}); ok && len(richText) > 0 {
						if textObj, ok := richText[0].(map[string]interface{}); ok {
							if plainText, ok := textObj["plain_text"].(string); ok {
								return plainText
							}
						}
					}
				}
			}
		}
	}
	return ""
}

func (s *NotionService) extractPostDate(properties map[string]interface{}) *time.Time {
	// Look for Post date property
	for propName, prop := range properties {
		if propName == "Post date" {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propMap["type"] == "date" {
					if dateObj, ok := propMap["date"].(map[string]interface{}); ok {
						if startStr, ok := dateObj["start"].(string); ok {
							if date, err := time.Parse("2006-01-02", startStr); err == nil {
								return &date
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *NotionService) extractOwner(properties map[string]interface{}) string {
	// Look for Owner people property
	for propName, prop := range properties {
		if propName == "Owner" {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propMap["type"] == "people" {
					if people, ok := propMap["people"].([]interface{}); ok && len(people) > 0 {
						var owners []string
						for _, person := range people {
							if personMap, ok := person.(map[string]interface{}); ok {
								if name, ok := personMap["name"].(string); ok {
									owners = append(owners, name)
								}
							}
						}
						if len(owners) > 0 {
							return fmt.Sprintf("%v", owners)
						}
					}
				}
			}
		}
	}
	return ""
}

func (s *NotionService) extractPlatforms(properties map[string]interface{}) string {
	// Look for Platform multi_select property
	for propName, prop := range properties {
		if propName == "Platform" {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propMap["type"] == "multi_select" {
					if platforms, ok := propMap["multi_select"].([]interface{}); ok {
						var platformNames []string
						for _, platform := range platforms {
							if platformMap, ok := platform.(map[string]interface{}); ok {
								if name, ok := platformMap["name"].(string); ok {
									platformNames = append(platformNames, name)
								}
							}
						}
						return fmt.Sprintf("%v", platformNames)
					}
				}
			}
		}
	}
	return ""
}

func (s *NotionService) extractContentType(properties map[string]interface{}) string {
	// Look for Content type multi_select property
	for propName, prop := range properties {
		if propName == "Content type" {
			if propMap, ok := prop.(map[string]interface{}); ok {
				if propMap["type"] == "multi_select" {
					if contentTypes, ok := propMap["multi_select"].([]interface{}); ok {
						var typeNames []string
						for _, contentType := range contentTypes {
							if typeMap, ok := contentType.(map[string]interface{}); ok {
								if name, ok := typeMap["name"].(string); ok {
									typeNames = append(typeNames, name)
								}
							}
						}
						return fmt.Sprintf("%v", typeNames)
					}
				}
			}
		}
	}
	return ""
}

func (s *NotionService) extractTextFromBlock(block map[string]interface{}) string {
	blockType, ok := block["type"].(string)
	if !ok {
		return ""
	}

	blockContent, ok := block[blockType].(map[string]interface{})
	if !ok {
		return ""
	}

	richText, ok := blockContent["rich_text"].([]interface{})
	if !ok {
		return ""
	}

	var text string
	for _, rt := range richText {
		if rtMap, ok := rt.(map[string]interface{}); ok {
			if plainText, ok := rtMap["plain_text"].(string); ok {
				text += plainText
			}
		}
	}

	return text
}

func (s *NotionService) GetAllPages() ([]models.NotionPage, error) {
	var pages []models.NotionPage
	if err := s.db.Find(&pages).Error; err != nil {
		return nil, fmt.Errorf("failed to get pages: %w", err)
	}
	return pages, nil
}