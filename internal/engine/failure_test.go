package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/user/clotho/internal/domain"
)

func TestClassifyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want domain.FailureClass
	}{
		{"deadline_exceeded", context.DeadlineExceeded, domain.FailureTimeout},
		{"canceled", context.Canceled, domain.FailureTimeout},
		{"io_eof", io.EOF, domain.FailureNetwork},
		{"io_unexpected_eof", io.ErrUnexpectedEOF, domain.FailureNetwork},
		{"net_closed", net.ErrClosed, domain.FailureNetwork},

		{
			name: "net_op_error",
			err:  &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			want: domain.FailureNetwork,
		},

		// go-openai typed APIError — exercise every branch.
		{
			name: "openai_401",
			err:  &openai.APIError{HTTPStatusCode: 401, Message: "invalid api key"},
			want: domain.FailureAuth,
		},
		{
			name: "openai_403",
			err:  &openai.APIError{HTTPStatusCode: 403, Message: "forbidden"},
			want: domain.FailureAuth,
		},
		{
			name: "openai_408",
			err:  &openai.APIError{HTTPStatusCode: 408, Message: "request timeout"},
			want: domain.FailureTimeout,
		},
		{
			name: "openai_429",
			err:  &openai.APIError{HTTPStatusCode: 429, Message: "rate limit"},
			want: domain.FailureRateLimit,
		},
		{
			name: "openai_500",
			err:  &openai.APIError{HTTPStatusCode: 500, Message: "server error"},
			want: domain.FailureProvider5xx,
		},
		{
			name: "openai_503",
			err:  &openai.APIError{HTTPStatusCode: 503, Message: "service unavailable"},
			want: domain.FailureProvider5xx,
		},
		{
			name: "openai_400",
			err:  &openai.APIError{HTTPStatusCode: 400, Message: "bad request"},
			want: domain.FailureProvider4xx,
		},
		{
			name: "openai_404",
			err:  &openai.APIError{HTTPStatusCode: 404, Message: "model not found"},
			want: domain.FailureProvider4xx,
		},
		// Wrapped APIError still classifies correctly via errors.As.
		{
			name: "wrapped_openai_429",
			err:  fmt.Errorf("upstream: %w", &openai.APIError{HTTPStatusCode: 429}),
			want: domain.FailureRateLimit,
		},

		// Gemini-style string errors — internal/llm/gemini.go formats as
		// "gemini complete: status 401: <body>" etc.
		{
			name: "gemini_status_401",
			err:  errors.New("gemini complete: status 401: API key not valid"),
			want: domain.FailureAuth,
		},
		{
			name: "gemini_status_429",
			err:  errors.New("gemini stream: status 429: quota exceeded"),
			want: domain.FailureRateLimit,
		},
		{
			name: "gemini_status_500",
			err:  errors.New("gemini complete: status 500: internal"),
			want: domain.FailureProvider5xx,
		},

		// String-matched network failures.
		{
			name: "string_connection_refused",
			err:  errors.New("dial tcp 127.0.0.1:8080: connect: connection refused"),
			want: domain.FailureNetwork,
		},
		{
			name: "string_dns",
			err:  errors.New("dial tcp: lookup api.example.com: no such host"),
			want: domain.FailureNetwork,
		},
		{
			name: "string_tls",
			err:  errors.New("Get \"https://api.example.com\": tls handshake failure"),
			want: domain.FailureNetwork,
		},

		// Unknown / catch-all.
		{
			name: "unknown",
			err:  errors.New("something weird happened"),
			want: domain.FailureInternal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyError(tc.err)
			if got != tc.want {
				t.Errorf("classifyError(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestClassifyProviderError_Envelope(t *testing.T) {
	t.Parallel()

	t.Run("nil error returns zero StepFailure", func(t *testing.T) {
		t.Parallel()
		got := ClassifyProviderError(nil, "openai", "gpt-4o")
		if got != (domain.StepFailure{}) {
			t.Errorf("nil err should return zero StepFailure, got %+v", got)
		}
	})

	t.Run("populates provider, model, hint, retryable", func(t *testing.T) {
		t.Parallel()
		err := &openai.APIError{HTTPStatusCode: 429, Message: "rate limited"}
		got := ClassifyProviderError(err, "openai", "gpt-4o-mini")

		if got.Provider != "openai" || got.Model != "gpt-4o-mini" {
			t.Errorf("provider/model not propagated: %+v", got)
		}
		if got.Class != domain.FailureRateLimit {
			t.Errorf("class = %q, want rate_limit", got.Class)
		}
		if !got.Retryable {
			t.Errorf("rate_limit should be retryable")
		}
		if got.Hint == "" {
			t.Errorf("rate_limit should have a hint")
		}
		if got.Stage != domain.StageProviderCall {
			t.Errorf("stage default should be provider_call")
		}
		if got.Attempts != 1 {
			t.Errorf("default Attempts = %d, want 1", got.Attempts)
		}
		if got.At.IsZero() {
			t.Errorf("At should be set")
		}
	})

	t.Run("auth class is not retryable", func(t *testing.T) {
		t.Parallel()
		got := ClassifyProviderError(
			&openai.APIError{HTTPStatusCode: 401, Message: "bad key"},
			"openai", "gpt-4o",
		)
		if got.Retryable {
			t.Errorf("auth class must NOT be retryable")
		}
	})

	t.Run("scrubs API key from cause", func(t *testing.T) {
		t.Parallel()
		// Use a long token-like string that the redact package will catch.
		// redact.Secrets matches the OpenAI key pattern sk-...
		// (49+ chars after sk-) and similar; pick something realistic.
		raw := "Bearer sk-proj-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		err := errors.New("auth failed: " + raw)
		got := ClassifyProviderError(err, "openai", "gpt-4o")

		if strings.Contains(got.Cause, raw) {
			t.Errorf("Cause leaked the raw secret: %q", got.Cause)
		}
	})
}

func TestExtractHTTPStatusFromMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want int
	}{
		{"gemini complete: status 401: bad key", 401},
		{"gemini stream: status 503: outage", 503},
		{"prefix status 200 ok suffix", 200},
		{"no status here", 0},
		{"status abc", 0},                // non-numeric
		{"status 12345 too long", 123},   // capped at 3 digits
		{"first status 200 second status 500", 200}, // first wins
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got := extractHTTPStatusFromMessage(tc.in)
			if got != tc.want {
				t.Errorf("extractHTTPStatusFromMessage(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
