package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ifuryst/ripple/internal/config"
	"github.com/ifuryst/ripple/internal/models"
	"github.com/ifuryst/ripple/internal/service"
	"github.com/ifuryst/ripple/internal/service/notion"
)

type Server struct {
	Config *config.Config
	DB     *gorm.DB
	Router *gin.Engine
	Logger *zap.Logger
	Server *http.Server

	// Services
	NotionService      *notion.Service
	PublisherService   *service.PublisherService
	MonitoringService  *service.MonitoringService
	StatsUpdater       *service.StatsUpdater
	Scheduler          *service.Scheduler
}

func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// Set gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database
	db, err := service.NewDatabase(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize services
	notionService := notion.NewService(&cfg.Notion, db, logger)
	publisherService := service.NewPublisherService(cfg, db, logger)
	monitoringService := service.NewMonitoringService(db, logger)
	statsUpdater := service.NewStatsUpdater(monitoringService, logger, 15*time.Minute) // Update every 15 minutes
	scheduler := service.NewScheduler(&cfg.Scheduler, logger, notionService, publisherService)

	// Create router
	router := gin.New()

	// Create server
	srv := &Server{
		Config:            cfg,
		DB:                db,
		Router:            router,
		Logger:            logger,
		NotionService:     notionService,
		PublisherService:  publisherService,
		MonitoringService: monitoringService,
		StatsUpdater:      statsUpdater,
		Scheduler:         scheduler,
	}

	// Setup middleware and routes
	srv.setupMiddleware()
	srv.setupRoutes()

	return srv, nil
}

func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.Router.Use(gin.Recovery())

	// Logger middleware
	s.Router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
				param.ClientIP,
				param.TimeStamp.Format(time.RFC3339),
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.UserAgent(),
				param.ErrorMessage,
			)
		},
	}))

	// CORS middleware
	s.Router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
}

func (s *Server) setupRoutes() {
	// Serve static files for dashboard
	s.Router.Static("/assets", "./web/dist/assets")
	s.Router.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	
	// Serve dashboard index.html for root path
	s.Router.GET("/", func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
	
	// Serve dashboard for SPA routes (overview, platforms, trends, errors)
	dashboardRoutes := []string{"/overview", "/platforms", "/trends", "/errors"}
	for _, route := range dashboardRoutes {
		s.Router.GET(route, func(c *gin.Context) {
			c.File("./web/dist/index.html")
		})
	}
	
	// Serve dashboard for any other route that doesn't start with /api
	s.Router.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.File("./web/dist/index.html")
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
		}
	})

	// Health check
	s.Router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	// API routes
	api := s.Router.Group("/api/v1")
	{
		// Notion routes
		notion := api.Group("/notion")
		{
			notion.GET("/pages", s.handleGetNotionPages)
			notion.POST("/sync", s.handleSyncNotionPages)
		}

		// Publisher routes
		publisher := api.Group("/publisher")
		{
			publisher.GET("/platforms", s.handleGetPlatforms)
			publisher.POST("/publish/:pageId", s.handlePublishPage)
			publisher.POST("/publish/:pageId/:platform", s.handlePublishPageToPlatform)
			publisher.POST("/draft/:pageId/:platform", s.handleSavePageToDraft)
			publisher.GET("/history/:pageId", s.handleGetPublishHistory)
			publisher.POST("/process-pending", s.handleProcessPendingPages)
		}

		// Dashboard routes
		dashboard := api.Group("/dashboard")
		{
			dashboard.GET("/summary", s.handleGetDashboardSummary)
			dashboard.GET("/platform-stats", s.handleGetPlatformStats)
			dashboard.GET("/recent-errors", s.handleGetRecentErrors)
			dashboard.GET("/system-stats", s.handleGetSystemStats)
			dashboard.GET("/recent-pages", s.handleGetRecentPages)
			dashboard.GET("/recent-jobs", s.handleGetRecentJobs)
			dashboard.GET("/jobs", s.handleGetJobs)
			dashboard.POST("/update-stats", s.handleUpdateStats)
			dashboard.POST("/resolve-error/:errorId", s.handleResolveError)
		}
	}
}

func (s *Server) handleGetNotionPages(c *gin.Context) {
	pages, err := s.NotionService.GetAllPages()
	if err != nil {
		s.Logger.Error("Failed to get notion pages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pages": pages})
}

func (s *Server) handleSyncNotionPages(c *gin.Context) {
	err := s.NotionService.SyncPages()
	if err != nil {
		s.Logger.Error("Failed to sync notion pages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync pages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sync completed successfully"})
}

func (s *Server) handleGetPlatforms(c *gin.Context) {
	platforms := s.PublisherService.GetAvailablePlatforms()
	c.JSON(http.StatusOK, gin.H{"platforms": platforms})
}

func (s *Server) handlePublishPage(c *gin.Context) {
	pageID := c.Param("pageId")
	if pageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Page ID is required"})
		return
	}

	results, err := s.PublisherService.PublishPage(c.Request.Context(), pageID)
	if err != nil {
		s.Logger.Error("Failed to publish page", zap.String("page_id", pageID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Page published successfully",
		"results": results,
	})
}

func (s *Server) handlePublishPageToPlatform(c *gin.Context) {
	pageID := c.Param("pageId")
	platform := c.Param("platform")

	if pageID == "" || platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Page ID and platform are required"})
		return
	}

	result, err := s.PublisherService.PublishPageToPlatform(c.Request.Context(), pageID, platform)
	if err != nil {
		s.Logger.Error("Failed to publish page to platform",
			zap.String("page_id", pageID),
			zap.String("platform", platform),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Page published to platform successfully",
		"result":  result,
	})
}

func (s *Server) handleSavePageToDraft(c *gin.Context) {
	pageID := c.Param("pageId")
	platform := c.Param("platform")

	if pageID == "" || platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Page ID and platform are required"})
		return
	}

	result, err := s.PublisherService.SavePageToDraft(c.Request.Context(), pageID, platform)
	if err != nil {
		s.Logger.Error("Failed to save page to draft",
			zap.String("page_id", pageID),
			zap.String("platform", platform),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Page saved to draft successfully",
		"result":  result,
	})
}

func (s *Server) handleGetPublishHistory(c *gin.Context) {
	pageID := c.Param("pageId")
	if pageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Page ID is required"})
		return
	}

	history, err := s.PublisherService.GetPublishHistory(c.Request.Context(), pageID)
	if err != nil {
		s.Logger.Error("Failed to get publish history", zap.String("page_id", pageID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

func (s *Server) handleProcessPendingPages(c *gin.Context) {
	err := s.PublisherService.ProcessPendingPages(c.Request.Context())
	if err != nil {
		s.Logger.Error("Failed to process pending pages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pending pages processed successfully"})
}

func (s *Server) Start(ctx context.Context) error {
	// Start stats updater
	s.StatsUpdater.Start(ctx)

	// Start scheduler
	if err := s.Scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", s.Config.Server.Host, s.Config.Server.Port)

	s.Server = &http.Server{
		Addr:    addr,
		Handler: s.Router,
	}

	s.Logger.Info("Starting HTTP server", zap.String("addr", addr))

	if s.Config.Server.CertFile != "" && s.Config.Server.KeyFile != "" {
		return s.Server.ListenAndServeTLS(s.Config.Server.CertFile, s.Config.Server.KeyFile)
	}

	return s.Server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Stop stats updater first
	s.StatsUpdater.Stop()

	// Stop scheduler
	s.Scheduler.Stop()

	if s.Server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.Server.Shutdown(shutdownCtx)
}

// Dashboard handlers

func (s *Server) handleGetDashboardSummary(c *gin.Context) {
	summary, err := s.MonitoringService.GetDashboardSummary()
	if err != nil {
		s.Logger.Error("Failed to get dashboard summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

func (s *Server) handleGetPlatformStats(c *gin.Context) {
	daysParam := c.DefaultQuery("days", "7")
	days := 7
	if d, err := strconv.Atoi(daysParam); err == nil && d > 0 {
		days = d
	}

	stats, err := s.MonitoringService.GetPlatformStats(days)
	if err != nil {
		s.Logger.Error("Failed to get platform stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get platform stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (s *Server) handleGetRecentErrors(c *gin.Context) {
	limitParam := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
		limit = l
	}

	errors, err := s.MonitoringService.GetRecentErrors(limit)
	if err != nil {
		s.Logger.Error("Failed to get recent errors", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent errors"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"errors": errors})
}

func (s *Server) handleGetSystemStats(c *gin.Context) {
	daysParam := c.DefaultQuery("days", "7")
	days := 7
	if d, err := strconv.Atoi(daysParam); err == nil && d > 0 {
		days = d
	}

	var stats []models.SystemStats
	startDate := time.Now().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	
	err := s.DB.Where("date >= ?", startDate).Order("date desc").Find(&stats).Error
	if err != nil {
		s.Logger.Error("Failed to get system stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get system stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

func (s *Server) handleUpdateStats(c *gin.Context) {
	// 更新系统统计
	if err := s.MonitoringService.UpdateSystemStats(); err != nil {
		s.Logger.Error("Failed to update system stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update system stats"})
		return
	}

	// 更新平台统计
	if err := s.MonitoringService.UpdatePlatformStats(); err != nil {
		s.Logger.Error("Failed to update platform stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update platform stats"})
		return
	}

	// 更新仪表板摘要
	if err := s.MonitoringService.UpdateDashboardSummary(); err != nil {
		s.Logger.Error("Failed to update dashboard summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update dashboard summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stats updated successfully"})
}

func (s *Server) handleResolveError(c *gin.Context) {
	errorIDParam := c.Param("errorId")
	errorID, err := strconv.ParseUint(errorIDParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid error ID"})
		return
	}

	now := time.Now()
	err = s.DB.Model(&models.ErrorLog{}).Where("id = ?", uint(errorID)).Updates(map[string]interface{}{
		"resolved":    true,
		"resolved_at": &now,
	}).Error

	if err != nil {
		s.Logger.Error("Failed to resolve error", zap.Uint64("error_id", errorID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Error resolved successfully"})
}

func (s *Server) handleGetRecentPages(c *gin.Context) {
	limitParam := c.DefaultQuery("limit", "5")
	limit := 5
	if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
		limit = l
	}

	var pages []models.NotionPage
	err := s.DB.Order("updated_at desc").Limit(limit).Find(&pages).Error
	if err != nil {
		s.Logger.Error("Failed to get recent pages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent pages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pages": pages})
}

func (s *Server) handleGetRecentJobs(c *gin.Context) {
	limitParam := c.DefaultQuery("limit", "5")
	limit := 5
	if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
		limit = l
	}

	var jobs []models.DistributionJob
	err := s.DB.Preload("Page").Preload("Platform").
		Order("updated_at desc").
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		s.Logger.Error("Failed to get recent jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (s *Server) handleGetJobs(c *gin.Context) {
	limitParam := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
		limit = l
	}

	offsetParam := c.DefaultQuery("offset", "0")
	offset := 0
	if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
		offset = o
	}

	status := c.Query("status") // pending, completed, failed

	query := s.DB.Preload("Page").Preload("Platform")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var jobs []models.DistributionJob
	var total int64
	
	// Get total count
	countQuery := s.DB.Model(&models.DistributionJob{})
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	countQuery.Count(&total)

	err := query.Order("updated_at desc").
		Offset(offset).
		Limit(limit).
		Find(&jobs).Error
	if err != nil {
		s.Logger.Error("Failed to get jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
