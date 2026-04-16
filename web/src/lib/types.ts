// ---------------------------------------------------------------------------
// Domain types mirroring Go backend
// ---------------------------------------------------------------------------

export type PortType =
  | 'text'
  | 'image_prompt'
  | 'video_prompt'
  | 'audio_prompt'
  | 'image'
  | 'video'
  | 'audio'
  | 'json'
  | 'any';

export type PortDirection = 'input' | 'output';

export type NodeType = 'agent' | 'tool' | 'media';

export type ToolType = 'text_box' | 'image_box' | 'video_box';

export type TaskType =
  | 'script'
  | 'image_prompt'
  | 'video_prompt'
  | 'audio_prompt'
  | 'character_prompt'
  | 'story'
  | 'prompt_enhancement'
  | 'story_to_prompt'
  | 'custom';

export type ExecutionStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'skipped';

// ---------------------------------------------------------------------------
// Port & Position
// ---------------------------------------------------------------------------

export interface Port {
  id: string;
  name: string;
  type: PortType;
  direction: PortDirection;
  required: boolean;
}

export interface Position {
  x: number;
  y: number;
}

// ---------------------------------------------------------------------------
// Node configuration
// ---------------------------------------------------------------------------

export interface RoleConfig {
  system_prompt: string;
  persona: string;
  variables?: Record<string, string>;
}

export interface TaskConfig {
  task_type: TaskType;
  output_type: PortType;
  template: string;
  output_schema?: unknown;
}

export interface AgentNodeConfig {
  provider: string;
  model: string;
  role: RoleConfig;
  task: TaskConfig;
  temperature: number;
  max_tokens: number;
  cost_cap?: number;
  credential_id?: string;

  // Near-universal sampling knobs — optional, provider-gated via
  // web/src/lib/llmCapabilities.ts. Undefined means "use provider default".
  top_p?: number;
  top_k?: number;
  stop_sequences?: string[];
  seed?: number;
  frequency_penalty?: number;
  presence_penalty?: number;

  // Reliability knobs (Phase A). step_timeout_sec overrides the engine's
  // 120s default; max_retries overrides the 3-attempt default for
  // retryable failures.
  step_timeout_sec?: number;
  max_retries?: number;

  // Free-form annotation surfaced in the inspector. Engine never reads
  // it; it's a breadcrumb for the creator (e.g. "this one breaks above
  // 4k tokens"). Stored as part of the node config blob.
  notes?: string;
}

export interface ToolNodeConfig {
  tool_type: ToolType;
  content?: string;
  media_url?: string;
}

// ---------------------------------------------------------------------------
// Media node configuration
// ---------------------------------------------------------------------------

export type MediaType = 'image' | 'video' | 'audio';

export interface MediaNodeConfig {
  media_type: MediaType;
  provider: string;
  model: string;
  credential_id?: string;
  prompt: string;
  aspect_ratio?: string;
  voice?: string;
  duration?: number;
  num_outputs?: number;
  cost_cap?: number;
}

// ---------------------------------------------------------------------------
// Graph primitives
// ---------------------------------------------------------------------------

export interface NodeInstance {
  id: string;
  type: NodeType;
  label: string;
  position: Position;
  ports: Port[];
  config: AgentNodeConfig | ToolNodeConfig | MediaNodeConfig;
}

export interface Edge {
  id: string;
  source: string;
  source_port: string;
  target: string;
  target_port: string;
}

export interface Viewport {
  x: number;
  y: number;
  zoom: number;
}

export interface PipelineGraph {
  nodes: NodeInstance[];
  edges: Edge[];
  viewport: Viewport;
}

// ---------------------------------------------------------------------------
// Presets removed — personalities dropped from the product.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// API resource types
// ---------------------------------------------------------------------------

export interface Project {
  id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface Pipeline {
  id: string;
  project_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface PipelineVersion {
  id: string;
  pipeline_id: string;
  version: number;
  graph: PipelineGraph;
  created_at: string;
}

// FailureClass mirrors internal/domain/failure.go. Adding a new value
// here also requires updating the badge color map in
// web/src/components/execution/FailureDrawer.tsx.
export type FailureClass =
  | 'network'
  | 'rate_limit'
  | 'timeout'
  | 'auth'
  | 'provider_5xx'
  | 'provider_4xx'
  | 'validation'
  | 'output_shape'
  | 'output_quality'
  | 'cost_cap'
  | 'circuit_open'
  | 'internal';

export type FailureStage =
  | 'input_resolve'
  | 'provider_call'
  | 'stream_parse'
  | 'output_validate'
  | 'persist';

export interface StepFailure {
  class: FailureClass;
  stage: FailureStage;
  provider?: string;
  model?: string;
  retryable: boolean;
  message: string;
  cause?: string;
  hint?: string;
  attempts: number;
  at: string;
}

export interface StepResult {
  node_id: string;
  status: ExecutionStatus;
  output?: string;
  /**
   * Structured failure payload populated when status === 'failed'.
   * Mirrors internal/domain/failure.go::StepFailure. Use this in the
   * FailureDrawer + node tooltips. The legacy `error` field still
   * carries the 1-line summary for back-compat.
   */
  failure?: StepFailure;
  /**
   * Optional `clotho://file/…` URL pointing to an on-disk artifact for
   * this node — set when an agent wrote its text to a .txt file or a
   * media node produced an image/audio/video asset. Tools have none.
   * Populated from the `output_file` field on the step_completed SSE
   * event.
   */
  output_file?: string;
  error?: string;
  tokens_used?: number;
  cost?: number;
  duration_ms?: number;
  started_at?: string;
  completed_at?: string;
}

export interface Execution {
  id: string;
  pipeline_id: string;
  version_id: string;
  status: ExecutionStatus;
  steps: StepResult[];
  total_cost: number;
  started_at: string;
  completed_at?: string;
}

export interface Credential {
  id: string;
  provider: string;
  label: string;
  created_at: string;
}

// ---------------------------------------------------------------------------
// React Flow node data shapes
//
// React Flow v12 requires node data to satisfy Record<string, unknown>.
// We use `type` with an explicit index signature so TS is happy.
// ---------------------------------------------------------------------------

export type AgentNodeData = {
  [key: string]: unknown;
  nodeType: 'agent';
  label: string;
  ports: Port[];
  config: AgentNodeConfig;
};

export type ToolNodeData = {
  [key: string]: unknown;
  nodeType: 'tool';
  label: string;
  ports: Port[];
  config: ToolNodeConfig;
};

export type MediaNodeData = {
  [key: string]: unknown;
  nodeType: 'media';
  label: string;
  ports: Port[];
  config: MediaNodeConfig;
};

export type PipelineNodeData = AgentNodeData | ToolNodeData | MediaNodeData;

// ---------------------------------------------------------------------------
// Provider info (from GET /api/providers)
// ---------------------------------------------------------------------------

export interface ProviderInfo {
  name: string;
  available: boolean;
  models: string[];
}
