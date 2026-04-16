package domain

import "encoding/json"

// NodeType discriminates node kinds on the canvas.
type NodeType string

const (
	NodeTypeAgent NodeType = "agent"
	NodeTypeTool  NodeType = "tool"
	NodeTypeMedia NodeType = "media"
)

// PortType defines the data type flowing through a connection.
type PortType string

const (
	PortTypeText        PortType = "text"
	PortTypeImagePrompt PortType = "image_prompt"
	PortTypeVideoPrompt PortType = "video_prompt"
	PortTypeAudioPrompt PortType = "audio_prompt"
	PortTypeImage       PortType = "image"
	PortTypeVideo       PortType = "video"
	PortTypeAudio       PortType = "audio"
	PortTypeJSON        PortType = "json"
	PortTypeAny         PortType = "any"
)

// PortDirection indicates whether a port is an input or output.
type PortDirection string

const (
	PortInput  PortDirection = "input"
	PortOutput PortDirection = "output"
)

// Port is a typed connection point on a node.
type Port struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Type      PortType      `json:"type"`
	Direction PortDirection `json:"direction"`
	Required  bool          `json:"required"`
}

// Position is an (x, y) coordinate on the canvas.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// OnFailurePolicy controls what happens when a node's execution fails.
//
//   "abort"    — default. Engine stops the whole execution.
//   "skip"     — emit step_failed but treat downstream inputs as null
//                and keep going. Useful in fan-out pipelines where one
//                branch failing shouldn't kill the rest.
//   "continue" — same as skip, but downstream sees the StepFailure JSON
//                as the input value so an error-handler agent can react.
type OnFailurePolicy string

const (
	OnFailureAbort    OnFailurePolicy = "abort"
	OnFailureSkip     OnFailurePolicy = "skip"
	OnFailureContinue OnFailurePolicy = "continue"
)

// NodeInstance is a placed node in a pipeline graph.
//
// Pinned + PinnedOutput let creators freeze a node's output across
// re-runs so iterating on a downstream prompt doesn't re-call (and
// re-pay for) every upstream agent. Engine consults Pinned BEFORE
// dispatching to the executor — see engine.executeNode.
//
// OnFailure controls failure propagation per node and falls back to
// OnFailureAbort when empty for back-compat with existing graphs.
type NodeInstance struct {
	ID           string          `json:"id"`
	Type         NodeType        `json:"type"`
	Label        string          `json:"label"`
	Position     Position        `json:"position"`
	Ports        []Port          `json:"ports"`
	Config       json.RawMessage `json:"config"` // AgentNodeConfig or ToolNodeConfig
	Pinned       bool            `json:"pinned,omitempty"`
	PinnedOutput json.RawMessage `json:"pinned_output,omitempty"`
	OnFailure    OnFailurePolicy `json:"on_failure,omitempty"`
}
