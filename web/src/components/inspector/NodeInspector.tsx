import { usePipelineStore } from '../../stores/pipelineStore';
import { useExecutionStore } from '../../stores/executionStore';
import type { AgentNodeConfig, ToolNodeConfig, MediaNodeConfig } from '../../lib/types';
import { AgentInspector } from './AgentInspector';
import { ToolInspector } from './ToolInspector';
import { MediaInspector } from './MediaInspector';
import { ExecutionInspector } from './ExecutionInspector';

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function NodeInspector() {
  const selectedNodeId = usePipelineStore((s) => s.selectedNodeId);
  const nodes = usePipelineStore((s) => s.nodes);
  const executionStatus = useExecutionStore((s) => s.status);
  const stepResults = useExecutionStore((s) => s.stepResults);

  if (!selectedNodeId) return null;

  const node = nodes.find((n) => n.id === selectedNodeId);
  if (!node) return null;

  const step = stepResults.get(selectedNodeId);
  const showExecution =
    step && (executionStatus === 'running' || executionStatus === 'completed');

  return (
    <aside
      style={{
        width: 300,
        minWidth: 300,
        height: '100%',
        background: '#12131f',
        borderLeft: '1px solid #1e2030',
        overflowY: 'auto',
        padding: '14px',
      }}
    >
      <h3
        style={{
          fontSize: 12,
          fontWeight: 600,
          textTransform: 'uppercase',
          color: '#64748b',
          marginBottom: 14,
          letterSpacing: '0.04em',
        }}
      >
        Inspector
      </h3>

      {/* Execution overlay when running/completed */}
      {showExecution && step && (
        <div style={{ marginBottom: 16 }}>
          <ExecutionInspector step={step} />
          <hr
            style={{
              border: 'none',
              borderTop: '1px solid #1e2030',
              marginTop: 14,
            }}
          />
        </div>
      )}

      {/* Node configuration */}
      {node.data.nodeType === 'agent' ? (
        <AgentInspector
          nodeId={node.id}
          label={node.data.label}
          config={node.data.config as AgentNodeConfig}
        />
      ) : node.data.nodeType === 'media' ? (
        <MediaInspector
          nodeId={node.id}
          label={node.data.label}
          config={node.data.config as MediaNodeConfig}
        />
      ) : (
        <ToolInspector
          nodeId={node.id}
          label={node.data.label}
          config={node.data.config as ToolNodeConfig}
        />
      )}
    </aside>
  );
}
