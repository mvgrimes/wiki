package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"wiki/internal/auth"
	"wiki/internal/db"
	"wiki/internal/web"
)

type Config struct {
	Address      string
	Port         int
	Environment  string
	TemplatesDir string
	JWTSecret    string
	SSOLoginURL  string
	JWTCookie    string
	Queries      *db.Queries
	UploadStore  ObjectStore
	UploadBucket string
	UploadMaxBytes int64
}

func Start(ctx context.Context, cfg Config) error {
	e, err := New(cfg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	slog.Info("http server listening", "addr", addr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = e.Shutdown(shutdownCtx)
	}()

	return e.Start(addr)
}

func New(cfg Config) (*echo.Echo, error) {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	renderer, err := newRenderer(cfg)
	if err != nil {
		return nil, err
	}
	e.Renderer = renderer

	e.Use(auth.Middleware(auth.Config{
		JWTSecret:   cfg.JWTSecret,
		CookieName:  cfg.JWTCookie,
		SSOLoginURL: cfg.SSOLoginURL,
		Skipper: func(c echo.Context) bool {
			path := c.Path()
			return path == "/healthz" || path == "/readyz" || strings.HasPrefix(path, "/assets/")
		},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "pages/home", map[string]any{"Title": "Wiki"})
	})
	e.GET("/healthz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/readyz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.Static("/assets", "internal/web/assets")
	e.GET("/wiki/*", getWikiHandler(cfg.Queries))
	e.POST("/wiki/*", postWikiHandler(cfg.Queries))
	e.POST("/api/uploads", uploadHandler(cfg))

	return e, nil
}

type wikiViewData struct {
	Title     string
	Slug      string
	Body      string
	UpdatedAt string
}

type wikiEditData struct {
	Title string
	Slug  string
	Body  string
	Error string
}

func getWikiHandler(queries *db.Queries) echo.HandlerFunc {
	return func(c echo.Context) error {
		if queries == nil {
			return c.String(http.StatusInternalServerError, "wiki storage not configured")
		}

		routePath := strings.Trim(c.Param("*"), "/")
		if strings.HasSuffix(routePath, "/edit") {
			return renderWikiEdit(c, queries, strings.TrimSuffix(routePath, "/edit"))
		}

		slug, err := normalizeWikiSlug(routePath)
		if err != nil {
			return c.String(http.StatusNotFound, "page not found")
		}

		page, err := queries.GetPageBySlug(c.Request().Context(), slug)
		if errors.Is(err, sql.ErrNoRows) {
			return c.String(http.StatusNotFound, "page not found")
		}
		if err != nil {
			return err
		}

		revisions, err := queries.ListRevisionsByPage(c.Request().Context(), page.ID)
		if err != nil {
			return err
		}
		if len(revisions) == 0 {
			return c.String(http.StatusNotFound, "page has no revisions")
		}

		return c.Render(http.StatusOK, "pages/wiki_view", wikiViewData{
			Title:     page.Title,
			Slug:      page.Slug,
			Body:      revisions[0].Body,
			UpdatedAt: revisions[0].CreatedAt,
		})
	}
}

func postWikiHandler(queries *db.Queries) echo.HandlerFunc {
	return func(c echo.Context) error {
		if queries == nil {
			return c.String(http.StatusInternalServerError, "wiki storage not configured")
		}
		routePath := strings.Trim(c.Param("*"), "/")
		if !strings.HasSuffix(routePath, "/edit") {
			return c.String(http.StatusNotFound, "page not found")
		}

		slug, err := normalizeWikiSlug(strings.TrimSuffix(routePath, "/edit"))
		if err != nil {
			return c.String(http.StatusNotFound, "page not found")
		}
		return saveWikiRevision(c, queries, slug)
	}
}

func renderWikiEdit(c echo.Context, queries *db.Queries, routePath string) error {
	slug, err := normalizeWikiSlug(routePath)
	if err != nil {
		return c.String(http.StatusNotFound, "page not found")
	}

	data := wikiEditData{Title: titleFromSlug(slug), Slug: slug}

	page, err := queries.GetPageBySlug(c.Request().Context(), slug)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		data.Title = page.Title
		revisions, revErr := queries.ListRevisionsByPage(c.Request().Context(), page.ID)
		if revErr != nil {
			return revErr
		}
		if len(revisions) > 0 {
			data.Body = revisions[0].Body
		}
	}

	return c.Render(http.StatusOK, "pages/wiki_edit", data)
}

func saveWikiRevision(c echo.Context, queries *db.Queries, slug string) error {
	body := c.FormValue("body")
	if strings.TrimSpace(body) == "" {
		return c.Render(http.StatusUnprocessableEntity, "pages/wiki_edit", wikiEditData{
			Title: titleFromSlug(slug),
			Slug:  slug,
			Body:  body,
			Error: "body is required",
		})
	}

	ctx := c.Request().Context()
	page, err := queries.GetPageBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		page, err = queries.CreatePage(ctx, db.CreatePageParams{Slug: slug, Title: titleFromSlug(slug)})
	}
	if err != nil {
		return err
	}

	author := "unknown"
	if principal, ok := auth.PrincipalFromContext(c); ok && strings.TrimSpace(principal.Subject) != "" {
		author = principal.Subject
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
	_, err = queries.CreateRevision(ctx, db.CreateRevisionParams{
		PageID:   page.ID,
		Body:     body,
		Author:   author,
		Checksum: checksum,
	})
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/wiki/"+slug)
}

func normalizeWikiSlug(raw string) (string, error) {
	raw = strings.TrimSpace(strings.Trim(raw, "/"))
	if raw == "" {
		return "", fmt.Errorf("empty slug")
	}
	return raw, nil
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

type templateRenderer struct {
	templates *template.Template
}

func (r templateRenderer) Render(w io.Writer, name string, data any, _ echo.Context) error {
	return r.templates.ExecuteTemplate(w, name, data)
}

func newRenderer(cfg Config) (echo.Renderer, error) {
	t, err := parseTemplates(cfg)
	if err != nil {
		return nil, err
	}
	return templateRenderer{templates: t}, nil
}

func parseTemplates(cfg Config) (*template.Template, error) {
	tmplFS := templateFS(cfg)
	paths, err := templatePaths(tmplFS, ".")
	if err != nil {
		return nil, err
	}

	t := template.New("wiki")
	for _, path := range paths {
		content, readErr := fs.ReadFile(tmplFS, path)
		if readErr != nil {
			return nil, fmt.Errorf("read template %q: %w", path, readErr)
		}
		if _, parseErr := t.Parse(string(content)); parseErr != nil {
			return nil, fmt.Errorf("parse template %q: %w", path, parseErr)
		}
	}

	return t, nil
}

func templateFS(cfg Config) fs.FS {
	if strings.EqualFold(cfg.Environment, "development") {
		dir := cfg.TemplatesDir
		if strings.TrimSpace(dir) == "" {
			dir = "internal/web/templates"
		}
		return os.DirFS(dir)
	}

	sub, err := fs.Sub(web.TemplatesFS, "templates")
	if err != nil {
		return web.TemplatesFS
	}

	return sub
}

func templatePaths(fsys fs.FS, root string) ([]string, error) {
	paths := make([]string, 0, 8)
	err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".tmpl") {
			paths = append(paths, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("no templates found")
	}
	sort.Strings(paths)
	return paths, nil
}
