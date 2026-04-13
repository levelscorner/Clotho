package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaultAgentPorts(t *testing.T) {
	t.Parallel()

	t.Run("text output type", func(t *testing.T) {
		t.Parallel()
		ports := DefaultAgentPorts(PortTypeText)

		if len(ports) != 2 {
			t.Fatalf("expected 2 ports, got %d", len(ports))
		}

		in := ports[0]
		if in.ID != "in" {
			t.Errorf("input port ID = %q, want %q", in.ID, "in")
		}
		if in.Type != PortTypeText {
			t.Errorf("input port Type = %q, want %q", in.Type, PortTypeText)
		}
		if in.Direction != PortInput {
			t.Errorf("input port Direction = %q, want %q", in.Direction, PortInput)
		}

		out := ports[1]
		if out.ID != "out" {
			t.Errorf("output port ID = %q, want %q", out.ID, "out")
		}
		if out.Type != PortTypeText {
			t.Errorf("output port Type = %q, want %q", out.Type, PortTypeText)
		}
		if out.Direction != PortOutput {
			t.Errorf("output port Direction = %q, want %q", out.Direction, PortOutput)
		}
	})

	t.Run("image_prompt output type", func(t *testing.T) {
		t.Parallel()
		ports := DefaultAgentPorts(PortTypeImagePrompt)

		if len(ports) != 2 {
			t.Fatalf("expected 2 ports, got %d", len(ports))
		}

		out := ports[1]
		if out.Type != PortTypeImagePrompt {
			t.Errorf("output port Type = %q, want %q", out.Type, PortTypeImagePrompt)
		}
	})
}

func TestAgentNodeConfigPresetCategoryRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("round-trips preset_category when set", func(t *testing.T) {
		t.Parallel()

		original := AgentNodeConfig{
			Provider: "openai",
			Model:    "gpt-4o",
			Role: RoleConfig{
				SystemPrompt: "You are a screenwriter.",
				Persona:      "Screenwriter",
			},
			Task: TaskConfig{
				TaskType:   TaskTypeScript,
				OutputType: PortTypeText,
				Template:   "Write a script from: {{input}}",
			},
			Temperature:    0.8,
			MaxTokens:      4096,
			PresetCategory: "script",
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		if !strings.Contains(string(data), `"preset_category":"script"`) {
			t.Errorf("expected serialized JSON to contain preset_category=script, got: %s", data)
		}

		var decoded AgentNodeConfig
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.PresetCategory != "script" {
			t.Errorf("PresetCategory = %q, want %q", decoded.PresetCategory, "script")
		}
		if decoded.Provider != original.Provider {
			t.Errorf("Provider = %q, want %q", decoded.Provider, original.Provider)
		}
		if decoded.Task.TaskType != original.Task.TaskType {
			t.Errorf("Task.TaskType = %q, want %q", decoded.Task.TaskType, original.Task.TaskType)
		}
	})

	t.Run("omits preset_category when empty", func(t *testing.T) {
		t.Parallel()

		cfg := AgentNodeConfig{
			Provider: "openai",
			Model:    "gpt-4o",
			Role:     RoleConfig{SystemPrompt: "x", Persona: "y"},
			Task: TaskConfig{
				TaskType:   TaskTypeCustom,
				OutputType: PortTypeText,
				Template:   "{{input}}",
			},
			Temperature: 0.5,
			MaxTokens:   1024,
		}

		data, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		if strings.Contains(string(data), "preset_category") {
			t.Errorf("expected preset_category to be omitted when empty, got: %s", data)
		}
	})

	t.Run("accepts crafter category", func(t *testing.T) {
		t.Parallel()

		raw := `{"provider":"openai","model":"gpt-4o","role":{"system_prompt":"","persona":""},"task":{"task_type":"image_prompt","output_type":"image_prompt","template":""},"temperature":0.7,"max_tokens":1024,"preset_category":"crafter"}`

		var decoded AgentNodeConfig
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if decoded.PresetCategory != "crafter" {
			t.Errorf("PresetCategory = %q, want %q", decoded.PresetCategory, "crafter")
		}
	})
}
