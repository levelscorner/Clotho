import { create } from 'zustand';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface AuthUser {
  id: string;
  email: string;
  name: string;
}

interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user: AuthUser;
}

interface AuthState {
  token: string | null;
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => void;
  refreshToken: () => Promise<void>;
  clearError: () => void;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const STORAGE_KEY_TOKEN = 'clotho_access_token';
const STORAGE_KEY_REFRESH = 'clotho_refresh_token';
const STORAGE_KEY_USER = 'clotho_user';

function loadPersistedAuth(): {
  token: string | null;
  user: AuthUser | null;
} {
  try {
    const token = localStorage.getItem(STORAGE_KEY_TOKEN);
    const userJson = localStorage.getItem(STORAGE_KEY_USER);
    const user = userJson ? (JSON.parse(userJson) as AuthUser) : null;
    return { token, user };
  } catch {
    return { token: null, user: null };
  }
}

function persistAuth(tokens: AuthTokens): void {
  localStorage.setItem(STORAGE_KEY_TOKEN, tokens.access_token);
  localStorage.setItem(STORAGE_KEY_REFRESH, tokens.refresh_token);
  localStorage.setItem(STORAGE_KEY_USER, JSON.stringify(tokens.user));
}

function clearPersistedAuth(): void {
  localStorage.removeItem(STORAGE_KEY_TOKEN);
  localStorage.removeItem(STORAGE_KEY_REFRESH);
  localStorage.removeItem(STORAGE_KEY_USER);
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

const NO_AUTH =
  (import.meta.env.VITE_NO_AUTH as string | undefined) === 'true';

const LOCAL_DEV_USER: AuthUser = {
  id: '00000000-0000-0000-0000-000000000001',
  email: 'you@local',
  name: 'Local Dev',
};

const initial = NO_AUTH
  ? { token: 'no-auth', user: LOCAL_DEV_USER }
  : loadPersistedAuth();

export const useAuthStore = create<AuthState>((set) => ({
  token: initial.token,
  user: initial.user,
  isAuthenticated: NO_AUTH ? true : initial.token !== null,
  isLoading: false,
  error: null,

  login: async (email: string, password: string) => {
    if (NO_AUTH) {
      set({ token: 'no-auth', user: LOCAL_DEV_USER, isAuthenticated: true });
      return;
    }
    set({ isLoading: true, error: null });
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      if (!res.ok) {
        const text = await res.text().catch(() => res.statusText);
        throw new Error(text || 'Login failed');
      }

      const tokens = (await res.json()) as AuthTokens;
      persistAuth(tokens);
      set({
        token: tokens.access_token,
        user: tokens.user,
        isAuthenticated: true,
        isLoading: false,
      });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Login failed';
      set({ isLoading: false, error: message });
      throw err;
    }
  },

  register: async (email: string, password: string, name: string) => {
    if (NO_AUTH) {
      set({ token: 'no-auth', user: LOCAL_DEV_USER, isAuthenticated: true });
      return;
    }
    set({ isLoading: true, error: null });
    try {
      const res = await fetch('/api/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, name }),
      });

      if (!res.ok) {
        const text = await res.text().catch(() => res.statusText);
        throw new Error(text || 'Registration failed');
      }

      const tokens = (await res.json()) as AuthTokens;
      persistAuth(tokens);
      set({
        token: tokens.access_token,
        user: tokens.user,
        isAuthenticated: true,
        isLoading: false,
      });
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Registration failed';
      set({ isLoading: false, error: message });
      throw err;
    }
  },

  logout: () => {
    if (NO_AUTH) {
      // No-op in unauthenticated mode — identity is ambient.
      return;
    }
    clearPersistedAuth();
    set({ token: null, user: null, isAuthenticated: false, error: null });
  },

  refreshToken: async () => {
    if (NO_AUTH) {
      return;
    }
    const refreshToken = localStorage.getItem(STORAGE_KEY_REFRESH);
    if (!refreshToken) {
      clearPersistedAuth();
      set({ token: null, user: null, isAuthenticated: false });
      return;
    }

    try {
      const res = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      if (!res.ok) {
        clearPersistedAuth();
        set({ token: null, user: null, isAuthenticated: false });
        return;
      }

      const tokens = (await res.json()) as AuthTokens;
      persistAuth(tokens);
      set({
        token: tokens.access_token,
        user: tokens.user,
        isAuthenticated: true,
      });
    } catch {
      clearPersistedAuth();
      set({ token: null, user: null, isAuthenticated: false });
    }
  },

  clearError: () => set({ error: null }),
}));
