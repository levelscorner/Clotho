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
var compatibility = map[PortType]map[PortType]bool{
	PortTypeText:        {PortTypeText: true, PortTypeAny: true},
	PortTypeImagePrompt: {PortTypeImagePrompt: true, PortTypeText: true, PortTypeAny: true},
	PortTypeVideoPrompt: {PortTypeVideoPrompt: true, PortTypeText: true, PortTypeAny: true},
	PortTypeAudioPrompt: {PortTypeAudioPrompt: true, PortTypeText: true, PortTypeAny: true},
	PortTypeImage:       {PortTypeImage: true, PortTypeAny: true},
	PortTypeVideo:       {PortTypeVideo: true, PortTypeAny: true},
	PortTypeAudio:       {PortTypeAudio: true, PortTypeAny: true},
	PortTypeJSON:        {PortTypeJSON: true, PortTypeAny: true},
	PortTypeAny:         {PortTypeText: true, PortTypeImagePrompt: true, PortTypeVideoPrompt: true, PortTypeAudioPrompt: true, PortTypeImage: true, PortTypeVideo: true, PortTypeAudio: true, PortTypeJSON: true, PortTypeAny: true},
}

// CanConnect checks if a source port type is compatible with a target port type.
func CanConnect(src, tgt PortType) bool {
	targets, ok := compatibility[src]
	if !ok {
		return false
	}
	return targets[tgt]
}
