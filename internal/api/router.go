package api

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"wa-proxy/internal/config"
	"wa-proxy/internal/webui"
	"wa-proxy/internal/ws"
)

// NewRouter builds the Gin engine: REST API (basic-auth protected), the
// WebSocket endpoint, media file serving and the embedded React SPA.
func NewRouter(cfg *config.Config, h *Handler, hub *ws.Hub) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(RequestLogger())

	// CORS so the Vite dev server (different origin) can call the API.
	if !cfg.IsProduction() {
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
		}))
	}

	// Health check (unauthenticated).
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth check — used by the dashboard login screen to validate credentials.
	r.GET("/api/auth/check", BasicAuth(cfg), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	// WebSocket endpoint (auth handled at connection via basic auth too).
	r.GET("/ws", BasicAuth(cfg), func(c *gin.Context) {
		ws.ServeWS(hub, c)
	})

	// REST API group — dashboard endpoints use Basic Auth; the messaging /
	// integration endpoints accept either X-Api-Key or Basic (APIAuth).
	api := r.Group("/api", BasicAuth(cfg))
	{
		// Device / session (F01, F02)
		device := api.Group("/device")
		{
			device.POST("/connect", h.Connect)
			device.GET("/qr", h.QR)
			device.GET("/status", h.Status)
			device.POST("/logout", h.Logout)
			// API key management (DB-backed)
			device.GET("/api-key", h.GetAPIKey)
			device.POST("/api-key/generate", h.GenerateAPIKey)
		}

		// Inbox / chats (F07, F08, F09)
		api.GET("/conversations", h.ListConversations)
		api.GET("/conversations/:id", h.ChatDetail)
		api.POST("/conversations/:id/reply", h.Reply)
		api.POST("/conversations/:id/reply-media", h.ReplyMedia)

		// Contacts (F10)
		api.GET("/contacts", h.ListContacts)
		api.GET("/contacts/:id", h.ContactDetail)

		// Broadcast (F06)
		api.POST("/broadcasts", h.CreateBroadcast)
		api.GET("/broadcasts", h.ListBroadcasts)
		api.GET("/broadcasts/:id", h.BroadcastDetail)

		// Dashboard stats
		api.GET("/dashboard/stats", h.DashboardStats)

		// Storage monitoring & file manager
		api.GET("/storage-admin/overview", h.StorageOverview)
		api.GET("/storage-admin/files/:category", h.StorageFiles)
		api.DELETE("/storage-admin/files/:category", h.StorageDeleteFile)
		api.DELETE("/storage-admin/category/:category", h.StorageClearCategory)
		api.DELETE("/storage-admin/all", h.StorageClearAll)

		// Logs viewer
		api.GET("/logs/days", h.LogDays)
		api.GET("/logs", h.LogEntries)
		api.GET("/logs/download", h.LogDownload)
	}

	// Integration API — usable by external apps via X-Api-Key (WAHA-style),
	// with dashboard Basic Auth as a fallback.
	ext := r.Group("/api", APIAuth(cfg, h.keys))
	{
		// Messaging (F03, F04, F05)
		ext.POST("/messages/send", h.SendMessage)
		ext.POST("/messages/send-image", h.SendImage)
		ext.POST("/messages/send-file", h.SendFile)
		ext.POST("/messages/send-buttons", h.SendButtons)

		// Anti-blocking / presence helpers (WAHA-inspired)
		ext.POST("/messages/check-exists", h.CheckExists)
		ext.POST("/messages/typing", h.SetTyping)
		ext.POST("/messages/seen", h.SendSeen)

		// Start a new 1:1 chat (single-target; auto-syncs existing contact)
		ext.POST("/chats/start", h.StartChat)
	}

	// Media files (basic-auth protected so chat attachments aren't public).
	r.GET("/storage/*filepath", BasicAuth(cfg), func(c *gin.Context) {
		fp := c.Param("filepath")
		c.FileFromFS(fp, gin.Dir(cfg.StoragePath, false))
	})

	// Embedded React SPA (catch-all, must be registered last).
	spa := webui.Handler()
	r.NoRoute(func(c *gin.Context) {
		// Don't let the SPA swallow API/storage/ws 404s.
		p := c.Request.URL.Path
		if isReservedPath(p) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "not found"})
			return
		}
		spa.ServeHTTP(c.Writer, c.Request)
	})

	return r
}

func isReservedPath(p string) bool {
	for _, prefix := range []string{"/api", "/ws", "/storage", "/healthz"} {
		if p == prefix || len(p) >= len(prefix) && p[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
