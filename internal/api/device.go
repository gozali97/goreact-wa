package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Connect initiates the WhatsApp login / reconnect flow. (F01)
func (h *Handler) Connect(c *gin.Context) {
	if err := h.wa.Connect(); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"status": h.wa.Status()})
}

// QR returns the latest QR code string (also pushed over WebSocket). (F01)
func (h *Handler) QR(c *gin.Context) {
	ok(c, gin.H{"code": h.wa.LastQR(), "status": h.wa.Status()})
}

// Status returns device status and account metadata. (F02)
func (h *Handler) Status(c *gin.Context) {
	session, _ := h.repo.GetSession()
	resp := gin.H{
		"status":    h.wa.Status(),
		"connected": h.wa.IsConnected(),
	}
	if session != nil {
		resp["phone"] = session.Phone
		resp["push_name"] = session.PushName
		resp["jid"] = session.JID
		resp["last_seen"] = session.LastSeen
	}
	ok(c, resp)
}

// Logout terminates the WhatsApp session. (F02)
func (h *Handler) Logout(c *gin.Context) {
	if err := h.wa.Logout(); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"status": h.wa.Status()})
}

// GetAPIKey returns the current integration API key (empty if none yet).
func (h *Handler) GetAPIKey(c *gin.Context) {
	ok(c, gin.H{"api_key": h.keys.Get()})
}

// GenerateAPIKey creates (or regenerates) the integration API key and returns
// the new value. Used by the Device page "Generate" / "Regenerate" buttons.
func (h *Handler) GenerateAPIKey(c *gin.Context) {
	key, err := h.keys.Generate()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"api_key": key})
}
