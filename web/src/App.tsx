import { useEffect, useState, useCallback, useRef } from 'react';
import '@xyflow/react/dist/style.css';
import './styles/nodes.css';

import { ReactFlowProvider } from '@xyflow/react';
import { PipelineCanvas } from './components/canvas/PipelineCanvas';
import { NodePalette } from './components/sidebar/NodePalette';
import { NodeInspector } from './components/inspector/NodeInspector';
import { RunButton } from './components/execution/RunButton';
import { ExecutionStatus } from './components/execution/ExecutionStatus';
import { ErrorBoundary } from './components/ErrorBoundary';
import { VersionPanel } from './components/versioning/VersionPanel';
import { LoginPage } from './components/auth/LoginPage';
import { RegisterPage } from './components/auth/RegisterPage';
import { SettingsPanel } from './components/settings/SettingsPanel';
import { TemplateGallery } from './components/templates/TemplateGallery';
import { useProjectStore } from './stores/projectStore';
import { usePipelineStore } from './stores/pipelineStore';
import { useHistoryStore } from './stores/historyStore';
import { useVersionStore } from './stores/versionStore';
import { useAuthStore } from './stores/authStore';
import { useSSE } from './hooks/useSSE';
import { useUndoRedo } from './hooks/useUndoRedo';
import { useExecutionStore } from './stores/executionStore';
import { api } from './lib/api';
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

  const undo = usePipelineStore((s) => s.undo);
  const redo = usePipelineStore((s) => s.redo);
  const canUndo = useHistoryStore((s) => s.canUndo);
  const canRedo = useHistoryStore((s) => s.canRedo);

  const versionPanelIsOpen = useVersionStore((s) => s.isOpen);
  const toggleVersionPanel = useVersionStore((s) => s.togglePanel);
  const fetchVersions = useVersionStore((s) => s.fetchVersions);

  const executionId = useExecutionStore((s) => s.executionId);

  const [loading, setLoading] = useState(true);
  const [noProvidersAvailable, setNoProvidersAvailable] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [templatesOpen, setTemplatesOpen] = useState(false);
  const importInputRef = useRef<HTMLInputElement>(null);

  // Connect SSE when we have an execution
  useSSE(executionId);

  // Keyboard shortcuts for undo/redo
  useUndoRedo();

  // Fetch providers and check availability
  useEffect(() => {
    api.get<ProviderInfo[]>('/providers')
      .then((data) => {
        const arr = Array.isArray(data) ? data : [];
        const hasAvailable = arr.some((p) => p.available);
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

  const handleExport = useCallback(async () => {
    if (!currentPipelineId) return;
    try {
      const blob = await api.exportPipeline(currentPipelineId);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${pipelineName || 'pipeline'}.clotho.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // Export failed silently — could add a toast in the future.
    }
  }, [currentPipelineId, pipelineName]);

  const handleImport = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file || !currentPipelineId) return;

      try {
        const text = await file.text();
        const data: unknown = JSON.parse(text);
        await api.importPipeline(currentPipelineId, data);
        await loadPipeline(currentPipelineId);
      } catch {
        // Import failed silently — could add a toast in the future.
      }

      // Reset file input so the same file can be re-imported.
      if (importInputRef.current) {
        importInputRef.current.value = '';
      }
    },
    [currentPipelineId, loadPipeline],
  );

  if (loading) {
    return (
      <div
        style={{
          width: '100vw',
          height: '100vh',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'var(--surface-base)',
          color: 'var(--text-muted)',
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
        background: 'var(--surface-base)',
      }}
    >
      {/* Provider warning banner */}
      {noProvidersAvailable && (
        <div
          style={{
            padding: '8px 16px',
            background: 'var(--accent-soft)',
            borderBottom: '1px solid var(--accent)',
            color: 'var(--accent)',
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
          gap: 12,
          height: 48,
          padding: '0 16px',
          borderBottom: '1px solid var(--surface-border)',
          background: 'var(--surface-overlay)',
          flexShrink: 0,
        }}
      >
        <span
          style={{
            fontFamily: 'var(--font-display)',
            fontWeight: 700,
            fontSize: 16,
            color: 'var(--accent)',
          }}
        >
          Clotho
        </span>

        {currentProjectId && (
          <select
            style={{
              background: 'var(--surface-base)',
              color: 'var(--text-primary)',
              border: '1px solid var(--surface-border)',
              borderRadius: 'var(--radius-sm)',
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
              background: 'var(--surface-base)',
              color: 'var(--text-primary)',
              border: '1px solid var(--surface-border)',
              borderRadius: 'var(--radius-sm)',
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
          style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 4 }}
        >
          {pipelineName}
        </span>

        <button
          onClick={undo}
          disabled={!canUndo}
          title="Undo (Ctrl+Z)"
          style={{
            padding: '6px 10px',
            minHeight: 32,
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--surface-border)',
            background: 'transparent',
            color: canUndo ? 'var(--text-secondary)' : 'var(--surface-border)',
            fontSize: 14,
            cursor: canUndo ? 'pointer' : 'default',
            lineHeight: 1,
          }}
        >
          {'↩'}
        </button>
        <button
          onClick={redo}
          disabled={!canRedo}
          title="Redo (Ctrl+Shift+Z)"
          style={{
            padding: '6px 10px',
            minHeight: 32,
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--surface-border)',
            background: 'transparent',
            color: canRedo ? 'var(--text-secondary)' : 'var(--surface-border)',
            fontSize: 14,
            cursor: canRedo ? 'pointer' : 'default',
            lineHeight: 1,
          }}
        >
          {'↪'}
        </button>

        <button
          onClick={handleSave}
          style={
            isDirty
              ? {
                  padding: '6px 12px',
                  minHeight: 32,
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--accent)',
                  background: 'var(--accent-soft)',
                  color: 'var(--accent)',
                  fontSize: 12,
                  cursor: 'pointer',
                  fontWeight: 600,
                }
              : {
                  padding: '6px 12px',
                  minHeight: 32,
                  borderRadius: 'var(--radius-sm)',
                  border: '1px solid var(--surface-border)',
                  background: 'transparent',
                  color: 'var(--text-muted)',
                  fontSize: 12,
                  cursor: 'default',
                }
          }
          disabled={!isDirty}
        >
          Save
        </button>

        <button
          onClick={() => setTemplatesOpen(true)}
          title="Browse pipeline templates"
          style={{
            padding: '6px 12px',
            minHeight: 32,
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--surface-border)',
            background: 'transparent',
            color: 'var(--accent)',
            fontSize: 12,
            cursor: 'pointer',
            fontWeight: 600,
          }}
        >
          Templates
        </button>

        <button
          onClick={handleExport}
          disabled={!currentPipelineId}
          title="Export pipeline as JSON"
          style={{
            padding: '6px 12px',
            minHeight: 32,
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--surface-border)',
            background: 'transparent',
            color: currentPipelineId ? 'var(--text-secondary)' : 'var(--surface-border)',
            fontSize: 12,
            cursor: currentPipelineId ? 'pointer' : 'default',
          }}
        >
          Export
        </button>

        <button
          onClick={() => importInputRef.current?.click()}
          disabled={!currentPipelineId}
          title="Import pipeline from JSON"
          style={{
            padding: '6px 12px',
            minHeight: 32,
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--surface-border)',
            background: 'transparent',
            color: currentPipelineId ? 'var(--text-secondary)' : 'var(--surface-border)',
            fontSize: 12,
            cursor: currentPipelineId ? 'pointer' : 'default',
          }}
        >
          Import
        </button>
        <input
          ref={importInputRef}
          type="file"
          accept=".json"
          onChange={handleImport}
          style={{ display: 'none' }}
        />

        <div style={{ flex: 1 }} />

        <button
          onClick={() => setSettingsOpen(true)}
          title="Settings"
          style={{
            padding: '6px 10px',
            minHeight: 32,
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--surface-border)',
            background: 'transparent',
            color: 'var(--text-secondary)',
            fontSize: 14,
            cursor: 'pointer',
            lineHeight: 1,
          }}
        >
          {'⚙'}
        </button>
        <button
          onClick={() => {
            toggleVersionPanel();
            if (!versionPanelIsOpen && currentPipelineId) {
              void fetchVersions(currentPipelineId);
            }
          }}
          style={{
            padding: '6px 10px',
            minHeight: 32,
            borderRadius: 'var(--radius-sm)',
            border: '1px solid var(--surface-border)',
            background: versionPanelIsOpen ? 'var(--accent-soft)' : 'transparent',
            color: versionPanelIsOpen ? 'var(--accent)' : 'var(--text-secondary)',
            fontSize: 12,
            cursor: 'pointer',
          }}
        >
          History
        </button>
        <ExecutionStatus />
        <RunButton />
      </header>

      {/* Main content */}
      <div style={{ display: 'flex', flex: 1, minHeight: 0, position: 'relative' }}>
        <NodePalette />
        <PipelineCanvas />
        <NodeInspector />
        <VersionPanel />
      </div>

      {/* Settings slide-over */}
      {settingsOpen && (
        <SettingsPanel onClose={() => setSettingsOpen(false)} />
      )}

      {/* Template gallery modal */}
      {templatesOpen && (
        <TemplateGallery onClose={() => setTemplatesOpen(false)} />
      )}
    </div>
  );
}

function AuthGate() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const [authView, setAuthView] = useState<'login' | 'register'>('login');

  if (!isAuthenticated) {
    return authView === 'login' ? (
      <LoginPage onSwitchToRegister={() => setAuthView('register')} />
    ) : (
      <RegisterPage onSwitchToLogin={() => setAuthView('login')} />
    );
  }

  return (
    <ReactFlowProvider>
      <AppContent />
    </ReactFlowProvider>
  );
}

export default function App() {
  return (
    <ErrorBoundary>
      <AuthGate />
    </ErrorBoundary>
  );
}
