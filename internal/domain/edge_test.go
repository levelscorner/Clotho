package domain

import "testing"

func TestCanConnect(t *testing.T) {
	t.Parallel()

	allTypes := []PortType{
		PortTypeText, PortTypeImagePrompt, PortTypeVideoPrompt,
		PortTypeAudioPrompt, PortTypeImage, PortTypeVideo,
		PortTypeAudio, PortTypeJSON, PortTypeAny,
	}

	tests := []struct {
		name string
		src  PortType
		tgt  PortType
		want bool
	}{
		// Specified cases
		{"text to text", PortTypeText, PortTypeText, true},
		{"text to any", PortTypeText, PortTypeAny, true},
		{"image_prompt to text (subtype)", PortTypeImagePrompt, PortTypeText, true},
		{"image_prompt to image_prompt", PortTypeImagePrompt, PortTypeImagePrompt, true},
		{"image to text (strict)", PortTypeImage, PortTypeText, false},
		{"video to audio", PortTypeVideo, PortTypeAudio, false},
		{"any to text (wildcard)", PortTypeAny, PortTypeText, true},
		{"any to any", PortTypeAny, PortTypeAny, true},
		{"unknown type to text", PortType("unknown"), PortTypeText, false},

		// Additional cross-type incompatibilities
		{"text to image", PortTypeText, PortTypeImage, false},
		{"image to video", PortTypeImage, PortTypeVideo, false},
		{"audio to json", PortTypeAudio, PortTypeJSON, false},
		{"json to text", PortTypeJSON, PortTypeText, false},

		// Prompt subtypes connect to text
		{"video_prompt to text", PortTypeVideoPrompt, PortTypeText, true},
		{"audio_prompt to text", PortTypeAudioPrompt, PortTypeText, true},

		// All types connect to any
		{"image to any", PortTypeImage, PortTypeAny, true},
		{"video to any", PortTypeVideo, PortTypeAny, true},
		{"audio to any", PortTypeAudio, PortTypeAny, true},
		{"json to any", PortTypeJSON, PortTypeAny, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := CanConnect(tt.src, tt.tgt)
			if got != tt.want {
				t.Errorf("CanConnect(%q, %q) = %v, want %v", tt.src, tt.tgt, got, tt.want)
			}
		})
	}

	// All 9 types to themselves should be true
	for _, pt := range allTypes {
		pt := pt
		t.Run("self_"+string(pt), func(t *testing.T) {
			t.Parallel()
			if !CanConnect(pt, pt) {
				t.Errorf("CanConnect(%q, %q) = false, want true (self-connection)", pt, pt)
			}
		})
	}
}
