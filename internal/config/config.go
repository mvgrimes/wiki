package config

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Environment string
	LogLevel    string

	ErrorMonitoringEnabled     bool
	HoneybadgerAPIKey          string
	TestErrorMonitoringEnabled bool
	Branding                   string
	LogoURL                    string
	LogoHeroURL                string
	FaviconURL                 string

	ServerAddress string
	ServerPort    int
	ServerURL     string

	DatabaseURL string
	JWTSecret   string

	SSOLoginURL string
	JWTCookie   string

	S3Endpoint        string
	S3Bucket          string
	S3Region          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3ForcePathStyle  bool
}

func Load() (*Config, error) {
	serverPort, err := intEnvOrDefault("WIKI_SERVER_PORT", 8080)
	if err != nil {
		return nil, err
	}

	s3ForcePathStyle, err := boolEnvOrDefault("WIKI_S3_FORCE_PATH_STYLE", true)
	if err != nil {
		return nil, err
	}

	testErrorMonitoringEnabled, err := boolEnvOrDefault("FILESHARE_TEST_ERROR_MONITORING", false)
	if err != nil {
		return nil, err
	}

	environment := envOrDefault("WIKI_ENVIRONMENT", "development")
	if legacyEnvironment := strings.TrimSpace(os.Getenv("FILESHARE_ENVIRONMENT")); legacyEnvironment != "" {
		environment = legacyEnvironment
	}

	errorMonitoringEnabled := strings.EqualFold(environment, "production") || testErrorMonitoringEnabled

	cfg := &Config{
		Environment:                environment,
		LogLevel:                   envOrDefault("WIKI_LOG_LEVEL", "info"),
		ErrorMonitoringEnabled:     errorMonitoringEnabled,
		HoneybadgerAPIKey:          strings.TrimSpace(os.Getenv("FILESHARE_HONEYBADGER_API_KEY")),
		TestErrorMonitoringEnabled: testErrorMonitoringEnabled,
		Branding:                   envOrDefault("WIKI_BRANDING", "Wiki"),
		LogoURL:                    imageEnvOrDataURL("WIKI_LOGO"),
		LogoHeroURL:                imageEnvOrDataURL("WIKI_LOGO_HERO"),
		FaviconURL:                 imageEnvOrDataURL("WIKI_FAVICON"),
		ServerAddress:              envOrDefault("WIKI_SERVER_ADDRESS", "0.0.0.0"),
		ServerPort:                 serverPort,
		ServerURL:                  strings.TrimSpace(os.Getenv("WIKI_SERVER_URL")),
		DatabaseURL:                envOrDefault("WIKI_DATABASE_URL", "./data/wiki.db"),
		JWTSecret:                  strings.TrimSpace(os.Getenv("WIKI_JWT_SECRET")),
		SSOLoginURL:                strings.TrimSpace(os.Getenv("WIKI_SSO_LOGIN_URL")),
		JWTCookie:                  envOrDefault("WIKI_JWT_COOKIE", "auth._token.local"),
		S3Endpoint:                 strings.TrimSpace(os.Getenv("WIKI_S3_ENDPOINT")),
		S3Bucket:                   envOrDefault("WIKI_S3_BUCKET", "wiki-uploads"),
		S3Region:                   envOrDefault("WIKI_AWS_REGION", "us-east-1"),
		S3AccessKeyID:              strings.TrimSpace(os.Getenv("WIKI_AWS_ACCESS_KEY_ID")),
		S3SecretAccessKey:          strings.TrimSpace(os.Getenv("WIKI_AWS_SECRET_ACCESS_KEY")),
		S3ForcePathStyle:           s3ForcePathStyle,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func imageEnvOrDataURL(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return ""
	}

	lower := strings.ToLower(v)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "data:") {
		return v
	}

	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return v
	}
	mimeType := http.DetectContentType(decoded)
	if mimeType == "application/octet-stream" {
		mimeType = "image/png"
	}

	return "data:" + mimeType + ";base64," + v
}

func (c *Config) Validate() error {
	missing := make([]string, 0, 2)
	if strings.TrimSpace(c.ServerURL) == "" {
		missing = append(missing, "WIKI_SERVER_URL")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		missing = append(missing, "WIKI_JWT_SECRET")
	}
	if strings.TrimSpace(c.SSOLoginURL) == "" {
		missing = append(missing, "WIKI_SSO_LOGIN_URL")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if c.ServerPort <= 0 {
		return fmt.Errorf("WIKI_SERVER_PORT must be greater than zero")
	}

	return nil
}

func envOrDefault(key, fallback string) string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func intEnvOrDefault(key string, fallback int) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return n, nil
}

func boolEnvOrDefault(key string, fallback bool) (bool, error) {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback, nil
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean (true/false)", key)
	}
}
