package models

import (
	"time"
)

// SystemStats 系统整体统计信息
type SystemStats struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Date                  time.Time `gorm:"uniqueIndex;not null" json:"date"` // 按日统计
	TotalNotionPages      int       `gorm:"default:0" json:"total_notion_pages"`
	TotalDistributionJobs int       `gorm:"default:0" json:"total_distribution_jobs"`
	SuccessfulJobs        int       `gorm:"default:0" json:"successful_jobs"`
	FailedJobs            int       `gorm:"default:0" json:"failed_jobs"`
	PendingJobs           int       `gorm:"default:0" json:"pending_jobs"`
	TotalPlatforms        int       `gorm:"default:0" json:"total_platforms"`
	ActivePlatforms       int       `gorm:"default:0" json:"active_platforms"`
	CreatedAt             time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// PlatformStats 平台级别统计信息
type PlatformStats struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Date             time.Time `gorm:"index;not null" json:"date"`
	PlatformID       uint      `gorm:"not null;index" json:"platform_id"`
	PlatformName     string    `gorm:"size:100;not null" json:"platform_name"`
	TotalJobs        int       `gorm:"default:0" json:"total_jobs"`
	SuccessfulJobs   int       `gorm:"default:0" json:"successful_jobs"`
	FailedJobs       int       `gorm:"default:0" json:"failed_jobs"`
	PendingJobs      int       `gorm:"default:0" json:"pending_jobs"`
	AvgProcessTime   float64   `gorm:"default:0" json:"avg_process_time"` // 平均处理时间(秒)
	LastSuccessAt    *time.Time `json:"last_success_at"`
	LastFailureAt    *time.Time `json:"last_failure_at"`
	ErrorCount       int       `gorm:"default:0" json:"error_count"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Platform Platform `gorm:"foreignKey:PlatformID" json:"platform"`
}

// ErrorLog 错误日志表
type ErrorLog struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Level        string     `gorm:"size:20;not null;index" json:"level"`          // ERROR, WARN, INFO
	Source       string     `gorm:"size:100;not null;index" json:"source"`        // notion, publisher, scheduler等
	PlatformName string     `gorm:"size:100;index" json:"platform_name"`          // 平台名称(如果是平台相关错误)
	PageID       *uint      `gorm:"index" json:"page_id"`                         // 相关的页面ID
	JobID        *uint      `gorm:"index" json:"job_id"`                          // 相关的任务ID
	Title        string     `gorm:"size:500;not null" json:"title"`               // 错误标题
	Message      string     `gorm:"type:text;not null" json:"message"`            // 错误信息
	StackTrace   string     `gorm:"type:text" json:"stack_trace"`                 // 堆栈信息
	Context      string     `gorm:"type:jsonb" json:"context"`                    // 额外上下文信息
	Resolved     bool       `gorm:"default:false;index" json:"resolved"`          // 是否已解决
	ResolvedAt   *time.Time `json:"resolved_at"`
	CreatedAt    time.Time  `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	Page *NotionPage     `gorm:"foreignKey:PageID" json:"page,omitempty"`
	Job  *DistributionJob `gorm:"foreignKey:JobID" json:"job,omitempty"`
}

// MetricsSample 指标采样数据
type MetricsSample struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	MetricName string    `gorm:"size:100;not null;index" json:"metric_name"`  // 指标名称
	MetricType string    `gorm:"size:50;not null" json:"metric_type"`         // gauge, counter, histogram
	Value      float64   `gorm:"not null" json:"value"`                       // 指标值
	Tags       string    `gorm:"type:jsonb" json:"tags"`                      // 标签信息
	Timestamp  time.Time `gorm:"not null;index" json:"timestamp"`             // 采样时间戳
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// DashboardSummary 仪表板汇总信息 (用于快速查询)
type DashboardSummary struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	TotalPages             int       `gorm:"default:0" json:"total_pages"`
	TotalJobsToday         int       `gorm:"default:0" json:"total_jobs_today"`
	SuccessfulJobsToday    int       `gorm:"default:0" json:"successful_jobs_today"`
	FailedJobsToday        int       `gorm:"default:0" json:"failed_jobs_today"`
	PendingJobsCount       int       `gorm:"default:0" json:"pending_jobs_count"`
	ActivePlatformsCount   int       `gorm:"default:0" json:"active_platforms_count"`
	TotalPlatformsCount    int       `gorm:"default:0" json:"total_platforms_count"`
	LastSyncTime           *time.Time `json:"last_sync_time"`
	LastPublishTime        *time.Time `json:"last_publish_time"`
	UnresolvedErrorsCount  int       `gorm:"default:0" json:"unresolved_errors_count"`
	AvgProcessTimeToday    float64   `gorm:"default:0" json:"avg_process_time_today"`
	UpdatedAt              time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}