package server

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	if !strings.Contains(w.Body.String(), "rolled_back") &&
		!strings.Contains(w.Body.String(), "Fix the failed call") {
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

// TestSessionBatchAtomicity verifies that a failed batch leaves zero
// partial effects on server-side state.
func TestSessionBatchAtomicity(t *testing.T) {
	s := New(Config{Port: "0"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/session", nil)
	s.router.ServeHTTP(w, req)
	var created struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created.SessionID

	// GET baseline state
	state := func() int {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/session/"+id, nil)
		s.router.ServeHTTP(w, req)
		var st struct {
			SlideCount int `json:"slide_count"`
		}
		json.Unmarshal(w.Body.Bytes(), &st)
		return st.SlideCount
	}
	before := state()

	// Batch: ok call + failing call → whole batch must roll back.
	body := `{"calls":[` +
		`{"name":"add_slide","arguments":{"layout":"blank"}},` +
		`{"name":"add_text","arguments":{"slide_id":"nope","text":"x"}}` +
		`]}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/v1/session/"+id+"/execute", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"rolled_back":true`) {
		t.Fatalf("response missing rolled_back flag: %s", w.Body.String())
	}
	if got := state(); got != before {
		t.Fatalf("atomicity violated: slide_count after failed batch = %d, want %d", got, before)
	}
}

// TestSessionIdempotency verifies that a retried request with the same
// idempotency_key is served from cache without re-executing.
func TestSessionIdempotency(t *testing.T) {
	s := New(Config{Port: "0"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/session", nil)
	s.router.ServeHTTP(w, req)
	var created struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created.SessionID

	exec := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/session/"+id+"/execute",
			strings.NewReader(`{"idempotency_key":"k1","calls":[{"name":"add_slide","arguments":{"layout":"blank"}}]}`))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		return w
	}

	if w := exec(); w.Code != http.StatusOK {
		t.Fatalf("first execute: %d %s", w.Code, w.Body.String())
	}
	w = exec() // retry with same key
	if w.Code != http.StatusOK {
		t.Fatalf("replayed execute: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"idempotent_replay":true`) {
		t.Fatalf("replay response missing idempotent_replay flag: %s", w.Body.String())
	}

	// State must contain exactly one slide: the retry did not re-execute.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/session/"+id, nil)
	s.router.ServeHTTP(w, req)
	var st struct {
		SlideCount int `json:"slide_count"`
	}
	json.Unmarshal(w.Body.Bytes(), &st)
	if st.SlideCount != 1 {
		t.Fatalf("slide_count = %d, want 1 (retry must not re-execute)", st.SlideCount)
	}
}

// Failed batches (422) are also cached: retrying the same bad batch with the
// same key returns the stored response instead of re-executing the calls.
func TestSessionIdempotencyCachesFailure(t *testing.T) {
	s := New(Config{Port: "0"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/session", nil)
	s.router.ServeHTTP(w, req)
	var created struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created.SessionID

	exec := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/session/"+id+"/execute",
			strings.NewReader(`{"idempotency_key":"bad1","calls":[{"name":"add_text","arguments":{"slide_id":"no-such-slide","text":"x"}}]}`))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		return w
	}

	if w := exec(); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("first execute: %d %s", w.Code, w.Body.String())
	}
	w = exec()
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("replayed execute: %d, want 422 from cache", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"idempotent_replay":true`) {
		t.Fatalf("replay response missing idempotent_replay flag: %s", w.Body.String())
	}
}

// FIFO eviction: with more distinct keys than the cache limit, the oldest
// key is evicted (not a random one).
func TestSessionIdempotencyFIFOEviction(t *testing.T) {
	s := New(Config{Port: "0"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/session", nil)
	s.router.ServeHTTP(w, req)
	var created struct {
		SessionID string `json:"session_id"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)
	id := created.SessionID

	exec := func(key string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		body := fmt.Sprintf(`{"idempotency_key":%q,"calls":[{"name":"undo","arguments":{}}]}`, key)
		req, _ := http.NewRequest("POST", "/api/v1/session/"+id+"/execute", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		s.router.ServeHTTP(w, req)
		return w
	}

	exec("k000") // oldest — must be evicted after limit is exceeded
	for i := 1; i < idempotencyCacheLimit; i++ {
		exec(fmt.Sprintf("k%03d", i))
	}
	exec("new") // exceeds limit -> evicts k000
	w = exec("k000")
	if strings.Contains(w.Body.String(), `"idempotent_replay":true`) {
		t.Fatalf("k000 should have been FIFO-evicted, got replay: %s", w.Body.String())
	}

	// Sanity: the newest key still replays.
	w = exec("new")
	if !strings.Contains(w.Body.String(), `"idempotent_replay":true`) {
		t.Fatalf("newest key should replay from cache: %s", w.Body.String())
	}
}
