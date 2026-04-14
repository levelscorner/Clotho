import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import type { AgentNodeConfig } from '../../../lib/types';

// ---------------------------------------------------------------------------
// Mocks — must be set up before importing AgentInspector
// ---------------------------------------------------------------------------

// Mock the pipeline store — AgentInspector reads two selectors from it.
vi.mock('../../../stores/pipelineStore', () => ({
  usePipelineStore: (selector: (s: unknown) => unknown) =>
    selector({
      updateNodeConfig: vi.fn(),
      updateNodeLabel: vi.fn(),
    }),
}));

// Mock the api module — AgentInspector calls api.get and api.credentials.list
// in a useEffect. Return empty datasets so it resolves synchronously-enough.
vi.mock('../../../lib/api', () => ({
  api: {
    get: vi.fn().mockResolvedValue([]),
    credentials: {
      list: vi.fn().mockResolvedValue([]),
    },
  },
}));

// OllamaModelDropdown is not exercised here (provider defaults to non-ollama).

import { AgentInspector } from '../AgentInspector';

// ---------------------------------------------------------------------------
// Fixture
// ---------------------------------------------------------------------------

function buildConfig(): AgentNodeConfig {
  return {
    provider: 'openai',
    model: 'gpt-4o',
    role: {
      system_prompt: 'You are a helpful assistant.',
      persona: 'Helpful writer',
    },
    task: {
      task_type: 'script',
      output_type: 'text',
      template: 'Write a scene based on {{input}}.',
    },
    temperature: 0.7,
    max_tokens: 1024,
  };
}

describe('AgentInspector — Prompt field promoted to Basics', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders a textarea labeled "Prompt" in the Basics group', () => {
    render(
      <AgentInspector
        nodeId="node_1"
        label="Scene Writer"
        config={buildConfig()}
      />,
    );

    // Find the Basics <details> group
    const basicsSummary = screen.getByText('Basics');
    const basicsGroup = basicsSummary.closest('details');
    expect(basicsGroup).not.toBeNull();

    // The Prompt label lives inside Basics.
    const promptLabel = within(basicsGroup as HTMLElement).getByText('Prompt');
    expect(promptLabel).toBeInTheDocument();

    // The sibling input/textarea should be a <textarea>.
    const textarea = promptLabel.parentElement?.querySelector('textarea');
    expect(textarea).not.toBeNull();
    expect(textarea?.value).toBe('Write a scene based on {{input}}.');
  });

  it('does not duplicate the Prompt field in the Advanced group', () => {
    render(
      <AgentInspector
        nodeId="node_1"
        label="Scene Writer"
        config={buildConfig()}
      />,
    );

    const advancedSummary = screen.getByText('Advanced');
    const advancedGroup = advancedSummary.closest('details');
    expect(advancedGroup).not.toBeNull();

    // The old "Template" label should be gone from Advanced.
    const templateInAdvanced = within(advancedGroup as HTMLElement).queryByText(
      /^Template$/,
    );
    expect(templateInAdvanced).toBeNull();

    // And the "Prompt" label should not exist inside Advanced either.
    const promptInAdvanced = within(advancedGroup as HTMLElement).queryByText(
      /^Prompt$/,
    );
    expect(promptInAdvanced).toBeNull();
  });

  it('orders Prompt before Persona in Basics DOM order', () => {
    render(
      <AgentInspector
        nodeId="node_1"
        label="Scene Writer"
        config={buildConfig()}
      />,
    );

    const basicsSummary = screen.getByText('Basics');
    const basicsGroup = basicsSummary.closest('details') as HTMLElement;

    const promptLabel = within(basicsGroup).getByText('Prompt');
    const personaLabel = within(basicsGroup).getByText('Persona');

    // compareDocumentPosition returns DOCUMENT_POSITION_FOLLOWING (4) when the
    // argument comes after the node.
    const relation = promptLabel.compareDocumentPosition(personaLabel);
    expect(relation & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('renders the {{input}} helper caption under the Prompt field', () => {
    render(
      <AgentInspector
        nodeId="node_1"
        label="Scene Writer"
        config={buildConfig()}
      />,
    );

    expect(
      screen.getByText(/Use \{\{input\}\} to reference incoming data\./),
    ).toBeInTheDocument();
  });
});
