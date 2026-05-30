package api

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"wa-proxy/internal/apikey"
	"wa-proxy/internal/config"
	"wa-proxy/internal/logsvc"
)

// validCredentials performs a constant-time comparison of the supplied
// username/password against the configured values.
func validCredentials(cfg *config.Config, user, pass string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(cfg.BasicAuthUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(cfg.BasicAuthPass)) == 1
	return userOK && passOK
}

// credsFromToken decodes a base64("user:pass") token (used for the WebSocket
// query parameter, since browsers cannot set headers on WebSocket requests).
func credsFromToken(token string) (user, pass string, ok bool) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// BasicAuth returns a Gin middleware enforcing authentication. It accepts
// credentials either via the standard HTTP Basic Authorization header or via an
// `access_token` query parameter (base64 of "user:pass"). The query-parameter
// path exists so the dashboard WebSocket can authenticate.
func BasicAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, pass, ok := c.Request.BasicAuth(); ok && validCredentials(cfg, user, pass) {
			c.Next()
			return
		}
		if token := c.Query("access_token"); token != "" {
			if user, pass, ok := credsFromToken(token); ok && validCredentials(cfg, user, pass) {
				c.Next()
				return
			}
		}

		c.Header("WWW-Authenticate", `Basic realm="WA Proxy"`)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized",
		})
	}
}

// APIAuth protects the external integration API. It accepts the WAHA-style
// `X-Api-Key` header (validated against the DB-stored key) and also falls back
// to the dashboard Basic Auth, so the dashboard can call the same endpoints.
func APIAuth(cfg *config.Config, keys *apikey.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if current := keys.Get(); current != "" {
			key := c.GetHeader("X-Api-Key")
			if key == "" {
				key = c.Query("api_key")
			}
			if key != "" && subtle.ConstantTimeCompare([]byte(key), []byte(current)) == 1 {
				c.Next()
				return
			}
		}
		// Fall back to dashboard credentials (Basic header or access_token).
		if user, pass, ok := c.Request.BasicAuth(); ok && validCredentials(cfg, user, pass) {
			c.Next()
			return
		}
		if token := c.Query("access_token"); token != "" {
			if user, pass, ok := credsFromToken(token); ok && validCredentials(cfg, user, pass) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "unauthorized: provide X-Api-Key or Basic auth",
		})
	}
}

// RequestLogger records every API request to the structured log service,
// classifying the outcome by HTTP status (2xx success, 4xx issue, 5xx failed).
// Static asset, WebSocket and log-viewer requests are skipped to avoid noise.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// Skip non-API and noisy paths.
		if !strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/api/logs") {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		status := c.Writer.Status()

		outcome := logsvc.StatusSuccess
		level := logsvc.LevelInfo
		switch {
		case status >= 500:
			outcome, level = logsvc.StatusFailed, logsvc.LevelError
		case status >= 400:
			outcome, level = logsvc.StatusIssue, logsvc.LevelWarn
		}

		fields := map[string]interface{}{
			"method":      c.Request.Method,
			"path":        path,
			"status":      status,
			"duration_ms": time.Since(start).Milliseconds(),
			"client_ip":   c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		src := c.Request.Method + " " + path
		logsvc.Default().Log(level, outcome, src,
			http.StatusText(status), fields)
	}
}
