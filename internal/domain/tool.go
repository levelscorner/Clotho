package domain

// ToolType identifies the kind of non-AI data source.
type ToolType string

const (
	ToolTypeTextBox  ToolType = "text_box"
	ToolTypeImageBox ToolType = "image_box"
	ToolTypeVideoBox ToolType = "video_box"
)

// ToolNodeConfig is the configuration for a tool node.
// Stored as JSON in NodeInstance.Config.
type ToolNodeConfig struct {
	ToolType ToolType `json:"tool_type"`
	Content  string   `json:"content,omitempty"`
	MediaURL string   `json:"media_url,omitempty"`
}

// DefaultToolPorts returns the output port for a tool node based on its type.
func DefaultToolPorts(tt ToolType) []Port {
	switch tt {
	case ToolTypeImageBox:
		return []Port{
			{ID: "out", Name: "Image", Type: PortTypeImage, Direction: PortOutput},
		}
	case ToolTypeVideoBox:
		return []Port{
			{ID: "out", Name: "Video", Type: PortTypeVideo, Direction: PortOutput},
		}
	default: // text_box
		return []Port{
			{ID: "out", Name: "Text", Type: PortTypeText, Direction: PortOutput},
		}
	}
}
