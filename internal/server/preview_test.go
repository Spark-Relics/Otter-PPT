package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	s := New(Config{Port: "0"})
	return s.router
}

func TestPreviewLifecycle(t *testing.T) {
	router := newTestRouter()

	// 1. Create session from a bare presentation JSON.
	createPres := map[string]any{
		"slides": []map[string]any{{
			"id":     "s1",
			"layout": "title",
			"elements": []map[string]any{{
				"id":   "e1",
				"type": "title",
				"rect": map[string]any{"x": 10, "y": 35, "w": 80, "h": 15},
				"text": "Hello Preview",
			}},
		}},
	}
	raw, _ := json.Marshal(createPres)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preview", bytes.NewReader(raw))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create: status %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		Token      string `json:"token"`
		SlideCount int    `json:"slide_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Token == "" || created.SlideCount != 1 {
		t.Fatalf("unexpected create result: %+v", created)
	}

	// 2. Viewer page renders the slide.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/preview/"+created.Token, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("viewer: status %d", w.Code)
	}
	page := w.Body.String()
	if !strings.Contains(page, "Hello Preview") {
		t.Error("viewer page does not contain slide title")
	}
	if !strings.Contains(page, "otter-viewer-bar") {
		t.Error("viewer chrome missing")
	}

	// 3. Version poll works.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/preview/"+created.Token+"/version", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("version: status %d", w.Code)
	}

	// 4. Push an update and see the version bump.
	updateBody := map[string]any{
		"calls": []map[string]any{
			{"name": "add_slide", "arguments": map[string]any{"layout": "title_content"}},
		},
	}
	raw, _ = json.Marshal(updateBody)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/preview/"+created.Token, bytes.NewReader(raw))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update: status %d: %s", w.Code, w.Body.String())
	}
	var updated struct {
		SlideCount int    `json:"slide_count"`
		Presentation struct {
			Slides []struct {
				ID string `json:"id"`
			} `json:"slides"`
		} `json:"presentation"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode update: %v", err)
	}
	if len(updated.Presentation.Slides) != 2 {
		t.Errorf("slide count after update = %d, want 2", len(updated.Presentation.Slides))
	}

	// 5. Viewer reflects the new slide.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/preview/"+created.Token, nil)
	router.ServeHTTP(w, req)
	if got := strings.Count(w.Body.String(), `class="slide"`); got < 2 {
		t.Errorf("viewer slide count = %d, want >= 2", got)
	}

	// 6. Unknown token → 404.
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/preview/nope", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown token status = %d, want 404", w.Code)
	}
}

func TestPreviewEmptySession(t *testing.T) {
	router := newTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preview", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create empty: status %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/preview/"+created.Token, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "No slides yet") {
		t.Errorf("empty viewer: status %d body %q", w.Code, w.Body.String())
	}
}

func TestPreviewBadPayload(t *testing.T) {
	router := newTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preview", strings.NewReader(`{"foo":1}`))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad payload status = %d, want 400", w.Code)
	}
}
