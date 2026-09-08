package pptoolkit

// batch.go — atomic batch execution.
//
// A batch of tool calls either fully applies or leaves the session exactly
// as it was before the batch started. One successful batch = one undo step.

// ToolCall is a named tool invocation with JSON-decoded arguments.
type ToolCall struct {
	Name      string
	Arguments map[string]any
}

// BatchResult reports the outcome of an atomic batch.
type BatchResult struct {
	Results     []ToolResult
	AllSuccess  bool
	FailedIndex int // -1 when AllSuccess; index of first failed call otherwise
	// RolledBack is true when the batch failed and the session was
	// restored to its pre-batch state.
	RolledBack bool
}

// ExecuteBatch runs calls sequentially against the session.
//
// Atomicity: a snapshot is taken before the batch; if any call fails, the
// snapshot is restored, so no partial effects survive. If every call
// succeeds, the pre-batch snapshot is pushed as a single undo step —
// one undo reverts the whole batch.
func (s *Session) ExecuteBatch(calls []ToolCall) BatchResult {
	if len(calls) == 0 {
		return BatchResult{Results: []ToolResult{}, AllSuccess: true, FailedIndex: -1}
	}

	snap, err := s.snapshot()
	if err != nil {
		return BatchResult{
			Results: []ToolResult{fail("batch checkpoint failed: " + err.Error())},
		}
	}

	// Suppress per-call history push: the batch commits one undo step at
	// the end (or none, on rollback).
	s.mu.Lock()
	suppressed := s.suppressHistory
	s.suppressHistory = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.suppressHistory = suppressed
		s.mu.Unlock()
	}()

	results := make([]ToolResult, 0, len(calls))
	for i, call := range calls {
		if call.Arguments == nil {
			call.Arguments = map[string]any{}
		}
		result := s.ExecuteTool(call.Name, call.Arguments)
		results = append(results, result)
		if !result.Success {
			// Roll the whole batch back to the pre-batch snapshot.
			if rbErr := s.restore(snap); rbErr != nil {
				results[i] = fail(result.Message + " (also: rollback failed: " + rbErr.Error() + ")")
			}
			return BatchResult{
				Results:     results,
				FailedIndex: i,
				RolledBack:  true,
			}
		}
	}

	s.pushHistory(snap)
	return BatchResult{
		Results:     results,
		AllSuccess:  true,
		FailedIndex: -1,
	}
}
