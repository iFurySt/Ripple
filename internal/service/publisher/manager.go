package publisher

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"

	"github.com/ifuryst/ripple/internal/models"
)

// Manager implements the Manager interface
type Manager struct {
	publishers map[string]Publisher
	logger     *zap.Logger
	db         *gorm.DB
	configs    map[string]PublishConfig
}

func NewPublishManager(logger *zap.Logger, db *gorm.DB) *Manager {
	return &Manager{
		publishers: make(map[string]Publisher),
		logger:     logger,
		db:         db,
		configs:    make(map[string]PublishConfig),
	}
}

func (m *Manager) RegisterPublisher(publisher Publisher) error {
	platformName := publisher.GetPlatformName()
	if _, exists := m.publishers[platformName]; exists {
		return fmt.Errorf("publisher for platform %s already registered", platformName)
	}

	m.publishers[platformName] = publisher
	m.logger.Info("Publisher registered", zap.String("platform", platformName))
	return nil
}

func (m *Manager) GetPublisher(platformName string) (Publisher, error) {
	publisher, exists := m.publishers[platformName]
	if !exists {
		return nil, fmt.Errorf("publisher for platform %s not found", platformName)
	}
	return publisher, nil
}

func (m *Manager) GetAvailablePublishers() []Publisher {
	var publishers []Publisher
	for _, publisher := range m.publishers {
		publishers = append(publishers, publisher)
	}
	return publishers
}

func (m *Manager) SetPlatformConfig(platformName string, config PublishConfig) {
	m.configs[platformName] = config
}

func (m *Manager) GetPlatformConfig(platformName string) (PublishConfig, error) {
	config, exists := m.configs[platformName]
	if !exists {
		return PublishConfig{}, fmt.Errorf("config for platform %s not found", platformName)
	}
	return config, nil
}

func (m *Manager) PublishToAll(ctx context.Context, page *models.NotionPage) (map[string]*PublishResult, error) {
	// Use platforms directly from page.Platforms (now a StringArray)
	notionPlatforms := []string(page.Platforms)

	// Map Notion platform names to system platform names
	var platforms []string
	for _, notionPlatform := range notionPlatforms {
		if systemPlatform := m.mapPlatformName(notionPlatform); systemPlatform != "" {
			platforms = append(platforms, systemPlatform)
		}
	}

	if len(platforms) == 0 {
		// If no platforms specified, publish to all available platforms
		for platformName := range m.publishers {
			platforms = append(platforms, platformName)
		}
	}

	return m.PublishToPlatforms(ctx, page, platforms)
}

func (m *Manager) PublishToPlatforms(ctx context.Context, page *models.NotionPage, platforms []string) (map[string]*PublishResult, error) {
	results := make(map[string]*PublishResult)
	content := FromNotionPage(page)

	for _, platformName := range platforms {
		publisher, err := m.GetPublisher(platformName)
		if err != nil {
			m.logger.Error("Publisher not found",
				zap.String("platform", platformName),
				zap.Error(err))
			results[platformName] = &PublishResult{
				Success: false,
				Error:   err,
			}
			continue
		}

		config, err := m.GetPlatformConfig(platformName)
		if err != nil {
			m.logger.Error("Platform config not found",
				zap.String("platform", platformName),
				zap.Error(err))
			results[platformName] = &PublishResult{
				Success: false,
				Error:   err,
			}
			continue
		}

		// Check if platform is enabled
		if !config.Enabled {
			m.logger.Info("Platform disabled, skipping",
				zap.String("platform", platformName))
			results[platformName] = &PublishResult{
				Success: false,
				Error:   fmt.Errorf("platform %s is disabled", platformName),
			}
			continue
		}

		// Get platform ID
		platformID := m.getPlatformID(platformName)
		if platformID == 0 {
			m.logger.Error("Failed to get platform ID",
				zap.String("platform", platformName))
			results[platformName] = &PublishResult{
				Success: false,
				Error:   fmt.Errorf("failed to get platform ID for %s", platformName),
			}
			continue
		}

		// Check if this platform already has a completed job
		var existingJob models.DistributionJob
		if err := m.db.Where("page_id = ? AND platform_id = ? AND status = ?", 
			page.ID, platformID, "completed").First(&existingJob).Error; err == nil {
			// Job already completed, skip
			m.logger.Info("Platform already completed, skipping",
				zap.String("platform", platformName),
				zap.Uint("page_id", page.ID))
			results[platformName] = &PublishResult{
				Success: true,
				PublishID: fmt.Sprintf("existing-job-%d", existingJob.ID),
			}
			continue
		}

		// Record distribution job start
		job := &models.DistributionJob{
			PageID:     page.ID,
			PlatformID: platformID,
			Status:     "in_progress",
			Content:    content.Content,
		}

		if err := m.db.Create(job).Error; err != nil {
			m.logger.Error("Failed to create distribution job",
				zap.String("platform", platformName),
				zap.Error(err))
		}

		// Initialize publisher
		if err := publisher.Initialize(ctx, config); err != nil {
			m.logger.Error("Failed to initialize publisher",
				zap.String("platform", platformName),
				zap.Error(err))

			m.updateJobStatus(job, "failed", err.Error())
			results[platformName] = &PublishResult{
				Success: false,
				Error:   err,
			}
			continue
		}

		// Publish content
		result, err := publisher.PublishDirect(ctx, *content, config)
		if err != nil {
			m.logger.Error("Failed to publish content",
				zap.String("platform", platformName),
				zap.Error(err))

			m.updateJobStatus(job, "failed", err.Error())
			results[platformName] = &PublishResult{
				Success: false,
				Error:   err,
			}
			continue
		}

		// Update job status
		if result.Success {
			m.updateJobStatus(job, "completed", "")
			job.PublishedAt = &result.PublishedAt
		} else {
			errorMsg := "unknown error"
			if result.Error != nil {
				errorMsg = result.Error.Error()
			}
			m.updateJobStatus(job, "failed", errorMsg)
		}

		// Cleanup
		if result.Success && result.PublishID != "" {
			if err := publisher.Cleanup(ctx, result.PublishID, config); err != nil {
				m.logger.Warn("Cleanup failed",
					zap.String("platform", platformName),
					zap.Error(err))
			}
		}

		results[platformName] = result

		m.logger.Info("Publishing completed",
			zap.String("platform", platformName),
			zap.Bool("success", result.Success),
			zap.String("publish_id", result.PublishID))
	}

	return results, nil
}

func (m *Manager) GetPublishHistory(ctx context.Context, pageID string) ([]*models.DistributionJob, error) {
	var jobs []*models.DistributionJob

	// Find page by notion_id
	var page models.NotionPage
	if err := m.db.Where("notion_id = ?", pageID).First(&page).Error; err != nil {
		return nil, fmt.Errorf("page not found: %w", err)
	}

	// Get distribution jobs for this page
	if err := m.db.Where("page_id = ?", page.ID).
		Preload("Platform").
		Order("created_at DESC").
		Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to get publish history: %w", err)
	}

	return jobs, nil
}

// PublishSinglePlatform publishes content to a single platform
func (m *Manager) PublishSinglePlatform(ctx context.Context, page *models.NotionPage, platformName string, isDraft bool) (*PublishResult, error) {
	publisher, err := m.GetPublisher(platformName)
	if err != nil {
		return &PublishResult{
			Success: false,
			Error:   err,
		}, nil
	}

	config, err := m.GetPlatformConfig(platformName)
	if err != nil {
		return &PublishResult{
			Success: false,
			Error:   err,
		}, nil
	}

	if !config.Enabled {
		return &PublishResult{
			Success: false,
			Error:   fmt.Errorf("platform %s is disabled", platformName),
		}, nil
	}

	content := FromNotionPage(page)

	// Initialize publisher
	if err := publisher.Initialize(ctx, config); err != nil {
		return &PublishResult{
			Success: false,
			Error:   err,
		}, nil
	}

	// Transform content
	transformedContent, err := publisher.TransformContent(ctx, *content)
	if err != nil {
		return &PublishResult{
			Success: false,
			Error:   err,
		}, nil
	}

	// Process resources
	if err := publisher.ProcessResources(ctx, transformedContent, config); err != nil {
		return &PublishResult{
			Success: false,
			Error:   err,
		}, nil
	}

	var result *PublishResult

	if isDraft {
		// Save as draft
		result, err = publisher.SaveToDraft(ctx, *transformedContent, config)
	} else {
		// Publish directly
		result, err = publisher.PublishDirect(ctx, *transformedContent, config)
	}

	if err != nil {
		return &PublishResult{
			Success: false,
			Error:   err,
		}, nil
	}

	// Record distribution job
	status := "completed"
	if isDraft {
		status = "draft"
	}
	if !result.Success {
		status = "failed"
	}

	// Get platform ID
	platformID := m.getPlatformID(platformName)
	if platformID == 0 {
		return &PublishResult{
			Success: false,
			Error:   fmt.Errorf("failed to get platform ID for %s", platformName),
		}, nil
	}

	job := &models.DistributionJob{
		PageID:     page.ID,
		PlatformID: platformID,
		Status:     status,
		Content:    transformedContent.Content,
	}

	if result.Success && !isDraft {
		job.PublishedAt = &result.PublishedAt
	}

	if result.Error != nil {
		job.Error = result.Error.Error()
	}

	if err := m.db.Create(job).Error; err != nil {
		m.logger.Error("Failed to record distribution job",
			zap.String("platform", platformName),
			zap.Error(err))
	}

	return result, nil
}

// Helper methods

func (m *Manager) mapPlatformName(notionPlatform string) string {
	// Map Notion platform names to system platform names
	platformMap := map[string]string{
		"Blog":       "al-folio",
		"blog":       "al-folio",
		"Jekyll":     "al-folio",
		"jekyll":     "al-folio",
		"微信公众号": "wechat-official",
		"WeChat":     "wechat-official",
		"wechat":     "wechat-official",
		"Substack":   "substack",
		"substack":   "substack",
		// Direct matches (already using system names)
		"al-folio":     "al-folio",
		"wechat-official": "wechat-official",
	}

	if systemName, exists := platformMap[notionPlatform]; exists {
		return systemName
	}

	m.logger.Warn("Unknown platform name", zap.String("notion_platform", notionPlatform))
	return ""
}

func (m *Manager) getPlatformID(platformName string) uint {
	// This is a simplified implementation
	// In a real system, you'd have a proper platform management system
	var platform models.Platform
	if err := m.db.Where("name = ?", platformName).First(&platform).Error; err != nil {
		// Create platform if it doesn't exist
		platform = models.Platform{
			Name:        platformName,
			DisplayName: strings.Title(platformName),
			Config:      "{}", // Empty JSON object for jsonb field
			Enabled:     true,
		}
		if createErr := m.db.Create(&platform).Error; createErr != nil {
			m.logger.Error("Failed to create platform",
				zap.String("platform_name", platformName),
				zap.Error(createErr))
			// Return 0 to indicate failure - caller should handle this
			return 0
		}
		m.logger.Info("Created new platform",
			zap.String("platform_name", platformName),
			zap.Uint("platform_id", platform.ID))
	}
	return platform.ID
}

func (m *Manager) updateJobStatus(job *models.DistributionJob, status, errorMsg string) {
	job.Status = status
	job.Error = errorMsg
	if err := m.db.Save(job).Error; err != nil {
		m.logger.Error("Failed to update job status",
			zap.Uint("job_id", job.ID),
			zap.Error(err))
	}
}
