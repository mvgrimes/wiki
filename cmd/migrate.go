package cmd

import (
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"

	"wiki/migrations"
)

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

func init() {
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	rootCmd.AddCommand(migrateCmd)
}
