import React, { useCallback, useMemo, type DragEvent } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  type NodeTypes,
  type Node,
  type Viewport,
} from '@xyflow/react';
import type {
  AgentNodeConfig,
  ToolNodeConfig,
  MediaNodeConfig,
  Port,
  PipelineNodeData,
  NodeType,
} from '../../lib/types';
import { usePipelineStore } from '../../stores/pipelineStore';
import {
  useExecutionStore,
  computeBlockedNodeIds,
} from '../../stores/executionStore';
import { AgentNode } from './nodes/AgentNode';
import { ToolNode } from './nodes/ToolNode';
import { MediaNode } from './nodes/MediaNode';
import { EmptyCanvasState } from './EmptyCanvasState';
import { CompletionToast } from './CompletionToast';

// ---------------------------------------------------------------------------
// Node type registry -- defined OUTSIDE the component to avoid recreation
// ---------------------------------------------------------------------------

const nodeTypes: NodeTypes = {
  agentNode: AgentNode,
  toolNode: ToolNode,
  mediaNode: MediaNode,
};

// ---------------------------------------------------------------------------
// DnD transfer data shape
// ---------------------------------------------------------------------------

interface DragPayload {
  nodeType: NodeType;
  config: AgentNodeConfig | ToolNodeConfig | MediaNodeConfig;
  ports: Port[];
  label?: string;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function PipelineCanvas() {
  const nodes = usePipelineStore((s) => s.nodes);
  const edges = usePipelineStore((s) => s.edges);
  const stepResults = useExecutionStore((s) => s.stepResults);

  // Blocked node ids: downstream of any failed step result.
  const blockedNodeIds = useMemo(
    () => computeBlockedNodeIds(stepResults, edges),
    [stepResults, edges],
  );

  // Inject .clotho-node-blocked class on any node flagged as downstream-blocked.
  // This lets nodes-states.css style them without touching BaseNode/AgentNode.
  const decoratedNodes = useMemo(() => {
    if (blockedNodeIds.size === 0) return nodes;
    return nodes.map((n) => {
      const isBlocked = blockedNodeIds.has(n.id);
      const baseClass = n.className ?? '';
      const hasBlocked = baseClass.split(/\s+/).includes('clotho-node-blocked');
      if (isBlocked && !hasBlocked) {
        return {
          ...n,
          className: `${baseClass} clotho-node-blocked`.trim(),
        };
      }
      if (!isBlocked && hasBlocked) {
        return {
          ...n,
          className: baseClass
            .split(/\s+/)
            .filter((c) => c !== 'clotho-node-blocked')
            .join(' '),
        };
      }
      return n;
    });
  }, [nodes, blockedNodeIds]);
  const onNodesChange = usePipelineStore((s) => s.onNodesChange);
  const onEdgesChange = usePipelineStore((s) => s.onEdgesChange);
  const onConnect = usePipelineStore((s) => s.onConnect);
  const addNode = usePipelineStore((s) => s.addNode);
  const removeNodes = usePipelineStore((s) => s.removeNodes);
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);

  const onNodesDelete = useCallback(
    (deleted: Node<PipelineNodeData>[]) => {
      if (deleted.length === 0) return;
      removeNodes(deleted.map((n) => n.id));
    },
    [removeNodes],
  );
  const onViewportChange = usePipelineStore((s) => s.onViewportChange);
  const storedViewport = usePipelineStore((s) => s.viewport);

  const onMoveEnd = useCallback(
    (_event: unknown, viewport: Viewport) => {
      onViewportChange({ x: viewport.x, y: viewport.y, zoom: viewport.zoom });
    },
    [onViewportChange],
  );

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, [setSelectedNode]);

  const onDragOver = useCallback((event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      event.preventDefault();

      const raw = event.dataTransfer.getData('application/clotho-node');
      if (!raw) return;

      let payload: DragPayload;
      try {
        payload = JSON.parse(raw) as DragPayload;
      } catch {
        return;
      }

      const bounds = (event.target as HTMLElement)
        .closest('.react-flow')
        ?.getBoundingClientRect();

      const position = {
        x: bounds ? event.clientX - bounds.left : event.clientX,
        y: bounds ? event.clientY - bounds.top : event.clientY,
      };

      addNode(
        payload.nodeType,
        position,
        payload.config,
        payload.ports,
        payload.label,
      );
    },
    [addNode],
  );

  const onNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node<PipelineNodeData>) => {
      setSelectedNode(node.id);
    },
    [setSelectedNode],
  );

  const defaultEdgeOptions = useMemo(
    () => ({
      style: { stroke: '#475569', strokeWidth: 2 },
      type: 'smoothstep' as const,
    }),
    [],
  );

  return (
    <div style={{ flex: 1, height: '100%', position: 'relative' }}>
      <ReactFlow<Node<PipelineNodeData>>
        nodes={decoratedNodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodesDelete={onNodesDelete}
        deleteKeyCode={['Backspace', 'Delete']}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        onMoveEnd={onMoveEnd}
        onDragOver={onDragOver}
        onDrop={onDrop}
        nodeTypes={nodeTypes}
        defaultEdgeOptions={defaultEdgeOptions}
        defaultViewport={storedViewport}
        fitView
        proOptions={{ hideAttribution: true }}
      >
        <Background color="var(--surface-border)" gap={24} size={1} />
        <Controls className="clotho-controls" />
      </ReactFlow>
      <EmptyCanvasState />
      <CompletionToast />
    </div>
  );
}
