package main

import (
	"context"
	"fmt"
	yamlenv "github.com/ifuryst/go-yaml-env"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ifuryst/ripple/internal/config"
	"github.com/ifuryst/ripple/internal/server"
	"github.com/ifuryst/ripple/pkg/logger"
)

var (
	configPath string
	version    = "0.1.0"
	gitCommit  = "unknown"
	buildTime  = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "ripple",
	Short: "Ripple - Content automation distribution tool",
	Long:  `Ripple processes structured notes from Notion and automatically distributes them to multiple platforms.`,
	RunE:  runServer,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Ripple %s\n", version)
		fmt.Printf("Git commit: %s\n", gitCommit)
		fmt.Printf("Build time: %s\n", buildTime)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "configs/server.yaml", "config file path")
	rootCmd.AddCommand(versionCmd)
}

func runServer(*cobra.Command, []string) error {
	// Load configuration
	cfg, err := yamlenv.LoadConfig[config.Config](configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	appLogger, err := logger.NewLogger(cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer appLogger.Sync()

	appLogger.Info("Starting Ripple server", zap.String("version", version))

	// Create server
	srv, err := server.NewServer(cfg, appLogger)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := srv.Start(ctx); err != nil {
			appLogger.Error("Server failed to start", zap.Error(err))
			cancel()
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		appLogger.Info("Shutting down server...")
	case <-ctx.Done():
		appLogger.Info("Server context cancelled")
	}

	// Graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	appLogger.Info("Server exited")
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
