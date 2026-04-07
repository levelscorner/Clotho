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

// NodeInstance is a placed node in a pipeline graph.
type NodeInstance struct {
	ID       string          `json:"id"`
	Type     NodeType        `json:"type"`
	Label    string          `json:"label"`
	Position Position        `json:"position"`
	Ports    []Port          `json:"ports"`
	Config   json.RawMessage `json:"config"` // AgentNodeConfig or ToolNodeConfig
}
