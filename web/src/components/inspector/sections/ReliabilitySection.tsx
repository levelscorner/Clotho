import type { OnFailurePolicy } from '../../../lib/types';
import { usePipelineStore } from '../../../stores/pipelineStore';
import { InspectorGroup } from '../InspectorGroup';
import { fieldGroup, helperTextStyle, inputStyle, labelStyle } from './sectionStyles';

interface ReliabilitySectionProps {
  nodeId: string;
}

/**
 * Per-node reliability controls. Pin freezes the node's last output so
 * downstream iteration doesn't re-call (and re-pay for) upstream agents.
 * On-failure picks the propagation policy when the node errors.
 *
 * Pinned/PinnedOutput live on the NodeInstance directly (not in the
 * config blob) since pinning is universal across node types. We read
 * the node from pipelineStore and patch via setNodePin / setNodeOnFailure
 * helpers below.
 */
export function ReliabilitySection({ nodeId }: ReliabilitySectionProps) {
  const node = usePipelineStore((s) => s.nodes.find((n) => n.id === nodeId));
  const setNodePin = usePipelineStore((s) => s.setNodePin);
  const setNodeOnFailure = usePipelineStore((s) => s.setNodeOnFailure);

  if (!node) return null;

  const pinned = !!node.data.pinned;
  const onFailure: OnFailurePolicy = node.data.onFailure ?? 'abort';
  const hasCachedOutput =
    node.data.pinnedOutput !== undefined && node.data.pinnedOutput !== null;

  return (
    <InspectorGroup title="Reliability">
      <div style={fieldGroup}>
        <label style={{ ...labelStyle, display: 'flex', alignItems: 'center', gap: 8, textTransform: 'none', letterSpacing: 0, fontSize: 12 }}>
          <input
            type="checkbox"
            checked={pinned}
            onChange={(e) => setNodePin(nodeId, e.target.checked)}
            disabled={!hasCachedOutput && !pinned}
            style={{ margin: 0 }}
          />
          <span>Pin output across runs</span>
        </label>
        <div style={helperTextStyle}>
          {hasCachedOutput
            ? 'Engine skips this node and serves the cached output. Save $$ while iterating downstream.'
            : 'Run the pipeline once first — the most recent successful output is what gets pinned.'}
        </div>
      </div>

      <div style={fieldGroup}>
        <label style={labelStyle}>On Failure</label>
        <select
          style={inputStyle}
          value={onFailure}
          onChange={(e) => setNodeOnFailure(nodeId, e.target.value as OnFailurePolicy)}
        >
          <option value="abort">Abort — stop the whole execution (default)</option>
          <option value="skip">Skip — record failure, downstream sees no input</option>
          <option value="continue">Continue — downstream receives the failure JSON</option>
        </select>
        <div style={helperTextStyle}>
          {onFailure === 'abort' && 'Hard stop: any downstream nodes are cancelled.'}
          {onFailure === 'skip' && 'Useful in fan-out pipelines where one branch can flake without blocking the rest.'}
          {onFailure === 'continue' && 'Useful when a downstream agent is your error-handler.'}
        </div>
      </div>
    </InspectorGroup>
  );
}
