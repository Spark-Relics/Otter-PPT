// Package server: live preview endpoints.
//
// A preview session holds a pptoolkit.Session behind a random token:
//
//	POST /api/v1/preview            create session (optional body: presentation JSON or {calls})
//	POST /api/v1/preview/:token     update session (body: presentation JSON or {calls:[{name,arguments}]})
//	GET  /preview/:token            viewer page (server-rendered HTML + auto-refresh)
//	GET  /api/v1/preview/:token/version  poll endpoint {version, slide_count}
//
// The typical loop: an AI agent (or SDK) creates a session, applies tool
// calls after each design iteration, and a human keeps the viewer open in
// a browser — the page polls the version and reloads itself when the
// presentation changes.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otter-ppt/otter-ppt/internal/model"
	"github.com/otter-ppt/otter-ppt/internal/pptoolkit"
	"github.com/otter-ppt/otter-ppt/internal/renderer"
)

// previewEntry is one live preview session.
type previewEntry struct {
	mu      sync.Mutex
	session *pptoolkit.Session
	version uint64
}

var previewStore sync.Map // token -> *previewEntry

// applyPreviewPayload updates a session from a request body.
// Accepted forms:
//   - {"presentation": {...}}            — replace presentation state
//   - {"calls": [{name, arguments}]}     — execute tool calls in order
//   - a bare Presentation object          — replace state
func applyPreviewPayload(e *previewEntry, body []byte) (applied string, err error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return "no-op", nil
	}

	// Form 3: bare Presentation (has "slides" but neither "presentation" nor "calls").
	var wrapper struct {
		Presentation *model.Presentation `json:"presentation"`
		Calls        []externalToolCall  `json:"calls"`
		Slides       json.RawMessage     `json:"slides"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return "", fmt.Errorf("invalid JSON body: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	switch {
	case len(wrapper.Calls) > 0:
		for i, call := range wrapper.Calls {
			if call.Arguments == nil {
				call.Arguments = map[string]any{}
			}
			if result := e.session.ExecuteTool(call.Name, call.Arguments); !result.Success {
				return "", fmt.Errorf("call %d (%s) failed: %s", i, call.Name, result.Message)
			}
		}
		e.version++
		return fmt.Sprintf("applied %d tool calls", len(wrapper.Calls)), nil

	case wrapper.Presentation != nil:
		e.session = pptoolkit.NewSessionFromPresentation(wrapper.Presentation)
		e.version++
		return "presentation replaced", nil

	case len(wrapper.Slides) > 0:
		var pres model.Presentation
		if err := json.Unmarshal(body, &pres); err != nil {
			return "", fmt.Errorf("invalid presentation JSON: %w", err)
		}
		e.session = pptoolkit.NewSessionFromPresentation(&pres)
		e.version++
		return "presentation replaced", nil

	default:
		return "", fmt.Errorf(`body must contain "calls" or a presentation`)
	}
}

// handlePreviewCreate registers a new preview session.
func (s *Server) handlePreviewCreate(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	e := &previewEntry{session: pptoolkit.NewSession()}
	token := uuid.NewString()[:12]
	if _, err := applyPreviewPayload(e, body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	previewStore.Store(token, e)

	c.JSON(http.StatusOK, gin.H{
		"token":       token,
		"viewer_url":  "/preview/" + token,
		"update_url":  "/api/v1/preview/" + token,
		"version":     e.version,
		"slide_count": len(e.session.Presentation().Slides),
	})
}

// handlePreviewUpdate pushes new state or tool calls into a session.
func (s *Server) handlePreviewUpdate(c *gin.Context) {
	token := c.Param("token")
	v, ok := previewStore.Load(token)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown preview token"})
		return
	}
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e := v.(*previewEntry)
	applied, err := applyPreviewPayload(e, body)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error(), "token": token})
		return
	}
	pres := e.session.Presentation()
	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"applied":      applied,
		"version":      e.version,
		"presentation": pres,
	})
}

// handlePreviewVersion is the lightweight poll endpoint for the viewer.
func (s *Server) handlePreviewVersion(c *gin.Context) {
	token := c.Param("token")
	v, ok := previewStore.Load(token)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown preview token"})
		return
	}
	e := v.(*previewEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{
		"version":     e.version,
		"slide_count": len(e.session.Presentation().Slides),
	})
}

// handlePreviewViewer serves the self-contained viewer page: the slides
// HTML produced by the Go renderer, plus injected viewer chrome
// (keyboard navigation, zoom, slide counter, auto-refresh on change).
func (s *Server) handlePreviewViewer(c *gin.Context) {
	token := c.Param("token")
	v, ok := previewStore.Load(token)
	if !ok {
		c.String(http.StatusNotFound, "unknown preview token: %s", token)
		return
	}
	e := v.(*previewEntry)
	e.mu.Lock()
	pres := e.session.Presentation()
	version := e.version
	e.mu.Unlock()

	if len(pres.Slides) == 0 {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(emptyViewerHTML))
		return
	}

	// Render slides to a self-contained HTML file, then read it back.
	tmp, err := os.CreateTemp("", "otter-preview-*.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "render failed: %v", err)
		return
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)
	if err := renderer.GenerateHTML(pres, tmpPath); err != nil {
		c.String(http.StatusInternalServerError, "render failed: %v", err)
		return
	}
	raw, err := os.ReadFile(tmpPath)
	if err != nil {
		c.String(http.StatusInternalServerError, "read render failed: %v", err)
		return
	}

	page := injectViewerChrome(string(raw), token, version)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

// injectViewerChrome splices viewer CSS/JS into a rendered slide document.
func injectViewerChrome(page, token string, version uint64) string {
	extra := fmt.Sprintf(`
<style>
#otter-viewer-bar {
  position: fixed; top: 12px; right: 16px; z-index: 9999;
  background: rgba(20,20,28,.82); color: #fff; border-radius: 10px;
  padding: 8px 12px; font: 13px/1.6 system-ui, sans-serif;
  display: flex; gap: 10px; align-items: center;
  box-shadow: 0 4px 16px rgba(0,0,0,.35);
  user-select: none;
}
#otter-viewer-bar button {
  background: #fff2; color: #fff; border: 1px solid #fff4; border-radius: 6px;
  padding: 2px 9px; cursor: pointer; font: inherit;
}
#otter-viewer-bar button:hover { background: #fff3; }
body { scroll-behavior: smooth; }
.slide { outline: 1px solid #ffffff14; }
html, body { background: #1b1b22 !important; }
body.zoom-100 .slide { zoom: 1; }
body.zoom-fit .slide { zoom: %.4f; }
</style>
<div id="otter-viewer-bar">
  <span id="otter-pos">1 / 1</span>
  <button id="otter-prev" title="Previous (←)">←</button>
  <button id="otter-next" title="Next (→)">→</button>
  <button id="otter-zoom" title="Toggle zoom (fit / 100%%)">fit</button>
  <span id="otter-live" title="auto-refresh every 2s">● live</span>
</div>
<script>
(function () {
  var token = %q, version = %d;
  var slides = Array.prototype.slice.call(document.querySelectorAll('.slide'));
  var pos = document.getElementById('otter-pos');
  var live = document.getElementById('otter-live');

  function updatePos() {
    var mid = window.scrollY + window.innerHeight / 2;
    var idx = 0;
    for (var i = 0; i < slides.length; i++) {
      var el = slides[i], top = el.offsetTop;
      if (top <= mid) idx = i;
    }
    pos.textContent = (idx + 1) + ' / ' + slides.length;
  }
  window.addEventListener('scroll', updatePos);
  updatePos();

  function go(delta) {
    var mid = window.scrollY + window.innerHeight / 2, idx = 0;
    for (var i = 0; i < slides.length; i++) if (slides[i].offsetTop <= mid) idx = i;
    var next = Math.min(Math.max(idx + delta, 0), slides.length - 1);
    window.scrollTo(0, slides[next].offsetTop);
  }
  document.getElementById('otter-prev').onclick = function () { go(-1); };
  document.getElementById('otter-next').onclick = function () { go(1); };

  var fit = true;
  var zoomBtn = document.getElementById('otter-zoom');
  function applyZoom() {
    document.body.className = fit ? 'zoom-fit' : 'zoom-100';
    zoomBtn.textContent = fit ? 'fit' : '100%%';
  }
  zoomBtn.onclick = function () { fit = !fit; applyZoom(); };
  applyZoom();

  document.addEventListener('keydown', function (ev) {
    if (ev.key === 'ArrowLeft' || ev.key === 'PageUp') go(-1);
    if (ev.key === 'ArrowRight' || ev.key === 'PageDown' || ev.key === ' ') go(1);
  });

  setInterval(function () {
    fetch('/api/v1/preview/' + token + '/version')
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (data) {
        if (!data) { live.textContent = '○ offline'; return; }
        if (data.version !== version) { location.reload(); }
        else { live.textContent = '● live'; }
      })
      .catch(function () { live.textContent = '○ offline'; });
  }, 2000);
})();
</script>
`, fitZoomFactor(), token, version)

	// Splice before the closing tags of the rendered document.
	if idx := strings.LastIndex(page, "</body>"); idx >= 0 {
		return page[:idx] + extra + page[idx:]
	}
	return page + extra
}

// fitZoomFactor computes the CSS zoom that fits one slide width into a
// typical viewport (assumes ~1600px design viewport for 13.33in slides).
func fitZoomFactor() float64 {
	w, _ := model.DefaultSlideSize()
	return 1600.0 / (w * 96.0)
}

const emptyViewerHTML = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Otter PPT Preview</title>
<style>body{background:#1b1b22;color:#9aa;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;font:16px system-ui,sans-serif}</style>
</head><body><div>⬦ No slides yet — push tool calls to <code>POST /api/v1/preview/&lt;token&gt;</code> and this page will refresh automatically.</div></body></html>`
