package api

import (
	"io"
	"net/http"
	"strings"
	"wa-proxy/internal/whatsapp"

	"github.com/gin-gonic/gin"
)

type sendMessageReq struct {
	Phone   string `json:"phone" binding:"required"`
	Message string `json:"message" binding:"required"`
}

// SendMessage sends a text message. (F03)
func (h *Handler) SendMessage(c *gin.Context) {
	var req sendMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "phone and message are required")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}

	res, err := h.wa.SendText(normalizePhone(req.Phone), req.Message)
	if err != nil {
		failSend(c, err)
		return
	}
	ok(c, gin.H{"message_id": res.MessageID})
}

// SendImage sends an image via multipart/form-data. (F04)
func (h *Handler) SendImage(c *gin.Context) {
	phone := c.PostForm("phone")
	caption := c.PostForm("caption")
	if phone == "" {
		fail(c, http.StatusBadRequest, "phone is required")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "file is required")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		fail(c, http.StatusInternalServerError, "cannot open file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		fail(c, http.StatusInternalServerError, "cannot read file")
		return
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	res, err := h.wa.SendImage(normalizePhone(phone), caption, data, mimeType)
	if err != nil {
		failSend(c, err)
		return
	}
	ok(c, gin.H{"message_id": res.MessageID})
}

// SendFile sends a document via multipart/form-data. (F05)
func (h *Handler) SendFile(c *gin.Context) {
	phone := c.PostForm("phone")
	caption := c.PostForm("caption")
	if phone == "" {
		fail(c, http.StatusBadRequest, "phone is required")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "file is required")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		fail(c, http.StatusInternalServerError, "cannot open file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		fail(c, http.StatusInternalServerError, "cannot read file")
		return
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	res, err := h.wa.SendDocument(normalizePhone(phone), caption, fileHeader.Filename, data, mimeType)
	if err != nil {
		failSend(c, err)
		return
	}
	ok(c, gin.H{"message_id": res.MessageID})
}

type buttonReq struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type sendButtonsReq struct {
	Phone   string      `json:"phone" binding:"required"`
	Message string      `json:"message" binding:"required"`
	Footer  string      `json:"footer"`
	Buttons []buttonReq `json:"buttons" binding:"required"`
}

// SendButtons sends an interactive message with URL call-to-action buttons.
func (h *Handler) SendButtons(c *gin.Context) {
	var req sendButtonsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "phone, message and buttons are required")
		return
	}
	if len(req.Buttons) == 0 {
		fail(c, http.StatusBadRequest, "at least one button is required")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}

	buttons := make([]whatsapp.URLButton, 0, len(req.Buttons))
	for _, b := range req.Buttons {
		if b.Text == "" || b.URL == "" {
			fail(c, http.StatusBadRequest, "each button needs text and url")
			return
		}
		buttons = append(buttons, whatsapp.URLButton{Text: b.Text, URL: b.URL})
	}

	res, err := h.wa.SendButtons(normalizePhone(req.Phone), req.Message, req.Footer, buttons)
	if err != nil {
		failSend(c, err)
		return
	}
	ok(c, gin.H{"message_id": res.MessageID})
}

// normalizePhone strips common formatting from a phone number.
func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	return phone
}

type checkExistsReq struct {
	Phones []string `json:"phones" binding:"required"`
}

// CheckExists reports which numbers are registered on WhatsApp (WAHA-style).
func (h *Handler) CheckExists(c *gin.Context) {
	var req checkExistsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "phones is required")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}
	phones := make([]string, 0, len(req.Phones))
	for _, p := range req.Phones {
		if np := normalizePhone(p); np != "" {
			phones = append(phones, np)
		}
	}
	results, err := h.wa.CheckOnWhatsApp(phones)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]gin.H, 0, len(results))
	for _, r := range results {
		out = append(out, gin.H{
			"phone":  r.Query,
			"exists": r.IsIn,
			"jid":    r.JID.String(),
		})
	}
	ok(c, gin.H{"results": out})
}

type typingReq struct {
	Phone  string `json:"phone" binding:"required"`
	Typing bool   `json:"typing"`
}

// SetTyping toggles the typing indicator for a chat (WAHA startTyping/stopTyping).
func (h *Handler) SetTyping(c *gin.Context) {
	var req typingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "phone is required")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}
	if err := h.wa.SendTyping(normalizePhone(req.Phone), req.Typing); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}

type seenReq struct {
	Phone      string   `json:"phone" binding:"required"`
	MessageIDs []string `json:"message_ids" binding:"required"`
}

// SendSeen marks messages as read (WAHA sendSeen).
func (h *Handler) SendSeen(c *gin.Context) {
	var req seenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "phone and message_ids are required")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}
	if err := h.wa.MarkSeen(normalizePhone(req.Phone), req.MessageIDs); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, nil)
}
