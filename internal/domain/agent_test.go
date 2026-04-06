package domain

import "testing"

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
