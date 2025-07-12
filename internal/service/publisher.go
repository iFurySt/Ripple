package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ifuryst/ripple/internal/config"
	"github.com/ifuryst/ripple/internal/models"
	"github.com/ifuryst/ripple/internal/service/publisher"
	"github.com/ifuryst/ripple/internal/service/publisher/al_folio"
	"github.com/ifuryst/ripple/internal/service/publisher/substack"
	"github.com/ifuryst/ripple/internal/service/publisher/wechat_official"
)

// PublisherService manages content publishing to various platforms
type PublisherService struct {
	logger             *zap.Logger
	db                 *gorm.DB
	config             *config.Config
	manager            *publisher.Manager
	monitoringService  *MonitoringService
}

func NewPublisherService(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *PublisherService {
	service := &PublisherService{
		logger:            logger,
		db:                db,
		config:            cfg,
		manager:           publisher.NewPublishManager(logger, db),
		monitoringService: NewMonitoringService(db, logger),
	}

	// Register publishers
	service.registerPublishers()

	return service
}

func (s *PublisherService) registerPublishers() {
	// Register Al-Folio Blog Publisher
	if s.config.Publisher.AlFolio.Enabled {
		alFolioPublisher := al_folio.NewAlFolioPublisher(s.logger)
		if err := s.manager.RegisterPublisher(alFolioPublisher); err != nil {
			s.logger.Error("Failed to register Al-Folio blog publisher", zap.Error(err))
		} else {
			// Set platform configuration
			cfg := publisher.PublishConfig{
				PlatformName: "al-folio",
				Enabled:      s.config.Publisher.AlFolio.Enabled,
				Config: map[string]string{
					"repo_url":       s.config.Publisher.AlFolio.RepoURL,
					"branch":         s.config.Publisher.AlFolio.Branch,
					"workspace_dir":  s.config.Publisher.AlFolio.WorkspaceDir,
					"base_url":       s.config.Publisher.AlFolio.BaseURL,
					"commit_message": s.config.Publisher.AlFolio.CommitMessage,
					"auto_publish":   fmt.Sprintf("%t", s.config.Publisher.AlFolio.AutoPublish),
				},
			}
			s.manager.SetPlatformConfig("al-folio", cfg)
			s.logger.Info("Al-Folio blog publisher registered and configured")
		}
	}

	// Register WeChat Official Account Publisher
	if s.config.Publisher.WeChatOfficial.Enabled {
		wechatPublisher := wechat_official.NewWeChatOfficialPublisher(s.logger)
		if err := s.manager.RegisterPublisher(wechatPublisher); err != nil {
			s.logger.Error("Failed to register WeChat Official Account publisher", zap.Error(err))
		} else {
			// Set platform configuration
			cfg := publisher.PublishConfig{
				PlatformName: "wechat-official",
				Enabled:      s.config.Publisher.WeChatOfficial.Enabled,
				Config: map[string]string{
					"app_id":                s.config.Publisher.WeChatOfficial.AppID,
					"app_secret":            s.config.Publisher.WeChatOfficial.AppSecret,
					"auto_publish":          fmt.Sprintf("%t", s.config.Publisher.WeChatOfficial.AutoPublish),
					"need_open_comment":     fmt.Sprintf("%d", s.config.Publisher.WeChatOfficial.NeedOpenComment),
					"only_fans_can_comment": fmt.Sprintf("%d", s.config.Publisher.WeChatOfficial.OnlyFansCanComment),
					"default_thumb_media_id": s.config.Publisher.WeChatOfficial.DefaultThumbMediaID,
				},
			}
			s.manager.SetPlatformConfig("wechat-official", cfg)
			s.logger.Info("WeChat Official Account publisher registered and configured")
		}
	}

	// Register Substack Publisher
	if s.config.Publisher.Substack.Enabled {
		substackPublisher := substack.NewSubstackPublisher(s.logger)
		if err := s.manager.RegisterPublisher(substackPublisher); err != nil {
			s.logger.Error("Failed to register Substack publisher", zap.Error(err))
		} else {
			// Set platform configuration
			cfg := publisher.PublishConfig{
				PlatformName: "substack",
				Enabled:      s.config.Publisher.Substack.Enabled,
				Config: map[string]string{
					"domain":       s.config.Publisher.Substack.Domain,
					"cookie":       s.config.Publisher.Substack.Cookie,
					"auto_publish": fmt.Sprintf("%t", s.config.Publisher.Substack.AutoPublish),
				},
			}
			s.manager.SetPlatformConfig("substack", cfg)
			s.logger.Info("Substack publisher registered and configured")
		}
	}
}

// PublishPage publishes a single page to all configured platforms
func (s *PublisherService) PublishPage(ctx context.Context, pageID string) (map[string]*publisher.PublishResult, error) {
	// Get the page from database
	var page models.NotionPage
	if err := s.db.Where("notion_id = ?", pageID).First(&page).Error; err != nil {
		return nil, fmt.Errorf("page not found: %w", err)
	}

	// Check if page is ready for publishing (status should be "Done")
	if page.Status != "Done" {
		return nil, fmt.Errorf("page status is not 'Done', current status: %s", page.Status)
	}

	s.logger.Info("Publishing page",
		zap.String("page_id", pageID),
		zap.String("title", page.Title),
		zap.Strings("platforms", page.Platforms))

	// Publish to all platforms
	results, err := s.manager.PublishToAll(ctx, &page)
	if err != nil {
		// Record error in monitoring
		s.monitoringService.RecordError("ERROR", "publisher", "Failed to publish page to all platforms", err.Error(),
			WithPage(page.ID),
			WithContext(map[string]interface{}{
				"page_id":   pageID,
				"title":     page.Title,
				"platforms": page.Platforms,
			}))
		return nil, fmt.Errorf("failed to publish page: %w", err)
	}

	// Record metrics for each platform
	for platformName, result := range results {
		if result.Success {
			s.monitoringService.RecordMetric("publish_success", "counter", 1, map[string]interface{}{
				"platform": platformName,
				"page_id":  pageID,
			})
		} else {
			s.monitoringService.RecordMetric("publish_failure", "counter", 1, map[string]interface{}{
				"platform": platformName,
				"page_id":  pageID,
			})
			// Record specific error
			if result.Error != nil {
				s.monitoringService.RecordError("ERROR", "publisher", fmt.Sprintf("Failed to publish to %s", platformName), result.Error.Error(),
					WithPlatform(platformName),
					WithPage(page.ID),
					WithContext(map[string]interface{}{
						"page_id": pageID,
						"title":   page.Title,
					}))
			}
		}
	}

	return results, nil
}

// PublishPageToPlatform publishes a page to a specific platform
func (s *PublisherService) PublishPageToPlatform(ctx context.Context, pageID string, platformName string) (*publisher.PublishResult, error) {
	// Get the page from database
	var page models.NotionPage
	if err := s.db.Where("notion_id = ?", pageID).First(&page).Error; err != nil {
		return nil, fmt.Errorf("page not found: %w", err)
	}

	// Check if page is ready for publishing
	if page.Status != "Done" {
		return nil, fmt.Errorf("page status is not 'Done', current status: %s", page.Status)
	}

	s.logger.Info("Publishing page to platform",
		zap.String("page_id", pageID),
		zap.String("title", page.Title),
		zap.String("platform", platformName))

	// Publish to specific platform
	result, err := s.manager.PublishSinglePlatform(ctx, &page, platformName, false)
	if err != nil {
		// Record error in monitoring
		s.monitoringService.RecordError("ERROR", "publisher", fmt.Sprintf("Failed to publish to platform %s", platformName), err.Error(),
			WithPlatform(platformName),
			WithPage(page.ID),
			WithContext(map[string]interface{}{
				"page_id": pageID,
				"title":   page.Title,
			}))
		return nil, fmt.Errorf("failed to publish to platform %s: %w", platformName, err)
	}

	// Record metrics
	if result.Success {
		s.monitoringService.RecordMetric("publish_success", "counter", 1, map[string]interface{}{
			"platform": platformName,
			"page_id":  pageID,
		})
	} else {
		s.monitoringService.RecordMetric("publish_failure", "counter", 1, map[string]interface{}{
			"platform": platformName,
			"page_id":  pageID,
		})
		if result.Error != nil {
			s.monitoringService.RecordError("ERROR", "publisher", fmt.Sprintf("Failed to publish to %s", platformName), result.Error.Error(),
				WithPlatform(platformName),
				WithPage(page.ID),
				WithContext(map[string]interface{}{
					"page_id": pageID,
					"title":   page.Title,
				}))
		}
	}

	return result, nil
}

// SavePageToDraft saves a page as draft to a specific platform
func (s *PublisherService) SavePageToDraft(ctx context.Context, pageID string, platformName string) (*publisher.PublishResult, error) {
	// Get the page from database
	var page models.NotionPage
	if err := s.db.Where("notion_id = ?", pageID).First(&page).Error; err != nil {
		return nil, fmt.Errorf("page not found: %w", err)
	}

	s.logger.Info("Saving page to draft",
		zap.String("page_id", pageID),
		zap.String("title", page.Title),
		zap.String("platform", platformName))

	// Save as draft
	result, err := s.manager.PublishSinglePlatform(ctx, &page, platformName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to save to draft on platform %s: %w", platformName, err)
	}

	return result, nil
}

// GetPublishHistory returns the publishing history for a page
func (s *PublisherService) GetPublishHistory(ctx context.Context, pageID string) ([]*models.DistributionJob, error) {
	return s.manager.GetPublishHistory(ctx, pageID)
}

// GetAvailablePlatforms returns all available publishing platforms
func (s *PublisherService) GetAvailablePlatforms() []string {
	publishers := s.manager.GetAvailablePublishers()
	var platforms []string

	for _, pub := range publishers {
		platforms = append(platforms, pub.GetPlatformName())
	}

	return platforms
}

// ProcessPendingPages processes all pages that are ready for publishing
func (s *PublisherService) ProcessPendingPages(ctx context.Context) error {
	// Find pages that are Done but haven't been fully published to all required platforms
	var pages []models.NotionPage

	// Get pages that are Done and either have no distribution jobs or have failed/pending jobs
	if err := s.db.Where("status = ?", "Done").
		Limit(10). // Process in batches
		Find(&pages).Error; err != nil {
		return fmt.Errorf("failed to get pending pages: %w", err)
	}

	// Filter pages that still need publishing
	var pendingPages []models.NotionPage
	for _, page := range pages {
		needsPublishing, err := s.needsPublishing(ctx, &page)
		if err != nil {
			s.logger.Error("Failed to check if page needs publishing",
				zap.String("page_id", page.NotionID),
				zap.Error(err))
			continue
		}
		if needsPublishing {
			pendingPages = append(pendingPages, page)
		}
	}

	pages = pendingPages

	s.logger.Info("Processing pending pages", zap.Int("count", len(pages)))

	for _, page := range pages {
		results, err := s.manager.PublishToAll(ctx, &page)
		if err != nil {
			s.logger.Error("Failed to publish page",
				zap.String("page_id", page.NotionID),
				zap.Error(err))
			continue
		}

		// Log results
		for platform, result := range results {
			s.logger.Info("Publish result",
				zap.String("page_id", page.NotionID),
				zap.String("platform", platform),
				zap.Bool("success", result.Success))
		}
	}

	return nil
}

// needsPublishing checks if a page needs publishing to any of its required platforms
func (s *PublisherService) needsPublishing(ctx context.Context, page *models.NotionPage) (bool, error) {
	// Get all distribution jobs for this page
	var jobs []models.DistributionJob
	if err := s.db.Preload("Platform").Where("page_id = ?", page.ID).Find(&jobs).Error; err != nil {
		return false, fmt.Errorf("failed to get distribution jobs: %w", err)
	}

	// Create a map of platform name to job status
	platformStatus := make(map[string]string)
	for _, job := range jobs {
		platformStatus[job.Platform.Name] = job.Status
	}

	// Check if all required platforms are completed
	for _, platformName := range page.Platforms {
		status, exists := platformStatus[platformName]
		if !exists || (status != "completed") {
			// Platform either has no job or job is not completed
			return true, nil
		}
	}

	return false, nil
}
