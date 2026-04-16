package domain

import "testing"

// TestCanConnect_81CellMatrix enumerates every (source, target) port-type
// pair against the compatibility matrix locked in
// docs/PIPELINE-PATTERNS.md §1. If an engine refactor changes
// domain.CanConnect, this test fails loudly and the doc must be updated
// together with the behaviour.
//
// The matrix encodes the rules:
//   - Text family (text, image_prompt, video_prompt, audio_prompt) is
//     one interchangeable group — every pair within the family connects.
//   - Media outputs are hermetic (image → video is rejected).
//   - json pairs only with itself or any.
//   - any accepts everything; nothing but any → X has any as source in
//     runtime graphs (Media's ref input uses any as a target).
func TestCanConnect_81CellMatrix(t *testing.T) {
	// Column / row order must match the doc's matrix for easy review.
	types := []PortType{
		PortTypeText,
		PortTypeImagePrompt,
		PortTypeVideoPrompt,
		PortTypeAudioPrompt,
		PortTypeImage,
		PortTypeVideo,
		PortTypeAudio,
		PortTypeJSON,
		PortTypeAny,
	}

	// expected[src][tgt] — true if src → tgt is allowed.
	// ALL 81 CELLS below; do NOT rely on defaults. If this layout goes
	// stale, compat rules have drifted.
	expected := map[PortType]map[PortType]bool{
		PortTypeText: {
			PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
			PortTypeAudioPrompt: true, PortTypeImage: false, PortTypeVideo: false,
			PortTypeAudio: false, PortTypeJSON: false, PortTypeAny: true,
		},
		PortTypeImagePrompt: {
			PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
			PortTypeAudioPrompt: true, PortTypeImage: false, PortTypeVideo: false,
			PortTypeAudio: false, PortTypeJSON: false, PortTypeAny: true,
		},
		PortTypeVideoPrompt: {
			PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
			PortTypeAudioPrompt: true, PortTypeImage: false, PortTypeVideo: false,
			PortTypeAudio: false, PortTypeJSON: false, PortTypeAny: true,
		},
		PortTypeAudioPrompt: {
			PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
			PortTypeAudioPrompt: true, PortTypeImage: false, PortTypeVideo: false,
			PortTypeAudio: false, PortTypeJSON: false, PortTypeAny: true,
		},
		PortTypeImage: {
			PortTypeText: false, PortTypeImagePrompt: false, PortTypeVideoPrompt: false,
			PortTypeAudioPrompt: false, PortTypeImage: true, PortTypeVideo: false,
			PortTypeAudio: false, PortTypeJSON: false, PortTypeAny: true,
		},
		PortTypeVideo: {
			PortTypeText: false, PortTypeImagePrompt: false, PortTypeVideoPrompt: false,
			PortTypeAudioPrompt: false, PortTypeImage: false, PortTypeVideo: true,
			PortTypeAudio: false, PortTypeJSON: false, PortTypeAny: true,
		},
		PortTypeAudio: {
			PortTypeText: false, PortTypeImagePrompt: false, PortTypeVideoPrompt: false,
			PortTypeAudioPrompt: false, PortTypeImage: false, PortTypeVideo: false,
			PortTypeAudio: true, PortTypeJSON: false, PortTypeAny: true,
		},
		PortTypeJSON: {
			PortTypeText: false, PortTypeImagePrompt: false, PortTypeVideoPrompt: false,
			PortTypeAudioPrompt: false, PortTypeImage: false, PortTypeVideo: false,
			PortTypeAudio: false, PortTypeJSON: true, PortTypeAny: true,
		},
		PortTypeAny: {
			PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
			PortTypeAudioPrompt: true, PortTypeImage: true, PortTypeVideo: true,
			PortTypeAudio: true, PortTypeJSON: true, PortTypeAny: true,
		},
	}

	// Sanity: the table covers every combination exactly once.
	if len(expected) != 9 {
		t.Fatalf("expected table rows = %d, want 9", len(expected))
	}
	for src, row := range expected {
		if len(row) != 9 {
			t.Fatalf("row %q has %d cols, want 9", src, len(row))
		}
	}

	// Exercise every cell.
	for _, src := range types {
		for _, tgt := range types {
			want := expected[src][tgt]
			got := CanConnect(src, tgt)
			if got != want {
				t.Errorf("CanConnect(%q → %q) = %v, want %v", src, tgt, got, want)
			}
		}
	}
}

// TestCanConnect_KeyInvariants spells out the rules a reader cares about
// more readable-language than the matrix. Fails if any invariant drifts.
func TestCanConnect_KeyInvariants(t *testing.T) {
	t.Run("text family is fully interchangeable", func(t *testing.T) {
		family := []PortType{
			PortTypeText, PortTypeImagePrompt, PortTypeVideoPrompt, PortTypeAudioPrompt,
		}
		for _, a := range family {
			for _, b := range family {
				if !CanConnect(a, b) {
					t.Errorf("%q → %q should be allowed (text family rule)", a, b)
				}
			}
		}
	})

	t.Run("media outputs are hermetic (no cross-media)", func(t *testing.T) {
		media := []PortType{PortTypeImage, PortTypeVideo, PortTypeAudio}
		for _, a := range media {
			for _, b := range media {
				if a == b {
					continue
				}
				if CanConnect(a, b) {
					t.Errorf("%q should NOT connect to %q (hermetic media rule)", a, b)
				}
			}
		}
	})

	t.Run("any accepts every source", func(t *testing.T) {
		all := []PortType{
			PortTypeText, PortTypeImagePrompt, PortTypeVideoPrompt,
			PortTypeAudioPrompt, PortTypeImage, PortTypeVideo,
			PortTypeAudio, PortTypeJSON, PortTypeAny,
		}
		for _, src := range all {
			if !CanConnect(src, PortTypeAny) {
				t.Errorf("%q → any should be allowed (universal sink)", src)
			}
		}
	})

	t.Run("json pairs only with itself or any", func(t *testing.T) {
		nonJSON := []PortType{
			PortTypeText, PortTypeImagePrompt, PortTypeVideoPrompt,
			PortTypeAudioPrompt, PortTypeImage, PortTypeVideo, PortTypeAudio,
		}
		for _, tgt := range nonJSON {
			if CanConnect(PortTypeJSON, tgt) {
				t.Errorf("json → %q should NOT be allowed (json is pair-only)", tgt)
			}
		}
	})
}
