package server

import (
	"context"
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

	return e, nil
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
