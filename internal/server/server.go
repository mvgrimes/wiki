package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
)

type Config struct {
	Address string
	Port    string
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

	return Config{Address: address, Port: port}
}

func Start(ctx context.Context, cfg Config) error {
	e := echo.New()
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
