package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
	"wiki/migrations"
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
	Long:  "Wiki service CLI for running the HTTP server and operational commands.",
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the wiki HTTP server",
	Args:  cobra.NoArgs,
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
			Environment: cfg.Environment,
			JWTSecret:   cfg.JWTSecret,
			SSOLoginURL: cfg.SSOLoginURL,
			JWTCookie:   cfg.JWTCookie,
		})
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long:  "Run database schema migrations against the configured sqlite database.",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply pending migrations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		goose.SetBaseFS(migrations.FS())
		return goose.Up(db, ".")
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback the latest migration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		goose.SetBaseFS(migrations.FS())
		return goose.Down(db, ".")
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := openDB()
		if err != nil {
			return err
		}
		defer db.Close()

		goose.SetBaseFS(migrations.FS())
		return goose.Status(db, ".")
	},
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Run migrate up and startup checks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, err := databasePath()
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return err
		}

		if err := migrateUpCmd.RunE(cmd, args); err != nil {
			return err
		}

		db, err := openDB()
		if err != nil {
			return err
		}
		return db.Close()
	},
}

func openDB() (*sql.DB, error) {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, err
	}

	dbPath, err := databasePath()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func databasePath() (string, error) {
	dbURL := strings.TrimSpace(os.Getenv("WIKI_DATABASE_URL"))
	if dbURL == "" {
		dbURL = "./data/wiki.db"
	}

	cleanPath := filepath.Clean(dbURL)
	if cleanPath == "." {
		return "", fmt.Errorf("WIKI_DATABASE_URL must point to a sqlite file path")
	}

	return cleanPath, nil
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(bootstrapCmd)
}
