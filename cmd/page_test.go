package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPageCreateRejectsUnexpectedArgs(t *testing.T) {
	err := pageCreateCmd.Args(pageCreateCmd, []string{"extra"})
	if err == nil {
		t.Fatal("expected argument validation error")
	}
}

func TestRootHasPageCommand(t *testing.T) {
	if _, _, err := rootCmd.Find([]string{"page", "create"}); err != nil {
		t.Fatalf("expected page create command to be registered: %v", err)
	}
}

func TestPageBodyFromFlagsWithInlineBody(t *testing.T) {
	cmd := testPageBodyCmd(t)
	if err := cmd.Flags().Set("body", "# hello"); err != nil {
		t.Fatalf("set body flag: %v", err)
	}

	body, err := pageBodyFromFlags(cmd)
	if err != nil {
		t.Fatalf("pageBodyFromFlags() unexpected error: %v", err)
	}
	if body != "# hello" {
		t.Fatalf("body = %q, want %q", body, "# hello")
	}
}

func TestPageBodyFromFlagsWithFile(t *testing.T) {
	cmd := testPageBodyCmd(t)
	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte("from file"), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	if err := cmd.Flags().Set("body-file", path); err != nil {
		t.Fatalf("set body-file flag: %v", err)
	}

	body, err := pageBodyFromFlags(cmd)
	if err != nil {
		t.Fatalf("pageBodyFromFlags() unexpected error: %v", err)
	}
	if body != "from file" {
		t.Fatalf("body = %q, want %q", body, "from file")
	}
}

func TestPageBodyFromFlagsWithStdin(t *testing.T) {
	cmd := testPageBodyCmd(t)
	cmd.SetIn(strings.NewReader("from stdin"))
	if err := cmd.Flags().Set("body-file", "-"); err != nil {
		t.Fatalf("set body-file flag: %v", err)
	}

	body, err := pageBodyFromFlags(cmd)
	if err != nil {
		t.Fatalf("pageBodyFromFlags() unexpected error: %v", err)
	}
	if body != "from stdin" {
		t.Fatalf("body = %q, want %q", body, "from stdin")
	}
}

func TestPageBodyFromFlagsRejectsMissingSource(t *testing.T) {
	cmd := testPageBodyCmd(t)

	_, err := pageBodyFromFlags(cmd)
	if err == nil {
		t.Fatal("expected error when neither --body nor --body-file is set")
	}
}

func TestPageBodyFromFlagsRejectsBothSources(t *testing.T) {
	cmd := testPageBodyCmd(t)
	if err := cmd.Flags().Set("body", "inline"); err != nil {
		t.Fatalf("set body flag: %v", err)
	}
	if err := cmd.Flags().Set("body-file", "x.md"); err != nil {
		t.Fatalf("set body-file flag: %v", err)
	}

	_, err := pageBodyFromFlags(cmd)
	if err == nil {
		t.Fatal("expected error when both --body and --body-file are set")
	}
}

func testPageBodyCmd(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.Flags().String("body", "", "")
	cmd.Flags().String("body-file", "", "")
	return cmd
}
