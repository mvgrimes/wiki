package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func TestMiddleware_AllowsBearerTokenAndSetsPrincipal(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/pages", nil)
	req.Header.Set(
		echo.HeaderAuthorization,
		"Bearer "+signedToken(t, "secret", time.Now().Add(time.Hour)),
	)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	h := Middleware(Config{JWTSecret: "secret"})(func(c echo.Context) error {
		nextCalled = true
		p, ok := PrincipalFromContext(c)
		if !ok {
			t.Fatal("expected principal in context")
		}
		if p.Subject != "user-1" || p.User != 42 || p.Role != "EMPLOYEE" ||
			len(p.Capabilities) != 2 {
			t.Fatalf("unexpected principal: %+v", p)
		}
		return c.NoContent(http.StatusNoContent)
	})

	if err := h(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
}

func TestMiddleware_AllowsCookieToken(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/pages", nil)
	req.AddCookie(
		&http.Cookie{
			Name:  "wiki_token",
			Value: signedToken(t, "secret", time.Now().Add(time.Hour)),
		},
	)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := Middleware(Config{JWTSecret: "secret"})(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := h(c); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestMiddleware_ExpiredTokenGetsUnauthorizedForAPI(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/pages", nil)
	req.Header.Set(
		echo.HeaderAuthorization,
		"Bearer "+signedToken(t, "secret", time.Now().Add(-time.Hour)),
	)
	req.Header.Set(echo.HeaderAccept, "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := Middleware(Config{JWTSecret: "secret"})(func(c echo.Context) error {
		t.Fatal("next should not be called")
		return nil
	})

	_ = h(c)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestMiddleware_BadSignatureRedirectsForHTML(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/pages/abc", nil)
	req.Header.Set(echo.HeaderAccept, "text/html")
	req.Header.Set(
		echo.HeaderAuthorization,
		"Bearer "+signedToken(t, "wrong-secret", time.Now().Add(time.Hour)),
	)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := Middleware(
		Config{JWTSecret: "secret", SSOLoginURL: "/sso/login"},
	)(
		func(c echo.Context) error {
			t.Fatal("next should not be called")
			return nil
		},
	)

	_ = h(c)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get(echo.HeaderLocation)
	if location != "/sso/login?redirectTo=%2Fpages%2Fabc" {
		t.Fatalf("unexpected redirect location: %s", location)
	}
}

func signedToken(t *testing.T, secret string, exp time.Time) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Data: claimsData{
			User: 42,
			Role: "EMPLOYEE",
		},
		Capabilities: []string{"page:read", "page:write"},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})

	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return s
}
