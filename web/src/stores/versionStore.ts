import { create } from 'zustand';
import type { Node, Edge as RFEdge } from '@xyflow/react';
import type {
  PipelineVersion,
  PipelineNodeData,
  AgentNodeConfig,
  ToolNodeConfig,
  Viewport,
} from '../lib/types';
import { api } from '../lib/api';
import { usePipelineStore } from './pipelineStore';
import { useHistoryStore } from './historyStore';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface VersionState {
  versions: PipelineVersion[];
  isOpen: boolean;
  isLoading: boolean;

  fetchVersions: (pipelineId: string) => Promise<void>;
  restoreVersion: (version: PipelineVersion) => void;
  togglePanel: () => void;
  close: () => void;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useVersionStore = create<VersionState>((set) => ({
  versions: [],
  isOpen: false,
  isLoading: false,

  fetchVersions: async (pipelineId) => {
    set({ isLoading: true });
    try {
      const versions = await api.get<PipelineVersion[]>(
        `/pipelines/${pipelineId}/versions`,
      );
      set({ versions, isLoading: false });
    } catch {
      set({ versions: [], isLoading: false });
    }
  },

  restoreVersion: (version) => {
    const graph = version.graph;
    const pipelineStore = usePipelineStore.getState();

    const nodes = graph.nodes.map((ni) => {
      const data =
        ni.type === 'agent'
          ? {
              nodeType: 'agent' as const,
              label: ni.label,
              ports: ni.ports,
              config: ni.config as import('../lib/types').AgentNodeConfig,
            }
          : ni.type === 'media'
            ? {
                nodeType: 'media' as const,
                label: ni.label,
                ports: ni.ports,
                config: ni.config as import('../lib/types').MediaNodeConfig,
              }
            : {
                nodeType: 'tool' as const,
                label: ni.label,
                ports: ni.ports,
                config: ni.config as import('../lib/types').ToolNodeConfig,
              };

      const rfTypeMap: Record<string, string> = {
        agent: 'agentNode',
        tool: 'toolNode',
        media: 'mediaNode',
      };

      return {
        id: ni.id,
        type: rfTypeMap[ni.type] ?? 'agentNode',
        position: ni.position,
        data,
      };
    });

    const edges = graph.edges.map((e) => ({
      id: e.id,
      source: e.source,
      sourceHandle: e.source_port,
      target: e.target,
      targetHandle: e.target_port,
    }));

    const viewport = graph.viewport ?? { x: 0, y: 0, zoom: 1 };

    // Clear history since we are restoring a saved version
    useHistoryStore.getState().clear();

    pipelineStore.setRestoredState(nodes, edges, viewport);
  },

  togglePanel: () => {
    set((state) => ({ isOpen: !state.isOpen }));
  },

  close: () => {
    set({ isOpen: false });
  },
}));
