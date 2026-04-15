import React, { useCallback, useMemo, type DragEvent } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  useReactFlow,
  type NodeTypes,
  type Node,
  type NodeChange,
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
  const lockedNodes = usePipelineStore((s) => s.lockedNodes);
  const stepResults = useExecutionStore((s) => s.stepResults);

  // Blocked node ids: downstream of any failed step result.
  const blockedNodeIds = useMemo(
    () => computeBlockedNodeIds(stepResults, edges),
    [stepResults, edges],
  );

  // Inject .clotho-node-blocked class on any node flagged as downstream-blocked.
  // Also stamp draggable:false / deletable:false on locked nodes so React Flow
  // blocks drag + native-delete interactions at the framework level.
  const decoratedNodes = useMemo(() => {
    if (blockedNodeIds.size === 0 && lockedNodes.size === 0) return nodes;
    return nodes.map((n) => {
      const isBlocked = blockedNodeIds.has(n.id);
      const isLocked = lockedNodes.has(n.id);
      const baseClass = n.className ?? '';
      const hasBlocked = baseClass.split(/\s+/).includes('clotho-node-blocked');

      let next = n;
      if (isBlocked && !hasBlocked) {
        next = { ...next, className: `${baseClass} clotho-node-blocked`.trim() };
      } else if (!isBlocked && hasBlocked) {
        next = {
          ...next,
          className: baseClass
            .split(/\s+/)
            .filter((c) => c !== 'clotho-node-blocked')
            .join(' '),
        };
      }

      if (isLocked) {
        next = { ...next, draggable: false, deletable: false };
      } else if (n.draggable === false || n.deletable === false) {
        // Strip the flags if a node was unlocked — otherwise React Flow keeps
        // treating it as locked even after the store flips.
        const { draggable: _d, deletable: _del, ...rest } = next;
        void _d;
        void _del;
        next = rest as typeof next;
      }

      return next;
    });
  }, [nodes, blockedNodeIds, lockedNodes]);
  const onNodesChangeStore = usePipelineStore((s) => s.onNodesChange);
  const onNodesChange = useCallback(
    (changes: NodeChange<Node<PipelineNodeData>>[]) => {
      if (lockedNodes.size === 0) {
        onNodesChangeStore(changes);
        return;
      }
      // Filter out destructive changes for locked nodes — defense in depth on
      // top of React Flow's draggable/deletable node flags.
      const filtered = changes.filter((change) => {
        if (change.type === 'remove' && lockedNodes.has(change.id)) {
          return false;
        }
        if (change.type === 'position' && lockedNodes.has(change.id)) {
          return false;
        }
        return true;
      });
      onNodesChangeStore(filtered);
    },
    [lockedNodes, onNodesChangeStore],
  );
  const onEdgesChange = usePipelineStore((s) => s.onEdgesChange);
  const onConnect = usePipelineStore((s) => s.onConnect);
  const addNode = usePipelineStore((s) => s.addNode);
  const removeNodes = usePipelineStore((s) => s.removeNodes);
  const setSelectedNode = usePipelineStore((s) => s.setSelectedNode);

  const onNodesDelete = useCallback(
    (deleted: Node<PipelineNodeData>[]) => {
      if (deleted.length === 0) return;
      const removable = deleted
        .map((n) => n.id)
        .filter((id) => !lockedNodes.has(id));
      if (removable.length === 0) return;
      removeNodes(removable);
    },
    [removeNodes, lockedNodes],
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

  const { setCenter, screenToFlowPosition } = useReactFlow();

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

      // Convert screen (clientX/Y) to flow coordinates. The previous code
      // used clientX - bounds.left, which was wrong whenever the viewport
      // was panned or zoomed — nodes landed near the canvas origin instead
      // of under the cursor. screenToFlowPosition accounts for the current
      // viewport transform.
      const position = screenToFlowPosition({
        x: event.clientX,
        y: event.clientY,
      });

      addNode(
        payload.nodeType,
        position,
        payload.config,
        payload.ports,
        payload.label,
      );
    },
    [addNode, screenToFlowPosition],
  );

  const onNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node<PipelineNodeData>) => {
      setSelectedNode(node.id);
      // Smoothly pan the canvas to center the clicked node so partially-
      // visible / clipped nodes come fully into view. Keep the current zoom
      // by passing the existing viewport zoom.
      const width = node.measured?.width ?? node.width ?? 220;
      const height = node.measured?.height ?? node.height ?? 150;
      const centerX = node.position.x + width / 2;
      const centerY = node.position.y + height / 2;
      setCenter(centerX, centerY, {
        duration: 300,
        zoom: storedViewport.zoom,
      });
    },
    [setSelectedNode, setCenter, storedViewport.zoom],
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
