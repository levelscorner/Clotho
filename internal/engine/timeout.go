package engine

import (
	"encoding/json"
	"time"

	"github.com/user/clotho/internal/domain"
)

// DefaultStepTimeout caps how long any single node may run before the
// engine cancels its context. Generous default since reasoning models
// (gemma4:26b, o3) can legitimately spend a minute thinking before the
// first visible token. Per-node overrides via AgentNodeConfig.StepTimeoutSec.
const DefaultStepTimeout = 120 * time.Second

// stepTimeoutFor returns the per-node timeout. Agent nodes may override
// the default via cfg.StepTimeoutSec; tool/media nodes get the default.
// Unmarshal errors fall back to the default rather than failing — the
// engine's normal config validation catches malformed configs elsewhere
// and we'd rather run the node than pre-fail it from the timeout helper.
func stepTimeoutFor(node domain.NodeInstance) time.Duration {
	if node.Type != domain.NodeTypeAgent {
		return DefaultStepTimeout
	}
	var cfg domain.AgentNodeConfig
	if err := json.Unmarshal(node.Config, &cfg); err != nil {
		return DefaultStepTimeout
	}
	if cfg.StepTimeoutSec != nil && *cfg.StepTimeoutSec > 0 {
		return time.Duration(*cfg.StepTimeoutSec) * time.Second
	}
	return DefaultStepTimeout
}
