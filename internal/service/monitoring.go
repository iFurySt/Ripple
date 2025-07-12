package service

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ifuryst/ripple/internal/models"
)

type MonitoringService struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewMonitoringService(db *gorm.DB, logger *zap.Logger) *MonitoringService {
	return &MonitoringService{
		db:     db,
		logger: logger,
	}
}

// RecordError 记录错误日志
func (m *MonitoringService) RecordError(level, source, title, message string, options ...ErrorLogOption) error {
	errorLog := &models.ErrorLog{
		Level:   level,
		Source:  source,
		Title:   title,
		Message: message,
	}

	// 应用选项
	for _, option := range options {
		option(errorLog)
	}

	return m.db.Create(errorLog).Error
}

// ErrorLogOption 错误日志选项
type ErrorLogOption func(*models.ErrorLog)

// WithPlatform 设置平台名称
func WithPlatform(platformName string) ErrorLogOption {
	return func(e *models.ErrorLog) {
		e.PlatformName = platformName
	}
}

// WithPage 设置页面ID
func WithPage(pageID uint) ErrorLogOption {
	return func(e *models.ErrorLog) {
		e.PageID = &pageID
	}
}

// WithJob 设置任务ID
func WithJob(jobID uint) ErrorLogOption {
	return func(e *models.ErrorLog) {
		e.JobID = &jobID
	}
}

// WithStackTrace 设置堆栈信息
func WithStackTrace(stackTrace string) ErrorLogOption {
	return func(e *models.ErrorLog) {
		e.StackTrace = stackTrace
	}
}

// WithContext 设置上下文信息
func WithContext(context map[string]interface{}) ErrorLogOption {
	return func(e *models.ErrorLog) {
		if contextBytes, err := json.Marshal(context); err == nil {
			e.Context = string(contextBytes)
		}
	}
}

// UpdateSystemStats 更新系统统计数据
func (m *MonitoringService) UpdateSystemStats() error {
	today := time.Now().Truncate(24 * time.Hour)

	var stats models.SystemStats
	result := m.db.Where("date = ?", today).First(&stats)

	// 查询各种统计数据
	var totalPages int64
	m.db.Model(&models.NotionPage{}).Count(&totalPages)

	var totalJobs, successfulJobs, failedJobs, pendingJobs int64
	m.db.Model(&models.DistributionJob{}).Count(&totalJobs)
	m.db.Model(&models.DistributionJob{}).Where("status = ?", "completed").Count(&successfulJobs)
	m.db.Model(&models.DistributionJob{}).Where("status = ?", "failed").Count(&failedJobs)
	m.db.Model(&models.DistributionJob{}).Where("status = ?", "pending").Count(&pendingJobs)

	var totalPlatforms, activePlatforms int64
	m.db.Model(&models.Platform{}).Count(&totalPlatforms)
	m.db.Model(&models.Platform{}).Where("enabled = ?", true).Count(&activePlatforms)

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		stats = models.SystemStats{
			Date:                  today,
			TotalNotionPages:      int(totalPages),
			TotalDistributionJobs: int(totalJobs),
			SuccessfulJobs:        int(successfulJobs),
			FailedJobs:            int(failedJobs),
			PendingJobs:           int(pendingJobs),
			TotalPlatforms:        int(totalPlatforms),
			ActivePlatforms:       int(activePlatforms),
		}
		return m.db.Create(&stats).Error
	} else {
		// 更新现有记录
		return m.db.Model(&stats).Updates(map[string]interface{}{
			"total_notion_pages":      totalPages,
			"total_distribution_jobs": totalJobs,
			"successful_jobs":         successfulJobs,
			"failed_jobs":             failedJobs,
			"pending_jobs":            pendingJobs,
			"total_platforms":         totalPlatforms,
			"active_platforms":        activePlatforms,
		}).Error
	}
}

// UpdatePlatformStats 更新平台统计数据
func (m *MonitoringService) UpdatePlatformStats() error {
	today := time.Now().Truncate(24 * time.Hour)

	var platforms []models.Platform
	if err := m.db.Find(&platforms).Error; err != nil {
		return err
	}

	for _, platform := range platforms {
		var stats models.PlatformStats
		result := m.db.Where("date = ? AND platform_id = ?", today, platform.ID).First(&stats)

		// 查询平台相关统计
		var totalJobs, successfulJobs, failedJobs, pendingJobs int64
		m.db.Model(&models.DistributionJob{}).Where("platform_id = ?", platform.ID).Count(&totalJobs)
		m.db.Model(&models.DistributionJob{}).Where("platform_id = ? AND status = ?", platform.ID, "completed").Count(&successfulJobs)
		m.db.Model(&models.DistributionJob{}).Where("platform_id = ? AND status = ?", platform.ID, "failed").Count(&failedJobs)
		m.db.Model(&models.DistributionJob{}).Where("platform_id = ? AND status = ?", platform.ID, "pending").Count(&pendingJobs)

		// 计算平均处理时间 (这里需要根据实际业务逻辑调整)
		var avgProcessTime float64
		// TODO: 实现平均处理时间计算

		// 获取最后成功和失败时间
		var lastSuccessJob, lastFailureJob models.DistributionJob
		m.db.Where("platform_id = ? AND status = ?", platform.ID, "completed").Order("published_at desc").First(&lastSuccessJob)
		m.db.Where("platform_id = ? AND status = ?", platform.ID, "failed").Order("updated_at desc").First(&lastFailureJob)

		// 计算错误数量
		var errorCount int64
		m.db.Model(&models.ErrorLog{}).Where("platform_name = ? AND created_at >= ?", platform.Name, today).Count(&errorCount)

		if result.Error == gorm.ErrRecordNotFound {
			// 创建新记录
			stats = models.PlatformStats{
				Date:           today,
				PlatformID:     platform.ID,
				PlatformName:   platform.Name,
				TotalJobs:      int(totalJobs),
				SuccessfulJobs: int(successfulJobs),
				FailedJobs:     int(failedJobs),
				PendingJobs:    int(pendingJobs),
				AvgProcessTime: avgProcessTime,
				ErrorCount:     int(errorCount),
			}

			if lastSuccessJob.ID != 0 {
				stats.LastSuccessAt = lastSuccessJob.PublishedAt
			}
			if lastFailureJob.ID != 0 {
				stats.LastFailureAt = &lastFailureJob.UpdatedAt
			}

			if err := m.db.Create(&stats).Error; err != nil {
				return err
			}
		} else {
			// 更新现有记录
			updates := map[string]interface{}{
				"total_jobs":      totalJobs,
				"successful_jobs": successfulJobs,
				"failed_jobs":     failedJobs,
				"pending_jobs":    pendingJobs,
				"avg_process_time": avgProcessTime,
				"error_count":     errorCount,
			}

			if lastSuccessJob.ID != 0 {
				updates["last_success_at"] = lastSuccessJob.PublishedAt
			}
			if lastFailureJob.ID != 0 {
				updates["last_failure_at"] = lastFailureJob.UpdatedAt
			}

			if err := m.db.Model(&stats).Updates(updates).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// UpdateDashboardSummary 更新仪表板摘要数据
func (m *MonitoringService) UpdateDashboardSummary() error {
	today := time.Now().Truncate(24 * time.Hour)

	var summary models.DashboardSummary
	result := m.db.First(&summary)

	// 查询各种统计数据
	var totalPages int64
	m.db.Model(&models.NotionPage{}).Count(&totalPages)

	var totalJobsToday, successfulJobsToday, failedJobsToday int64
	m.db.Model(&models.DistributionJob{}).Where("created_at >= ?", today).Count(&totalJobsToday)
	m.db.Model(&models.DistributionJob{}).Where("created_at >= ? AND status = ?", today, "completed").Count(&successfulJobsToday)
	m.db.Model(&models.DistributionJob{}).Where("created_at >= ? AND status = ?", today, "failed").Count(&failedJobsToday)

	var pendingJobsCount int64
	m.db.Model(&models.DistributionJob{}).Where("status = ?", "pending").Count(&pendingJobsCount)

	var activePlatformsCount, totalPlatformsCount int64
	m.db.Model(&models.Platform{}).Where("enabled = ?", true).Count(&activePlatformsCount)
	m.db.Model(&models.Platform{}).Count(&totalPlatformsCount)

	// 获取最后同步时间和发布时间
	var lastSyncPage models.NotionPage
	var lastPublishJob models.DistributionJob
	m.db.Order("updated_at desc").First(&lastSyncPage)
	m.db.Where("status = ?", "completed").Order("published_at desc").First(&lastPublishJob)

	// 未解决错误数量
	var unresolvedErrorsCount int64
	m.db.Model(&models.ErrorLog{}).Where("resolved = ?", false).Count(&unresolvedErrorsCount)

	// 今日平均处理时间
	var avgProcessTimeToday float64
	// TODO: 实现今日平均处理时间计算

	summaryData := models.DashboardSummary{
		TotalPages:             int(totalPages),
		TotalJobsToday:         int(totalJobsToday),
		SuccessfulJobsToday:    int(successfulJobsToday),
		FailedJobsToday:        int(failedJobsToday),
		PendingJobsCount:       int(pendingJobsCount),
		ActivePlatformsCount:   int(activePlatformsCount),
		TotalPlatformsCount:    int(totalPlatformsCount),
		UnresolvedErrorsCount:  int(unresolvedErrorsCount),
		AvgProcessTimeToday:    avgProcessTimeToday,
	}

	if lastSyncPage.ID != 0 {
		summaryData.LastSyncTime = &lastSyncPage.UpdatedAt
	}
	if lastPublishJob.ID != 0 {
		summaryData.LastPublishTime = lastPublishJob.PublishedAt
	}

	if result.Error == gorm.ErrRecordNotFound {
		// 创建新记录
		summaryData.ID = 1 // 确保只有一条记录
		return m.db.Create(&summaryData).Error
	} else {
		// 更新现有记录
		return m.db.Model(&summary).Updates(summaryData).Error
	}
}

// RecordMetric 记录指标数据
func (m *MonitoringService) RecordMetric(name, metricType string, value float64, tags map[string]interface{}) error {
	var tagsJSON string
	if tags != nil {
		if tagsBytes, err := json.Marshal(tags); err == nil {
			tagsJSON = string(tagsBytes)
		}
	}

	metric := &models.MetricsSample{
		MetricName: name,
		MetricType: metricType,
		Value:      value,
		Tags:       tagsJSON,
		Timestamp:  time.Now(),
	}

	return m.db.Create(metric).Error
}

// GetDashboardSummary 获取仪表板摘要数据
func (m *MonitoringService) GetDashboardSummary() (*models.DashboardSummary, error) {
	var summary models.DashboardSummary
	if err := m.db.First(&summary).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果没有记录，则创建一个空的摘要
			if err := m.UpdateDashboardSummary(); err != nil {
				return nil, err
			}
			return m.GetDashboardSummary()
		}
		return nil, err
	}
	return &summary, nil
}

// GetRecentErrors 获取最近的错误日志
func (m *MonitoringService) GetRecentErrors(limit int) ([]models.ErrorLog, error) {
	var errors []models.ErrorLog
	err := m.db.Preload("Page").Preload("Job").
		Order("created_at desc").
		Limit(limit).
		Find(&errors).Error
	return errors, err
}

// GetPlatformStats 获取平台统计数据
func (m *MonitoringService) GetPlatformStats(days int) ([]models.PlatformStats, error) {
	var stats []models.PlatformStats
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)

	err := m.db.Preload("Platform").
		Where("date >= ?", startDate).
		Order("date desc, platform_name").
		Find(&stats).Error
	return stats, err
}

// CleanupOldData 清理旧数据
func (m *MonitoringService) CleanupOldData(daysToKeep int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	// 清理旧的指标数据
	if err := m.db.Where("timestamp < ?", cutoffDate).Delete(&models.MetricsSample{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup metrics samples: %w", err)
	}

	// 清理旧的系统统计数据
	if err := m.db.Where("date < ?", cutoffDate).Delete(&models.SystemStats{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup system stats: %w", err)
	}

	// 清理旧的平台统计数据
	if err := m.db.Where("date < ?", cutoffDate).Delete(&models.PlatformStats{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup platform stats: %w", err)
	}

	// 清理已解决的旧错误日志
	if err := m.db.Where("created_at < ? AND resolved = ?", cutoffDate, true).Delete(&models.ErrorLog{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup resolved errors: %w", err)
	}

	return nil
}