// Package ai provides AI/LLM client utilities.
//
// transport.go implements a compatibility HTTP transport for non-OpenAI
// providers (e.g. Google Gemini) that offer OpenAI-compatible endpoints
// but have subtle differences that break the go-openai library.
package ai

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CompatTransport wraps an http.RoundTripper to fix Gemini quirks:
//
//  1. Error responses are JSON arrays:  [{"error":{...}}]  instead of {"error":{...}}
//  2. Tool calls include extra_content.google.thought_signature that
//     go-openai can't handle. We strip them on both request and response.
//  3. Request pacing: enforce MinInterval between requests to stay under
//     free-tier rate limits (e.g. Gemini allows 5 RPM = 12s interval).
//  4. Smart 429 retry: parse "Please retry in Xs" from the error body.
type CompatTransport struct {
	base http.RoundTripper

	// MinInterval is the minimum duration between HTTP requests.
	// 0 means no pacing. Set to 12*time.Second for Gemini free tier.
	MinInterval time.Duration

	mu       sync.Mutex
	lastReq  time.Time
}

// NewCompatTransport creates a transport wrapping a custom HTTP/1.1 transport.
// We force HTTP/1.1 because many third-party LLM proxies (e.g. agnes-ai)
// don't fully support HTTP/2, causing EOF errors on larger request bodies.
func NewCompatTransport() *CompatTransport {
	base := &http.Transport{
		ForceAttemptHTTP2:       false,
		DisableKeepAlives:       true, // prevent stale connection reuse → EOF
		MaxIdleConns:            10,
		IdleConnTimeout:         90 * time.Second,
		TLSHandshakeTimeout:     15 * time.Second,
		ExpectContinueTimeout:   1 * time.Second,
		ResponseHeaderTimeout:   120 * time.Second, // thinking models can be slow
		TLSClientConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &CompatTransport{base: base}
}

func (t *CompatTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// ── Pacing: enforce minimum interval between requests ──
	if t.MinInterval > 0 {
		t.mu.Lock()
		if !t.lastReq.IsZero() {
			elapsed := time.Since(t.lastReq)
			if wait := t.MinInterval - elapsed; wait > 0 {
				log.Printf("[CompatTransport] Pacing: waiting %.0fs before next request", wait.Seconds())
				time.Sleep(wait)
			}
		}
		t.lastReq = time.Now()
		t.mu.Unlock()
	}

	var patchedBody []byte

	// ── Request: strip extra_content from messages ──
	if req.Body != nil && req.Method == http.MethodPost {
		reqBody, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err == nil {
			patchedBody = patchRequest(reqBody)
			req.Body = io.NopCloser(bytes.NewReader(patchedBody))
			req.ContentLength = int64(len(patchedBody))
		}
	}

	// Retry loop: handles both 429 rate limits and transient connection errors (EOF).
	const maxRetries = 5
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Restore body for retry
		if patchedBody != nil {
			req.Body = io.NopCloser(bytes.NewReader(patchedBody))
			req.ContentLength = int64(len(patchedBody))
		}

		resp, err := t.base.RoundTrip(req)
		if err != nil {
			// Retry on transient connection errors (EOF, connection reset, broken pipe).
			if attempt < maxRetries && isTransientError(err) {
				delay := time.Duration(attempt+1) * 2 * time.Second
				log.Printf("[CompatTransport] Connection error (attempt %d/%d): %v — retrying in %.0fs",
					attempt+1, maxRetries, err, delay.Seconds())
				time.Sleep(delay)
				continue
			}
			return nil, err
		}

		// Retry on 429 Too Many Requests.
		if resp.StatusCode == 429 && attempt < maxRetries {
			// Read body to parse retry hint, then close.
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			delay := parseRetryDelay(bodyBytes, attempt)
			log.Printf("[CompatTransport] 429 rate limited, retry %d/%d after %.0fs",
				attempt+1, maxRetries, delay.Seconds())
			time.Sleep(delay)
			continue
		}

		// ── Response: fix error format + strip extra_content ──
		if resp.Body != nil {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				resp.Body = io.NopCloser(bytes.NewReader(nil))
				return resp, nil
			}

			// Fix Gemini's array-wrapped error first.
			body = fixErrorArray(body)

			// Strip unknown extra_content fields from tool_calls
			// so go-openai's strict decoder doesn't choke.
			if resp.StatusCode == 200 {
				body = stripExtraContent(body)
			}

			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
		}

		return resp, nil
	}

	return nil, fmt.Errorf("unexpected: exhausted retry loop")
}

// ─────── request patching ───────

// patchRequest strips extra_content from any tool_calls in the messages.
// Gemini adds extra_content.google.thought_signature to tool_calls when
// thinking mode is enabled (2.5+ models). We strip it because go-openai
// silently drops unknown fields. Removing it on both sides avoids mismatch.
func patchRequest(body []byte) []byte {
	var req map[string]json.RawMessage
	if json.Unmarshal(body, &req) != nil {
		return body
	}

	if msgsRaw, ok := req["messages"]; ok {
		var messages []map[string]json.RawMessage
		if json.Unmarshal(msgsRaw, &messages) == nil {
			changed := false
			for i := range messages {
				tcRaw, ok := messages[i]["tool_calls"]
				if !ok {
					continue
				}
				var toolCalls []map[string]json.RawMessage
				if json.Unmarshal(tcRaw, &toolCalls) != nil {
					continue
				}
				for j := range toolCalls {
					if _, ok := toolCalls[j]["extra_content"]; ok {
						delete(toolCalls[j], "extra_content")
						changed = true
					}
				}
				if changed {
					newTC, _ := json.Marshal(toolCalls)
					messages[i]["tool_calls"] = newTC
				}
			}
			if changed {
				newMsgs, _ := json.Marshal(messages)
				req["messages"] = newMsgs
			}
		}
	}

	out, err := json.Marshal(req)
	if err != nil {
		return body
	}
	return out
}

// ─────── response helpers ───────

// stripExtraContent removes extra_content from tool_calls in the response
// so go-openai's strict deserialization doesn't choke on unknown fields.
func stripExtraContent(body []byte) []byte {
	var resp map[string]json.RawMessage
	if json.Unmarshal(body, &resp) != nil {
		return body
	}

	choicesRaw, ok := resp["choices"]
	if !ok {
		return body
	}

	var choices []map[string]json.RawMessage
	if json.Unmarshal(choicesRaw, &choices) != nil {
		return body
	}

	changed := false
	for i := range choices {
		msgRaw, ok := choices[i]["message"]
		if !ok {
			continue
		}

		var msg map[string]json.RawMessage
		if json.Unmarshal(msgRaw, &msg) != nil {
			continue
		}

		tcRaw, ok := msg["tool_calls"]
		if !ok {
			continue
		}

		var toolCalls []map[string]json.RawMessage
		if json.Unmarshal(tcRaw, &toolCalls) != nil {
			continue
		}

		tcChanged := false
		for j := range toolCalls {
			if _, ok := toolCalls[j]["extra_content"]; ok {
				delete(toolCalls[j], "extra_content")
				tcChanged = true
			}
		}

		if tcChanged {
			newTC, _ := json.Marshal(toolCalls)
			msg["tool_calls"] = newTC
			newMsg, _ := json.Marshal(msg)
			choices[i]["message"] = newMsg
			changed = true
		}
	}

	if !changed {
		return body
	}

	newChoices, _ := json.Marshal(choices)
	resp["choices"] = newChoices
	out, err := json.Marshal(resp)
	if err != nil {
		return body
	}
	return out
}

// ─────── error format helpers ───────

// fixErrorArray converts Gemini's array-wrapped error to a plain object.
//
//	Gemini returns:  [{"error":{"code":400,"message":"...","status":"..."}}]
//	OpenAI expects:  {"error":{"message":"..."}}
func fixErrorArray(body []byte) []byte {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return body
	}

	var arr []json.RawMessage
	if json.Unmarshal(trimmed, &arr) != nil || len(arr) == 0 {
		return body
	}

	first := bytes.TrimSpace(arr[0])
	if len(first) == 0 {
		return body
	}
	return first
}

// ─────── retry helpers ───────

// isTransientError reports whether the error is likely a transient
// connection issue (EOF, reset, broken pipe) that warrants a retry.
func isTransientError(err error) bool {
	msg := err.Error()
	// Common patterns: "EOF", "connection reset by peer", "broken pipe",
	// "unexpected EOF", "tls: handshake failure"
	for _, pat := range []string{"EOF", "connection reset", "broken pipe", "use of closed"} {
		if strings.Contains(msg, pat) {
			return true
		}
	}
	return false
}

// retryAfterRegexp matches "Please retry in 48.5s" or "retry in 48s".
var retryAfterRegexp = regexp.MustCompile(`retry in (\d+\.?\d*)\s*s`)

// parseRetryDelay extracts the retry delay from a Gemini 429 response body.
// The body typically contains: "Please retry in 48.488610727s."
// Falls back to exponential backoff: 15s, 30s, 60s, 60s, 60s.
func parseRetryDelay(body []byte, attempt int) time.Duration {
	matches := retryAfterRegexp.FindSubmatch(body)
	if len(matches) >= 2 {
		if secs, err := strconv.ParseFloat(string(matches[1]), 64); err == nil && secs > 0 {
			// Add 2s buffer to the parsed value.
			return time.Duration(secs+2) * time.Second
		}
	}
	// Fallback: exponential backoff capped at 60s.
	backoffs := []time.Duration{15 * time.Second, 30 * time.Second, 60 * time.Second, 60 * time.Second, 60 * time.Second}
	if attempt >= len(backoffs) {
		return backoffs[len(backoffs)-1]
	}
	return backoffs[attempt]
}
