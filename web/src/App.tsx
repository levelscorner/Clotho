import { useEffect, useState, useCallback } from 'react';
import '@xyflow/react/dist/style.css';
import './styles/nodes.css';

import { ReactFlowProvider } from '@xyflow/react';
import { PipelineCanvas } from './components/canvas/PipelineCanvas';
import { NodePalette } from './components/sidebar/NodePalette';
import { NodeInspector } from './components/inspector/NodeInspector';
import { RunButton } from './components/execution/RunButton';
import { ExecutionStatus } from './components/execution/ExecutionStatus';
import { ErrorBoundary } from './components/ErrorBoundary';
import { useProjectStore } from './stores/projectStore';
import { usePipelineStore } from './stores/pipelineStore';
import { useSSE } from './hooks/useSSE';
import { useExecutionStore } from './stores/executionStore';
import type { ProviderInfo } from './lib/types';

// ---------------------------------------------------------------------------
// App
// ---------------------------------------------------------------------------

function AppContent() {
  const fetchProjects = useProjectStore((s) => s.fetchProjects);
  const createProject = useProjectStore((s) => s.createProject);
  const projects = useProjectStore((s) => s.projects);
  const currentProjectId = useProjectStore((s) => s.currentProjectId);
  const setCurrentProject = useProjectStore((s) => s.setCurrentProject);
  const fetchPipelines = useProjectStore((s) => s.fetchPipelines);
  const createPipeline = useProjectStore((s) => s.createPipeline);
  const pipelines = useProjectStore((s) => s.pipelines);
  const currentPipelineId = useProjectStore((s) => s.currentPipelineId);
  const setCurrentPipeline = useProjectStore((s) => s.setCurrentPipeline);

  const pipelineName = usePipelineStore((s) => s.pipelineName);
  const setPipeline = usePipelineStore((s) => s.setPipeline);
  const loadPipeline = usePipelineStore((s) => s.load);
  const savePipeline = usePipelineStore((s) => s.save);
  const isDirty = usePipelineStore((s) => s.isDirty);

  const executionId = useExecutionStore((s) => s.executionId);

  const [loading, setLoading] = useState(true);
  const [noProvidersAvailable, setNoProvidersAvailable] = useState(false);

  // Connect SSE when we have an execution
  useSSE(executionId);

  // Fetch providers and check availability
  useEffect(() => {
    fetch('/api/providers')
      .then((res) => res.json())
      .then((data: ProviderInfo[]) => {
        const hasAvailable = data.some((p) => p.available);
        setNoProvidersAvailable(!hasAvailable);
      })
      .catch(() => {
        setNoProvidersAvailable(true);
      });
  }, []);

  // Bootstrap: fetch projects, auto-create if empty, load first pipeline
  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      try {
        await fetchProjects();
      } catch {
        // API not available yet
        setLoading(false);
        return;
      }

      if (cancelled) return;

      const store = useProjectStore.getState();
      let projectId = store.projects[0]?.id;

      if (!projectId) {
        try {
          const project = await createProject('Default Project');
          projectId = project.id;
        } catch {
          setLoading(false);
          return;
        }
      }

      if (cancelled) return;
      setCurrentProject(projectId);
      await fetchPipelines(projectId);

      if (cancelled) return;

      const pStore = useProjectStore.getState();
      let pipelineId = pStore.pipelines[0]?.id;

      if (!pipelineId) {
        try {
          const pipeline = await createPipeline(projectId, 'My Pipeline');
          pipelineId = pipeline.id;
        } catch {
          setLoading(false);
          return;
        }
      }

      if (cancelled) return;
      setCurrentPipeline(pipelineId);
      const pipeline = useProjectStore
        .getState()
        .pipelines.find((p) => p.id === pipelineId);
      setPipeline(pipelineId, pipeline?.name ?? 'Pipeline');

      try {
        await loadPipeline(pipelineId);
      } catch {
        // No version yet -- fresh pipeline
      }

      setLoading(false);
    }

    void bootstrap();

    return () => {
      cancelled = true;
    };
    // Run once on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSave = useCallback(async () => {
    await savePipeline();
  }, [savePipeline]);

  if (loading) {
    return (
      <div
        style={{
          width: '100vw',
          height: '100vh',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: '#0f1117',
          color: '#64748b',
          fontSize: 14,
        }}
      >
        Loading...
      </div>
    );
  }

  return (
    <div
      style={{
        width: '100vw',
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        background: '#0f1117',
      }}
    >
      {/* Provider warning banner */}
      {noProvidersAvailable && (
        <div
          style={{
            padding: '8px 16px',
            background: '#422006',
            borderBottom: '1px solid #854d0e',
            color: '#fef08a',
            fontSize: 13,
            textAlign: 'center',
            flexShrink: 0,
          }}
        >
          No LLM providers configured. Add at least one API key: GEMINI_API_KEY
          (free), OPENAI_API_KEY, or OPENROUTER_API_KEY
        </div>
      )}

      {/* Top bar */}
      <header
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 16,
          padding: '8px 16px',
          borderBottom: '1px solid #1e2030',
          background: '#12131f',
          flexShrink: 0,
        }}
      >
        <span
          style={{ fontWeight: 700, fontSize: 15, color: '#e2e8f0' }}
        >
          Clotho
        </span>
        <span style={{ color: '#475569' }}>|</span>

        {currentProjectId && (
          <select
            style={{
              background: '#1a1c2e',
              color: '#e2e8f0',
              border: '1px solid #334155',
              borderRadius: 4,
              padding: '4px 8px',
              fontSize: 12,
            }}
            value={currentProjectId}
            onChange={(e) => {
              setCurrentProject(e.target.value);
              void fetchPipelines(e.target.value);
            }}
          >
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        )}

        {currentPipelineId && (
          <select
            style={{
              background: '#1a1c2e',
              color: '#e2e8f0',
              border: '1px solid #334155',
              borderRadius: 4,
              padding: '4px 8px',
              fontSize: 12,
            }}
            value={currentPipelineId}
            onChange={(e) => {
              const id = e.target.value;
              setCurrentPipeline(id);
              const pl = pipelines.find((p) => p.id === id);
              setPipeline(id, pl?.name ?? 'Pipeline');
              void loadPipeline(id);
            }}
          >
            {pipelines.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        )}

        <span
          style={{ fontSize: 13, color: '#94a3b8', marginLeft: 4 }}
        >
          {pipelineName}
        </span>

        {isDirty && (
          <button
            onClick={handleSave}
            style={{
              padding: '4px 12px',
              borderRadius: 4,
              border: '1px solid #334155',
              background: 'transparent',
              color: '#94a3b8',
              fontSize: 12,
              cursor: 'pointer',
            }}
          >
            Save
          </button>
        )}

        <div style={{ flex: 1 }} />

        <ExecutionStatus />
        <RunButton />
      </header>

      {/* Main content */}
      <div style={{ display: 'flex', flex: 1, minHeight: 0 }}>
        <NodePalette />
        <PipelineCanvas />
        <NodeInspector />
      </div>
    </div>
  );
}

export default function App() {
  return (
    <ErrorBoundary>
      <ReactFlowProvider>
        <AppContent />
      </ReactFlowProvider>
    </ErrorBoundary>
  );
}
