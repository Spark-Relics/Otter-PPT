package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSessionLifecycle covers the full HTTP editing loop:
// create → execute calls → undo → redo → delete, plus 404 behavior.
func TestSessionLifecycle(t *testing.T) {
	s := New(Config{Port: "0"})

	// 1. Create session
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/session", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.SessionID == "" {
		t.Fatal("no session_id in response")
	}
	id := created.SessionID

	// 2. Execute tool calls against the session
	exec := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/session/"+id+"/execute", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		return w
	}

	w = exec(`{"calls":[{"name":"add_slide","arguments":{"layout":"blank"}}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("execute add_slide: %d %s", w.Code, w.Body.String())
	}
	var execResult struct {
		Results []struct {
			Success bool           `json:"success"`
			Data    map[string]any `json:"data"`
		} `json:"results"`
	}
	json.Unmarshal(w.Body.Bytes(), &execResult)
	if len(execResult.Results) != 1 || !execResult.Results[0].Success {
		t.Fatalf("unexpected results: %s", w.Body.String())
	}
	slideID, _ := execResult.Results[0].Data["slide_id"].(string)
	if slideID == "" {
		t.Fatalf("structured data missing slide_id: %s", w.Body.String())
	}

	// add text so undo has something to undo
	textCall := map[string]any{
		"calls": []map[string]any{{
			"name": "add_text",
			"arguments": map[string]any{
				"slide_id": slideID, "x": 10, "y": 10, "w": 80, "h": 10,
				"text": "hello session", "font_size": 24,
			},
		}},
	}
	body, _ := json.Marshal(textCall)
	if w := exec(string(body)); w.Code != http.StatusOK {
		t.Fatalf("execute add_text: %d %s", w.Code, w.Body.String())
	}

	// 3. Undo — text should be gone, slide remains
	if w := exec(`{"calls":[{"name":"undo","arguments":{}}]}`); w.Code != http.StatusOK {
		t.Fatalf("undo via execute: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/session/"+id, nil)
	s.router.ServeHTTP(w, req)
	var state struct {
		Presentation struct {
			Slides []struct {
				Elements []struct {
					Text string `json:"text"`
				} `json:"elements"`
			} `json:"slides"`
		} `json:"presentation"`
	}
	json.Unmarshal(w.Body.Bytes(), &state)
	if len(state.Presentation.Slides) != 1 {
		t.Fatalf("undo should keep 1 slide, got %d", len(state.Presentation.Slides))
	}
	for _, e := range state.Presentation.Slides[0].Elements {
		if strings.Contains(e.Text, "hello session") {
			t.Fatal("undo did not remove the added text")
		}
	}

	// 4. Direct undo endpoint also works
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/session/"+id+"/undo", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("undo endpoint: %d %s", w.Code, w.Body.String())
	}

	// 5. Redo endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/session/"+id+"/redo", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("redo endpoint: %d %s", w.Code, w.Body.String())
	}

	// 6. Failed call returns 422 with partial results and a helpful hint
	w = exec(`{"calls":[{"name":"add_text","arguments":{"slide_id":"nonexistent","text":"x"}}]}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "session state is preserved") &&
		!strings.Contains(w.Body.String(), "retry the failed call") {
		t.Fatalf("missing retry hint: %s", w.Body.String())
	}

	// 7. Build returns a pptx
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/session/"+id+"/build", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("build: %d %s", w.Code, w.Body.String())
	}
	if !bytes.HasPrefix(w.Body.Bytes(), []byte("PK")) {
		t.Fatal("build response is not a zip (pptx)")
	}

	// 8. Delete + subsequent access 404s
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/v1/session/"+id, nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete: %d", w.Code)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/session/"+id, nil)
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", w.Code)
	}
}

// TestSessionBadPayload verifies the self-healing 400 with an example payload.
func TestSessionBadPayload(t *testing.T) {
	s := New(Config{Port: "0"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/session/xxx/execute", strings.NewReader(`{"nope":1}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound { // unknown session should win over bad body
		t.Fatalf("expected 404 for unknown session, got %d: %s", w.Code, w.Body.String())
	}

	// Create a session, then send a bad execute body
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/session", nil)
	s.router.ServeHTTP(w, req)
	var created struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/session/"+created.SessionID+"/execute", strings.NewReader(`{"calls":[]}`))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty calls, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "example") {
		t.Fatalf("400 response missing example payload: %s", w.Body.String())
	}
}
