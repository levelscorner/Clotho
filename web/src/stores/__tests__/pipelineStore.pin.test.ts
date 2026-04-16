import { describe, it, expect, beforeEach, vi } from 'vitest';
import { usePipelineStore } from '../pipelineStore';
import { useExecutionStore } from '../executionStore';
import type { AgentNodeConfig } from '../../lib/types';

// Avoid hitting the real network when the pipelineStore touches api.post.
vi.mock('../../lib/api', () => ({
  api: {
    post: vi.fn().mockResolvedValue({}),
    get: vi.fn().mockResolvedValue({}),
  },
}));

describe('pipelineStore — pin + on_failure', () => {
  beforeEach(() => {
    usePipelineStore.getState().reset();
    useExecutionStore.getState().reset();
  });

  function seedAgentNode(id: string) {
    const cfg: AgentNodeConfig = {
      provider: 'ollama',
      model: 'llama3.1',
      role: { system_prompt: '', persona: '' },
      task: { task_type: 'custom', output_type: 'text', template: '' },
      temperature: 0.7,
      max_tokens: 2048,
    };
    usePipelineStore.getState().addNode(
      'agent',
      { x: 0, y: 0 },
      cfg,
      [
        { id: 'in', name: 'In', type: 'any', direction: 'input', required: false },
        { id: 'out', name: 'Out', type: 'text', direction: 'output', required: false },
      ],
      'agent ' + id,
    );
    return usePipelineStore.getState().nodes[
      usePipelineStore.getState().nodes.length - 1
    ].id;
  }

  describe('setNodePin', () => {
    it('does not snapshot output when no executionStore result exists', () => {
      const id = seedAgentNode('a');
      usePipelineStore.getState().setNodePin(id, true);
      const node = usePipelineStore.getState().nodes.find((n) => n.id === id);
      expect(node?.data.pinned).toBe(true);
      // No prior execution → no pinned output.
      expect(node?.data.pinnedOutput).toBeUndefined();
    });

    it('snapshots the most recent successful output from executionStore', () => {
      const id = seedAgentNode('a');
      useExecutionStore.getState().updateStep({
        node_id: id,
        status: 'completed',
        output: 'frozen value from a prior run',
      });

      usePipelineStore.getState().setNodePin(id, true);
      const node = usePipelineStore.getState().nodes.find((n) => n.id === id);
      expect(node?.data.pinned).toBe(true);
      expect(node?.data.pinnedOutput).toBe('frozen value from a prior run');
    });

    it('does NOT snapshot a failed step (status != completed)', () => {
      const id = seedAgentNode('a');
      useExecutionStore.getState().updateStep({
        node_id: id,
        status: 'failed',
        output: 'partial garbage',
      });

      usePipelineStore.getState().setNodePin(id, true);
      const node = usePipelineStore.getState().nodes.find((n) => n.id === id);
      // pinned flag flips on but no output is snapshotted; the inspector
      // disables the checkbox until a successful run exists.
      expect(node?.data.pinned).toBe(true);
      expect(node?.data.pinnedOutput).toBeUndefined();
    });

    it('unpinning clears pinnedOutput', () => {
      const id = seedAgentNode('a');
      useExecutionStore.getState().updateStep({
        node_id: id,
        status: 'completed',
        output: 'cached',
      });
      usePipelineStore.getState().setNodePin(id, true);
      usePipelineStore.getState().setNodePin(id, false);

      const node = usePipelineStore.getState().nodes.find((n) => n.id === id);
      expect(node?.data.pinned).toBe(false);
      expect(node?.data.pinnedOutput).toBeUndefined();
    });

    it('marks the pipeline dirty on toggle', () => {
      const id = seedAgentNode('a');
      // addNode already marks dirty; clear it via save() being mocked
      // works imperfectly here, so just check that setNodePin sets it
      // back to true after we clear it directly.
      usePipelineStore.setState({ isDirty: false });
      usePipelineStore.getState().setNodePin(id, true);
      expect(usePipelineStore.getState().isDirty).toBe(true);
    });
  });

  describe('setNodeOnFailure', () => {
    it('sets the policy on the right node', () => {
      const a = seedAgentNode('a');
      const b = seedAgentNode('b');

      usePipelineStore.getState().setNodeOnFailure(a, 'skip');

      const aNode = usePipelineStore.getState().nodes.find((n) => n.id === a);
      const bNode = usePipelineStore.getState().nodes.find((n) => n.id === b);
      expect(aNode?.data.onFailure).toBe('skip');
      expect(bNode?.data.onFailure).toBeUndefined();
    });

    it('accepts each policy value', () => {
      const id = seedAgentNode('a');
      const policies = ['abort', 'skip', 'continue'] as const;
      for (const p of policies) {
        usePipelineStore.getState().setNodeOnFailure(id, p);
        const node = usePipelineStore.getState().nodes.find((n) => n.id === id);
        expect(node?.data.onFailure).toBe(p);
      }
    });

    it('marks the pipeline dirty', () => {
      const id = seedAgentNode('a');
      usePipelineStore.setState({ isDirty: false });
      usePipelineStore.getState().setNodeOnFailure(id, 'continue');
      expect(usePipelineStore.getState().isDirty).toBe(true);
    });
  });
});
