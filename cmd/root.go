package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
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
		return server.Start(cmd.Context(), server.ConfigFromEnv())
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
	dsn := os.Getenv("WIKI_DATABASE_URL")
	if dsn == "" {
		dsn = filepath.Join("data", "wiki.db")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
