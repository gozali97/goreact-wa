package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"wa-proxy/internal/logsvc"
)

// StorageOverview returns disk usage + per-category storage stats.
func (h *Handler) StorageOverview(c *gin.Context) {
	cats, total, err := h.store.Stats()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp := gin.H{
		"categories": cats,
		"total":      total,
	}
	if d, err := h.store.Disk(); err == nil {
		resp["disk"] = d
	}
	ok(c, resp)
}

// StorageFiles lists files within a category.
func (h *Handler) StorageFiles(c *gin.Context) {
	category := c.Param("category")
	files, err := h.store.ListFiles(category)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ok(c, gin.H{"files": files})
}

// StorageDeleteFile removes a single file.
func (h *Handler) StorageDeleteFile(c *gin.Context) {
	category := c.Param("category")
	name := c.Query("name")
	if name == "" {
		fail(c, http.StatusBadRequest, "name is required")
		return
	}
	if err := h.store.DeleteFile(category, name); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	logsvc.Info("storage.delete", "deleted file", map[string]interface{}{"category": category, "name": name})
	ok(c, nil)
}

// StorageClearCategory empties one category.
func (h *Handler) StorageClearCategory(c *gin.Context) {
	category := c.Param("category")
	n, err := h.store.ClearCategory(category)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	logsvc.Issue("storage.clear", "cleared category", map[string]interface{}{"category": category, "removed": n})
	ok(c, gin.H{"removed": n})
}

// StorageClearAll empties every managed category (clear storage / cache).
func (h *Handler) StorageClearAll(c *gin.Context) {
	n, err := h.store.ClearAll()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	logsvc.Issue("storage.clear_all", "cleared all storage", map[string]interface{}{"removed": n})
	ok(c, gin.H{"removed": n})
}
