package config

import (
	"strings"
	"testing"
)

func TestLoadSuccess(t *testing.T) {
	t.Setenv("WIKI_SERVER_ADDRESS", "127.0.0.1")
	t.Setenv("WIKI_SERVER_PORT", "9090")
	t.Setenv("WIKI_SERVER_URL", "https://wiki.test")
	t.Setenv("WIKI_ENVIRONMENT", "test")
	t.Setenv("WIKI_LOG_LEVEL", "debug")
	t.Setenv("WIKI_BRANDING", "Acme Wiki")
	t.Setenv("WIKI_LOGO", "https://cdn.example.com/logo.svg")
	t.Setenv("WIKI_LOGO_HERO", "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+X4Z8AAAAASUVORK5CYII=")
	t.Setenv("WIKI_FAVICON", "https://cdn.example.com/favicon.ico")
	t.Setenv("WIKI_DATABASE_URL", "./data/wiki.test.db")
	t.Setenv("WIKI_JWT_SECRET", "jwt-secret")
	t.Setenv("WIKI_SSO_LOGIN_URL", "/custom/sso/login")
	t.Setenv("WIKI_JWT_COOKIE", "wiki_auth")
	t.Setenv("WIKI_S3_ENDPOINT", "http://localhost:9100")
	t.Setenv("WIKI_S3_BUCKET", "wiki-test")
	t.Setenv("WIKI_AWS_REGION", "us-west-2")
	t.Setenv("WIKI_AWS_ACCESS_KEY_ID", "wiki")
	t.Setenv("WIKI_AWS_SECRET_ACCESS_KEY", "wiki1234")
	t.Setenv("WIKI_S3_FORCE_PATH_STYLE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.ServerPort != 9090 {
		t.Fatalf("ServerPort = %d, want %d", cfg.ServerPort, 9090)
	}
	if cfg.ServerAddress != "127.0.0.1" {
		t.Fatalf("ServerAddress = %q, want %q", cfg.ServerAddress, "127.0.0.1")
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.Branding != "Acme Wiki" {
		t.Fatalf("Branding = %q, want %q", cfg.Branding, "Acme Wiki")
	}
	if cfg.LogoURL != "https://cdn.example.com/logo.svg" {
		t.Fatalf("LogoURL = %q, want %q", cfg.LogoURL, "https://cdn.example.com/logo.svg")
	}
	if cfg.LogoHeroURL != "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+X4Z8AAAAASUVORK5CYII=" {
		t.Fatalf("LogoHeroURL = %q, want %q", cfg.LogoHeroURL, "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+X4Z8AAAAASUVORK5CYII=")
	}
	if cfg.FaviconURL != "https://cdn.example.com/favicon.ico" {
		t.Fatalf("FaviconURL = %q, want %q", cfg.FaviconURL, "https://cdn.example.com/favicon.ico")
	}
	if cfg.SSOLoginURL != "/custom/sso/login" {
		t.Fatalf("SSOLoginURL = %q, want %q", cfg.SSOLoginURL, "/custom/sso/login")
	}
	if !cfg.S3ForcePathStyle {
		t.Fatal("S3ForcePathStyle = false, want true")
	}
}

func TestLoadMissingRequiredEnv(t *testing.T) {
	t.Setenv("WIKI_SERVER_URL", "")
	t.Setenv("WIKI_JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want missing required environment variables")
	}

	msg := err.Error()
	for _, key := range []string{"WIKI_SERVER_URL", "WIKI_JWT_SECRET"} {
		if !strings.Contains(msg, key) {
			t.Fatalf("error %q does not contain %q", msg, key)
		}
	}
}

func TestLoadInvalidIntEnv(t *testing.T) {
	t.Setenv("WIKI_SERVER_PORT", "not-a-number")
	t.Setenv("WIKI_SERVER_URL", "https://wiki.test")
	t.Setenv("WIKI_JWT_SECRET", "jwt-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want invalid integer error")
	}
	if !strings.Contains(err.Error(), "WIKI_SERVER_PORT") {
		t.Fatalf("error = %q, want reference to WIKI_SERVER_PORT", err.Error())
	}
}

func TestLoadInvalidBoolEnv(t *testing.T) {
	t.Setenv("WIKI_SERVER_URL", "https://wiki.test")
	t.Setenv("WIKI_JWT_SECRET", "jwt-secret")
	t.Setenv("WIKI_S3_FORCE_PATH_STYLE", "not-a-bool")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want invalid boolean error")
	}
	if !strings.Contains(err.Error(), "WIKI_S3_FORCE_PATH_STYLE") {
		t.Fatalf("error = %q, want reference to WIKI_S3_FORCE_PATH_STYLE", err.Error())
	}
}
