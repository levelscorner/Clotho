import { create } from 'zustand';
import {
  type Node,
  type Edge as RFEdge,
  type NodeChange,
  type EdgeChange,
  type Connection,
  applyNodeChanges,
  applyEdgeChanges,
} from '@xyflow/react';
import type {
  Port,
  Position,
  AgentNodeConfig,
  ToolNodeConfig,
  MediaNodeConfig,
  PipelineNodeData,
  PipelineGraph,
  NodeType,
  Viewport,
} from '../lib/types';
import { canConnect } from '../lib/portCompatibility';
import { api } from '../lib/api';
import { useHistoryStore } from './historyStore';

// ---------------------------------------------------------------------------
// Custom node type alias
// ---------------------------------------------------------------------------

type PipelineNode = Node<PipelineNodeData>;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

let nodeIdCounter = 0;
function nextNodeId(): string {
  nodeIdCounter += 1;
  return `node_${Date.now()}_${nodeIdCounter}`;
}

let edgeIdCounter = 0;
function nextEdgeId(): string {
  edgeIdCounter += 1;
  return `edge_${Date.now()}_${edgeIdCounter}`;
}

// ---------------------------------------------------------------------------
// Debounced history push for config/label changes
// ---------------------------------------------------------------------------

let configDebounceTimer: ReturnType<typeof setTimeout> | null = null;

function debouncedHistoryPush(nodes: PipelineNode[], edges: RFEdge[]): void {
  if (configDebounceTimer) {
    clearTimeout(configDebounceTimer);
  }
  configDebounceTimer = setTimeout(() => {
    useHistoryStore.getState().push({ nodes, edges });
    configDebounceTimer = null;
  }, 300);
}

function pushHistory(nodes: PipelineNode[], edges: RFEdge[]): void {
  useHistoryStore.getState().push({ nodes, edges });
}

function findPort(
  nodes: PipelineNode[],
  nodeId: string,
  portId: string,
): Port | undefined {
  const node = nodes.find((n) => n.id === nodeId);
  if (!node) return undefined;
  return node.data.ports.find((p) => p.id === portId);
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

interface PipelineState {
  nodes: PipelineNode[];
  edges: RFEdge[];
  viewport: Viewport;
  pipelineId: string | null;
  pipelineName: string;
  isDirty: boolean;
  selectedNodeId: string | null;

  onViewportChange: (viewport: Viewport) => void;
  addNode: (
    nodeType: NodeType,
    position: Position,
    config: AgentNodeConfig | ToolNodeConfig | MediaNodeConfig,
    ports: Port[],
    label?: string,
  ) => void;
  removeNodes: (ids: string[]) => void;
  updateNodeConfig: (
    nodeId: string,
    updater: (
      prev: AgentNodeConfig | ToolNodeConfig | MediaNodeConfig,
    ) => AgentNodeConfig | ToolNodeConfig | MediaNodeConfig,
  ) => void;
  updateNodeLabel: (nodeId: string, label: string) => void;
  onNodesChange: (changes: NodeChange<PipelineNode>[]) => void;
  onEdgesChange: (changes: EdgeChange<RFEdge>[]) => void;
  onConnect: (connection: Connection) => void;
  setSelectedNode: (id: string | null) => void;
  undo: () => void;
  redo: () => void;
  setRestoredState: (
    nodes: PipelineNode[],
    edges: RFEdge[],
    viewport: Viewport,
  ) => void;
  save: () => Promise<void>;
  load: (pipelineId: string) => Promise<void>;
  setPipeline: (id: string, name: string) => void;
  reset: () => void;
}

export const usePipelineStore = create<PipelineState>((set, get) => ({
  nodes: [],
  edges: [],
  viewport: { x: 0, y: 0, zoom: 1 },
  pipelineId: null,
  pipelineName: 'Untitled Pipeline',
  isDirty: false,
  selectedNodeId: null,

  onViewportChange: (viewport) => {
    set({ viewport });
  },

  addNode: (nodeType, position, config, ports, label) => {
    const { nodes, edges } = get();
    pushHistory(nodes, edges);
    const id = nextNodeId();
    const data: PipelineNodeData =
      nodeType === 'agent'
        ? {
            nodeType: 'agent' as const,
            label: label ?? 'New Agent',
            ports,
            config: config as AgentNodeConfig,
          }
        : nodeType === 'media'
          ? {
              nodeType: 'media' as const,
              label: label ?? 'Media',
              ports,
              config: config as MediaNodeConfig,
            }
          : {
              nodeType: 'tool' as const,
              label: label ?? 'Tool',
              ports,
              config: config as ToolNodeConfig,
            };

    const rfTypeMap: Record<NodeType, string> = {
      agent: 'agentNode',
      tool: 'toolNode',
      media: 'mediaNode',
    };

    const newNode: PipelineNode = {
      id,
      type: rfTypeMap[nodeType],
      position,
      data,
    };
    set((state) => ({
      nodes: [...state.nodes, newNode],
      isDirty: true,
    }));
  },

  removeNodes: (ids) => {
    const { nodes, edges } = get();
    pushHistory(nodes, edges);
    const idSet = new Set(ids);
    set((state) => ({
      nodes: state.nodes.filter((n) => !idSet.has(n.id)),
      edges: state.edges.filter(
        (e) => !idSet.has(e.source) && !idSet.has(e.target),
      ),
      isDirty: true,
      selectedNodeId:
        state.selectedNodeId && idSet.has(state.selectedNodeId)
          ? null
          : state.selectedNodeId,
    }));
  },

  updateNodeConfig: (nodeId, updater) => {
    const { nodes, edges } = get();
    debouncedHistoryPush(nodes, edges);
    set((state) => ({
      nodes: state.nodes.map((n): PipelineNode => {
        if (n.id !== nodeId) return n;
        const prevConfig = n.data.config as AgentNodeConfig | ToolNodeConfig | MediaNodeConfig;
        const newConfig = updater(prevConfig);
        const newData: PipelineNodeData =
          n.data.nodeType === 'agent'
            ? { ...n.data, nodeType: 'agent' as const, config: newConfig as AgentNodeConfig }
            : n.data.nodeType === 'media'
              ? { ...n.data, nodeType: 'media' as const, config: newConfig as MediaNodeConfig }
              : { ...n.data, nodeType: 'tool' as const, config: newConfig as ToolNodeConfig };
        return { ...n, data: newData };
      }),
      isDirty: true,
    }));
  },

  updateNodeLabel: (nodeId, label) => {
    const { nodes, edges } = get();
    debouncedHistoryPush(nodes, edges);
    set((state) => ({
      nodes: state.nodes.map((n): PipelineNode =>
        n.id === nodeId ? { ...n, data: { ...n.data, label } } : n,
      ),
      isDirty: true,
    }));
  },

  onNodesChange: (changes) => {
    set((state) => ({
      nodes: applyNodeChanges(changes, state.nodes) as PipelineNode[],
      isDirty: true,
    }));
  },

  onEdgesChange: (changes) => {
    set((state) => ({
      edges: applyEdgeChanges(changes, state.edges),
      isDirty: true,
    }));
  },

  onConnect: (connection) => {
    const { nodes } = get();
    const sourceHandle = connection.sourceHandle;
    const targetHandle = connection.targetHandle;

    if (!sourceHandle || !targetHandle) return;

    const sourcePort = findPort(nodes, connection.source, sourceHandle);
    const targetPort = findPort(nodes, connection.target, targetHandle);

    if (!sourcePort || !targetPort) return;

    if (!canConnect(sourcePort.type, targetPort.type)) {
      return;
    }

    pushHistory(nodes, get().edges);

    const newEdge: RFEdge = {
      id: nextEdgeId(),
      source: connection.source,
      sourceHandle,
      target: connection.target,
      targetHandle,
    };

    set((state) => ({
      edges: [...state.edges, newEdge],
      isDirty: true,
    }));
  },

  setSelectedNode: (id) => {
    set({ selectedNodeId: id });
  },

  undo: () => {
    const snapshot = useHistoryStore.getState().undo();
    if (snapshot) {
      set({
        nodes: snapshot.nodes as PipelineNode[],
        edges: snapshot.edges,
        isDirty: true,
      });
    }
  },

  redo: () => {
    const snapshot = useHistoryStore.getState().redo();
    if (snapshot) {
      set({
        nodes: snapshot.nodes as PipelineNode[],
        edges: snapshot.edges,
        isDirty: true,
      });
    }
  },

  setRestoredState: (nodes, edges, viewport) => {
    set({
      nodes,
      edges,
      viewport,
      isDirty: false,
      selectedNodeId: null,
    });
  },

  save: async () => {
    const { pipelineId, nodes, edges, viewport } = get();
    if (!pipelineId) return;

    const graph: PipelineGraph = {
      nodes: nodes.map((n) => ({
        id: n.id,
        type: n.data.nodeType,
        label: n.data.label,
        position: { x: n.position.x, y: n.position.y },
        ports: n.data.ports,
        config: n.data.config as AgentNodeConfig | ToolNodeConfig,
      })),
      edges: edges.map((e) => ({
        id: e.id,
        source: e.source,
        source_port: e.sourceHandle ?? '',
        target: e.target,
        target_port: e.targetHandle ?? '',
      })),
      viewport,
    };

    await api.post(`/pipelines/${pipelineId}/versions`, { graph });
    set({ isDirty: false });
    useHistoryStore.getState().clear();
  },

  load: async (pipelineId) => {
    const version = await api.get<{ graph: PipelineGraph }>(
      `/pipelines/${pipelineId}/versions/latest`,
    );

    const graph = version.graph;

    const nodes: PipelineNode[] = graph.nodes.map((ni): PipelineNode => {
      const data: PipelineNodeData =
        ni.type === 'agent'
          ? {
              nodeType: 'agent' as const,
              label: ni.label,
              ports: ni.ports,
              config: ni.config as AgentNodeConfig,
            }
          : {
              nodeType: 'tool' as const,
              label: ni.label,
              ports: ni.ports,
              config: ni.config as ToolNodeConfig,
            };

      return {
        id: ni.id,
        type: ni.type === 'agent' ? 'agentNode' : 'toolNode',
        position: ni.position,
        data,
      };
    });

    const edges: RFEdge[] = graph.edges.map((e) => ({
      id: e.id,
      source: e.source,
      sourceHandle: e.source_port,
      target: e.target,
      targetHandle: e.target_port,
    }));

    const loadedViewport = graph.viewport ?? { x: 0, y: 0, zoom: 1 };

    useHistoryStore.getState().clear();
    set({
      pipelineId,
      nodes,
      edges,
      viewport: loadedViewport,
      isDirty: false,
      selectedNodeId: null,
    });
  },

  setPipeline: (id, name) => {
    set({ pipelineId: id, pipelineName: name });
  },

  reset: () => {
    useHistoryStore.getState().clear();
    set({
      nodes: [],
      edges: [],
      viewport: { x: 0, y: 0, zoom: 1 },
      pipelineId: null,
      pipelineName: 'Untitled Pipeline',
      isDirty: false,
      selectedNodeId: null,
    });
  },
}));
