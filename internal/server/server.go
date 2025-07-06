package server

import (
	"context"
	"fmt"
	"github.com/ifuryst/ripple/internal/service/notion"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ifuryst/ripple/internal/config"
	"github.com/ifuryst/ripple/internal/service"
)

type Server struct {
	Config *config.Config
	DB     *gorm.DB
	Router *gin.Engine
	Logger *zap.Logger
	Server *http.Server

	// Services
	NotionService    *notion.Service
	PublisherService *service.PublisherService
	Scheduler        *service.Scheduler
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
	scheduler := service.NewScheduler(&cfg.Scheduler, logger, notionService, publisherService)

	// Create router
	router := gin.New()

	// Create server
	srv := &Server{
		Config:           cfg,
		DB:               db,
		Router:           router,
		Logger:           logger,
		NotionService:    notionService,
		PublisherService: publisherService,
		Scheduler:        scheduler,
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
	// Stop scheduler first
	s.Scheduler.Stop()

	if s.Server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return s.Server.Shutdown(shutdownCtx)
}
