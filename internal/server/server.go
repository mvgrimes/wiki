package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"wiki/internal/auth"
)

type Config struct {
	Address     string
	Port        int
	JWTSecret   string
	SSOLoginURL string
	JWTCookie   string
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
