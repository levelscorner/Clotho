import { create } from 'zustand';
import type { Project, Pipeline } from '../lib/types';
import { api } from '../lib/api';

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

interface ProjectState {
  projects: Project[];
  currentProjectId: string | null;
  pipelines: Pipeline[];
  currentPipelineId: string | null;

  fetchProjects: () => Promise<void>;
  createProject: (name: string, description?: string) => Promise<Project>;
  fetchPipelines: (projectId: string) => Promise<void>;
  createPipeline: (projectId: string, name: string) => Promise<Pipeline>;
  setCurrentProject: (id: string) => void;
  setCurrentPipeline: (id: string) => void;
}

export const useProjectStore = create<ProjectState>((set) => ({
  projects: [],
  currentProjectId: null,
  pipelines: [],
  currentPipelineId: null,

  fetchProjects: async () => {
    const projects = await api.get<Project[]>('/projects');
    set({ projects });
  },

  createProject: async (name, description) => {
    const project = await api.post<Project>('/projects', {
      name,
      description: description ?? '',
    });
    set((state) => ({ projects: [...state.projects, project] }));
    return project;
  },

  fetchPipelines: async (projectId) => {
    const pipelines = await api.get<Pipeline[]>(
      `/projects/${projectId}/pipelines`,
    );
    set({ pipelines, currentProjectId: projectId });
  },

  createPipeline: async (projectId, name) => {
    const pipeline = await api.post<Pipeline>(
      `/projects/${projectId}/pipelines`,
      { name },
    );
    set((state) => ({ pipelines: [...state.pipelines, pipeline] }));
    return pipeline;
  },

  setCurrentProject: (id) => {
    set({ currentProjectId: id });
  },

  setCurrentPipeline: (id) => {
    set({ currentPipelineId: id });
  },
}));
