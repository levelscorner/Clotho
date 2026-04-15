// Package redact scrubs API keys and secrets from strings before they are
// logged or returned to clients. Centralized so every provider error path
// goes through the same sieve — if a new provider adds a key format we
// only need to teach one file about it.
//
// Redaction is intentionally conservative: we keep the first few
// characters for debugging context ("sk-ant-api03-a…") and replace the
// rest with "...[redacted]". Callers should treat the result as
// display-only and never try to recover the original value.
package redact

import (
	"regexp"
	"strings"
)

// secretPatterns enumerates every provider-key prefix and format we've seen.
// Patterns are ordered most-specific-first so longer matches win.
var secretPatterns = []*regexp.Regexp{
	// Anthropic: sk-ant-<version>-<40+ base64-ish chars>
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{6,}[A-Za-z0-9]{20,}`),
	// OpenAI project keys: sk-proj-<...>  (long modern format)
	regexp.MustCompile(`sk-proj-[A-Za-z0-9_-]{20,}`),
	// OpenRouter: sk-or-v1-<64 hex>
	regexp.MustCompile(`sk-or-[A-Za-z0-9_-]{20,}`),
	// Legacy OpenAI: sk-<48 alnum>
	regexp.MustCompile(`\bsk-[A-Za-z0-9]{30,}\b`),
	// Google AI Studio / Gemini: AIza<35 chars>
	regexp.MustCompile(`AIza[0-9A-Za-z_-]{20,}`),
	// Replicate: r8_<alnum>
	regexp.MustCompile(`r8_[A-Za-z0-9]{20,}`),
	// Generic Bearer tokens in headers or error echoes.
	regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._\-+=/]{16,}`),
}

// Secrets scrubs every recognised secret out of s. Safe to call with any
// string, including empty. Non-matches pass through unchanged.
func Secrets(s string) string {
	if s == "" {
		return s
	}
	out := s
	for _, re := range secretPatterns {
		out = re.ReplaceAllStringFunc(out, maskMatch)
	}
	return out
}

// Error returns err's message with secrets redacted. Returns empty string
// on nil so callers can inline it in fmt calls without a nil guard.
func Error(err error) string {
	if err == nil {
		return ""
	}
	return Secrets(err.Error())
}

// maskMatch keeps enough leading characters to identify the key family
// (e.g. "sk-ant-api03-") and replaces the rest with a fixed placeholder.
func maskMatch(match string) string {
	// Preserve common prefixes in full so "sk-ant-" etc remain diagnostic.
	prefixes := []string{
		"sk-ant-api03-",
		"sk-ant-api01-",
		"sk-ant-",
		"sk-proj-",
		"sk-or-v1-",
		"sk-or-",
		"AIza",
		"r8_",
		"sk-",
	}
	lower := strings.ToLower(match)
	for _, p := range prefixes {
		if strings.HasPrefix(lower, strings.ToLower(p)) {
			// Keep prefix + up to 4 chars of the actual secret for traceability.
			keep := len(p)
			if len(match) > keep+4 {
				return match[:keep+4] + "…[redacted]"
			}
			return match[:keep] + "[redacted]"
		}
	}
	// Bearer / other — keep first 8 chars.
	if len(match) > 8 {
		return match[:8] + "…[redacted]"
	}
	return "[redacted]"
}
