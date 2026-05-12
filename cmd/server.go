package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"

	"wiki/internal/config"
	"wiki/internal/db"
	"wiki/internal/logger"
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

		return server.Start(cmd.Context(), server.Config{
			Address:        cfg.ServerAddress,
			Port:           cfg.ServerPort,
			Environment:    cfg.Environment,
			JWTSecret:      cfg.JWTSecret,
			SSOLoginURL:    cfg.SSOLoginURL,
			JWTCookie:      cfg.JWTCookie,
			Queries:        db.New(sqlDB),
			UploadStore:    uploadStore,
			UploadBucket:   cfg.S3Bucket,
			UploadMaxBytes: 10 << 20,
		})
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
