package service

import (
	"context"
	"github.com/ifuryst/ripple/internal/service/notion"
	"time"

	"go.uber.org/zap"

	"github.com/ifuryst/ripple/internal/config"
)

type Scheduler struct {
	config           *config.SchedulerConfig
	logger           *zap.Logger
	notionService    *notion.Service
	publisherService *PublisherService
	ticker           *time.Ticker
	stopCh           chan struct{}
}

func NewScheduler(cfg *config.SchedulerConfig, logger *zap.Logger, notionService *notion.Service, publisherService *PublisherService) *Scheduler {
	return &Scheduler{
		config:           cfg,
		logger:           logger,
		notionService:    notionService,
		publisherService: publisherService,
		stopCh:           make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Scheduler is disabled")
		return nil
	}

	s.logger.Info("Starting scheduler", zap.String("sync_interval", s.config.SyncInterval.String()))

	s.ticker = time.NewTicker(s.config.SyncInterval)

	// Run first sync immediately
	go func() {
		s.logger.Info("Running initial sync")
		if err := s.runSync(); err != nil {
			s.logger.Error("Initial sync failed", zap.Error(err))
		}
	}()

	// Start periodic sync
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.logger.Info("Running scheduled sync")
				if err := s.runSync(); err != nil {
					s.logger.Error("Scheduled sync failed", zap.Error(err))
				}
			case <-s.stopCh:
				s.logger.Info("Scheduler stopped")
				return
			case <-ctx.Done():
				s.logger.Info("Scheduler context cancelled")
				return
			}
		}
	}()

	return nil
}

func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopCh)
	s.logger.Info("Scheduler shutdown completed")
}

func (s *Scheduler) runSync() error {
	start := time.Now()

	// First sync pages from Notion
	err := s.notionService.SyncPages()
	if err != nil {
		syncDuration := time.Since(start)
		s.logger.Error("Notion sync failed",
			zap.Error(err),
			zap.Duration("sync_duration", syncDuration))
		return err
	}

	syncDuration := time.Since(start)
	s.logger.Info("Notion sync completed successfully",
		zap.Duration("sync_duration", syncDuration))

	// Then process pending pages for publishing
	publishStart := time.Now()
	if s.publisherService != nil {
		err = s.publisherService.ProcessPendingPages(context.Background())
		publishDuration := time.Since(publishStart)

		if err != nil {
			s.logger.Error("Publishing pending pages failed",
				zap.Error(err),
				zap.Duration("publish_duration", publishDuration))
			// Don't return error here - sync was successful, just publishing failed
		} else {
			s.logger.Info("Publishing pending pages completed successfully",
				zap.Duration("publish_duration", publishDuration))
		}
	}

	totalDuration := time.Since(start)
	s.logger.Info("Full sync and publish cycle completed",
		zap.Duration("total_duration", totalDuration))
	return nil
}
