package service

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// StatsUpdater handles periodic statistics updates
type StatsUpdater struct {
	monitoringService *MonitoringService
	logger            *zap.Logger
	ticker            *time.Ticker
	done              chan bool
}

// NewStatsUpdater creates a new stats updater
func NewStatsUpdater(monitoringService *MonitoringService, logger *zap.Logger, interval time.Duration) *StatsUpdater {
	return &StatsUpdater{
		monitoringService: monitoringService,
		logger:            logger,
		ticker:            time.NewTicker(interval),
		done:              make(chan bool),
	}
}

// Start begins the periodic stats update process
func (s *StatsUpdater) Start(ctx context.Context) {
	go func() {
		s.logger.Info("Starting stats updater")
		for {
			select {
			case <-s.done:
				s.logger.Info("Stats updater stopped")
				return
			case <-ctx.Done():
				s.logger.Info("Stats updater stopped due to context cancellation")
				return
			case <-s.ticker.C:
				s.updateStats()
			}
		}
	}()
}

// Stop stops the stats updater
func (s *StatsUpdater) Stop() {
	s.ticker.Stop()
	close(s.done)
}

// updateStats performs the actual stats update
func (s *StatsUpdater) updateStats() {
	s.logger.Debug("Updating statistics")

	// Update system stats
	if err := s.monitoringService.UpdateSystemStats(); err != nil {
		s.logger.Error("Failed to update system stats", zap.Error(err))
	}

	// Update platform stats
	if err := s.monitoringService.UpdatePlatformStats(); err != nil {
		s.logger.Error("Failed to update platform stats", zap.Error(err))
	}

	// Update dashboard summary
	if err := s.monitoringService.UpdateDashboardSummary(); err != nil {
		s.logger.Error("Failed to update dashboard summary", zap.Error(err))
	}

	// Clean up old data (keep last 90 days)
	if err := s.monitoringService.CleanupOldData(90); err != nil {
		s.logger.Error("Failed to cleanup old data", zap.Error(err))
	}

	s.logger.Debug("Statistics updated successfully")
}