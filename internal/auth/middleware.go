package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const principalContextKey = "auth.principal"

type Config struct {
	JWTSecret   string
	CookieName  string
	SSOLoginURL string
	ServerURL   string
	Skipper     func(echo.Context) bool
}

type Principal struct {
	Subject      string
	Email        string
	User         int64
	Role         string
	Capabilities []string
}

type claims struct {
	Data         claimsData `json:"data"`
	Capabilities []string   `json:"capabilities"`
	jwt.RegisteredClaims
}
type claimsData struct {
	// Email        string   `json:"email"`
	User int64  `json:"user"`
	Role string `json:"role"`
}

func Middleware(cfg Config) echo.MiddlewareFunc {
	if cfg.CookieName == "" {
		cfg.CookieName = "wiki_token"
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			tokenString, err := tokenFromRequest(c, cfg.CookieName)
			if err != nil {
				return unauthenticated(c, cfg)
			}

			principal, err := parsePrincipal(tokenString, cfg.JWTSecret)
			if err != nil {
				slog.Warn("invalid token", "error", err)
				return unauthenticated(c, cfg)
			}

			slog.Info(
				"authenticated user",
				"user",
				principal.User,
				"email",
				principal.Email,
				"role",
				principal.Role,
			)
			if principal.Role != "EMPLOYEE" {
				return unauthenticated(c, cfg)
			}

			c.Set(principalContextKey, principal)
			return next(c)
		}
	}
}

func PrincipalFromContext(c echo.Context) (Principal, bool) {
	p, ok := c.Get(principalContextKey).(Principal)
	return p, ok
}

func tokenFromRequest(c echo.Context, cookieName string) (string, error) {
	authorization := c.Request().Header.Get(echo.HeaderAuthorization)
	if authorization != "" {
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "" {
			return parts[1], nil
		}
		return "", fmt.Errorf("invalid authorization header")
	}

	cookie, err := c.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return "", fmt.Errorf("missing token")
	}

	return cookie.Value, nil
}

func parsePrincipal(tokenString, jwtSecret string) (Principal, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return Principal{}, err
	}

	parsedClaims, ok := token.Claims.(*claims)
	if !ok || !token.Valid {
		return Principal{}, fmt.Errorf("invalid token claims")
	}

	return Principal{
		Subject: parsedClaims.Subject,
		// Email:        parsedClaims.Data.Email,
		User:         parsedClaims.Data.User,
		Role:         parsedClaims.Data.Role,
		Capabilities: parsedClaims.Capabilities,
	}, nil
}

func unauthenticated(c echo.Context, cfg Config) error {
	if wantsHTML(c) {
		redirectURL := withReturnTo(
			cfg.SSOLoginURL,
			cfg.ServerURL+c.Request().URL.RequestURI(),
		)
		return c.Redirect(http.StatusFound, redirectURL)
	}

	return c.NoContent(http.StatusUnauthorized)
}

func wantsHTML(c echo.Context) bool {
	accept := c.Request().Header.Get(echo.HeaderAccept)
	if strings.Contains(accept, "text/html") {
		return true
	}

	if strings.HasPrefix(c.Request().URL.Path, "/api") {
		return false
	}

	if strings.Contains(accept, "application/json") {
		return false
	}

	return true
}

func withReturnTo(baseURL, returnTo string) string {
	if baseURL == "" {
		baseURL = "/sso/login"
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	query := parsed.Query()
	query.Set("redirectTo", returnTo)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
