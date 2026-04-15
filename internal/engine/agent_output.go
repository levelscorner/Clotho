package engine

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/user/clotho/internal/domain"
	"github.com/user/clotho/internal/storage"
)

// writeAgentOutputFile persists an agent node's text output as a .txt file
// alongside manifest.json and any media assets, so users can browse the
// whole run in Finder and open their generated scripts/prompts without
// going through the DB.
//
// Filename uses the slugified node label (falling back to the node ID),
// suffixed with .txt. Writes go through the same storage.Store that media
// providers use — Location scoping is identical, so everything lands
// under {DataDir}/{project}/{pipeline}/{execution}/.
//
// Non-fatal: any write failure is logged at warn level and returns an
// empty string. The execution itself is unaffected because step_results
// already has the authoritative inline output.
func writeAgentOutputFile(
	ctx context.Context,
	store storage.Store,
	loc storage.Location,
	node domain.NodeInstance,
	data json.RawMessage,
) string {
	if store == nil || len(data) == 0 {
		return ""
	}

	// Agent outputs are JSON-encoded strings (the executor Marshals
	// strconv'd text). If decode fails we're either dealing with a
	// structured output or a non-agent caller — bail quietly.
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return ""
	}
	if strings.TrimSpace(text) == "" {
		return ""
	}
	// If the data turned out to be a file reference already, skip — media
	// nodes write their own files elsewhere; we only want to fill the gap
	// for agent text output.
	if strings.HasPrefix(text, "clotho://") {
		return ""
	}

	name := storage.Slugify(node.Label)
	if name == "" {
		name = node.ID
	}
	filename := name + ".txt"

	rel, _, err := store.Write(ctx, loc, filename, []byte(text))
	if err != nil {
		slog.Warn("agent output file write failed",
			"node_id", node.ID,
			"filename", filename,
			"error", err)
		return ""
	}
	return rel
}

// deriveOutputFileURL produces the clotho://file/{rel} URL the frontend
// uses to reveal a node's artifact in Finder — or an empty string when
// the node has nothing on disk.
//
//   - Agent: the caller hands in the rel path returned by
//     writeAgentOutputFile and this function minks the URL.
//   - Media: the executor has already stored a clotho://file/ URL as
//     the node's output; we extract and return it unchanged.
//   - Tool and everything else: empty string — no on-disk artifact.
func deriveOutputFileURL(nodeType domain.NodeType, outputData json.RawMessage, agentRel string) string {
	switch nodeType {
	case domain.NodeTypeAgent:
		if agentRel == "" {
			return ""
		}
		return "clotho://file/" + agentRel
	case domain.NodeTypeMedia:
		if len(outputData) == 0 {
			return ""
		}
		var s string
		if err := json.Unmarshal(outputData, &s); err != nil {
			return ""
		}
		if strings.HasPrefix(s, "clotho://file/") {
			return s
		}
		return ""
	default:
		return ""
	}
}
