package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"wa-proxy/internal/logsvc"
)

// LogDays returns the list of available log dates (newest first).
func (h *Handler) LogDays(c *gin.Context) {
	svc := logsvc.Default()
	if svc == nil {
		fail(c, http.StatusServiceUnavailable, "log service not available")
		return
	}
	days, err := svc.Days()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"days": days})
}

// LogEntries returns filtered, paginated log entries for a day.
func (h *Handler) LogEntries(c *gin.Context) {
	svc := logsvc.Default()
	if svc == nil {
		fail(c, http.StatusServiceUnavailable, "log service not available")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))

	res, err := svc.QueryEntries(logsvc.Query{
		Day:     c.Query("day"),
		Search:  c.Query("search"),
		Status:  c.Query("status"),
		Level:   c.Query("level"),
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"result": res})
}

// LogDownload streams a day's raw log file as an attachment.
func (h *Handler) LogDownload(c *gin.Context) {
	svc := logsvc.Default()
	if svc == nil {
		fail(c, http.StatusServiceUnavailable, "log service not available")
		return
	}
	day := c.Query("day")
	path := svc.FilePath(day)
	name := "wa-proxy-" + day + ".log"
	if day == "" {
		name = "wa-proxy.log"
	}
	c.Header("Content-Disposition", "attachment; filename=\""+name+"\"")
	c.File(path)
}
