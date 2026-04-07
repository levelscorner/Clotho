// ---------------------------------------------------------------------------
// Typed fetch wrapper for the Clotho backend API
// ---------------------------------------------------------------------------

import type { Credential } from './types';

const BASE = '/api';

class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// ---------------------------------------------------------------------------
// Auth header + 401 refresh logic
// ---------------------------------------------------------------------------

let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

function getAccessToken(): string | null {
  return localStorage.getItem('clotho_access_token');
}

async function tryRefreshToken(): Promise<boolean> {
  const refreshToken = localStorage.getItem('clotho_refresh_token');
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return false;

    const data = (await res.json()) as {
      access_token: string;
      refresh_token: string;
      user: { id: string; email: string; name: string };
    };

    localStorage.setItem('clotho_access_token', data.access_token);
    localStorage.setItem('clotho_refresh_token', data.refresh_token);
    localStorage.setItem('clotho_user', JSON.stringify(data.user));
    return true;
  } catch {
    return false;
  }
}

function clearAuthAndRedirect(): void {
  localStorage.removeItem('clotho_access_token');
  localStorage.removeItem('clotho_refresh_token');
  localStorage.removeItem('clotho_user');
  window.location.reload();
}

// ---------------------------------------------------------------------------
// Core request
// ---------------------------------------------------------------------------

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const url = `${BASE}${path}`;

  const headers: Record<string, string> = {};
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
  }

  const token = getAccessToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(url, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  // Handle 401 — attempt token refresh once, then retry
  if (res.status === 401 && token) {
    if (!isRefreshing) {
      isRefreshing = true;
      refreshPromise = tryRefreshToken().finally(() => {
        isRefreshing = false;
        refreshPromise = null;
      });
    }

    const refreshed = await (refreshPromise ?? Promise.resolve(false));

    if (refreshed) {
      // Retry original request with new token
      const retryHeaders: Record<string, string> = {};
      if (body !== undefined) {
        retryHeaders['Content-Type'] = 'application/json';
      }
      const newToken = getAccessToken();
      if (newToken) {
        retryHeaders['Authorization'] = `Bearer ${newToken}`;
      }

      const retryRes = await fetch(url, {
        method,
        headers: retryHeaders,
        body: body !== undefined ? JSON.stringify(body) : undefined,
      });

      if (!retryRes.ok) {
        if (retryRes.status === 401) {
          clearAuthAndRedirect();
        }
        const text = await retryRes.text().catch(() => retryRes.statusText);
        throw new ApiError(retryRes.status, text);
      }

      const contentLength = retryRes.headers.get('Content-Length');
      if (retryRes.status === 204 || contentLength === '0') {
        return undefined as T;
      }
      return retryRes.json() as Promise<T>;
    }

    clearAuthAndRedirect();
    throw new ApiError(401, 'Session expired');
  }

  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new ApiError(res.status, text);
  }

  // 204 No Content or empty body
  const contentLength = res.headers.get('Content-Length');
  if (res.status === 204 || contentLength === '0') {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

// ---------------------------------------------------------------------------
// Convenience methods
// ---------------------------------------------------------------------------

async function get<T>(path: string): Promise<T> {
  return request<T>('GET', path);
}

async function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>('POST', path, body);
}

async function put<T>(path: string, body: unknown): Promise<T> {
  return request<T>('PUT', path, body);
}

async function del(path: string): Promise<void> {
  await request<void>('DELETE', path);
}

// ---------------------------------------------------------------------------
// Credential API
// ---------------------------------------------------------------------------

const credentials = {
  list: () => request<Credential[]>('GET', '/credentials'),
  create: (data: { provider: string; label: string; api_key: string }) =>
    request<Credential>('POST', '/credentials', data),
  delete: (id: string) => request<void>('DELETE', `/credentials/${id}`),
};

// ---------------------------------------------------------------------------
// Pipeline Export / Import
// ---------------------------------------------------------------------------

async function exportPipeline(id: string): Promise<Blob> {
  const url = `${BASE}/pipelines/${id}/export`;
  const headers: Record<string, string> = {};
  const token = getAccessToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const res = await fetch(url, { headers });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new ApiError(res.status, text);
  }
  return res.blob();
}

function importPipeline(id: string, data: unknown) {
  return request<{ id: string }>('POST', `/pipelines/${id}/import`, data);
}

// ---------------------------------------------------------------------------
// Template API
// ---------------------------------------------------------------------------

export interface TemplateSummary {
  id: string;
  name: string;
  description: string;
  category: string;
  node_count: number;
}

export interface TemplateDetail {
  id: string;
  name: string;
  description: string;
  category: string;
  node_count: number;
  graph: import('./types').PipelineGraph;
}

const templateApi = {
  list: () => get<TemplateSummary[]>('/templates'),
  get: (id: string) => get<TemplateDetail>(`/templates/${id}`),
};

export const api = { get, post, put, del, credentials, exportPipeline, importPipeline, templates: templateApi };

export { ApiError };
