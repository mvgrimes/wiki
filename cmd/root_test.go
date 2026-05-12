package cmd

import (
	"strings"
	"testing"
)

func TestDatabasePathRejectsCurrentDirectory(t *testing.T) {
	t.Setenv("WIKI_DATABASE_URL", ".")

	_, err := databasePath()
	if err == nil {
		t.Fatal("expected error for invalid WIKI_DATABASE_URL")
	}

	if !strings.Contains(err.Error(), "WIKI_DATABASE_URL") {
		t.Fatalf("expected WIKI_DATABASE_URL in error, got %q", err)
	}
}

func TestMigrateUpRejectsUnexpectedArgs(t *testing.T) {
	err := migrateUpCmd.Args(migrateUpCmd, []string{"extra"})
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestMigrateDownRejectsUnexpectedArgs(t *testing.T) {
	err := migrateDownCmd.Args(migrateDownCmd, []string{"extra"})
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestMigrateStatusRejectsUnexpectedArgs(t *testing.T) {
	err := migrateStatusCmd.Args(migrateStatusCmd, []string{"extra"})
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestBootstrapRejectsUnexpectedArgs(t *testing.T) {
	err := bootstrapCmd.Args(bootstrapCmd, []string{"extra"})
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}
