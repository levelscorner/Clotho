package domain

import "time"

// FailureClass groups every way a step can fail into a small enum so the UI
// can surface a deterministic badge + recovery hint instead of dumping a
// raw error string. Adding a new class here also requires updating
// (a) ClassifyProviderError in internal/engine/failure.go, (b) the badge
// color map in web/src/components/execution/FailureDrawer.tsx, and
// (c) docs/PIPELINE-PATTERNS.md §failure-classes.
type FailureClass string

const (
	// Transport-layer issues — usually retryable.
	FailureNetwork   FailureClass = "network"    // connection refused, DNS, EOF mid-stream
	FailureTimeout   FailureClass = "timeout"    // step or per-call deadline exceeded
	FailureRateLimit FailureClass = "rate_limit" // 429 from provider; retry will back off

	// Provider responses.
	FailureAuth        FailureClass = "auth"         // 401/403; user must fix credential
	FailureProvider5xx FailureClass = "provider_5xx" // upstream broken; retryable
	FailureProvider4xx FailureClass = "provider_4xx" // bad request; not retryable as-is

	// Local validation.
	FailureValidation    FailureClass = "validation"     // input or graph schema rejected
	FailureOutputShape   FailureClass = "output_shape"   // TTS returned text, image returned JSON, etc.
	FailureOutputQuality FailureClass = "output_quality" // toxicity / PII / scoring (deferred)

	// Engine-level.
	FailureCostCap     FailureClass = "cost_cap"     // would exceed configured cap
	FailureCircuitOpen FailureClass = "circuit_open" // breaker tripped — short-circuited
	FailureInternal    FailureClass = "internal"     // bug in Clotho itself
)

// FailureStage names where in the per-step pipeline the failure happened.
// Useful for both the UI hint and metric labels.
type FailureStage string

const (
	StageInputResolve   FailureStage = "input_resolve"
	StageProviderCall   FailureStage = "provider_call"
	StageStreamParse    FailureStage = "stream_parse"
	StageOutputValidate FailureStage = "output_validate"
	StagePersist        FailureStage = "persist"
)

// StepFailure is the structured payload that flows from the executor →
// engine → SSE event → frontend store → FailureDrawer. The shape
// intentionally mirrors what creators need at the moment of breakage:
// "what kind of problem", "where did it happen", "is it worth retrying",
// "what should I do now".
//
// Both Message and Cause MUST be passed through redact.Secrets before
// landing here so credentials in upstream errors never leak to the UI or
// the persisted failure_json column.
type StepFailure struct {
	Class     FailureClass `json:"class"`
	Stage     FailureStage `json:"stage"`
	Provider  string       `json:"provider,omitempty"`
	Model     string       `json:"model,omitempty"`
	Retryable bool         `json:"retryable"`
	Message   string       `json:"message"`         // human-readable, scrubbed
	Cause     string       `json:"cause,omitempty"` // raw upstream error, scrubbed
	Hint      string       `json:"hint,omitempty"`  // call-to-action, e.g. "Verify API key in Settings"
	Attempts  int          `json:"attempts"`        // how many times we tried before giving up
	At        time.Time    `json:"at"`
}
