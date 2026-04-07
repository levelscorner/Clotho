import { create } from 'zustand';
import { api } from '../lib/api';
import type { TemplateSummary, TemplateDetail } from '../lib/api';
import type {
  AgentNodeConfig,
  ToolNodeConfig,
  MediaNodeConfig,
  PipelineNodeData,
} from '../lib/types';
import type { Node, Edge as RFEdge } from '@xyflow/react';
import { usePipelineStore } from './pipelineStore';
import { useHistoryStore } from './historyStore';

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

interface TemplateState {
  templates: TemplateSummary[];
  loading: boolean;
  error: string | null;
  fetchTemplates: () => Promise<void>;
  applyTemplate: (id: string) => Promise<void>;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useTemplateStore = create<TemplateState>((set) => ({
  templates: [],
  loading: false,
  error: null,

  fetchTemplates: async () => {
    set({ loading: true, error: null });
    try {
      const list = await api.templates.list();
      set({ templates: list, loading: false });
    } catch {
      set({ error: 'Failed to load templates', loading: false });
    }
  },

  applyTemplate: async (id: string) => {
    try {
      const detail: TemplateDetail = await api.templates.get(id);
      const graph = detail.graph;

      const rfTypeMap: Record<string, string> = {
        agent: 'agentNode',
        tool: 'toolNode',
        media: 'mediaNode',
      };

      const nodes: Node<PipelineNodeData>[] = graph.nodes.map((ni) => {
        const data: PipelineNodeData =
          ni.type === 'agent'
            ? {
                nodeType: 'agent' as const,
                label: ni.label,
                ports: ni.ports,
                config: ni.config as AgentNodeConfig,
              }
            : ni.type === 'media'
              ? {
                  nodeType: 'media' as const,
                  label: ni.label,
                  ports: ni.ports,
                  config: ni.config as MediaNodeConfig,
                }
              : {
                  nodeType: 'tool' as const,
                  label: ni.label,
                  ports: ni.ports,
                  config: ni.config as ToolNodeConfig,
                };

        return {
          id: ni.id,
          type: rfTypeMap[ni.type] ?? 'agentNode',
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

      const viewport = graph.viewport ?? { x: 0, y: 0, zoom: 1 };

      useHistoryStore.getState().clear();
      usePipelineStore.setState({
        nodes,
        edges,
        viewport,
        isDirty: true,
        selectedNodeId: null,
      });
    } catch {
      set({ error: 'Failed to apply template' });
    }
  },
}));
