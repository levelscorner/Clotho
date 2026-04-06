package domain

import "testing"

func TestDefaultToolPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		toolType ToolType
		wantType PortType
		wantName string
	}{
		{"text_box returns text output", ToolTypeTextBox, PortTypeText, "Text"},
		{"image_box returns image output", ToolTypeImageBox, PortTypeImage, "Image"},
		{"video_box returns video output", ToolTypeVideoBox, PortTypeVideo, "Video"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ports := DefaultToolPorts(tt.toolType)

			if len(ports) != 1 {
				t.Fatalf("expected 1 port, got %d", len(ports))
			}

			p := ports[0]
			if p.ID != "out" {
				t.Errorf("port ID = %q, want %q", p.ID, "out")
			}
			if p.Direction != PortOutput {
				t.Errorf("port Direction = %q, want %q", p.Direction, PortOutput)
			}
			if p.Type != tt.wantType {
				t.Errorf("port Type = %q, want %q", p.Type, tt.wantType)
			}
			if p.Name != tt.wantName {
				t.Errorf("port Name = %q, want %q", p.Name, tt.wantName)
			}
		})
	}

	t.Run("unknown tool type defaults to text", func(t *testing.T) {
		t.Parallel()
		ports := DefaultToolPorts(ToolType("unknown"))
		if len(ports) != 1 {
			t.Fatalf("expected 1 port, got %d", len(ports))
		}
		if ports[0].Type != PortTypeText {
			t.Errorf("port Type = %q, want %q", ports[0].Type, PortTypeText)
		}
	})
}
