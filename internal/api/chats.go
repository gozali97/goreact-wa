package api

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"wa-proxy/internal/whatsapp"

	"github.com/gin-gonic/gin"
)

// ListConversations returns the inbox: conversations with last message. (F07)
func (h *Handler) ListConversations(c *gin.Context) {
	convs, err := h.repo.ListConversations()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"conversations": convs})
}

// ChatDetail returns message history for a conversation. (F08)
func (h *Handler) ChatDetail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid conversation id")
		return
	}
	conv, err := h.repo.GetConversation(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if conv == nil {
		fail(c, http.StatusNotFound, "conversation not found")
		return
	}
	msgs, err := h.repo.ListMessages(id, 500)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.repo.MarkConversationRead(id)
	ok(c, gin.H{"conversation": conv, "messages": msgs})
}

type replyReq struct {
	Message string `json:"message" binding:"required"`
}

// Reply sends a message within an existing conversation. (F09)
func (h *Handler) Reply(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid conversation id")
		return
	}
	var req replyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "message is required")
		return
	}
	conv, err := h.repo.GetConversation(id)
	if err != nil || conv == nil || conv.Contact == nil {
		fail(c, http.StatusNotFound, "conversation not found")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}

	res, err := h.wa.SendText(conv.Contact.Phone, req.Message)
	if err != nil {
		failSend(c, err)
		return
	}
	ok(c, gin.H{"message_id": res.MessageID})
}

// ReplyMedia sends an image or document within an existing conversation via
// multipart/form-data (fields: caption, file). The type is chosen from the
// uploaded file's MIME type: image/* → image, everything else → document. (F09)
func (h *Handler) ReplyMedia(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid conversation id")
		return
	}
	conv, err := h.repo.GetConversation(id)
	if err != nil || conv == nil || conv.Contact == nil {
		fail(c, http.StatusNotFound, "conversation not found")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}

	caption := c.PostForm("caption")
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
	phone := conv.Contact.Phone

	var res *whatsapp.SendResult
	if strings.HasPrefix(mimeType, "image/") {
		res, err = h.wa.SendImage(phone, caption, data, mimeType)
	} else {
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		res, err = h.wa.SendDocument(phone, caption, fileHeader.Filename, data, mimeType)
	}
	if err != nil {
		failSend(c, err)
		return
	}
	ok(c, gin.H{"message_id": res.MessageID})
}

func parseUintParam(c *gin.Context, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}

type startChatReq struct {
	Phone   string `json:"phone" binding:"required"`
	Message string `json:"message" binding:"required"`
}

// StartChat begins a new conversation with a single number (or reuses the
// existing one if the contact already exists). It optionally verifies the
// number is on WhatsApp, sends the first message, and returns the resulting
// conversation so the UI can open it. (Single-target counterpart to broadcast.)
func (h *Handler) StartChat(c *gin.Context) {
	var req startChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "phone and message are required")
		return
	}
	if !h.wa.IsConnected() {
		fail(c, http.StatusServiceUnavailable, "whatsapp not connected")
		return
	}

	phone := normalizePhone(req.Phone)
	if phone == "" {
		fail(c, http.StatusBadRequest, "invalid phone number")
		return
	}

	// Verify the number is registered on WhatsApp (avoids ban risk + dead sends).
	if registered, err := h.wa.IsRegistered(phone); err == nil && !registered {
		fail(c, http.StatusUnprocessableEntity, "number is not registered on WhatsApp")
		return
	}

	res, err := h.wa.SendText(phone, req.Message)
	if err != nil {
		failSend(c, err)
		return
	}

	// SendText already upserted the contact + conversation (auto-sync if the
	// number was already a contact). Return the conversation so the UI opens it.
	conv, _ := h.repo.GetConversationByPhone(phone)
	resp := gin.H{"message_id": res.MessageID}
	if conv != nil {
		resp["conversation"] = conv
		resp["conversation_id"] = conv.ID
	}
	ok(c, resp)
}
