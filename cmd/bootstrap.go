package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

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

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}
