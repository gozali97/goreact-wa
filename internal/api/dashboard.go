package api

import (
	"github.com/gin-gonic/gin"
)

// DashboardStats returns aggregate counters for the dashboard home page.
func (h *Handler) DashboardStats(c *gin.Context) {
	contacts, _ := h.repo.CountContacts()
	chats, _ := h.repo.CountConversations()
	broadcasts, _ := h.repo.CountBroadcasts()

	session, _ := h.repo.GetSession()
	device := gin.H{
		"status":    h.wa.Status(),
		"connected": h.wa.IsConnected(),
	}
	if session != nil {
		device["phone"] = session.Phone
		device["push_name"] = session.PushName
	}

	ok(c, gin.H{
		"total_contacts":   contacts,
		"total_chats":      chats,
		"total_broadcasts": broadcasts,
		"device":           device,
	})
}
