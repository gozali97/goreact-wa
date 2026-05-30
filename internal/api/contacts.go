package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListContacts returns contacts with optional search filter. (F10)
func (h *Handler) ListContacts(c *gin.Context) {
	search := c.Query("search")
	contacts, err := h.repo.ListContacts(search)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"contacts": contacts})
}

// ContactDetail returns a single contact. (F10)
func (h *Handler) ContactDetail(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid contact id")
		return
	}
	contact, err := h.repo.GetContact(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if contact == nil {
		fail(c, http.StatusNotFound, "contact not found")
		return
	}
	ok(c, gin.H{"contact": contact})
}
