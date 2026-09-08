// Package server: stateful editing sessions for the HTTP API.
//
// Stateless /execute forces callers to ship the full presentation JSON on
// every request, and makes undo/redo impossible over HTTP (the session dies
// with the request). These endpoints keep the session server-side:
//
//	POST /api/v1/session            create session {presentation?} -> {session_id}
//	GET  /api/v1/session/:id        {presentation, version}
//	POST /api/v1/session/:id/execute   {calls:[{name,arguments}]} (partial results on failure)
//	POST /api/v1/session/:id/render    -> slide images (same shape as /api/v1/render)
//	POST /api/v1/session/:id/build     -> .pptx download
//	POST /api/v1/session/:id/undo|redo undo/redo the last mutating tool call
//	DELETE /api/v1/session/:id         drop the session
//
// Sessions expire after 30 minutes of inactivity.
package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otter-ppt/otter-ppt/internal/builder"
	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
)

const sessionTTL = 30 * time.Minute

type editingSession struct {
	mu         sync.Mutex
	session    *pptoolkit.Session
	lastActive time.Time
}

var (
	sessionStore sync.Map // session_id -> *editingSession
)

// reapExpiredSessions removes sessions idle for longer than the TTL.
// Called opportunistically on session creation — good enough for a
// single-process server.
func reapExpiredSessions() {
	now := time.Now()
	sessionStore.Range(func(key, value any) bool {
		if e, ok := value.(*editingSession); ok {
			e.mu.Lock()
			idle := now.Sub(e.lastActive)
			e.mu.Unlock()
			if idle > sessionTTL {
				sessionStore.Delete(key)
			}
		}
		return true
	})
}

// loadSession fetches a session by ID and refreshes its TTL. Returns nil
// (and writes a 404) when not found or expired.
func loadSession(c *gin.Context) *editingSession {
	id := c.Param("id")
	v, ok := sessionStore.Load(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "unknown or expired session_id — create one with POST /api/v1/session",
		})
		return nil
	}
	e := v.(*editingSession)
	e.mu.Lock()
	e.lastActive = time.Now()
	e.mu.Unlock()
	return e
}

// ──────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────

// POST /api/v1/session — create an editing session.
// Body (optional): {"presentation": {...}} to seed state; otherwise empty.
func (s *Server) handleSessionCreate(c *gin.Context) {
	reapExpiredSessions()

	var body struct {
		Presentation *model.Presentation `json:"presentation"`
	}
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid JSON body: " + err.Error(),
				"example": gin.H{
					"presentation": gin.H{"title": "My Deck", "slides": []any{}},
				},
			})
			return
		}
	}

	e := &editingSession{
		session:    pptoolkit.NewSessionFromPresentation(body.Presentation),
		lastActive: time.Now(),
	}
	id := uuid.NewString()[:12]
	sessionStore.Store(id, e)

	c.JSON(http.StatusOK, gin.H{
		"session_id":  id,
		"expires_in":  sessionTTL.String(),
		"slide_count": len(e.session.Presentation().Slides),
		"usage": gin.H{
			"execute": "POST /api/v1/session/" + id + "/execute  {\"calls\":[{\"name\":\"add_slide\",\"arguments\":{\"layout\":\"blank\"}}]}",
			"undo":    "POST /api/v1/session/" + id + "/undo",
			"render":  "POST /api/v1/session/" + id + "/render",
			"build":   "POST /api/v1/session/" + id + "/build   (downloads .pptx)",
			"state":   "GET  /api/v1/session/" + id,
		},
	})
}

// GET /api/v1/session/:id — fetch current state.
func (s *Server) handleSessionGet(c *gin.Context) {
	e := loadSession(c)
	if e == nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"session_id":    c.Param("id"),
		"presentation":  e.session.Presentation(),
		"slide_count":   len(e.session.Presentation().Slides),
		"last_active":   e.lastActive.UTC().Format(time.RFC3339),
	})
}

// POST /api/v1/session/:id/execute — run tool calls against the session.
// Same contract as stateless /api/v1/execute, minus the presentation round-trip.
func (s *Server) handleSessionExecute(c *gin.Context) {
	e := loadSession(c)
	if e == nil {
		return
	}

	var req executeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"example": gin.H{
				"calls": []gin.H{
					{"name": "add_slide", "arguments": gin.H{"layout": "blank"}},
					{"name": "add_text", "arguments": gin.H{
						"slide_id": "REPLACE_WITH_SLIDE_ID",
						"x": 10, "y": 10, "w": 80, "h": 10,
						"text": "Hello world", "font_size": 24,
					}},
				},
			},
		})
		return
	}
	if len(req.Calls) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "calls must be a non-empty array of {name, arguments}",
		})
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	results := make([]pptoolkit.ToolResult, 0, len(req.Calls))
	for index, call := range req.Calls {
		if call.Arguments == nil {
			call.Arguments = map[string]any{}
		}
		result := e.session.ExecuteTool(call.Name, call.Arguments)
		results = append(results, result)
		if !result.Success {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":             result.Message,
				"failed_call_index": index,
				"results":           results,
				// Session survives — no need to resend state, just fix the call.
				"hint":              "server-side state is preserved; retry the failed call via POST /api/v1/session/" + c.Param("id") + "/execute",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":   c.Param("id"),
		"results":      results,
		"presentation": e.session.Presentation(),
	})
}

// POST /api/v1/session/:id/render — render current session state.
// Response shape matches stateless /api/v1/render.
func (s *Server) handleSessionRender(c *gin.Context) {
	e := loadSession(c)
	if e == nil {
		return
	}
	s.renderPresentation(c, e.session.Presentation())
}

// POST /api/v1/session/:id/build — build PPTX from current state.
func (s *Server) handleSessionBuild(c *gin.Context) {
	e := loadSession(c)
	if e == nil {
		return
	}
	pres := e.session.Presentation()

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	c.Header("Content-Disposition", "attachment; filename="+fileSafeTitle(pres)+".pptx")
	if err := builder.New(pres).Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// POST /api/v1/session/:id/undo — undo the last mutating tool call.
func (s *Server) handleSessionUndo(c *gin.Context) {
	s.sessionHistoryCall(c, "undo")
}

// POST /api/v1/session/:id/redo — redo.
func (s *Server) handleSessionRedo(c *gin.Context) {
	s.sessionHistoryCall(c, "redo")
}

func (s *Server) sessionHistoryCall(c *gin.Context, tool string) {
	e := loadSession(c)
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	result := e.session.ExecuteTool(tool, map[string]any{})
	status := http.StatusOK
	if !result.Success {
		status = http.StatusUnprocessableEntity
	}
	c.JSON(status, gin.H{
		"session_id": c.Param("id"),
		"tool":       tool,
		"result":     result,
	})
}

// DELETE /api/v1/session/:id — drop the session.
func (s *Server) handleSessionDelete(c *gin.Context) {
	id := c.Param("id")
	if _, ok := sessionStore.Load(id); !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown session_id"})
		return
	}
	sessionStore.Delete(id)
	c.JSON(http.StatusOK, gin.H{"deleted": id})
}

// fileSafeTitle derives a download filename from the deck title.
func fileSafeTitle(pres *model.Presentation) string {
	title := pres.Title
	if title == "" {
		title = "presentation"
	}
	safe := make([]rune, 0, len(title))
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == ' ', r >= 0x4e00 && r <= 0x9fff: // keep CJK
			safe = append(safe, r)
		}
	}
	if len(safe) == 0 {
		return "presentation"
	}
	// trim spaces
	out := string(safe)
	if len(out) > 60 {
		out = out[:60]
	}
	return out
}
