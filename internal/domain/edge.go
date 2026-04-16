package domain

// Edge connects an output port of one node to an input port of another.
type Edge struct {
	ID         string `json:"id"`
	Source     string `json:"source"`      // source node ID
	SourcePort string `json:"source_port"` // source port ID
	Target     string `json:"target"`      // target node ID
	TargetPort string `json:"target_port"` // target port ID
}

// compatibility maps a source PortType to the set of target PortTypes it can connect to.
//
// Rules:
//   - Text family (text, image_prompt, video_prompt, audio_prompt) is a
//     single interchangeable group. All four carry strings; downstream
//     agents route behavior by the node's own task_type, not by the
//     incoming port specialization. This unblocks the canonical flow
//     "LLM writes a prompt → image/video/audio node generates media".
//   - Media outputs (image, video, audio) are hermetic. Cross-media
//     remixing is an explicit executor's job, never free piping.
//   - json is pair-only. It's a structured channel — connect json→json
//     or json→any. Callers that want to flatten JSON into text set the
//     upstream agent's output_type to text instead.
//   - any accepts everything; used for optional reference inputs on
//     Media nodes and the palette-default agent input.
var compatibility = map[PortType]map[PortType]bool{
	PortTypeText: {
		PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
		PortTypeAudioPrompt: true, PortTypeAny: true,
	},
	PortTypeImagePrompt: {
		PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
		PortTypeAudioPrompt: true, PortTypeAny: true,
	},
	PortTypeVideoPrompt: {
		PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
		PortTypeAudioPrompt: true, PortTypeAny: true,
	},
	PortTypeAudioPrompt: {
		PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
		PortTypeAudioPrompt: true, PortTypeAny: true,
	},
	PortTypeImage: {PortTypeImage: true, PortTypeAny: true},
	PortTypeVideo: {PortTypeVideo: true, PortTypeAny: true},
	PortTypeAudio: {PortTypeAudio: true, PortTypeAny: true},
	PortTypeJSON:  {PortTypeJSON: true, PortTypeAny: true},
	PortTypeAny: {
		PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true,
		PortTypeAudioPrompt: true, PortTypeImage: true, PortTypeVideo: true,
		PortTypeAudio: true, PortTypeJSON: true, PortTypeAny: true,
	},
}

// CanConnect checks if a source port type is compatible with a target port type.
func CanConnect(src, tgt PortType) bool {
	targets, ok := compatibility[src]
	if !ok {
		return false
	}
	return targets[tgt]
}
