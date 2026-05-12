package server

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type failingRenderer struct{}

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
	e, err := New(Config{Environment: "production", JWTSecret: "secret", JWTCookie: "wiki_token", SSOLoginURL: "/sso/login"})
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
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", notFoundRec.Code, http.StatusNotFound)
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

func signedToken(t *testing.T, secret string) string {
	t.Helper()
	claims := jwt.RegisteredClaims{
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
