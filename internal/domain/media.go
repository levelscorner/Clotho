package domain

// MediaType discriminates media generation kinds.
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
	MediaTypeAudio MediaType = "audio"
)

// MediaNodeConfig is the configuration for a media generator node.
type MediaNodeConfig struct {
	MediaType    MediaType `json:"media_type"`
	Provider     string    `json:"provider"`      // replicate, openai, elevenlabs
	Model        string    `json:"model"`          // e.g. "flux-1.1-pro", "dall-e-3", "tts-1"
	CredentialID string    `json:"credential_id"`  // optional, for user-provided API keys
	Prompt       string    `json:"prompt"`         // prompt template (supports {{input}})
	AspectRatio  string    `json:"aspect_ratio"`   // e.g. "16:9", "1:1" (image/video)
	Voice        string    `json:"voice"`          // TTS voice name (audio only)
	Duration     int       `json:"duration"`       // max duration in seconds (video/audio)
	NumOutputs   int       `json:"num_outputs"`    // number of images/clips to generate
	CostCap      *float64  `json:"cost_cap"`       // optional cost cap per execution
}

// DefaultMediaPorts returns the default ports for a media node.
func DefaultMediaPorts(mt MediaType) []Port {
	inputType := PortTypeText
	switch mt {
	case MediaTypeImage:
		inputType = PortTypeImagePrompt
	case MediaTypeVideo:
		inputType = PortTypeVideoPrompt
	case MediaTypeAudio:
		inputType = PortTypeAudioPrompt
	}

	outputType := PortTypeImage
	switch mt {
	case MediaTypeVideo:
		outputType = PortTypeVideo
	case MediaTypeAudio:
		outputType = PortTypeAudio
	}

	return []Port{
		{ID: "in_prompt", Name: "Prompt", Type: inputType, Direction: PortInput, Required: true},
		{ID: "in_ref", Name: "Reference", Type: PortTypeAny, Direction: PortInput, Required: false},
		{ID: "out_media", Name: "Output", Type: outputType, Direction: PortOutput, Required: false},
	}
}
