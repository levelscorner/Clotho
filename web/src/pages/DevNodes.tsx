/* ---------------------------------------------------------------------------
   DevNodes — dev-only visual testbed at /dev/nodes.

   Shows every node kind × state in a tabbed grid so designers and engineers
   can eyeball the full personality/state matrix without touching a real graph.

   Instantiation approach:
   -----------------------
   The node components (AgentNode / MediaNode / ToolNode) render React Flow
   <Handle> elements via BaseNode, which REQUIRES a ReactFlowProvider in the
   tree. We therefore wrap the whole page in <ReactFlowProvider> and render
   each fixture as a plain component call with a hand-built NodeProps object.
   This is simpler than mounting a 1-node React Flow canvas per card and keeps
   the grid fully responsive.

   The nodes also read step-result data from useExecutionStore. On mount we
   seed the store with a Map of every fixture's pre-baked StepResult so the
   running/complete/failed/empty states render as intended.

   Gating: the page is only reachable when `import.meta.env.DEV === true` —
   see the route wiring in App.tsx.
   --------------------------------------------------------------------------- */

import React, { useEffect, useMemo, useState } from 'react';
import { ReactFlowProvider } from '@xyflow/react';
import type { NodeProps, Node } from '@xyflow/react';

import { AgentNode } from '../components/canvas/nodes/AgentNode';
import { MediaNode } from '../components/canvas/nodes/MediaNode';
import { ToolNode } from '../components/canvas/nodes/ToolNode';
import {
  ALL_FIXTURES,
  type NodeFixture,
  type MediaNodeFixture,
  type ToolNodeFixture,
  type FixtureState,
} from '../components/canvas/nodes/__mocks__/node-fixtures';
import { useExecutionStore } from '../stores/executionStore';
import type {
  AgentNodeData,
  MediaNodeData,
  StepResult,
  ToolNodeData,
} from '../lib/types';

import './DevNodes.css';

// ---------------------------------------------------------------------------
// Tab model
// ---------------------------------------------------------------------------

type TabId =
  | 'agent-script'
  | 'agent-crafter'
  | 'agent-generic'
  | 'media-image'
  | 'media-video'
  | 'media-audio'
  | 'tool';

interface Tab {
  id: TabId;
  label: string;
}

const TABS: Tab[] = [
  { id: 'agent-script', label: 'Agent · Script' },
  { id: 'agent-crafter', label: 'Agent · Crafter' },
  { id: 'agent-generic', label: 'Agent · Generic' },
  { id: 'media-image', label: 'Media · Image' },
  { id: 'media-video', label: 'Media · Video' },
  { id: 'media-audio', label: 'Media · Audio' },
  { id: 'tool', label: 'Tool' },
];

const STATE_BADGES: Record<FixtureState, string> = {
  queued: 'QUEUED',
  running: 'RUNNING',
  complete: 'COMPLETE',
  'empty-complete': 'EMPTY-COMPLETE',
  failed: 'FAILED',
};

// ---------------------------------------------------------------------------
// Fake NodeProps builder — fills fields the components ignore.
// ---------------------------------------------------------------------------

function fakeNodeProps<TData extends Record<string, unknown>>(
  id: string,
  data: TData,
  selected: boolean,
  type: string,
): NodeProps<Node<TData>> {
  return {
    id,
    data,
    selected,
    type,
    dragging: false,
    isConnectable: true,
    positionAbsoluteX: 0,
    positionAbsoluteY: 0,
    zIndex: 0,
    deletable: true,
    selectable: true,
    draggable: true,
  } as unknown as NodeProps<Node<TData>>;
}

// ---------------------------------------------------------------------------
// Card + grid
// ---------------------------------------------------------------------------

interface FixtureCardProps {
  badge: string;
  caption: string;
  children: React.ReactNode;
}

function FixtureCard({ badge, caption, children }: FixtureCardProps) {
  return (
    <article className="dev-nodes__card">
      <span className="dev-nodes__state-badge">{badge}</span>
      <div className="dev-nodes__node-wrap">{children}</div>
      <p className="dev-nodes__caption">{caption}</p>
    </article>
  );
}

// ---------------------------------------------------------------------------
// Per-tab renderers
// ---------------------------------------------------------------------------

function renderAgentCards(fixtures: NodeFixture[]): React.ReactNode {
  return fixtures.map((f) => (
    <FixtureCard
      key={f.id}
      badge={STATE_BADGES[f.state]}
      caption={`${f.id} · preset=${f.presetCategory}${
        f.stepResult?.tokens_used != null
          ? ` · tokens=${f.stepResult.tokens_used}`
          : ''
      }`}
    >
      <AgentNode
        {...fakeNodeProps<AgentNodeData>(f.id, f.data, f.selected, 'agent')}
      />
    </FixtureCard>
  ));
}

function renderMediaCards(fixtures: MediaNodeFixture[]): React.ReactNode {
  return fixtures.map((f) => (
    <FixtureCard
      key={f.id}
      badge={STATE_BADGES[f.state]}
      caption={`${f.id} · ${f.data.config.provider} · ${f.data.config.model}`}
    >
      <MediaNode
        {...fakeNodeProps<MediaNodeData>(f.id, f.data, f.selected, 'media')}
      />
    </FixtureCard>
  ));
}

function renderToolCards(fixtures: ToolNodeFixture[]): React.ReactNode {
  return fixtures.map((f) => (
    <FixtureCard
      key={f.id}
      badge={STATE_BADGES[f.state]}
      caption={`${f.id} · tool_type=${f.toolType}`}
    >
      <ToolNode
        {...fakeNodeProps<ToolNodeData>(f.id, f.data, f.selected, 'tool')}
      />
    </FixtureCard>
  ));
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

function DevNodesInner() {
  const [activeTab, setActiveTab] = useState<TabId>('agent-script');

  // Seed the execution store with every fixture's step result once on mount.
  // The nodes read from this store via `stepResults.get(id)` so they'll pick
  // up the queued/running/complete/empty-complete/failed state automatically.
  useEffect(() => {
    const all: [string, StepResult][] = [];
    for (const f of ALL_FIXTURES.agent) {
      if (f.stepResult) all.push([f.id, f.stepResult]);
    }
    for (const f of ALL_FIXTURES.media) {
      if (f.stepResult) all.push([f.id, f.stepResult]);
    }
    for (const f of ALL_FIXTURES.tool) {
      if (f.stepResult) all.push([f.id, f.stepResult]);
    }
    useExecutionStore.setState({
      executionId: 'dev-nodes-fixture',
      status: 'running',
      stepResults: new Map(all),
      totalCost: 0,
      isStreaming: false,
    });
    return () => {
      // On unmount clear the seeded state so the rest of the app is clean.
      useExecutionStore.setState({
        executionId: null,
        status: null,
        stepResults: new Map(),
        totalCost: 0,
        isStreaming: false,
      });
    };
  }, []);

  const content = useMemo(() => {
    switch (activeTab) {
      case 'agent-script':
        return renderAgentCards(
          ALL_FIXTURES.agent.filter((f) => f.presetCategory === 'script'),
        );
      case 'agent-crafter':
        return renderAgentCards(
          ALL_FIXTURES.agent.filter((f) => f.presetCategory === 'crafter'),
        );
      case 'agent-generic':
        return renderAgentCards(
          ALL_FIXTURES.agent.filter((f) => f.presetCategory === 'generic'),
        );
      case 'media-image':
        return renderMediaCards(
          ALL_FIXTURES.media.filter((f) => f.mediaType === 'image'),
        );
      case 'media-video':
        return renderMediaCards(
          ALL_FIXTURES.media.filter((f) => f.mediaType === 'video'),
        );
      case 'media-audio':
        return renderMediaCards(
          ALL_FIXTURES.media.filter((f) => f.mediaType === 'audio'),
        );
      case 'tool':
        return renderToolCards(ALL_FIXTURES.tool);
    }
  }, [activeTab]);

  return (
    <div className="dev-nodes">
      <header className="dev-nodes__header">
        <h1 className="dev-nodes__title">Clotho · Node Fixtures (dev)</h1>
        <p className="dev-nodes__subtitle">
          Visual matrix of every node kind × state. Not production.
        </p>
      </header>

      <nav className="dev-nodes__tabs" aria-label="Node fixture categories">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            className={
              'dev-nodes__tab' +
              (tab.id === activeTab ? ' dev-nodes__tab--active' : '')
            }
            aria-pressed={tab.id === activeTab}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </nav>

      <main className="dev-nodes__grid" data-testid="dev-nodes-grid">
        {content}
      </main>
    </div>
  );
}

export default function DevNodes() {
  return (
    <ReactFlowProvider>
      <DevNodesInner />
    </ReactFlowProvider>
  );
}
