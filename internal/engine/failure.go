package engine

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/util/redact"
)

// FailureError wraps a structured StepFailure as an error so executors
// can return it through the existing error-channel contract while
// preserving the rich payload. Engine's failure path uses errors.As to
// recover the StepFailure for SSE + persistence; callers without that
// awareness see the same string they always did.
type FailureError struct {
	Failure domain.StepFailure
	Cause   error
}

func (f *FailureError) Error() string {
	if f.Failure.Message != "" {
		return f.Failure.Message
	}
	if f.Cause != nil {
		return f.Cause.Error()
	}
	return string(f.Failure.Class)
}

func (f *FailureError) Unwrap() error { return f.Cause }

// AsFailure pulls the structured StepFailure out of err, returning
// (failure, true) if found. Callers that need to surface the rich
// payload (engine SSE handler, store persistence) call this; callers
// that only need a string fall back to err.Error().
func AsFailure(err error) (domain.StepFailure, bool) {
	if err == nil {
		return domain.StepFailure{}, false
	}
	var fe *FailureError
	if errors.As(err, &fe) {
		return fe.Failure, true
	}
	if errors.Is(err, ErrCircuitOpen) {
		return domain.StepFailure{
			Class:     domain.FailureCircuitOpen,
			Stage:     domain.StageProviderCall,
			Retryable: false,
			Message:   "Circuit breaker open — provider had too many recent failures.",
			Hint:      hintFor[domain.FailureCircuitOpen],
			At:        time.Now().UTC(),
		}, true
	}
	return domain.StepFailure{}, false
}

// ClassifyExecutionError is the engine-side fallback when the executor
// returned a plain error rather than a *FailureError. It runs the same
// classification we'd do at the provider boundary, then redacts.
func ClassifyExecutionError(err error, provider, model string) domain.StepFailure {
	if err == nil {
		return domain.StepFailure{}
	}
	if f, ok := AsFailure(err); ok {
		return f
	}
	return ClassifyProviderError(err, provider, model)
}

// hintFor returns the call-to-action surfaced in the FailureDrawer for a
// given class. Keep these short — they're rendered as a single-line hint
// next to the class badge. If you add a class to domain.FailureClass, add
// a hint here too (the empty string is fine and just hides the line).
var hintFor = map[domain.FailureClass]string{
	domain.FailureAuth:           "Verify the API key in Settings → Credentials.",
	domain.FailureRateLimit:      "Provider is throttling. Retries will back off automatically.",
	domain.FailureTimeout:        "Increase the step timeout or pick a smaller / faster model.",
	domain.FailureNetwork:        "Network blip. Check your connection or the provider's status page.",
	domain.FailureProvider5xx:    "The provider is having an outage. Retries will keep trying.",
	domain.FailureProvider4xx:    "The provider rejected the request. Check model name and parameters.",
	domain.FailureValidation:     "Input did not match the schema. Inspect the upstream node's output.",
	domain.FailureOutputShape:    "The model returned the wrong content type for this output port.",
	domain.FailureCostCap:        "This step would exceed the configured cost cap. Raise the cap or pick a cheaper model.",
	domain.FailureCircuitOpen:    "Too many recent failures from this provider+model. Cooling down before retry.",
	domain.FailureInternal:       "Clotho hit an internal bug. Copy the diagnostic and file an issue.",
	domain.FailureOutputQuality:  "The output failed a quality check.",
}

// retryableClasses lists every class whose default policy is "try again".
// Used by the retry loop in agent_executor.go and by the breaker to decide
// whether a failed call counts toward the open threshold (auth failures
// shouldn't trip the breaker — they need a human, not a wait).
var retryableClasses = map[domain.FailureClass]bool{
	domain.FailureNetwork:     true,
	domain.FailureTimeout:     true,
	domain.FailureRateLimit:   true,
	domain.FailureProvider5xx: true,
}

// ClassifyProviderError maps any error coming back from a provider call
// (or the engine wrapper around it) into a structured StepFailure with
// scrubbed Message + Cause, hint, and retryable flag.
//
// Order of checks matters — we look at sentinel errors and typed SDK
// errors before falling back to string matching, because string matching
// is brittle across SDK versions.
func ClassifyProviderError(err error, provider, model string) domain.StepFailure {
	if err == nil {
		return domain.StepFailure{}
	}

	class := classifyError(err)

	failure := domain.StepFailure{
		Class:     class,
		Stage:     domain.StageProviderCall,
		Provider:  provider,
		Model:     model,
		Retryable: retryableClasses[class],
		Message:   redact.Secrets(humanMessage(class, err)),
		Cause:     redact.Secrets(err.Error()),
		Hint:      hintFor[class],
		Attempts:  1, // caller bumps this when wrapping inside a retry loop
		At:        time.Now().UTC(),
	}

	return failure
}

// classifyError is the pure routing function — given an error, which
// FailureClass best describes it? Split out so tests can target the
// classification logic without going through the full ClassifyProviderError
// envelope (which adds redaction + timestamps).
func classifyError(err error) domain.FailureClass {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return domain.FailureTimeout
	case errors.Is(err, context.Canceled):
		// Caller cancellation isn't really a "failure" but if we're being
		// asked to classify it, treat it as timeout-class (the user gave
		// up). The engine should usually short-circuit before reaching here.
		return domain.FailureTimeout
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, net.ErrClosed):
		return domain.FailureNetwork
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		// Includes connection refused, DNS errors, broken pipe.
		return domain.FailureNetwork
	}

	// go-openai surfaces typed APIError for both real OpenAI and any
	// OpenAI-compatible upstream (Ollama, OpenRouter), so we get one
	// classifier branch for three providers.
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		return classifyHTTPStatus(apiErr.HTTPStatusCode)
	}

	// Gemini's HTTP client returns plain errors with status codes embedded
	// in the message — we string-match those as a last resort. The exact
	// formats live in internal/llm/gemini.go::Complete: "gemini complete:
	// status %d: %s".
	msg := err.Error()
	if status := extractHTTPStatusFromMessage(msg); status > 0 {
		return classifyHTTPStatus(status)
	}

	// Network errors that didn't surface as typed *net.OpError sometimes
	// look like "dial tcp:" or "connection refused" in plain strings.
	lc := strings.ToLower(msg)
	if strings.Contains(lc, "connection refused") ||
		strings.Contains(lc, "no such host") ||
		strings.Contains(lc, "i/o timeout") ||
		strings.Contains(lc, "tls handshake") {
		return domain.FailureNetwork
	}

	return domain.FailureInternal
}

// classifyHTTPStatus turns an HTTP status code into a FailureClass. Used
// by both typed APIError and string-extracted Gemini status paths.
func classifyHTTPStatus(code int) domain.FailureClass {
	switch {
	case code == 401, code == 403:
		return domain.FailureAuth
	case code == 408:
		return domain.FailureTimeout
	case code == 429:
		return domain.FailureRateLimit
	case code >= 500 && code < 600:
		return domain.FailureProvider5xx
	case code >= 400 && code < 500:
		return domain.FailureProvider4xx
	}
	return domain.FailureInternal
}

// extractHTTPStatusFromMessage looks for the "status N:" pattern that
// internal/llm/gemini.go uses when the upstream returns non-200. Returns
// 0 if nothing parseable is found.
func extractHTTPStatusFromMessage(msg string) int {
	// Pattern: "...status NNN:..." anywhere in the message.
	const marker = "status "
	idx := strings.Index(msg, marker)
	if idx < 0 {
		return 0
	}
	rest := msg[idx+len(marker):]
	// Read up to 3 digits.
	var code int
	digits := 0
	for i := 0; i < len(rest) && i < 3; i++ {
		c := rest[i]
		if c < '0' || c > '9' {
			break
		}
		code = code*10 + int(c-'0')
		digits++
	}
	if digits == 0 {
		return 0
	}
	return code
}

// humanMessage produces the short summary shown in the FailureDrawer
// header. Class-specific to give the user an immediately useful hint
// about what happened, instead of dumping the raw upstream error string.
func humanMessage(class domain.FailureClass, err error) string {
	switch class {
	case domain.FailureAuth:
		return "Authentication failed (invalid or missing API key)."
	case domain.FailureRateLimit:
		return "Provider rate limit reached."
	case domain.FailureTimeout:
		return "The provider call timed out."
	case domain.FailureNetwork:
		return "Network error reaching the provider."
	case domain.FailureProvider5xx:
		return "The provider returned a server error."
	case domain.FailureProvider4xx:
		return "The provider rejected the request."
	case domain.FailureValidation:
		return "Input failed validation."
	case domain.FailureOutputShape:
		return "Output type did not match the declared port type."
	case domain.FailureCostCap:
		return "Step would exceed the configured cost cap."
	case domain.FailureCircuitOpen:
		return "Circuit breaker is open for this provider+model."
	case domain.FailureInternal:
		// Fall through to the actual error text since we have no better
		// summary — but still scrubbed by the caller.
		return err.Error()
	}
	return err.Error()
}
