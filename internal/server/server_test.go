package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"wiki/internal/db"
	"wiki/migrations"
)

type failingRenderer struct{}

type objectStoreStub struct {
	err         error
	bucket      string
	key         string
	contentType string
	body        []byte
	size        int64
}

func (s *objectStoreStub) PutObject(
	_ context.Context,
	bucket, key, contentType string,
	body io.Reader,
	size int64,
) error {
	s.bucket = bucket
	s.key = key
	s.contentType = contentType
	s.size = size
	if body != nil {
		buf, _ := io.ReadAll(body)
		s.body = buf
	}
	return s.err
}

func (failingRenderer) Render(_ io.Writer, _ string, _ any, _ echo.Context) error {
	return errRenderFailed
}

var errRenderFailed = errors.New("render failed")

func TestNewServerHealthAndReadinessBypassAuth(t *testing.T) {
	e, err := New(Config{Environment: "production", JWTSecret: "secret", JWTCookie: "wiki_token"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", path, rec.Code, http.StatusOK)
		}
	}
}

func TestRootRouteRequiresAuthAndRendersTemplate(t *testing.T) {
	e, err := New(
		Config{
			Environment: "production",
			JWTSecret:   "secret",
			JWTCookie:   "wiki_token",
			SSOLoginURL: "/sso/login",
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	unauthReq := httptest.NewRequest(http.MethodGet, "/", nil)
	unauthReq.Header.Set(echo.HeaderAccept, "text/html")
	unauthRec := httptest.NewRecorder()
	e.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusFound {
		t.Fatalf("unauthenticated status = %d, want %d", unauthRec.Code, http.StatusFound)
	}

	token := signedToken(t, "secret")
	authReq := httptest.NewRequest(http.MethodGet, "/", nil)
	authReq.AddCookie(&http.Cookie{Name: "wiki_token", Value: token})
	authRec := httptest.NewRecorder()
	e.ServeHTTP(authRec, authReq)

	if authRec.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d, want %d", authRec.Code, http.StatusOK)
	}
	if !strings.Contains(authRec.Body.String(), "Welcome to the wiki service") {
		t.Fatalf("response missing expected body: %q", authRec.Body.String())
	}
}

func TestRouteAndRenderErrorPaths(t *testing.T) {
	e, err := New(Config{Environment: "production", JWTSecret: "secret", JWTCookie: "wiki_token"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	notFoundReq := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	notFoundReq.AddCookie(&http.Cookie{Name: "wiki_token", Value: signedToken(t, "secret")})
	notFoundRec := httptest.NewRecorder()
	e.ServeHTTP(notFoundRec, notFoundReq)
	if notFoundRec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", notFoundRec.Code, http.StatusInternalServerError)
	}

	e.Renderer = failingRenderer{}
	renderReq := httptest.NewRequest(http.MethodGet, "/", nil)
	renderReq.AddCookie(&http.Cookie{Name: "wiki_token", Value: signedToken(t, "secret")})
	renderRec := httptest.NewRecorder()
	e.ServeHTTP(renderRec, renderReq)
	if renderRec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", renderRec.Code, http.StatusInternalServerError)
	}
}

func TestWikiPageLifecycleRoutes(t *testing.T) {
	sqlDB := openMigratedDB(t)
	t.Cleanup(func() { _ = sqlDB.Close() })
	queries := db.New(sqlDB)

	e, err := New(
		Config{
			Environment: "production",
			JWTSecret:   "secret",
			JWTCookie:   "wiki_token",
			Queries:     queries,
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	auth := &http.Cookie{Name: "wiki_token", Value: signedToken(t, "secret")}

	missingReq := httptest.NewRequest(http.MethodGet, "/wiki/guides/getting-started", nil)
	missingReq.AddCookie(auth)
	missingRec := httptest.NewRecorder()
	e.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing page status = %d, want %d", missingRec.Code, http.StatusNotFound)
	}

	editReq := httptest.NewRequest(http.MethodGet, "/wiki/guides/getting-started/edit", nil)
	editReq.AddCookie(auth)
	editRec := httptest.NewRecorder()
	e.ServeHTTP(editRec, editReq)
	if editRec.Code != http.StatusOK {
		t.Fatalf("edit page status = %d, want %d", editRec.Code, http.StatusOK)
	}

	body := "body=%23+Getting+Started%0Ahello+world"
	postReq := httptest.NewRequest(
		http.MethodPost,
		"/wiki/guides/getting-started/edit",
		strings.NewReader(body),
	)
	postReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	postReq.AddCookie(auth)
	postRec := httptest.NewRecorder()
	e.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("post status = %d, want %d", postRec.Code, http.StatusSeeOther)
	}

	viewReq := httptest.NewRequest(http.MethodGet, "/wiki/guides/getting-started", nil)
	viewReq.AddCookie(auth)
	viewRec := httptest.NewRecorder()
	e.ServeHTTP(viewRec, viewReq)
	if viewRec.Code != http.StatusOK {
		t.Fatalf("view page status = %d, want %d", viewRec.Code, http.StatusOK)
	}
	if !strings.Contains(viewRec.Body.String(), "hello world") {
		t.Fatalf("view response missing revision body: %q", viewRec.Body.String())
	}

	page, err := queries.GetPageBySlug(context.Background(), "guides/getting-started")
	if err != nil {
		t.Fatalf("GetPageBySlug() unexpected error: %v", err)
	}
	revisions, err := queries.ListRevisionsByPage(context.Background(), page.ID)
	if err != nil {
		t.Fatalf("ListRevisionsByPage() unexpected error: %v", err)
	}
	if len(revisions) != 1 {
		t.Fatalf("revisions len = %d, want 1", len(revisions))
	}
	if revisions[0].Author != "u-1" {
		t.Fatalf("revision author = %q, want %q", revisions[0].Author, "u-1")
	}
}

func TestWikiEditValidationAndLatestRevision(t *testing.T) {
	sqlDB := openMigratedDB(t)
	t.Cleanup(func() { _ = sqlDB.Close() })
	queries := db.New(sqlDB)

	e, err := New(
		Config{
			Environment: "production",
			JWTSecret:   "secret",
			JWTCookie:   "wiki_token",
			Queries:     queries,
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	auth := &http.Cookie{Name: "wiki_token", Value: signedToken(t, "secret")}

	invalidReq := httptest.NewRequest(
		http.MethodPost,
		"/wiki/notes/edit",
		strings.NewReader("body=   "),
	)
	invalidReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	invalidReq.AddCookie(auth)
	invalidRec := httptest.NewRecorder()
	e.ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("validation status = %d, want %d", invalidRec.Code, http.StatusUnprocessableEntity)
	}

	firstReq := httptest.NewRequest(
		http.MethodPost,
		"/wiki/notes/edit",
		strings.NewReader("body=first"),
	)
	firstReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	firstReq.AddCookie(auth)
	firstRec := httptest.NewRecorder()
	e.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusSeeOther {
		t.Fatalf("first post status = %d, want %d", firstRec.Code, http.StatusSeeOther)
	}

	secondReq := httptest.NewRequest(
		http.MethodPost,
		"/wiki/notes/edit",
		strings.NewReader("body=second"),
	)
	secondReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	secondReq.AddCookie(auth)
	secondRec := httptest.NewRecorder()
	e.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusSeeOther {
		t.Fatalf("second post status = %d, want %d", secondRec.Code, http.StatusSeeOther)
	}

	viewReq := httptest.NewRequest(http.MethodGet, "/wiki/notes", nil)
	viewReq.AddCookie(auth)
	viewRec := httptest.NewRecorder()
	e.ServeHTTP(viewRec, viewReq)
	if viewRec.Code != http.StatusOK {
		t.Fatalf("view status = %d, want %d", viewRec.Code, http.StatusOK)
	}
	if !strings.Contains(viewRec.Body.String(), "second") {
		t.Fatalf("expected latest revision body in response, got %q", viewRec.Body.String())
	}
}

func TestUploadEndpointSuccessAndFailurePaths(t *testing.T) {
	sqlDB := openMigratedDB(t)
	t.Cleanup(func() { _ = sqlDB.Close() })
	queries := db.New(sqlDB)
	store := &objectStoreStub{}

	e, err := New(Config{
		Environment:    "production",
		JWTSecret:      "secret",
		JWTCookie:      "wiki_token",
		Queries:        queries,
		UploadStore:    store,
		UploadBucket:   "wiki-uploads",
		UploadMaxBytes: 1024,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	auth := &http.Cookie{Name: "wiki_token", Value: signedToken(t, "secret")}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("page_slug", "guides/uploads")
	part, _ := writer.CreateFormFile("file", "hello.txt")
	_, _ = part.Write([]byte("hello upload"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/uploads", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	req.AddCookie(auth)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"upload status = %d, want %d (%s)",
			rec.Code,
			http.StatusCreated,
			rec.Body.String(),
		)
	}
	if store.bucket != "wiki-uploads" ||
		!strings.HasPrefix(store.key, "attachments/guides-uploads/") {
		t.Fatalf("store call bucket/key = %q/%q", store.bucket, store.key)
	}
	if store.contentType != "text/plain; charset=utf-8" {
		t.Fatalf("content type = %q", store.contentType)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["storage_key"] == "" {
		t.Fatalf("storage_key missing in response: %v", payload)
	}

	store.err = errors.New("s3 unavailable")
	failBody := &bytes.Buffer{}
	failWriter := multipart.NewWriter(failBody)
	_ = failWriter.WriteField("page_slug", "guides/uploads")
	failPart, _ := failWriter.CreateFormFile("file", "again.txt")
	_, _ = failPart.Write([]byte("hello again"))
	_ = failWriter.Close()

	failReq := httptest.NewRequest(http.MethodPost, "/api/uploads", failBody)
	failReq.Header.Set(echo.HeaderContentType, failWriter.FormDataContentType())
	failReq.AddCookie(auth)
	failRec := httptest.NewRecorder()
	e.ServeHTTP(failRec, failReq)
	if failRec.Code != http.StatusBadGateway {
		t.Fatalf("failure status = %d, want %d", failRec.Code, http.StatusBadGateway)
	}

	badTypeBody := &bytes.Buffer{}
	badTypeWriter := multipart.NewWriter(badTypeBody)
	_ = badTypeWriter.WriteField("page_slug", "guides/uploads")
	badTypePart, _ := badTypeWriter.CreateFormFile("file", "payload.bin")
	_, _ = badTypePart.Write([]byte{0x00, 0x01, 0x02, 0x03})
	_ = badTypeWriter.Close()

	badTypeReq := httptest.NewRequest(http.MethodPost, "/api/uploads", badTypeBody)
	badTypeReq.Header.Set(echo.HeaderContentType, badTypeWriter.FormDataContentType())
	badTypeReq.AddCookie(auth)
	badTypeRec := httptest.NewRecorder()
	e.ServeHTTP(badTypeRec, badTypeReq)
	if badTypeRec.Code != http.StatusBadRequest {
		t.Fatalf("bad-type status = %d, want %d", badTypeRec.Code, http.StatusBadRequest)
	}
}

func signedToken(t *testing.T, secret string) string {
	t.Helper()
	claims := struct {
		Data struct {
			User int64  `json:"user"`
			Role string `json:"role"`
		} `json:"data"`
		jwt.RegisteredClaims
	}{}
	claims.Data.User = 1
	claims.Data.Role = "EMPLOYEE"
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   "u-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return signed
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
