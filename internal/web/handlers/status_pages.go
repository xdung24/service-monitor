package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xdung24/conductor/internal/models"
)

func (h *Handler) statusPageStore(c *gin.Context) *models.StatusPageStore {
	return models.NewStatusPageStore(h.userDB(c))
}

// StatusPageList renders the status pages management list.
func (h *Handler) StatusPageList(c *gin.Context) {
	pages, _ := h.statusPageStore(c).List()
	flash, _ := c.Cookie("sm_flash")
	if flash != "" {
		c.SetCookie("sm_flash", "", -1, "/", "", false, true)
	}
	c.HTML(http.StatusOK, "status_page_list.html", h.pageData(c, gin.H{
		"Pages": pages,
		"Flash": flash,
	}))
}

// StatusPageNew renders the create form.
func (h *Handler) StatusPageNew(c *gin.Context) {
	monitors, _ := h.monitorStore(c).List()
	c.HTML(http.StatusOK, "status_page_form.html", h.pageData(c, gin.H{
		"Page":             &models.StatusPage{},
		"IsNew":            true,
		"AllMonitors":      monitors,
		"LinkedMonitorIDs": map[int64]bool{},
		"Error":            "",
	}))
}

// StatusPageCreate handles the create form submission.
func (h *Handler) StatusPageCreate(c *gin.Context) {
	page, monitorIDs, err := statusPageFromForm(c)
	if err != nil {
		monitors, _ := h.monitorStore(c).List()
		c.HTML(http.StatusBadRequest, "status_page_form.html", gin.H{
			"Page": page, "IsNew": true, "AllMonitors": monitors,
			"LinkedMonitorIDs": map[int64]bool{}, "Error": err.Error(),
		})
		return
	}

	spStore := h.statusPageStore(c)
	id, err := spStore.Create(page)
	if err != nil {
		monitors, _ := h.monitorStore(c).List()
		c.HTML(http.StatusInternalServerError, "status_page_form.html", gin.H{
			"Page": page, "IsNew": true, "AllMonitors": monitors,
			"LinkedMonitorIDs": map[int64]bool{}, "Error": err.Error(),
		})
		return
	}

	_ = spStore.SetMonitors(id, monitorIDs)
	c.Redirect(http.StatusFound, "/status-pages")
}

// StatusPageEdit renders the edit form.
func (h *Handler) StatusPageEdit(c *gin.Context) {
	page, ok := h.getStatusPage(c)
	if !ok {
		return
	}
	spStore := h.statusPageStore(c)
	monitors, _ := h.monitorStore(c).List()
	linkedIDs, _ := spStore.ListMonitorIDs(page.ID)
	linked := make(map[int64]bool, len(linkedIDs))
	for _, id := range linkedIDs {
		linked[id] = true
	}
	c.HTML(http.StatusOK, "status_page_form.html", h.pageData(c, gin.H{
		"Page":             page,
		"IsNew":            false,
		"AllMonitors":      monitors,
		"LinkedMonitorIDs": linked,
		"Error":            "",
	}))
}

// StatusPageUpdate handles the edit form submission.
func (h *Handler) StatusPageUpdate(c *gin.Context) {
	existing, ok := h.getStatusPage(c)
	if !ok {
		return
	}
	updated, monitorIDs, err := statusPageFromForm(c)
	if err != nil {
		monitors, _ := h.monitorStore(c).List()
		c.HTML(http.StatusBadRequest, "status_page_form.html", gin.H{
			"Page": existing, "IsNew": false, "AllMonitors": monitors,
			"LinkedMonitorIDs": map[int64]bool{}, "Error": err.Error(),
		})
		return
	}
	updated.ID = existing.ID

	spStore := h.statusPageStore(c)
	if err := spStore.Update(updated); err != nil {
		monitors, _ := h.monitorStore(c).List()
		c.HTML(http.StatusInternalServerError, "status_page_form.html", gin.H{
			"Page": updated, "IsNew": false, "AllMonitors": monitors,
			"LinkedMonitorIDs": map[int64]bool{}, "Error": err.Error(),
		})
		return
	}
	_ = spStore.SetMonitors(updated.ID, monitorIDs)
	c.Redirect(http.StatusFound, "/status-pages")
}

// StatusPageDelete removes a status page.
func (h *Handler) StatusPageDelete(c *gin.Context) {
	page, ok := h.getStatusPage(c)
	if !ok {
		return
	}
	if err := h.statusPageStore(c).Delete(page.ID); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"Error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, "/status-pages")
}

// StatusPagePublic renders the unauthenticated public status page.
// Route: GET /status/:username/:slug
func (h *Handler) StatusPagePublic(c *gin.Context) {
	username := c.Param("username")
	slug := c.Param("slug")

	db, err := h.registry.Get(username)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Error": "Status page not found"})
		return
	}

	spStore := models.NewStatusPageStore(db)
	page, err := spStore.GetBySlug(slug)
	if err != nil || page == nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Error": "Status page not found"})
		return
	}

	monitorIDs, _ := spStore.ListMonitorIDs(page.ID)
	mStore := models.NewMonitorStore(db)
	bStore := models.NewHeartbeatStore(db)

	type entry struct {
		Monitor      *models.Monitor
		Uptime24h    float64
		LatestStatus int
		Sparkline    template.HTML
	}
	now := time.Now().UTC()
	var monitors []entry
	allOperational := true
	for _, mid := range monitorIDs {
		m, err := mStore.Get(mid)
		if err != nil || m == nil {
			continue
		}
		latestStatus := -1 // -1 = pending/unknown
		beats, _ := bStore.Latest(m.ID, 50)
		if len(beats) > 0 {
			m.LastStatus = &beats[0].Status
			m.LastLatency = &beats[0].LatencyMs
			latestStatus = beats[0].Status
		}
		if latestStatus != 1 {
			allOperational = false
		}
		uptime24h, _ := bStore.UptimePercent(m.ID, now.Add(-24*time.Hour))
		monitors = append(monitors, entry{
			Monitor:      m,
			Uptime24h:    uptime24h,
			LatestStatus: latestStatus,
			Sparkline:    computeSparklineSVG(beats),
		})
	}

	c.HTML(http.StatusOK, "status_page_public.html", gin.H{
		"Page":           page,
		"Monitors":       monitors,
		"AllOperational": allOperational && len(monitors) > 0,
		"Now":            now.Format("2006-01-02 15:04:05 UTC"),
		"Username":       username,
		"Slug":           slug,
	})
}

// StatusPagePublicChartData is a public JSON endpoint that returns heartbeat
// history for a monitor, but only if it belongs to the given status page.
// Route: GET /status/:username/:slug/chart-data/:id
func (h *Handler) StatusPagePublicChartData(c *gin.Context) {
	username := c.Param("username")
	slug := c.Param("slug")
	idStr := c.Param("id")
	monitorID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid monitor id"})
		return
	}

	db, err := h.registry.Get(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Verify the monitor is actually linked to this status page.
	spStore := models.NewStatusPageStore(db)
	page, err := spStore.GetBySlug(slug)
	if err != nil || page == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	linkedIDs, _ := spStore.ListMonitorIDs(page.ID)
	found := false
	for _, id := range linkedIDs {
		if id == monitorID {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	span := c.DefaultQuery("since", "24h")
	dur, ok := allowedChartSpans[span]
	if !ok {
		dur = 24 * time.Hour
		span = "24h"
	}

	// Public callers get a cached response — protects the DB from flooding.
	// Authenticated owners (MonitorChartData) bypass this endpoint entirely.
	cacheKey := chartCacheKey(username, idStr, span)
	if cached, hit := h.chartCache.get(cacheKey); hit {
		c.Data(http.StatusOK, "application/json; charset=utf-8", cached)
		return
	}

	beats, err := models.NewHeartbeatStore(db).LatestSince(monitorID, time.Now().Add(-dur), 500)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type point struct {
		TS      string `json:"ts"`
		Latency int    `json:"latency"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	// beats is newest-first; reverse to oldest-first for chart rendering
	pts := make([]point, len(beats))
	for i, b := range beats {
		pts[len(beats)-1-i] = point{
			TS:      b.CreatedAt.UTC().Format(time.RFC3339),
			Latency: b.LatencyMs,
			Status:  b.Status,
			Message: b.Message,
		}
	}

	type downtimeBand struct {
		Start string  `json:"start"`
		End   *string `json:"end"`
	}
	dtEvents, _ := models.NewDowntimeEventStore(db).ListSince(monitorID, time.Now().Add(-dur))
	bands := make([]downtimeBand, len(dtEvents))
	for i, e := range dtEvents {
		var end *string
		if e.EndedAt != nil {
			s := e.EndedAt.UTC().Format(time.RFC3339)
			end = &s
		}
		bands[i] = downtimeBand{Start: e.StartedAt.UTC().Format(time.RFC3339), End: end}
	}

	payload, err := json.Marshal(gin.H{"points": pts, "downtime": bands})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.chartCache.set(cacheKey, payload, chartCacheTTL)
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

func (h *Handler) getStatusPage(c *gin.Context) (*models.StatusPage, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"Error": "invalid status page id"})
		return nil, false
	}
	page, err := h.statusPageStore(c).Get(id)
	if err != nil || page == nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"Error": "status page not found"})
		return nil, false
	}
	return page, true
}

// statusPageFromForm parses a status page and the linked monitor IDs from a POST form.
func statusPageFromForm(c *gin.Context) (*models.StatusPage, []int64, error) {
	name := c.PostForm("name")
	slug := c.PostForm("slug")
	desc := c.PostForm("description")

	page := &models.StatusPage{Name: name, Slug: slug, Description: desc}
	if name == "" {
		return page, nil, &formError{"name is required"}
	}
	if slug == "" {
		return page, nil, &formError{"slug is required"}
	}

	var monitorIDs []int64
	for _, v := range c.PostFormArray("monitors") {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			monitorIDs = append(monitorIDs, id)
		}
	}
	return page, monitorIDs, nil
}
