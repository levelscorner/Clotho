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
import { AgentNode } from './nodes/AgentNode';
import { ToolNode } from './nodes/ToolNode';
import { MediaNode } from './nodes/MediaNode';

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
  const onNodesChange = usePipelineStore((s) => s.onNodesChange);
  const onEdgesChange = usePipelineStore((s) => s.onEdgesChange);
  const onConnect = usePipelineStore((s) => s.onConnect);
  const addNode = usePipelineStore((s) => s.addNode);
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);
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
    <div style={{ flex: 1, height: '100%' }}>
      <ReactFlow<Node<PipelineNodeData>>
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
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
    </div>
  );
}
