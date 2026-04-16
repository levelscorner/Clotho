package engine

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/h2non/filetype"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/user/clotho/internal/domain"
)

// ValidateOutput runs structural + semantic checks against a node's
// output and returns a populated StepFailure if anything looks off. The
// returned StepFailure has Class set; callers wrap or persist it as
// needed. A nil return means the output passed.
//
// Two kinds of validation:
//   1. Structural — for media-typed output ports (image/audio/video) we
//      sniff the magic bytes and confirm the MIME family matches.
//      Catches the canonical failure mode where a TTS endpoint returns
//      a JSON apology string instead of audio bytes.
//   2. Semantic   — for ports whose schema is set (json output_type
//      with cfg.Task.OutputSchema), validate the JSON body against the
//      JSON Schema. Catches LLMs returning structurally-correct text
//      that doesn't conform to the requested shape.
func ValidateOutput(node domain.NodeInstance, output json.RawMessage) *domain.StepFailure {
	// Pick the output port we care about — the first declared output.
	// All the executor types we ship today produce a single output;
	// when that changes we'll need to take a port ID parameter.
	port, ok := primaryOutputPort(node)
	if !ok {
		return nil
	}

	switch port.Type {
	case domain.PortTypeImage, domain.PortTypeAudio, domain.PortTypeVideo:
		return validateMediaShape(port.Type, output)
	case domain.PortTypeJSON:
		schema := extractOutputSchema(node)
		if schema == nil {
			// No schema declared — text-as-json is acceptable; the user
			// chose not to constrain the output.
			return nil
		}
		return validateJSONSchema(output, schema)
	}

	return nil
}

// primaryOutputPort returns the first output port on the node, if any.
func primaryOutputPort(node domain.NodeInstance) (domain.Port, bool) {
	for _, p := range node.Ports {
		if p.Direction == domain.PortOutput {
			return p, true
		}
	}
	return domain.Port{}, false
}

// extractOutputSchema pulls cfg.Task.OutputSchema off an agent node,
// returning nil if the node isn't an agent or no schema is declared.
func extractOutputSchema(node domain.NodeInstance) *json.RawMessage {
	if node.Type != domain.NodeTypeAgent {
		return nil
	}
	var cfg domain.AgentNodeConfig
	if err := json.Unmarshal(node.Config, &cfg); err != nil {
		return nil
	}
	return cfg.Task.OutputSchema
}

// validateMediaShape unwraps the raw JSON bytes (which may be a quoted
// string or a clotho://file/ URL or a data URI) and confirms the actual
// payload matches the expected media family.
//
// The output column stores agent text outputs as JSON-encoded strings
// and media outputs typically as a clotho://file/ reference produced by
// the media provider. For URL refs we trust the upstream provider — the
// failure mode this function targets is the inline-bytes case where a
// model returns base64 / data URI / raw text.
func validateMediaShape(want domain.PortType, output json.RawMessage) *domain.StepFailure {
	if len(bytes.TrimSpace(output)) == 0 {
		return failureOutputShape(want, "empty output for media port")
	}

	// Try to unmarshal into a string. If output is a JSON string we
	// inspect its decoded form (e.g., "Sorry, I can't generate audio").
	var asString string
	if err := json.Unmarshal(output, &asString); err == nil {
		// URL-like refs are passed through — the provider already wrote
		// real bytes to disk. Magic-byte checking on a URL string makes
		// no sense and would false-positive the legitimate happy path.
		if isURLRef(asString) {
			return nil
		}
		// Data URI: base64-decode the data segment, then sniff.
		if strings.HasPrefix(asString, "data:") {
			if data, ok := decodeDataURI(asString); ok {
				return sniffAndCompare(want, data)
			}
		}
		// Plain text string in a media port = the canonical "TTS
		// returned text" failure. Flag it.
		return failureOutputShape(want,
			"model returned text/plain in a "+string(want)+" output port")
	}

	// Not a JSON string — probably raw bytes embedded as a JSON array
	// or similar. Try to extract bytes via best-effort.
	return sniffAndCompare(want, []byte(output))
}

// sniffAndCompare uses h2non/filetype to detect the MIME family of data
// and reports a StepFailure if the family doesn't match the port's
// declared type. h2non/filetype only needs the first few hundred bytes
// to detect the format reliably for every common image/audio/video type.
func sniffAndCompare(want domain.PortType, data []byte) *domain.StepFailure {
	if len(data) == 0 {
		return failureOutputShape(want, "empty bytes for media port")
	}
	kind, err := filetype.Match(data)
	if err != nil || kind == filetype.Unknown {
		return failureOutputShape(want,
			"unrecognized binary format for "+string(want)+" port")
	}

	got := mimeFamily(kind.MIME.Type)
	if !mediaFamilyMatches(want, got) {
		return failureOutputShape(want, fmt.Sprintf(
			"model returned %s/%s but port expects %s/*",
			kind.MIME.Type, kind.MIME.Subtype, mimeFamilyForPort(want),
		))
	}
	return nil
}

// validateJSONSchema compiles the schema (caching is the caller's job
// for now) and validates the JSON output. Returns a FailureValidation
// StepFailure with the offending JSON path on mismatch.
func validateJSONSchema(output json.RawMessage, schema *json.RawMessage) *domain.StepFailure {
	if schema == nil || len(bytes.TrimSpace(*schema)) == 0 {
		return nil
	}

	c := jsonschema.NewCompiler()
	schemaURL := "clotho://node/output_schema.json"
	if err := c.AddResource(schemaURL, mustJSON(*schema)); err != nil {
		return &domain.StepFailure{
			Class:     domain.FailureValidation,
			Stage:     domain.StageOutputValidate,
			Message:   "Output schema is invalid JSON",
			Cause:     err.Error(),
			Hint:      "Fix the JSON Schema on the agent's output port.",
			Retryable: false,
		}
	}
	compiled, err := c.Compile(schemaURL)
	if err != nil {
		return &domain.StepFailure{
			Class:     domain.FailureValidation,
			Stage:     domain.StageOutputValidate,
			Message:   "Output schema does not compile",
			Cause:     err.Error(),
			Hint:      "Check that the schema is valid JSON Schema (draft 2020-12).",
			Retryable: false,
		}
	}

	value, err := mustJSONErr(output)
	if err != nil {
		return &domain.StepFailure{
			Class:     domain.FailureValidation,
			Stage:     domain.StageOutputValidate,
			Message:   "Output is not valid JSON",
			Cause:     err.Error(),
			Hint:      "Try a more deterministic temperature or use Output Format = json.",
			Retryable: true, // an LLM could plausibly succeed on retry
		}
	}

	if err := compiled.Validate(value); err != nil {
		return &domain.StepFailure{
			Class:     domain.FailureValidation,
			Stage:     domain.StageOutputValidate,
			Message:   "Output failed JSON Schema validation",
			Cause:     err.Error(),
			Hint:      "Inspect the schema and the upstream prompt for mismatches.",
			Retryable: true, // LLM may produce a conformant output on retry
		}
	}
	return nil
}

// failureOutputShape builds the canonical StepFailure for a structural
// mismatch. Class = output_shape, Stage = output_validate, Retryable =
// true (a different model or a retry might produce the right format).
func failureOutputShape(want domain.PortType, message string) *domain.StepFailure {
	return &domain.StepFailure{
		Class:     domain.FailureOutputShape,
		Stage:     domain.StageOutputValidate,
		Message:   message,
		Hint:      hintFor[domain.FailureOutputShape] + " Expected family: " + mimeFamilyForPort(want) + ".",
		Retryable: false, // shape mismatches usually mean wrong model — human picks new model
	}
}

// mediaFamilyMatches checks if the detected MIME family matches what
// the declared port type expects. Conservative: image port accepts only
// image/*, audio port only audio/*, video port only video/*.
func mediaFamilyMatches(port domain.PortType, family string) bool {
	switch port {
	case domain.PortTypeImage:
		return family == "image"
	case domain.PortTypeAudio:
		return family == "audio"
	case domain.PortTypeVideo:
		return family == "video"
	}
	return false
}

// mimeFamily extracts the family ("image", "audio", "video") from a
// MIME type string like "image/png".
func mimeFamily(mime string) string {
	if i := strings.Index(mime, "/"); i > 0 {
		return mime[:i]
	}
	return mime
}

// mimeFamilyForPort maps a port type to its expected MIME family for
// human-friendly error messages.
func mimeFamilyForPort(p domain.PortType) string {
	switch p {
	case domain.PortTypeImage:
		return "image"
	case domain.PortTypeAudio:
		return "audio"
	case domain.PortTypeVideo:
		return "video"
	}
	return string(p)
}

// isURLRef detects URL-like strings the executor passes through
// untouched. Matches clotho://file/, http(s)://, and provider-specific
// blob:// schemes.
func isURLRef(s string) bool {
	return strings.HasPrefix(s, "clotho://") ||
		strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "blob:")
}

// decodeDataURI extracts the binary payload from a data: URI like
// "data:image/png;base64,iVBOR...". Returns false when the URI doesn't
// parse cleanly.
func decodeDataURI(s string) ([]byte, bool) {
	const prefix = "data:"
	if !strings.HasPrefix(s, prefix) {
		return nil, false
	}
	rest := s[len(prefix):]
	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		return nil, false
	}
	header := rest[:commaIdx]
	body := rest[commaIdx+1:]
	if !strings.Contains(header, "base64") {
		// Treat percent-encoded body as opaque text bytes.
		return []byte(body), true
	}
	out, err := decodeBase64(body)
	if err != nil {
		return nil, false
	}
	return out, true
}

// decodeBase64 wraps StdEncoding so the caller stays tidy.
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// mustJSON unmarshals raw bytes into any. Panics only on malformed
// input; callers that can't tolerate a panic use mustJSONErr.
func mustJSON(raw json.RawMessage) any {
	var v any
	_ = json.Unmarshal(raw, &v)
	return v
}

// mustJSONErr is the error-returning version. Used in validation paths
// where invalid JSON is a real failure mode, not a precondition.
func mustJSONErr(raw json.RawMessage) (any, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return v, nil
}
