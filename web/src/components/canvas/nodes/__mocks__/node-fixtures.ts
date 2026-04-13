/* ---------------------------------------------------------------------------
   Node fixtures for every node kind × state combination.
   Consumed by the Wave 4 dev route (/dev/nodes) and by unit tests.

   Agent (Lane E): 3 preset_categories × 5 states = 15
   Media (Lane K): 3 media_types × 5 states        = 15
   Tool  (Lane K): 3 tool_types × 5 states         = 15

   All fixtures are immutable data — consumers that need execution state
   should seed the execution store via `seedStoreFromFixture`.
   --------------------------------------------------------------------------- */

import type {
  AgentNodeConfig,
  AgentNodeData,
  MediaNodeConfig,
  MediaNodeData,
  MediaType,
  Port,
  StepResult,
  ToolNodeConfig,
  ToolNodeData,
  ToolType,
} from '../../../../lib/types';

export type FixturePresetCategory = 'script' | 'crafter' | 'generic';

export type FixtureState =
  | 'queued'
  | 'running'
  | 'complete'
  | 'empty-complete'
  | 'failed';

export interface NodeFixture {
  id: string;
  presetCategory: FixturePresetCategory;
  state: FixtureState;
  data: AgentNodeData;
  stepResult?: StepResult;
  selected: boolean;
}

// ---------------------------------------------------------------------------
// Base configs
// ---------------------------------------------------------------------------

const baseRole = (persona: string, prompt: string) => ({
  system_prompt: prompt,
  persona,
  variables: {},
});

const SCRIPT_CONFIG: AgentNodeConfig = {
  provider: 'openai',
  model: 'gpt-4o',
  role: baseRole('Script Writer', 'You are a cinematic screenwriter.'),
  task: {
    task_type: 'script',
    output_type: 'text',
    template: 'Write a short opening scene.',
  },
  temperature: 0.7,
  max_tokens: 2048,
  preset_category: 'script',
};

const CRAFTER_CONFIG: AgentNodeConfig = {
  provider: 'openai',
  model: 'gpt-4o-mini',
  role: baseRole('Image Prompt Crafter', 'You craft precise image prompts.'),
  task: {
    task_type: 'image_prompt',
    output_type: 'image_prompt',
    template: 'Turn the scene into a single image prompt.',
  },
  temperature: 0.3,
  max_tokens: 512,
  preset_category: 'crafter',
};

const GENERIC_CONFIG: AgentNodeConfig = {
  provider: 'openai',
  model: 'gpt-4o',
  role: baseRole('Custom Agent', 'You are a helpful agent.'),
  task: {
    task_type: 'custom',
    output_type: 'text',
    template: 'Do the thing.',
  },
  temperature: 0.6,
  max_tokens: 1024,
  // preset_category intentionally omitted → generic
};

const CONFIGS: Record<FixturePresetCategory, AgentNodeConfig> = {
  script: SCRIPT_CONFIG,
  crafter: CRAFTER_CONFIG,
  generic: GENERIC_CONFIG,
};

const LABELS: Record<FixturePresetCategory, string> = {
  script: 'Script Writer',
  crafter: 'Image Prompt Crafter',
  generic: 'Custom Agent',
};

const SAMPLE_OUTPUTS: Record<FixturePresetCategory, string> = {
  script:
    '"In the year of the great drought, the river spoke to her again — a low hum beneath dry reeds..."',
  crafter:
    'cinematic wide shot, warm amber dawn over cracked riverbed, reeds silhouetted, dust in the air, 35mm film grain',
  generic: 'The assistant produced a short summary of the requested item.',
};

// ---------------------------------------------------------------------------
// Port skeleton — shared shape; real graphs may differ.
// ---------------------------------------------------------------------------

const DEFAULT_PORTS: Port[] = [
  {
    id: 'in-1',
    name: 'input',
    type: 'text',
    direction: 'input',
    required: false,
  },
  {
    id: 'out-1',
    name: 'output',
    type: 'text',
    direction: 'output',
    required: false,
  },
];

// ---------------------------------------------------------------------------
// Builders
// ---------------------------------------------------------------------------

function buildData(category: FixturePresetCategory): AgentNodeData {
  return {
    nodeType: 'agent',
    label: LABELS[category],
    ports: DEFAULT_PORTS,
    config: CONFIGS[category],
  };
}

function buildStepResult(
  nodeId: string,
  category: FixturePresetCategory,
  state: FixtureState,
): StepResult | undefined {
  switch (state) {
    case 'queued':
      return {
        node_id: nodeId,
        status: 'pending',
      };
    case 'running':
      return {
        node_id: nodeId,
        status: 'running',
        output: SAMPLE_OUTPUTS[category].slice(0, 40),
        tokens_used: 64,
      };
    case 'complete':
      return {
        node_id: nodeId,
        status: 'completed',
        output: SAMPLE_OUTPUTS[category],
        tokens_used: 142,
        cost: 0.012,
        duration_ms: 2400,
      };
    case 'empty-complete':
      return {
        node_id: nodeId,
        status: 'completed',
        output: '',
        tokens_used: 0,
        cost: 0,
        duration_ms: 600,
      };
    case 'failed':
      return {
        node_id: nodeId,
        status: 'failed',
        error: 'Provider returned 429 — rate limited.',
        duration_ms: 1200,
      };
  }
}

// ---------------------------------------------------------------------------
// Fixture table — 3 × 5 = 15 combos
// ---------------------------------------------------------------------------

const CATEGORIES: FixturePresetCategory[] = ['script', 'crafter', 'generic'];
const STATES: FixtureState[] = [
  'queued',
  'running',
  'complete',
  'empty-complete',
  'failed',
];

export const NODE_FIXTURES: NodeFixture[] = CATEGORIES.flatMap((category) =>
  STATES.map<NodeFixture>((state) => {
    const id = `fixture-${category}-${state}`;
    return {
      id,
      presetCategory: category,
      state,
      data: buildData(category),
      stepResult: buildStepResult(id, category, state),
      selected: false,
    };
  }),
);

export function getFixture(
  category: FixturePresetCategory,
  state: FixtureState,
): NodeFixture {
  const match = NODE_FIXTURES.find(
    (f) => f.presetCategory === category && f.state === state,
  );
  if (!match) {
    throw new Error(`No fixture for ${category} × ${state}`);
  }
  return match;
}

// ===========================================================================
// MEDIA fixtures (Lane K)
// ---------------------------------------------------------------------------
// 3 media types × 5 states = 15 fixtures.
// ===========================================================================

export interface MediaNodeFixture {
  id: string;
  mediaType: MediaType;
  state: FixtureState;
  data: MediaNodeData;
  stepResult?: StepResult;
  selected: boolean;
}

const MEDIA_CONFIGS: Record<MediaType, MediaNodeConfig> = {
  image: {
    media_type: 'image',
    provider: 'openai',
    model: 'dall-e-3',
    prompt: 'cinematic wide shot, warm amber dawn',
    aspect_ratio: '16:9',
    num_outputs: 1,
  },
  video: {
    media_type: 'video',
    provider: 'replicate',
    model: 'kling-v1.5',
    prompt: 'slow push-in on a river at dawn',
    aspect_ratio: '16:9',
    duration: 5,
  },
  audio: {
    media_type: 'audio',
    provider: 'openai',
    model: 'tts-1-hd',
    prompt: 'warm narrator reading the opening line',
    voice: 'alloy',
  },
};

const MEDIA_LABELS: Record<MediaType, string> = {
  image: 'Image Generator',
  video: 'Video Generator',
  audio: 'Audio Generator',
};

// Tiny 1×1 transparent PNG so the browser renders something without a network call.
const SAMPLE_IMAGE_URL =
  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==';

const MEDIA_SAMPLE_OUTPUT: Record<MediaType, string> = {
  image: SAMPLE_IMAGE_URL,
  video: SAMPLE_IMAGE_URL, // stand-in thumbnail for reel frames
  audio: 'audio://generated-clip',
};

const MEDIA_PORTS: Port[] = [
  { id: 'in-1', name: 'prompt', type: 'text', direction: 'input', required: true },
  { id: 'out-1', name: 'media', type: 'image', direction: 'output', required: false },
];

function buildMediaData(mediaType: MediaType): MediaNodeData {
  return {
    nodeType: 'media',
    label: MEDIA_LABELS[mediaType],
    ports: MEDIA_PORTS,
    config: MEDIA_CONFIGS[mediaType],
  };
}

function buildMediaStepResult(
  nodeId: string,
  mediaType: MediaType,
  state: FixtureState,
): StepResult | undefined {
  switch (state) {
    case 'queued':
      return { node_id: nodeId, status: 'pending' };
    case 'running':
      return {
        node_id: nodeId,
        status: 'running',
        tokens_used: 0,
      };
    case 'complete':
      return {
        node_id: nodeId,
        status: 'completed',
        output: MEDIA_SAMPLE_OUTPUT[mediaType],
        cost: 0.04,
        duration_ms: 6200,
      };
    case 'empty-complete':
      return {
        node_id: nodeId,
        status: 'completed',
        output: '',
        cost: 0,
        duration_ms: 900,
      };
    case 'failed':
      return {
        node_id: nodeId,
        status: 'failed',
        error: 'Provider rejected prompt (safety filter).',
        duration_ms: 1400,
      };
  }
}

const MEDIA_TYPES: MediaType[] = ['image', 'video', 'audio'];

export const MEDIA_FIXTURES: MediaNodeFixture[] = MEDIA_TYPES.flatMap((mediaType) =>
  STATES.map<MediaNodeFixture>((state) => {
    const id = `fixture-media-${mediaType}-${state}`;
    return {
      id,
      mediaType,
      state,
      data: buildMediaData(mediaType),
      stepResult: buildMediaStepResult(id, mediaType, state),
      selected: false,
    };
  }),
);

// ===========================================================================
// TOOL fixtures (Lane K)
// ---------------------------------------------------------------------------
// ToolNode doesn't read execution state, but we still keep state-keyed fixtures
// so the dev route can render the full matrix consistently. The state is
// represented purely visually by changing the content/label.
// 3 tool types × 5 states = 15 fixtures.
// ===========================================================================

export interface ToolNodeFixture {
  id: string;
  toolType: ToolType;
  state: FixtureState;
  data: ToolNodeData;
  stepResult?: StepResult;
  selected: boolean;
}

const TOOL_TYPES: ToolType[] = ['text_box', 'image_box', 'video_box'];

const TOOL_LABELS: Record<ToolType, string> = {
  text_box: 'Text Box',
  image_box: 'Image Box',
  video_box: 'Video Box',
};

const TOOL_PORTS: Port[] = [
  { id: 'out-1', name: 'content', type: 'text', direction: 'output', required: false },
];

function buildToolData(toolType: ToolType, state: FixtureState): ToolNodeData {
  const config: ToolNodeConfig = (() => {
    if (toolType === 'text_box') {
      return {
        tool_type: 'text_box',
        content:
          state === 'empty-complete'
            ? ''
            : 'Opening scene: a dry riverbed at dawn. The wind carries the sound of bells from the valley below.',
      };
    }
    // image_box / video_box
    return {
      tool_type: toolType,
      media_url: state === 'empty-complete' ? '' : SAMPLE_IMAGE_URL,
    };
  })();

  return {
    nodeType: 'tool',
    label: TOOL_LABELS[toolType],
    ports: TOOL_PORTS,
    config,
  };
}

function buildToolStepResult(
  nodeId: string,
  state: FixtureState,
): StepResult | undefined {
  // ToolNode itself doesn't render step results, but the BaseNode does pick up
  // the status class for styling — so we still seed one.
  switch (state) {
    case 'queued':
      return { node_id: nodeId, status: 'pending' };
    case 'running':
      return { node_id: nodeId, status: 'running' };
    case 'complete':
      return {
        node_id: nodeId,
        status: 'completed',
        output: 'passthrough',
        duration_ms: 50,
      };
    case 'empty-complete':
      return {
        node_id: nodeId,
        status: 'completed',
        output: '',
        duration_ms: 20,
      };
    case 'failed':
      return {
        node_id: nodeId,
        status: 'failed',
        error: 'Tool content missing.',
      };
  }
}

export const TOOL_FIXTURES: ToolNodeFixture[] = TOOL_TYPES.flatMap((toolType) =>
  STATES.map<ToolNodeFixture>((state) => {
    const id = `fixture-tool-${toolType}-${state}`;
    return {
      id,
      toolType,
      state,
      data: buildToolData(toolType, state),
      stepResult: buildToolStepResult(id, state),
      selected: false,
    };
  }),
);

// ===========================================================================
// Unified export grouped by kind
// ===========================================================================

export interface AllFixtures {
  agent: NodeFixture[];
  media: MediaNodeFixture[];
  tool: ToolNodeFixture[];
}

export const ALL_FIXTURES: AllFixtures = {
  agent: NODE_FIXTURES,
  media: MEDIA_FIXTURES,
  tool: TOOL_FIXTURES,
};
