package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"wa-proxy/internal/apikey"
	"wa-proxy/internal/config"
	"wa-proxy/internal/repository"
	"wa-proxy/internal/storage"
	"wa-proxy/internal/whatsapp"
	"wa-proxy/internal/worker"
	"wa-proxy/internal/ws"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	cfg       *config.Config
	repo      *repository.Repository
	wa        *whatsapp.Manager
	hub       *ws.Hub
	store     *storage.Store
	sender    *worker.MessageSender
	broadcast *worker.BroadcastWorker
	keys      *apikey.Service
}

// NewHandler builds an API Handler.
func NewHandler(
	cfg *config.Config,
	repo *repository.Repository,
	wa *whatsapp.Manager,
	hub *ws.Hub,
	store *storage.Store,
	sender *worker.MessageSender,
	broadcast *worker.BroadcastWorker,
	keys *apikey.Service,
) *Handler {
	return &Handler{
		cfg:       cfg,
		repo:      repo,
		wa:        wa,
		hub:       hub,
		store:     store,
		sender:    sender,
		broadcast: broadcast,
		keys:      keys,
	}
}

// ok writes a standard success envelope.
func ok(c *gin.Context, data gin.H) {
	if data == nil {
		data = gin.H{}
	}
	data["success"] = true
	c.JSON(http.StatusOK, data)
}

// fail writes a standard error envelope.
func fail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"success": false, "error": msg})
}

// failSend maps a send error to the right HTTP response, returning 429 with a
// retry hint for rate-limit errors.
func failSend(c *gin.Context, err error) {
	if rl, ok := whatsapp.IsRateLimited(err); ok {
		retry := int(rl.RetryAfter.Seconds())
		if retry < 1 {
			retry = 1
		}
		c.Header("Retry-After", strconv.Itoa(retry))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"success":     false,
			"error":       "rate limited: please wait before messaging this number again",
			"retry_after": retry,
		})
		return
	}
	fail(c, http.StatusInternalServerError, err.Error())
}
