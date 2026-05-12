package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
	"wiki/internal/config"
	"wiki/internal/logger"
	"wiki/internal/server"
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Wiki service CLI",
}

func init() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(bootstrapCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		slog.SetDefault(logger.New(cfg.LogLevel))
		slog.Info("starting server", "address", cfg.ServerAddress, "port", cfg.ServerPort, "environment", cfg.Environment)

		return server.Start(cmd.Context(), server.Config{
			Address:     cfg.ServerAddress,
			Port:        cfg.ServerPort,
			JWTSecret:   cfg.JWTSecret,
			SSOLoginURL: cfg.SSOLoginURL,
			JWTCookie:   cfg.JWTCookie,
		})
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate [up|down|status]",
	Short: "Run database migrations",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		goose.SetDialect("sqlite3")
		switch args[0] {
		case "up":
			return goose.Up(db, "migrations")
		case "down":
			return goose.Down(db, "migrations")
		case "status":
			return goose.Status(db, "migrations")
		default:
			return fmt.Errorf("unsupported migrate command: %s", args[0])
		}
	},
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Create local data dir and apply migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll("data", 0o755); err != nil {
			return err
		}
		return migrateCmd.RunE(cmd, []string{"up"})
	},
}

func openDB() (*sql.DB, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("sqlite", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
