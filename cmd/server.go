package cmd

import (
	"context"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"wiki/internal/config"
	"wiki/internal/db"
	"wiki/internal/logger"
	"wiki/internal/monitoring"
	"wiki/internal/server"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the wiki HTTP server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		uploadStore, err := server.NewS3ObjectStore(
			cfg.S3Region,
			cfg.S3Endpoint,
			cfg.S3AccessKeyID,
			cfg.S3SecretAccessKey,
			cfg.S3ForcePathStyle,
		)
		if err != nil {
			return err
		}

		sqlDB, err := openDB()
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		slog.SetDefault(logger.New(cfg.LogLevel))
		slog.Info(
			"starting server",
			"address", cfg.ServerAddress,
			"port", cfg.ServerPort,
			"environment", cfg.Environment,
		)

		var reportError func(ctx context.Context, err error, fields map[string]any)
		if cfg.ErrorMonitoringEnabled {
			if strings.TrimSpace(cfg.HoneybadgerAPIKey) == "" {
				slog.Warn("error monitoring enabled but WIKI_HONEYBADGER_API_KEY is not set")
			} else {
				monitoring.ConfigureHoneybadger(cfg.HoneybadgerAPIKey, cfg.Environment)
				reportError = monitoring.NotifyError
				slog.Info("honeybadger error monitoring enabled")
			}
		}

		return server.Start(cmd.Context(), server.Config{
			Address:        cfg.ServerAddress,
			Port:           cfg.ServerPort,
			Environment:    cfg.Environment,
			Branding:       cfg.Branding,
			LogoURL:        cfg.LogoURL,
			LogoHeroURL:    cfg.LogoHeroURL,
			FaviconURL:     cfg.FaviconURL,
			JWTSecret:      cfg.JWTSecret,
			SSOLoginURL:    cfg.SSOLoginURL,
			ServerURL:      cfg.ServerURL,
			JWTCookie:      cfg.JWTCookie,
			Queries:        db.New(sqlDB),
			UploadStore:    uploadStore,
			UploadBucket:   cfg.S3Bucket,
			UploadMaxBytes: 10 << 20,
			ReportError:    reportError,
		})
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
