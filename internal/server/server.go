package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/mermaid"
	// "go.abhg.dev/goldmark/toc"
	"go.abhg.dev/goldmark/wikilink"

	"wiki/internal/auth"
	"wiki/internal/db"
	"wiki/internal/web"
)

type Config struct {
	Address        string
	Port           int
	Environment    string
	Branding       string
	LogoURL        string
	LogoHeroURL    string
	FaviconURL     string
	TemplatesDir   string
	JWTSecret      string
	SSOLoginURL    string
	ServerURL      string
	JWTCookie      string
	Queries        *db.Queries
	UploadStore    ObjectStore
	UploadBucket   string
	UploadMaxBytes int64
	ReportError    func(context.Context, error, map[string]any)
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
	if strings.TrimSpace(cfg.Branding) == "" {
		cfg.Branding = "Wiki"
	}

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		method := c.Request().Method
		uri := c.Request().RequestURI
		slog.Error("http internal error",
			"request_id", requestID,
			"method", method,
			"uri", uri,
			"error", err,
		)
		if cfg.ReportError != nil {
			cfg.ReportError(c.Request().Context(), err, map[string]any{
				"request_id": requestID,
				"method":     method,
				"url":        uri,
			})
		}
		_ = c.String(http.StatusInternalServerError, "internal server error")
	}
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogURI:       true,
		LogMethod:    true,
		LogLatency:   true,
		LogRequestID: true,
		LogError:     true,
		HandleError:  true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []any{
				"request_id", v.RequestID,
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency.String(),
			}
			if v.Error != nil {
				attrs = append(attrs, "error", v.Error)
			}
			if requestErr, ok := c.Get("request_error").(error); ok && requestErr != nil {
				attrs = append(attrs, "request_error", requestErr)
			}
			if requestErrContext, ok := c.Get("request_error_context").(map[string]any); ok {
				for k, v := range requestErrContext {
					attrs = append(attrs, k, v)
				}
			}

			switch {
			case v.Status >= http.StatusInternalServerError:
				slog.Error("http request failed", attrs...)
			case v.Status >= http.StatusBadRequest:
				slog.Warn("http request failed", attrs...)
			default:
				slog.Info("http request", attrs...)
			}
			return nil
		},
	}))
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
		ServerURL:   cfg.ServerURL,
		Skipper: func(c echo.Context) bool {
			path := c.Path()
			return path == "/healthz" || path == "/readyz" || strings.HasPrefix(path, "/assets/")
		},
	}))

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "layout", homeData{
			Title:           "Wiki",
			Branding:        cfg.Branding,
			LogoURL:         cfg.LogoURL,
			LogoHeroURL:     cfg.LogoHeroURL,
			FaviconURL:      cfg.FaviconURL,
			ContentTemplate: "pages/home",
		})
	})
	e.GET("/healthz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/readyz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.Static("/assets", "internal/web/assets")
	e.GET("/wiki/*", getWikiHandler(cfg))
	e.POST("/wiki/*", postWikiHandler(cfg))
	e.POST("/api/markdown/preview", markdownPreviewHandler())
	e.POST("/api/uploads", uploadHandler(cfg))

	return e, nil
}

type wikiViewData struct {
	Title           string
	Slug            string
	Body            template.HTML
	UpdatedAt       string
	Branding        string
	LogoURL         string
	LogoHeroURL     string
	FaviconURL      string
	ContentTemplate string
}

type wikiEditData struct {
	Title           string
	Slug            string
	Body            string
	Error           string
	Branding        string
	LogoURL         string
	LogoHeroURL     string
	FaviconURL      string
	ContentTemplate string
}

type wikiMissingData struct {
	Title           string
	Slug            string
	Branding        string
	LogoURL         string
	LogoHeroURL     string
	FaviconURL      string
	ContentTemplate string
}

type homeData struct {
	Title           string
	Slug            string
	Branding        string
	LogoURL         string
	LogoHeroURL     string
	FaviconURL      string
	ContentTemplate string
}

func getWikiHandler(cfg Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		queries := cfg.Queries
		if queries == nil {
			return c.String(http.StatusInternalServerError, "wiki storage not configured")
		}

		routePath := strings.Trim(c.Param("*"), "/")
		if strings.HasSuffix(routePath, "/edit") {
			return renderWikiEdit(c, cfg, strings.TrimSuffix(routePath, "/edit"))
		}

		slug, err := normalizeWikiSlug(routePath)
		if err != nil {
			return c.String(http.StatusNotFound, "page not found")
		}

		page, err := queries.GetPageBySlug(c.Request().Context(), slug)
		if errors.Is(err, sql.ErrNoRows) {
			return c.Render(http.StatusOK, "layout", wikiMissingData{
				Title:           titleFromSlug(slug),
				Slug:            slug,
				Branding:        cfg.Branding,
				LogoURL:         cfg.LogoURL,
				LogoHeroURL:     cfg.LogoHeroURL,
				FaviconURL:      cfg.FaviconURL,
				ContentTemplate: "pages/wiki_missing",
			})
		}
		if err != nil {
			return requestError(c, err, map[string]any{
				"handler":   "get_wiki",
				"operation": "get_page_by_slug",
				"page_slug": slug,
			})
		}

		revisions, err := queries.ListRevisionsByPage(c.Request().Context(), page.ID)
		if err != nil {
			return requestError(c, err, map[string]any{
				"handler":   "get_wiki",
				"operation": "list_revisions_by_page",
				"page_id":   page.ID,
				"page_slug": slug,
			})
		}
		if len(revisions) == 0 {
			return c.String(http.StatusNotFound, "page has no revisions")
		}

		htmlBody, err := renderMarkdownToHTML(revisions[0].Body)
		if err != nil {
			return requestError(c, err, map[string]any{
				"handler":   "get_wiki",
				"operation": "render_markdown",
				"page_id":   page.ID,
				"page_slug": slug,
			})
		}

		return c.Render(http.StatusOK, "layout", wikiViewData{
			Title:           page.Title,
			Slug:            page.Slug,
			Body:            template.HTML(htmlBody),
			UpdatedAt:       revisions[0].CreatedAt,
			Branding:        cfg.Branding,
			LogoURL:         cfg.LogoURL,
			LogoHeroURL:     cfg.LogoHeroURL,
			FaviconURL:      cfg.FaviconURL,
			ContentTemplate: "pages/wiki_view",
		})
	}
}

type markdownPreviewRequest struct {
	Markdown string `json:"markdown"`
}

type markdownPreviewResponse struct {
	HTML string `json:"html"`
}

func markdownPreviewHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		var req markdownPreviewRequest
		if err := c.Bind(&req); err != nil {
			setRequestError(c, err, map[string]any{
				"handler":   "markdown_preview",
				"operation": "bind_request",
			})
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}

		htmlBody, err := renderMarkdownToHTML(req.Markdown)
		if err != nil {
			return requestError(c, err, map[string]any{
				"handler":   "markdown_preview",
				"operation": "render_markdown",
			})
		}

		return c.JSON(http.StatusOK, markdownPreviewResponse{HTML: htmlBody})
	}
}

func renderMarkdownToHTML(markdown string) (string, error) {
	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Typographer,
			// &toc.Extender{
			// 	MaxDepth: 4,
			// 	Compact:  true,
			// },
			&mermaid.Extender{},
			&wikilink.Extender{
				Resolver: CustomResolver,
			},
		),
	)
	if err := md.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var CustomResolver wikilink.Resolver = customResolver{}

type customResolver struct{}

func (customResolver) ResolveWikilink(n *wikilink.Node) ([]byte, error) {
	targetStr := string(n.Target)
	slug := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(targetStr)), " ", "_")
	return []byte(fmt.Sprintf("/wiki/%s", slug)), nil
}

func postWikiHandler(cfg Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		queries := cfg.Queries
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
		return saveWikiRevision(c, cfg, slug)
	}
}

func renderWikiEdit(c echo.Context, cfg Config, routePath string) error {
	queries := cfg.Queries
	slug, err := normalizeWikiSlug(routePath)
	if err != nil {
		return c.String(http.StatusNotFound, "page not found")
	}

	data := wikiEditData{
		Title:           titleFromSlug(slug),
		Slug:            slug,
		Branding:        cfg.Branding,
		LogoURL:         cfg.LogoURL,
		LogoHeroURL:     cfg.LogoHeroURL,
		FaviconURL:      cfg.FaviconURL,
		ContentTemplate: "pages/wiki_edit",
	}

	page, err := queries.GetPageBySlug(c.Request().Context(), slug)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return requestError(c, err, map[string]any{
			"handler":   "render_wiki_edit",
			"operation": "get_page_by_slug",
			"page_slug": slug,
		})
	}
	if err == nil {
		data.Title = page.Title
		revisions, revErr := queries.ListRevisionsByPage(c.Request().Context(), page.ID)
		if revErr != nil {
			return requestError(c, revErr, map[string]any{
				"handler":   "render_wiki_edit",
				"operation": "list_revisions_by_page",
				"page_id":   page.ID,
				"page_slug": slug,
			})
		}
		if len(revisions) > 0 {
			data.Body = revisions[0].Body
		}
	}

	return c.Render(http.StatusOK, "layout", data)
}

func saveWikiRevision(c echo.Context, cfg Config, slug string) error {
	queries := cfg.Queries
	body := c.FormValue("body")
	if strings.TrimSpace(body) == "" {
		return c.Render(http.StatusUnprocessableEntity, "layout", wikiEditData{
			Title:           titleFromSlug(slug),
			Slug:            slug,
			Body:            body,
			Error:           "body is required",
			Branding:        cfg.Branding,
			LogoURL:         cfg.LogoURL,
			LogoHeroURL:     cfg.LogoHeroURL,
			FaviconURL:      cfg.FaviconURL,
			ContentTemplate: "pages/wiki_edit",
		})
	}

	ctx := c.Request().Context()
	page, err := queries.GetPageBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		page, err = queries.CreatePage(
			ctx,
			db.CreatePageParams{Slug: slug, Title: titleFromSlug(slug)},
		)
	}
	if err != nil {
		return requestError(c, err, map[string]any{
			"handler":   "save_wiki_revision",
			"operation": "get_or_create_page",
			"page_slug": slug,
		})
	}

	author := "unknown"
	if principal, ok := auth.PrincipalFromContext(
		c,
	); ok &&
		strings.TrimSpace(principal.Subject) != "" {
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
		return requestError(c, err, map[string]any{
			"handler":   "save_wiki_revision",
			"operation": "create_revision",
			"page_id":   page.ID,
			"page_slug": slug,
		})
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

func setRequestError(c echo.Context, cause error, contextFields map[string]any) {
	if cause != nil {
		c.Set("request_error", cause)
	}
	if contextFields != nil {
		c.Set("request_error_context", contextFields)
	}
}

func requestError(c echo.Context, cause error, contextFields map[string]any) error {
	setRequestError(c, cause, contextFields)
	return cause
}

type templateRenderer struct {
	templates   *template.Template
	development bool
	cfg         Config
}

func (r templateRenderer) Render(w io.Writer, name string, data any, _ echo.Context) error {
	t := r.templates
	if r.development {
		var err error
		t, err = parseTemplates(r.cfg)
		if err != nil {
			return err
		}
	}
	return t.ExecuteTemplate(w, name, data)
}

func newRenderer(cfg Config) (echo.Renderer, error) {
	t, err := parseTemplates(cfg)
	if err != nil {
		return nil, err
	}
	return templateRenderer{
		templates:   t,
		development: strings.EqualFold(cfg.Environment, "development"),
		cfg:         cfg,
	}, nil
}

func parseTemplates(cfg Config) (*template.Template, error) {
	var t *template.Template

	tmplFS := templateFS(cfg)
	paths, err := templatePaths(tmplFS, ".")
	if err != nil {
		return nil, err
	}

	funcs := template.FuncMap{
		"renderContent": func(name string, data any) (template.HTML, error) {
			name = strings.TrimSpace(name)
			if name == "" {
				return "", fmt.Errorf("missing content template name")
			}

			var b strings.Builder
			if err := t.ExecuteTemplate(&b, name, data); err != nil {
				return "", err
			}
			return template.HTML(b.String()), nil
		},
		"attr": func(s string) template.HTMLAttr {
			return template.HTMLAttr(s)
		},
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"url": func(s string) template.URL {
			return template.URL(s)
		},
	}

	t = template.New("wiki").Funcs(funcs)
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
