package cmd

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"wiki/internal/db"
)

var pageCmd = &cobra.Command{
	Use:   "page",
	Short: "Manage wiki pages",
}

var pageCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a wiki page revision from markdown",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, err := cmd.Flags().GetString("slug")
		if err != nil {
			return err
		}
		slug = normalizeSlug(slug)
		if slug == "" {
			return fmt.Errorf("--slug is required")
		}

		title, err := cmd.Flags().GetString("title")
		if err != nil {
			return err
		}
		title = strings.TrimSpace(title)
		if title == "" {
			title = titleFromSlug(slug)
		}

		body, err := pageBodyFromFlags(cmd)
		if err != nil {
			return err
		}
		if strings.TrimSpace(body) == "" {
			return fmt.Errorf("page body is required")
		}

		author, err := cmd.Flags().GetString("author")
		if err != nil {
			return err
		}
		author = strings.TrimSpace(author)
		if author == "" {
			author = "cli"
		}

		sqlDB, err := openDB()
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		queries := db.New(sqlDB)
		ctx := cmd.Context()

		page, err := queries.GetPageBySlug(ctx, slug)
		if errors.Is(err, sql.ErrNoRows) {
			page, err = queries.CreatePage(ctx, db.CreatePageParams{Slug: slug, Title: title})
		}
		if err != nil {
			return err
		}

		checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
		revision, err := queries.CreateRevision(ctx, db.CreateRevisionParams{
			PageID:   page.ID,
			Body:     body,
			Author:   author,
			Checksum: checksum,
		})
		if err != nil {
			return err
		}

		cmd.Printf(
			"created revision %d for page %q (id=%d)\n",
			revision.ID,
			page.Slug,
			page.ID,
		)

		return nil
	},
}

func pageBodyFromFlags(cmd *cobra.Command) (string, error) {
	bodyInline, err := cmd.Flags().GetString("body")
	if err != nil {
		return "", err
	}
	bodyFile, err := cmd.Flags().GetString("body-file")
	if err != nil {
		return "", err
	}

	hasInline := strings.TrimSpace(bodyInline) != ""
	hasFile := strings.TrimSpace(bodyFile) != ""

	if hasInline == hasFile {
		return "", fmt.Errorf("provide exactly one of --body or --body-file")
	}

	if hasInline {
		return bodyInline, nil
	}

	bodyFile = strings.TrimSpace(bodyFile)
	if bodyFile == "-" {
		data, readErr := io.ReadAll(cmd.InOrStdin())
		if readErr != nil {
			return "", fmt.Errorf("read stdin: %w", readErr)
		}
		return string(data), nil
	}

	data, readErr := os.ReadFile(bodyFile)
	if readErr != nil {
		return "", fmt.Errorf("read body file %q: %w", bodyFile, readErr)
	}

	return string(data), nil
}

func normalizeSlug(raw string) string {
	return strings.Trim(strings.TrimSpace(raw), "/")
}

func titleFromSlug(slug string) string {
	parts := strings.Split(slug, "/")
	name := parts[len(parts)-1]
	name = strings.ReplaceAll(name, "-", " ")
	if name == "" {
		return "Untitled"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func init() {
	pageCreateCmd.Flags().String("slug", "", "Page slug (for example: guides/getting-started)")
	pageCreateCmd.Flags().String("title", "", "Page title (defaults from slug)")
	pageCreateCmd.Flags().String("body", "", "Markdown body content")
	pageCreateCmd.Flags().String("body-file", "", "Read markdown body from file path, or '-' for stdin")
	pageCreateCmd.Flags().String("author", "cli", "Revision author")

	pageCmd.AddCommand(pageCreateCmd)
	rootCmd.AddCommand(pageCmd)
}
