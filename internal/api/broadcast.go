package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"wa-proxy/internal/model"
)

type createBroadcastReq struct {
	Name    string   `json:"name"`
	Message string   `json:"message" binding:"required"`
	Phones  []string `json:"phones"`
}

// CreateBroadcast queues a new broadcast campaign. (F06)
//
// Only `message` is required. If `phones` is omitted/empty, the broadcast is
// sent to all known contacts. Recipients are de-duplicated.
func (h *Handler) CreateBroadcast(c *gin.Context) {
	var req createBroadcastReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "message is required")
		return
	}

	// Build the recipient list. Explicit phones take precedence; otherwise
	// fall back to all saved contacts.
	var raw []string
	if len(req.Phones) > 0 {
		raw = req.Phones
	} else {
		contacts, err := h.repo.AllContactPhones()
		if err != nil {
			fail(c, http.StatusInternalServerError, err.Error())
			return
		}
		raw = contacts
	}

	// Normalize + de-duplicate.
	seen := make(map[string]struct{}, len(raw))
	phones := make([]string, 0, len(raw))
	for _, p := range raw {
		np := normalizePhone(p)
		if np == "" {
			continue
		}
		if _, dup := seen[np]; dup {
			continue
		}
		seen[np] = struct{}{}
		phones = append(phones, np)
	}

	if len(phones) == 0 {
		fail(c, http.StatusBadRequest, "no recipients: provide phones or add contacts first")
		return
	}

	name := req.Name
	if name == "" {
		name = "Broadcast"
	}
	b := &model.Broadcast{
		Name:        name,
		Message:     req.Message,
		Status:      model.BroadcastPending,
		TotalTarget: len(phones),
	}
	if err := h.repo.CreateBroadcast(b, phones); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.broadcast.Trigger(b.ID)
	ok(c, gin.H{"broadcast_id": b.ID, "total_target": b.TotalTarget})
}

// ListBroadcasts returns broadcast history. (F06)
func (h *Handler) ListBroadcasts(c *gin.Context) {
	bs, err := h.repo.ListBroadcasts()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"broadcasts": bs})
}

// BroadcastDetail returns a broadcast and its per-recipient details. (F06)
func (h *Handler) BroadcastDetail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid broadcast id")
		return
	}
	b, err := h.repo.GetBroadcast(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if b == nil {
		fail(c, http.StatusNotFound, "broadcast not found")
		return
	}
	details, err := h.repo.ListBroadcastDetails(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"broadcast": b, "details": details})
}
