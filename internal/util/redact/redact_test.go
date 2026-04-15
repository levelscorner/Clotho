package redact

import (
	"errors"
	"strings"
	"testing"
)

func TestSecrets(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		mustStrip  string // substring that MUST be absent from output
		mustContain string // substring that must remain (prefix trace)
	}{
		{
			name:        "anthropic api key",
			in:          "401 Unauthorized: key sk-ant-api03-aBcDeFgHiJkLmNoPqRsTuVwXyZ1234567890ABCDEFGH rejected",
			mustStrip:   "aBcDeFgHiJkLmNoPqRsTuVwXyZ",
			mustContain: "sk-ant-api03-",
		},
		{
			name:        "openai project key",
			in:          `{"error":"invalid api key sk-proj-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMN"}`,
			mustStrip:   "abcdefghijklmnopqrstuvwxyz",
			mustContain: "sk-proj-",
		},
		{
			name:        "openrouter key",
			in:          "header: Authorization: Bearer sk-or-v1-0123456789abcdef0123456789abcdef",
			mustStrip:   "0123456789abcdef0123456789",
			mustContain: "sk-or-",
		},
		{
			name:        "gemini AIza key",
			in:          "gemini stream failed: key AIzaSyAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA expired",
			mustStrip:   "SyAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			mustContain: "AIza",
		},
		{
			name:        "replicate token",
			in:          "replicate 401: token r8_abc123def456ghi789jkl012mno345 denied",
			mustStrip:   "abc123def456ghi789jkl012mno345",
			mustContain: "r8_",
		},
		{
			name:        "bearer token generic",
			in:          "Authorization: Bearer abcdef1234567890abcdef1234567890xyz",
			mustStrip:   "abcdef1234567890abcdef1234567890xyz",
			mustContain: "Bearer",
		},
		{
			name:        "legacy openai key",
			in:          "error from provider: sk-abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGH is invalid",
			mustStrip:   "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGH",
			mustContain: "sk-",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Secrets(tc.in)
			if tc.mustStrip != "" && strings.Contains(got, tc.mustStrip) {
				t.Errorf("secret substring leaked into output\n  input:  %q\n  output: %q\n  leaked: %q", tc.in, got, tc.mustStrip)
			}
			if tc.mustContain != "" && !strings.Contains(got, tc.mustContain) {
				t.Errorf("prefix marker dropped from output\n  input:    %q\n  output:   %q\n  expected: %q", tc.in, got, tc.mustContain)
			}
			if !strings.Contains(got, "[redacted]") {
				t.Errorf("redaction marker missing from output: %q", got)
			}
		})
	}
}

func TestSecretsEmpty(t *testing.T) {
	if got := Secrets(""); got != "" {
		t.Fatalf("empty input should return empty output, got %q", got)
	}
}

func TestSecretsPassthrough(t *testing.T) {
	in := "generic rate-limit error with no key attached"
	if got := Secrets(in); got != in {
		t.Fatalf("non-secret input should pass through unchanged\n  input:  %q\n  output: %q", in, got)
	}
}

func TestErrorNil(t *testing.T) {
	if got := Error(nil); got != "" {
		t.Fatalf("Error(nil) should return empty string, got %q", got)
	}
}

func TestErrorWraps(t *testing.T) {
	err := errors.New("401: AIzaSyTESTTESTTESTTESTTESTTESTTESTTEST-xx expired")
	got := Error(err)
	if strings.Contains(got, "TESTTESTTESTTESTTEST") {
		t.Fatalf("Error should scrub secret: %q", got)
	}
	if !strings.Contains(got, "AIza") {
		t.Fatalf("Error should keep AIza prefix for diagnostics: %q", got)
	}
}
