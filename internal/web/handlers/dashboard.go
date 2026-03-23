package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xdung24/conductor/internal/config"
	"github.com/xdung24/conductor/internal/database"
	"github.com/xdung24/conductor/internal/mailer"
	"github.com/xdung24/conductor/internal/models"
	"github.com/xdung24/conductor/internal/scheduler"
)

// sparklineTmpl is parsed once at package init; html/template auto-escapes all dynamic values.
var sparklineTmpl = template.Must(template.New("sparkline").Parse(
	`<svg width="{{.W}}" height="{{.H}}" viewBox="0 0 {{.W}} {{.H}}" xmlns="http://www.w3.org/2000/svg" style="display:block;">` +
		`<polyline points="{{.Pts}}" fill="none" stroke="#38bdf8" stroke-width="1.5" stroke-linejoin="round" stroke-linecap="round"/></svg>`,
))

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	usersDB    *sql.DB                   // shared users database (auth + push_tokens)
	registry   *database.Registry        // per-user data DB registry
	msched     *scheduler.MultiScheduler // per-user schedulers
	cfg        *config.Config
	users      *models.UserStore  // backed by usersDB
	mailer     *mailer.Mailer     // system transactional email (nil = disabled)
	docsHTML   template.HTML      // pre-rendered docs markdown (rendered once at startup)
	chartCache *chartCache        // TTL cache for public chart JSON responses
	pageCache  *chartCache        // TTL cache for rendered public status page HTML
	tmpl       *template.Template // reference for off-response rendering (cache fill)
}

// New creates a Handler.
func New(usersDB *sql.DB, registry *database.Registry, msched *scheduler.MultiScheduler, cfg *config.Config, tmpl *template.Template, m *mailer.Mailer) *Handler {
	return &Handler{
		usersDB:    usersDB,
		registry:   registry,
		msched:     msched,
		docsHTML:   renderDocsMarkdown(),
		cfg:        cfg,
		users:      models.NewUserStore(usersDB),
		mailer:     m,
		chartCache: newChartCache(),
		pageCache:  newChartCache(),
		tmpl:       tmpl,
	}
}

// ---------------------------------------------------------------------------
// Per-request context helpers
// ---------------------------------------------------------------------------

func (h *Handler) userDB(c *gin.Context) *sql.DB {
	db, _ := c.Get(ctxKeyUserDB)
	return db.(*sql.DB)
}

func (h *Handler) username(c *gin.Context) string {
	return c.GetString(ctxKeyUsername)
}

func (h *Handler) monitorStore(c *gin.Context) *models.MonitorStore {
	return models.NewMonitorStore(h.userDB(c))
}

func (h *Handler) heartbeatStore(c *gin.Context) *models.HeartbeatStore {
	return models.NewHeartbeatStore(h.userDB(c))
}

func (h *Handler) notifStore(c *gin.Context) *models.NotificationStore {
	return models.NewNotificationStore(h.userDB(c))
}

func (h *Handler) notifLogStore(c *gin.Context) *models.NotificationLogStore {
	return models.NewNotificationLogStore(h.userDB(c))
}

func (h *Handler) schedFor(c *gin.Context) *scheduler.Scheduler {
	return h.msched.ForUser(h.username(c))
}

// pageData returns a gin.H with common authenticated-page fields (IsAdmin, Username)
// merged with any page-specific fields supplied by the caller.
func (h *Handler) pageData(c *gin.Context, extra gin.H) gin.H {
	data := gin.H{
		"IsAdmin":  h.isAdmin(c),
		"Username": c.GetString(ctxKeyUsername),
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

// Dashboard renders the main monitor list page.
func (h *Handler) Dashboard(c *gin.Context) {
	mstore := h.monitorStore(c)
	bstore := h.heartbeatStore(c)
	tstore := h.tagStore(c)

	monitors, err := mstore.List()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.gohtml", gin.H{"Error": err.Error()})
		return
	}

	now := time.Now()
	monitorIDs := make([]int64, 0, len(monitors))
	for _, m := range monitors {
		monitorIDs = append(monitorIDs, m.ID)
	}

	sparklines := make(map[int64]template.HTML, len(monitors))
	tagMap, _ := tstore.TagMapForMonitors(monitorIDs)

	for _, m := range monitors {
		beats, _ := bstore.Latest(m.ID, 50)
		if len(beats) > 0 {
			m.LastStatus = &beats[0].Status
			m.LastLatency = &beats[0].LatencyMs
			m.LastMessage = &beats[0].Message
		}
		m.Uptime24h, _ = bstore.UptimePercent(m.ID, now.Add(-24*time.Hour))
		m.Uptime30d, _ = bstore.UptimePercent(m.ID, now.Add(-30*24*time.Hour))
		sparklines[m.ID] = computeSparklineSVG(beats)
	}

	c.HTML(http.StatusOK, "dashboard.gohtml", h.pageData(c, gin.H{
		"Monitors":   monitors,
		"Sparklines": sparklines,
		"Tags":       tagMap,
		"Username":   h.username(c),
	}))
}

// computeSparklineSVG generates an inline SVG polyline from heartbeat data.
// beats should be in newest-first order (as returned by bstore.Latest).
func computeSparklineSVG(beats []*models.Heartbeat) template.HTML {
	if len(beats) == 0 {
		return ""
	}

	// Reverse to oldest-first for left-to-right rendering.
	n := len(beats)
	points := make([]int, n)
	for i, b := range beats {
		points[n-1-i] = b.LatencyMs
	}

	// Find max latency for scaling (minimum 1 to avoid divide-by-zero).
	maxVal := 1
	for _, v := range points {
		if v > maxVal {
			maxVal = v
		}
	}

	const svgW, svgH = 80, 24
	var pts string
	for i, v := range points {
		x := float64(i) * float64(svgW) / float64(len(points)-1)
		y := float64(svgH) - (float64(v)/float64(maxVal))*float64(svgH-2) - 1
		if i == 0 {
			pts += fmt.Sprintf("%.1f,%.1f", x, y)
		} else {
			pts += fmt.Sprintf(" %.1f,%.1f", x, y)
		}
	}

	var buf bytes.Buffer
	if err := sparklineTmpl.Execute(&buf, struct {
		W, H int
		Pts  string
	}{svgW, svgH, pts}); err != nil {
		return ""
	}
	return template.HTML(buf.String()) //nolint:gosec // content produced by html/template; all dynamic values are numeric
}
