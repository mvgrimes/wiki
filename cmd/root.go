package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Wiki service CLI",
	Long:  "Wiki service CLI for running the HTTP server and operational commands.",
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
