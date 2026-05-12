package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"wiki/internal/auth"
)

type Config struct {
	Address     string
	Port        string
	JWTSecret   string
	SSOLoginURL string
	JWTCookie   string
}

func ConfigFromEnv() Config {
	address := os.Getenv("WIKI_SERVER_ADDRESS")
	if address == "" {
		address = "0.0.0.0"
	}

	port := os.Getenv("WIKI_SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("WIKI_JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret"
	}

	ssoLoginURL := os.Getenv("WIKI_SSO_LOGIN_URL")
	if ssoLoginURL == "" {
		ssoLoginURL = "/sso/login"
	}

	jwtCookie := os.Getenv("WIKI_JWT_COOKIE")
	if jwtCookie == "" {
		jwtCookie = "wiki_token"
	}

	return Config{
		Address:     address,
		Port:        port,
		JWTSecret:   jwtSecret,
		SSOLoginURL: ssoLoginURL,
		JWTCookie:   jwtCookie,
	}
}

func Start(ctx context.Context, cfg Config) error {
	e := echo.New()
	e.Use(auth.Middleware(auth.Config{
		JWTSecret:   cfg.JWTSecret,
		CookieName:  cfg.JWTCookie,
		SSOLoginURL: cfg.SSOLoginURL,
		Skipper: func(c echo.Context) bool {
			path := c.Path()
			return path == "/healthz" || path == "/readyz"
		},
	}))
	e.GET("/healthz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.GET("/readyz", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	addr := fmt.Sprintf("%s:%s", cfg.Address, cfg.Port)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = e.Shutdown(shutdownCtx)
	}()

	return e.Start(addr)
}
