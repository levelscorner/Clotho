package engine

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/user/clotho/internal/domain"
)

// Golden binary headers — small enough to embed inline. h2non/filetype
// detects each from the first ~32 bytes.

// 1x1 transparent PNG.
const minimalPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII="

// First 64 bytes of a real ID3-tagged MP3 (synthesized; sufficient for
// magic-byte sniffing).
var mp3Header = []byte{
	0x49, 0x44, 0x33, 0x03, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x21, 0x54, 0x49, 0x54, 0x32, 0x00, 0x00,
	0x00, 0x05, 0x00, 0x00, 0x00, 0x74, 0x65, 0x73,
	0x74, 0xff, 0xfb, 0x90, 0x00, 0x00, 0x00, 0x00,
}

// MP4 ftyp box header.
var mp4Header = []byte{
	0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70,
	0x69, 0x73, 0x6f, 0x6d, 0x00, 0x00, 0x02, 0x00,
	0x69, 0x73, 0x6f, 0x6d, 0x69, 0x73, 0x6f, 0x32,
	0x61, 0x76, 0x63, 0x31, 0x6d, 0x70, 0x34, 0x31,
}

func mediaNode(t *testing.T, port domain.PortType) domain.NodeInstance {
	t.Helper()
	return domain.NodeInstance{
		Type: domain.NodeTypeMedia,
		Ports: []domain.Port{
			{ID: "out", Direction: domain.PortOutput, Type: port},
		},
	}
}

func agentJSONNode(t *testing.T, schema string) domain.NodeInstance {
	t.Helper()
	cfg := domain.AgentNodeConfig{
		Provider: "openai",
		Model:    "gpt-4o",
		Task: domain.TaskConfig{
			TaskType:   domain.TaskTypeCustom,
			OutputType: domain.PortTypeJSON,
		},
	}
	if schema != "" {
		raw := json.RawMessage(schema)
		cfg.Task.OutputSchema = &raw
	}
	bs, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal cfg: %v", err)
	}
	return domain.NodeInstance{
		Type:   domain.NodeTypeAgent,
		Config: bs,
		Ports: []domain.Port{
			{ID: "out", Direction: domain.PortOutput, Type: domain.PortTypeJSON},
		},
	}
}

func jsonString(s string) json.RawMessage {
	bs, _ := json.Marshal(s)
	return bs
}

func TestValidateOutput_PassesForURLRefs(t *testing.T) {
	t.Parallel()
	node := mediaNode(t, domain.PortTypeImage)
	if got := ValidateOutput(node, jsonString("clotho://file/proj/img.png")); got != nil {
		t.Errorf("clotho:// ref should pass, got %+v", got)
	}
	if got := ValidateOutput(node, jsonString("https://cdn.example.com/x.png")); got != nil {
		t.Errorf("https:// ref should pass, got %+v", got)
	}
}

func TestValidateOutput_FailsForTextInMediaPort(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		port domain.PortType
	}{
		{"image", domain.PortTypeImage},
		{"audio", domain.PortTypeAudio},
		{"video", domain.PortTypeVideo},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := mediaNode(t, tc.port)
			got := ValidateOutput(node, jsonString("Sorry, I can't generate that."))
			if got == nil {
				t.Fatalf("text in %s port should fail", tc.port)
			}
			if got.Class != domain.FailureOutputShape {
				t.Errorf("class = %q, want output_shape", got.Class)
			}
			if !strings.Contains(got.Message, "text/plain") {
				t.Errorf("message should explain text/plain mismatch: %q", got.Message)
			}
		})
	}
}

func TestValidateOutput_PassesForCorrectMediaDataURI(t *testing.T) {
	t.Parallel()
	node := mediaNode(t, domain.PortTypeImage)
	uri := "data:image/png;base64," + minimalPNGBase64
	if got := ValidateOutput(node, jsonString(uri)); got != nil {
		t.Errorf("PNG data URI in image port should pass, got %+v", got)
	}
}

func TestValidateOutput_FailsForMismatchedMediaFamily(t *testing.T) {
	t.Parallel()
	// PNG bytes in an audio port.
	node := mediaNode(t, domain.PortTypeAudio)
	uri := "data:image/png;base64," + minimalPNGBase64
	got := ValidateOutput(node, jsonString(uri))
	if got == nil {
		t.Fatal("PNG in audio port should fail")
	}
	if got.Class != domain.FailureOutputShape {
		t.Errorf("class = %q, want output_shape", got.Class)
	}
	if !strings.Contains(got.Message, "audio") {
		t.Errorf("message should mention expected family: %q", got.Message)
	}
}

func TestValidateOutput_DetectsAudioFromBytes(t *testing.T) {
	t.Parallel()
	node := mediaNode(t, domain.PortTypeAudio)
	uri := "data:audio/mpeg;base64," + base64.StdEncoding.EncodeToString(mp3Header)
	if got := ValidateOutput(node, jsonString(uri)); got != nil {
		t.Errorf("MP3 data URI in audio port should pass, got %+v", got)
	}
}

func TestValidateOutput_DetectsVideoFromBytes(t *testing.T) {
	t.Parallel()
	node := mediaNode(t, domain.PortTypeVideo)
	uri := "data:video/mp4;base64," + base64.StdEncoding.EncodeToString(mp4Header)
	if got := ValidateOutput(node, jsonString(uri)); got != nil {
		t.Errorf("MP4 data URI in video port should pass, got %+v", got)
	}
}

func TestValidateOutput_JSONSchemaPasses(t *testing.T) {
	t.Parallel()
	schema := `{
		"type": "object",
		"properties": {
			"title": {"type": "string"},
			"tags":  {"type": "array", "items": {"type": "string"}}
		},
		"required": ["title", "tags"]
	}`
	node := agentJSONNode(t, schema)
	output := json.RawMessage(`{"title":"Hello","tags":["a","b"]}`)
	if got := ValidateOutput(node, output); got != nil {
		t.Errorf("conforming object should pass, got %+v", got)
	}
}

func TestValidateOutput_JSONSchemaFailsOnMissingField(t *testing.T) {
	t.Parallel()
	schema := `{
		"type": "object",
		"properties": {"title": {"type": "string"}},
		"required": ["title"]
	}`
	node := agentJSONNode(t, schema)
	output := json.RawMessage(`{"different":"field"}`)
	got := ValidateOutput(node, output)
	if got == nil {
		t.Fatal("missing required field should fail")
	}
	if got.Class != domain.FailureValidation {
		t.Errorf("class = %q, want validation", got.Class)
	}
	if !got.Retryable {
		t.Errorf("schema mismatch should be retryable (LLM may comply on retry)")
	}
}

func TestValidateOutput_JSONSchemaFailsOnInvalidJSON(t *testing.T) {
	t.Parallel()
	schema := `{"type":"object"}`
	node := agentJSONNode(t, schema)
	output := json.RawMessage(`not actually json {`)
	got := ValidateOutput(node, output)
	if got == nil {
		t.Fatal("invalid JSON should fail")
	}
	if got.Class != domain.FailureValidation {
		t.Errorf("class = %q, want validation", got.Class)
	}
}

func TestValidateOutput_NoSchemaIsAcceptable(t *testing.T) {
	t.Parallel()
	node := agentJSONNode(t, "")
	output := json.RawMessage(`"any string is fine when no schema declared"`)
	if got := ValidateOutput(node, output); got != nil {
		t.Errorf("no schema = no validation, got %+v", got)
	}
}

func TestValidateOutput_TextPortIsAlwaysAcceptable(t *testing.T) {
	t.Parallel()
	node := domain.NodeInstance{
		Type: domain.NodeTypeAgent,
		Ports: []domain.Port{
			{ID: "out", Direction: domain.PortOutput, Type: domain.PortTypeText},
		},
	}
	if got := ValidateOutput(node, jsonString("anything goes")); got != nil {
		t.Errorf("text port should never fail validation here, got %+v", got)
	}
}
