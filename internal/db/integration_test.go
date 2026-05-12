package db_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"wiki/internal/db"
	"wiki/migrations"
)

func TestMigrationsCreateExpectedTables(t *testing.T) {
	sqlDB := openMigratedDB(t)
	t.Cleanup(func() { _ = sqlDB.Close() })

	tables := []string{"pages", "revisions", "attachments"}
	for _, table := range tables {
		var name string
		err := sqlDB.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).
			Scan(&name)
		if err != nil {
			t.Fatalf("expected table %q to exist: %v", table, err)
		}
	}
}

func TestQueriesRoundTripPageRevisionAttachment(t *testing.T) {
	sqlDB := openMigratedDB(t)
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	q := db.New(sqlDB)

	page, err := q.CreatePage(ctx, db.CreatePageParams{Slug: "home", Title: "Home"})
	if err != nil {
		t.Fatalf("CreatePage() unexpected error: %v", err)
	}

	revision, err := q.CreateRevision(ctx, db.CreateRevisionParams{
		PageID:   page.ID,
		Body:     "# Home",
		Author:   "tester",
		Checksum: "sha256:abc123",
	})
	if err != nil {
		t.Fatalf("CreateRevision() unexpected error: %v", err)
	}

	_, err = q.CreateAttachment(ctx, db.CreateAttachmentParams{
		PageID:      page.ID,
		RevisionID:  sql.NullInt64{Int64: revision.ID, Valid: true},
		FileName:    "diagram.png",
		ContentType: "image/png",
		SizeBytes:   4096,
		StorageKey:  "attachments/home/diagram.png",
	})
	if err != nil {
		t.Fatalf("CreateAttachment() unexpected error: %v", err)
	}

	fetchedPage, err := q.GetPageBySlug(ctx, page.Slug)
	if err != nil {
		t.Fatalf("GetPageBySlug() unexpected error: %v", err)
	}
	if fetchedPage.ID != page.ID {
		t.Fatalf("GetPageBySlug() ID = %d, want %d", fetchedPage.ID, page.ID)
	}

	revisions, err := q.ListRevisionsByPage(ctx, page.ID)
	if err != nil {
		t.Fatalf("ListRevisionsByPage() unexpected error: %v", err)
	}
	if len(revisions) != 1 {
		t.Fatalf("ListRevisionsByPage() len = %d, want 1", len(revisions))
	}

	attachments, err := q.ListAttachmentsByPage(ctx, page.ID)
	if err != nil {
		t.Fatalf("ListAttachmentsByPage() unexpected error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("ListAttachmentsByPage() len = %d, want 1", len(attachments))
	}
}

func openMigratedDB(t *testing.T) *sql.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "wiki.db")
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() unexpected error: %v", err)
	}

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("goose.SetDialect() unexpected error: %v", err)
	}
	goose.SetBaseFS(migrations.FS())
	if err := goose.Up(sqlDB, "."); err != nil {
		t.Fatalf("goose.Up() unexpected error: %v", err)
	}

	return sqlDB
}
