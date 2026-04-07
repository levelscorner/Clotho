package media

import "context"

// MediaRequest contains parameters for a media generation request.
type MediaRequest struct {
	Model       string            // model identifier
	Prompt      string            // text prompt
	ImageURL    string            // reference image URL (optional)
	AspectRatio string            // e.g. "16:9"
	Voice       string            // TTS voice name
	Duration    int               // max duration seconds
	NumOutputs  int               // number of outputs
	Extra       map[string]string // provider-specific params
}

// MediaStatus is the current state of a media generation job.
type MediaStatus struct {
	State  string // pending, processing, succeeded, failed, cancelled
	Output string // result URL or base64 data
	Error  string // error message if failed
}

// Provider generates media (images, video, audio) via async or sync APIs.
type Provider interface {
	// Submit starts a media generation job and returns a job ID.
	// For synchronous providers, this completes immediately and returns a synthetic job ID.
	Submit(ctx context.Context, req MediaRequest) (jobID string, err error)

	// Poll checks the status of a submitted job.
	Poll(ctx context.Context, jobID string) (MediaStatus, error)
}
