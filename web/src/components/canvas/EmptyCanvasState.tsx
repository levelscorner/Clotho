import { useCallback, useEffect, useState } from 'react';
import { useReactFlow } from '@xyflow/react';
import { usePipelineStore } from '../../stores/pipelineStore';
import type {
  AgentNodeConfig,
  MediaNodeConfig,
  Port,
} from '../../lib/types';
import './EmptyCanvasState.css';

// ---------------------------------------------------------------------------
// Dismissal persistence
// ---------------------------------------------------------------------------

const DISMISS_KEY = 'clotho.empty-state.dismissed';

function getStorage(): Storage | null {
  const ls =
    typeof globalThis !== 'undefined'
      ? (globalThis as unknown as { localStorage?: Storage }).localStorage
      : undefined;
  return ls ?? null;
}

function readDismissed(): boolean {
  const storage = getStorage();
  if (!storage) return false;
  try {
    return storage.getItem(DISMISS_KEY) === '1';
  } catch {
    return false;
  }
}

function markDismissed(): void {
  const storage = getStorage();
  if (!storage) return;
  try {
    storage.setItem(DISMISS_KEY, '1');
  } catch {
    // Ignore quota errors — nothing to persist if storage is unavailable.
  }
}

// ---------------------------------------------------------------------------
// Sample pipeline: Script Writer → Image Prompt Crafter → Image media node
// ---------------------------------------------------------------------------

function scriptWriterConfig(): {
  config: AgentNodeConfig;
  ports: Port[];
  label: string;
} {
  const config: AgentNodeConfig = {
    provider: 'openai',
    model: 'gpt-4o',
    role: {
      system_prompt:
        'You are a vivid, concise screenwriter who writes short cinematic scenes.',
      persona: 'Cinematic screenwriter',
    },
    task: {
      task_type: 'script',
      output_type: 'text',
      template: 'Write a 3-sentence opening scene.',
    },
    temperature: 0.8,
    max_tokens: 1024,
  };
  const ports: Port[] = [
    { id: 'in_text', name: 'Input', type: 'any', direction: 'input', required: false },
    { id: 'out_text', name: 'Script', type: 'text', direction: 'output', required: false },
  ];
  return { config, ports, label: 'Script Writer' };
}

function imagePromptCrafterConfig(): {
  config: AgentNodeConfig;
  ports: Port[];
  label: string;
} {
  const config: AgentNodeConfig = {
    provider: 'openai',
    model: 'gpt-4o',
    role: {
      system_prompt:
        'You turn narrative scenes into vivid image generation prompts.',
      persona: 'Image prompt crafter',
    },
    task: {
      task_type: 'image_prompt',
      output_type: 'image_prompt',
      template: 'Turn this scene into an image prompt.',
    },
    temperature: 0.6,
    max_tokens: 512,
  };
  const ports: Port[] = [
    { id: 'in_text', name: 'Scene', type: 'text', direction: 'input', required: true },
    { id: 'out_prompt', name: 'Prompt', type: 'image_prompt', direction: 'output', required: false },
  ];
  return { config, ports, label: 'Image Prompt Crafter' };
}

function imageMediaConfig(): {
  config: MediaNodeConfig;
  ports: Port[];
  label: string;
} {
  const config: MediaNodeConfig = {
    media_type: 'image',
    provider: 'replicate',
    model: 'flux-1.1-pro',
    prompt: '',
    aspect_ratio: '16:9',
    num_outputs: 1,
  };
  const ports: Port[] = [
    { id: 'in_prompt', name: 'Prompt', type: 'image_prompt', direction: 'input', required: true },
    { id: 'out_image', name: 'Image', type: 'image', direction: 'output', required: false },
  ];
  return { config, ports, label: 'Image' };
}

function loadSamplePipeline(): void {
  const store = usePipelineStore.getState();

  const script = scriptWriterConfig();
  const crafter = imagePromptCrafterConfig();
  const image = imageMediaConfig();

  store.addNode('agent', { x: 80, y: 160 }, script.config, script.ports, script.label);
  store.addNode('agent', { x: 360, y: 160 }, crafter.config, crafter.ports, crafter.label);
  store.addNode('media', { x: 640, y: 160 }, image.config, image.ports, image.label);

  // Nodes just added — read fresh state to connect them.
  const nodes = usePipelineStore.getState().nodes;
  const [n1, n2, n3] = nodes.slice(-3);
  if (!n1 || !n2 || !n3) return;

  store.onConnect({
    source: n1.id,
    sourceHandle: 'out_text',
    target: n2.id,
    targetHandle: 'in_text',
  });
  store.onConnect({
    source: n2.id,
    sourceHandle: 'out_prompt',
    target: n3.id,
    targetHandle: 'in_prompt',
  });
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function EmptyCanvasState(): JSX.Element | null {
  const nodeCount = usePipelineStore((s) => s.nodes.length);
  const pipelineId = usePipelineStore((s) => s.pipelineId);
  const [dismissed, setDismissed] = useState<boolean>(() => readDismissed());
  // Track whether the component has ever been visible during this session.
  // We only auto-dismiss and mark localStorage when the user adds nodes
  // while the empty state is actively shown — not when the pipeline loads
  // with pre-existing nodes (isDirty=false on initial load).
  const [wasVisible, setWasVisible] = useState<boolean>(false);

  // Reset dismissed when the pipeline switches and the new pipeline is empty
  // so a fresh empty pipeline shows the empty state regardless of prior dismissal.
  useEffect(() => {
    if (nodeCount === 0) {
      // Clear the global flag so empty pipelines always show the onboarding state.
      const storage = getStorage();
      try { storage?.removeItem(DISMISS_KEY); } catch { /* ignore */ }
      setDismissed(false);
    }
    setWasVisible(false);
  // Only re-run when the pipeline identity changes, not on node changes.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pipelineId]);

  useEffect(() => {
    // Only auto-dismiss via localStorage if the empty state was shown first
    // (wasVisible) and the user then added nodes (nodeCount > 0).
    // This prevents auto-dismissal when the pipeline loads pre-populated.
    if (nodeCount > 0 && wasVisible && !dismissed) {
      markDismissed();
      setDismissed(true);
    }
  }, [nodeCount, dismissed, wasVisible]);

  const { fitView } = useReactFlow();
  const onLoadSample = useCallback(() => {
    loadSamplePipeline();
    markDismissed();
    setDismissed(true);
    // Wait a frame for React Flow to register the new nodes, then fit so
    // the entire sample graph is visible on the current viewport regardless
    // of prior zoom/pan state.
    requestAnimationFrame(() => {
      fitView({ padding: 0.15, duration: 300 });
    });
  }, [fitView]);

  // Determine visibility before effects so the "wasVisible" marker is accurate.
  const shouldShow =
    nodeCount === 0 && Boolean(pipelineId) && !dismissed;

  // Mark as visible once we've shown the empty state at least once.
  useEffect(() => {
    if (shouldShow && !wasVisible) {
      setWasVisible(true);
    }
  }, [shouldShow, wasVisible]);

  // When a pipeline loaded from the API already has nodes, hide without
  // marking dismissed so a later empty pipeline shows the state correctly.
  if (!shouldShow) return null;

  return (
    <>
      <div className="empty-canvas" aria-hidden="true">
        <div className="empty-canvas__cluster">
          <Ghost icon="✎" title="Script Writer" variant="script">
            &ldquo;In the year of the…&rdquo;
          </Ghost>
          <span className="empty-canvas__conn empty-canvas__conn--text" />
          <Ghost icon="◉" title="Image" variant="matte" />
          <span className="empty-canvas__conn empty-canvas__conn--image" />
          <Ghost icon="▶" title="Video" variant="reel" />
        </div>

        <div className="empty-canvas__cta-wrap">
          <button
            type="button"
            className="empty-canvas__cta"
            onClick={onLoadSample}
            aria-label="Load sample pipeline"
          >
            LOAD SAMPLE PIPELINE
          </button>
          <span className="empty-canvas__hint">
            or drag your own node from the left
          </span>
        </div>
      </div>
      <div className="empty-canvas__hint-corner" aria-hidden="true">
        or press ⌘K for templates
      </div>
    </>
  );
}

interface GhostProps {
  icon: string;
  title: string;
  variant: 'script' | 'matte' | 'reel';
  children?: React.ReactNode;
}

function Ghost({ icon, title, variant, children }: GhostProps) {
  return (
    <div className="empty-canvas__ghost">
      <div className="empty-canvas__ghost-header">
        <span className="empty-canvas__ghost-icon">{icon}</span>
        <span className="empty-canvas__ghost-title">{title}</span>
        <span className="empty-canvas__ghost-dot" />
      </div>
      <div className={`empty-canvas__ghost-body empty-canvas__ghost-body--${variant}`}>
        {variant === 'reel' ? (
          <div className="empty-canvas__ghost-frames">
            <span />
            <span />
            <span />
          </div>
        ) : (
          children
        )}
      </div>
    </div>
  );
}
