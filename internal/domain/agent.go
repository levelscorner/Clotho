package domain

import "encoding/json"

// TaskType identifies what kind of creative output an agent produces.
type TaskType string

const (
	TaskTypeScript            TaskType = "script"
	TaskTypeImagePrompt       TaskType = "image_prompt"
	TaskTypeVideoPrompt       TaskType = "video_prompt"
	TaskTypeAudioPrompt       TaskType = "audio_prompt"
	TaskTypeCharacterPrompt   TaskType = "character_prompt"
	TaskTypeStory             TaskType = "story"
	TaskTypePromptEnhancement TaskType = "prompt_enhancement"
	TaskTypeStoryToPrompt     TaskType = "story_to_prompt"
	TaskTypeCustom            TaskType = "custom"
)

// RoleConfig defines the personality/job injected into an agent.
type RoleConfig struct {
	SystemPrompt string            `json:"system_prompt"`
	Persona      string            `json:"persona"`
	Variables    map[string]string `json:"variables,omitempty"`
}

// TaskConfig defines what the agent produces.
type TaskConfig struct {
	TaskType     TaskType         `json:"task_type"`
	OutputType   PortType         `json:"output_type"`
	Template     string           `json:"template"`
	OutputSchema *json.RawMessage `json:"output_schema,omitempty"`
}

// AgentNodeConfig is the configuration for an agent node.
// This is stored as JSON in NodeInstance.Config.
type AgentNodeConfig struct {
	Provider     string     `json:"provider"`
	Model        string     `json:"model"`
	Role         RoleConfig `json:"role"`
	Task         TaskConfig `json:"task"`
	Temperature  float64    `json:"temperature"`
	MaxTokens    int        `json:"max_tokens"`
	CostCap      *float64   `json:"cost_cap,omitempty"`
	CredentialID string     `json:"credential_id,omitempty"`
}

// DefaultAgentPorts returns the standard input/output ports for an agent node.
func DefaultAgentPorts(outputType PortType) []Port {
	return []Port{
		{ID: "in", Name: "Input", Type: PortTypeText, Direction: PortInput, Required: false},
		{ID: "out", Name: "Output", Type: outputType, Direction: PortOutput, Required: false},
	}
}
