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

export type NodeType = 'agent' | 'tool';

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
}

export interface ToolNodeConfig {
  tool_type: ToolType;
  content?: string;
  media_url?: string;
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
  config: AgentNodeConfig | ToolNodeConfig;
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
// Presets
// ---------------------------------------------------------------------------

export interface AgentPreset {
  id: string;
  name: string;
  description: string;
  category: string;
  config: AgentNodeConfig;
  icon: string;
  is_built_in: boolean;
}

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

export interface StepResult {
  node_id: string;
  status: ExecutionStatus;
  output?: string;
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

export type PipelineNodeData = AgentNodeData | ToolNodeData;

// ---------------------------------------------------------------------------
// Provider info (from GET /api/providers)
// ---------------------------------------------------------------------------

export interface ProviderInfo {
  name: string;
  available: boolean;
  models: string[];
}
