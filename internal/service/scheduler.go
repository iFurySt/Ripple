package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/ifuryst/ripple/internal/config"
)

type Scheduler struct {
	config        *config.SchedulerConfig
	logger        *zap.Logger
	notionService *NotionService
	ticker        *time.Ticker
	stopCh        chan struct{}
}

func NewScheduler(cfg *config.SchedulerConfig, logger *zap.Logger, notionService *NotionService) *Scheduler {
	return &Scheduler{
		config:        cfg,
		logger:        logger,
		notionService: notionService,
		stopCh:        make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Scheduler is disabled")
		return nil
	}

	interval, err := time.ParseDuration(s.config.SyncInterval)
	if err != nil {
		s.logger.Error("Invalid sync interval", zap.String("interval", s.config.SyncInterval), zap.Error(err))
		return err
	}

	s.logger.Info("Starting scheduler", zap.String("sync_interval", s.config.SyncInterval))

	s.ticker = time.NewTicker(interval)

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
	err := s.notionService.SyncPages()
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("Sync failed", 
			zap.Error(err), 
			zap.Duration("duration", duration))
		return err
	}

	s.logger.Info("Sync completed successfully", 
		zap.Duration("duration", duration))
	return nil
}