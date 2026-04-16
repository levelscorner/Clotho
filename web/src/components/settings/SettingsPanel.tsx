import { useState, useEffect, useCallback, useRef } from 'react';
import type { Credential } from '../../lib/types';
import { api, type CredentialTestResult } from '../../lib/api';
import { useAuthStore } from '../../stores/authStore';
import { useFocusTrap } from '../../hooks/useFocusTrap';

// ---------------------------------------------------------------------------
// Styles (DESIGN.md tokens)
// ---------------------------------------------------------------------------

// zIndex migrated to token — see global.css .clotho-z-modal
const overlayStyle: React.CSSProperties = {
  position: 'fixed',
  inset: 0,
  background: 'rgba(0, 0, 0, 0.6)',
  display: 'flex',
  justifyContent: 'flex-end',
};

const panelStyle: React.CSSProperties = {
  width: 400,
  height: '100%',
  background: '#1a1a20',
  borderLeft: '1px solid #2e2e38',
  overflowY: 'auto',
  padding: 24,
  fontFamily: "'Inter', sans-serif",
};

const headingStyle: React.CSSProperties = {
  fontSize: 14,
  fontWeight: 600,
  color: '#ececf0',
  marginBottom: 20,
};

const sectionHeadingStyle: React.CSSProperties = {
  fontSize: 11,
  fontWeight: 600,
  color: '#8888a0',
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  marginBottom: 12,
  marginTop: 24,
};

const credCardStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  padding: '10px 12px',
  background: '#222228',
  borderRadius: 6,
  marginBottom: 8,
  border: '1px solid #2e2e38',
};

const credProviderStyle: React.CSSProperties = {
  fontSize: 12,
  fontWeight: 600,
  color: '#ececf0',
  textTransform: 'capitalize',
};

const credLabelStyle: React.CSSProperties = {
  fontSize: 11,
  color: '#8888a0',
  marginTop: 2,
};

const deleteButtonStyle: React.CSSProperties = {
  padding: '4px 10px',
  borderRadius: 4,
  border: '1px solid rgba(248, 113, 113, 0.3)',
  background: 'transparent',
  color: '#f87171',
  fontSize: 11,
  cursor: 'pointer',
  fontFamily: "'Inter', sans-serif",
};

const fieldGroup: React.CSSProperties = {
  marginBottom: 12,
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 11,
  fontWeight: 600,
  color: '#8888a0',
  marginBottom: 6,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
};

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '8px 10px',
  borderRadius: 6,
  border: '1px solid #2e2e38',
  background: '#222228',
  color: '#ececf0',
  fontSize: 13,
  fontFamily: "'Inter', sans-serif",
  outline: 'none',
  boxSizing: 'border-box',
};

const addButtonStyle: React.CSSProperties = {
  padding: '8px 16px',
  borderRadius: 6,
  border: 'none',
  background: '#e5a84b',
  color: '#121216',
  fontSize: 12,
  fontWeight: 600,
  fontFamily: "'Inter', sans-serif",
  cursor: 'pointer',
  width: '100%',
  marginTop: 4,
};

const closeButtonStyle: React.CSSProperties = {
  position: 'absolute',
  top: 20,
  right: 20,
  padding: '4px 10px',
  borderRadius: 4,
  border: '1px solid #2e2e38',
  background: 'transparent',
  color: '#8888a0',
  fontSize: 14,
  cursor: 'pointer',
  fontFamily: "'Inter', sans-serif",
};

const errorStyle: React.CSSProperties = {
  padding: '8px 12px',
  borderRadius: 6,
  background: 'rgba(248, 113, 113, 0.12)',
  border: '1px solid rgba(248, 113, 113, 0.25)',
  color: '#f87171',
  fontSize: 12,
  marginBottom: 12,
};

const PROVIDERS = ['openai', 'gemini', 'openrouter'] as const;

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface SettingsPanelProps {
  onClose: () => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function SettingsPanel({ onClose }: SettingsPanelProps) {
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  const dialogRef = useRef<HTMLDivElement>(null);
  useFocusTrap(dialogRef, true);

  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Add form state
  const [showAddForm, setShowAddForm] = useState(false);
  const [newProvider, setNewProvider] = useState<string>(PROVIDERS[0]);
  const [newLabel, setNewLabel] = useState('');
  const [newApiKey, setNewApiKey] = useState('');
  const [adding, setAdding] = useState(false);

  const fetchCredentials = useCallback(async () => {
    try {
      const creds = await api.credentials.list();
      setCredentials(creds);
      setError(null);
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Failed to load credentials';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchCredentials();
  }, [fetchCredentials]);

  const handleAdd = useCallback(async () => {
    if (!newLabel.trim() || !newApiKey.trim()) return;

    setAdding(true);
    setError(null);
    try {
      await api.credentials.create({
        provider: newProvider,
        label: newLabel.trim(),
        api_key: newApiKey.trim(),
      });
      setNewLabel('');
      setNewApiKey('');
      setShowAddForm(false);
      await fetchCredentials();
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : 'Failed to add credential';
      setError(message);
    } finally {
      setAdding(false);
    }
  }, [newProvider, newLabel, newApiKey, fetchCredentials]);

  const handleDelete = useCallback(
    async (id: string) => {
      setError(null);
      try {
        await api.credentials.delete(id);
        await fetchCredentials();
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : 'Failed to delete credential';
        setError(message);
      }
    },
    [fetchCredentials],
  );

  // Per-credential test state. Keyed by credential ID so test buttons on
  // separate rows don't share status. Reset whenever the user re-tests.
  const [testing, setTesting] = useState<Record<string, boolean>>({});
  const [testResults, setTestResults] = useState<
    Record<string, CredentialTestResult>
  >({});

  const handleTest = useCallback(async (id: string) => {
    setTesting((prev) => ({ ...prev, [id]: true }));
    try {
      const result = await api.credentials.test(id);
      setTestResults((prev) => ({ ...prev, [id]: result }));
    } catch (err: unknown) {
      // Network-level failure (server down, 500, etc.) — synthesize a
      // not-ok result so the row badge still renders.
      setTestResults((prev) => ({
        ...prev,
        [id]: {
          ok: false,
          latency_ms: 0,
          message: err instanceof Error ? err.message : 'Test request failed',
        },
      }));
    } finally {
      setTesting((prev) => ({ ...prev, [id]: false }));
    }
  }, []);

  return (
    <div
      ref={dialogRef}
      className="clotho-z-modal"
      role="dialog"
      aria-modal="true"
      aria-label="Settings"
      style={overlayStyle}
      onClick={onClose}
    >
      <div
        style={{ ...panelStyle, position: 'relative' }}
        onClick={(e) => e.stopPropagation()}
      >
        <button style={closeButtonStyle} onClick={onClose}>
          ✕
        </button>

        <div style={headingStyle}>Settings</div>

        {/* User info */}
        {user && (
          <div
            style={{
              padding: '10px 12px',
              background: '#222228',
              borderRadius: 6,
              border: '1px solid #2e2e38',
              marginBottom: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}
          >
            <div>
              <div style={{ fontSize: 13, color: '#ececf0', fontWeight: 600 }}>
                {user.name}
              </div>
              <div style={{ fontSize: 11, color: '#8888a0', marginTop: 2 }}>
                {user.email}
              </div>
            </div>
            <button
              style={{
                ...deleteButtonStyle,
                color: '#8888a0',
                borderColor: '#2e2e38',
              }}
              onClick={logout}
            >
              Sign out
            </button>
          </div>
        )}

        {/* API Keys section */}
        <div style={sectionHeadingStyle}>API Keys</div>

        {error && <div style={errorStyle}>{error}</div>}

        {loading ? (
          <div style={{ fontSize: 12, color: '#55556a', padding: '12px 0' }}>
            Loading credentials...
          </div>
        ) : credentials.length === 0 && !showAddForm ? (
          <div
            style={{
              padding: '20px 16px',
              background: '#222228',
              borderRadius: 6,
              border: '1px solid #2e2e38',
              textAlign: 'center',
              marginBottom: 12,
            }}
          >
            <div style={{ fontSize: 13, color: '#8888a0', marginBottom: 8 }}>
              No API keys configured yet
            </div>
            <div style={{ fontSize: 12, color: '#55556a' }}>
              Add a key to start running workflows
            </div>
          </div>
        ) : (
          credentials.map((cred) => {
            const result = testResults[cred.id];
            const inFlight = testing[cred.id];
            const badgeText = inFlight
              ? 'Testing…'
              : result
                ? result.ok
                  ? `OK · ${result.latency_ms}ms`
                  : `Fail · ${result.message ?? 'unknown'}`
                : null;
            const badgeColor = result
              ? result.ok
                ? '#22c55e'
                : '#f87171'
              : '#8888a0';
            return (
              <div key={cred.id} style={credCardStyle}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={credProviderStyle}>{cred.provider}</div>
                  <div style={credLabelStyle}>{cred.label}</div>
                  {badgeText && (
                    <div
                      style={{
                        marginTop: 4,
                        fontSize: 10,
                        color: badgeColor,
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                      }}
                      title={badgeText}
                    >
                      {badgeText}
                    </div>
                  )}
                </div>
                <div style={{ display: 'flex', gap: 6 }}>
                  <button
                    style={{
                      ...deleteButtonStyle,
                      borderColor: '#2e2e38',
                      color: '#8888a0',
                      opacity: inFlight ? 0.6 : 1,
                    }}
                    disabled={inFlight}
                    onClick={() => void handleTest(cred.id)}
                    title="Send a 1-token ping to verify the key is valid"
                  >
                    {inFlight ? '…' : 'Test'}
                  </button>
                  <button
                    style={deleteButtonStyle}
                    onClick={() => void handleDelete(cred.id)}
                  >
                    Delete
                  </button>
                </div>
              </div>
            );
          })
        )}

        {/* Add form */}
        {showAddForm ? (
          <div
            style={{
              padding: 16,
              background: '#222228',
              borderRadius: 6,
              border: '1px solid #2e2e38',
              marginTop: 12,
            }}
          >
            <div style={fieldGroup}>
              <label style={labelStyle}>Provider</label>
              <select
                style={inputStyle}
                value={newProvider}
                onChange={(e) => setNewProvider(e.target.value)}
              >
                {PROVIDERS.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>

            <div style={fieldGroup}>
              <label style={labelStyle}>Label</label>
              <input
                style={inputStyle}
                type="text"
                value={newLabel}
                onChange={(e) => setNewLabel(e.target.value)}
                placeholder="e.g. Production key"
              />
            </div>

            <div style={fieldGroup}>
              <label style={labelStyle}>API Key</label>
              <input
                style={inputStyle}
                type="password"
                value={newApiKey}
                onChange={(e) => setNewApiKey(e.target.value)}
                placeholder="sk-..."
              />
            </div>

            <div style={{ display: 'flex', gap: 8 }}>
              <button
                style={{
                  ...addButtonStyle,
                  opacity: adding ? 0.6 : 1,
                }}
                disabled={adding}
                onClick={() => void handleAdd()}
              >
                {adding ? 'Saving...' : 'Save Key'}
              </button>
              <button
                style={{
                  ...addButtonStyle,
                  background: 'transparent',
                  border: '1px solid #2e2e38',
                  color: '#8888a0',
                }}
                onClick={() => {
                  setShowAddForm(false);
                  setNewLabel('');
                  setNewApiKey('');
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <button
            style={{
              ...addButtonStyle,
              background: 'transparent',
              border: '1px solid #2e2e38',
              color: '#e5a84b',
              marginTop: 8,
            }}
            onClick={() => setShowAddForm(true)}
          >
            + Add API Key
          </button>
        )}
      </div>
    </div>
  );
}
