import { useState, useCallback } from 'react';
import { useAuthStore } from '../../stores/authStore';

// ---------------------------------------------------------------------------
// Styles (DESIGN.md tokens)
// ---------------------------------------------------------------------------

const pageStyle: React.CSSProperties = {
  width: '100vw',
  height: '100vh',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  background: 'var(--surface-base)',
  fontFamily: 'var(--font-body)',
};

const cardStyle: React.CSSProperties = {
  width: 380,
  padding: 32,
  background: 'var(--surface-raised)',
  borderRadius: 'var(--radius-lg)',
  border: '1px solid var(--surface-border)',
};

const logoStyle: React.CSSProperties = {
  fontFamily: 'var(--font-display)',
  fontWeight: 700,
  fontSize: 28,
  color: 'var(--accent)',
  letterSpacing: '-0.5px',
  textAlign: 'center',
  marginBottom: 8,
};

const subtitleStyle: React.CSSProperties = {
  fontSize: 13,
  color: 'var(--text-secondary)',
  textAlign: 'center',
  marginBottom: 28,
};

const fieldGroup: React.CSSProperties = {
  marginBottom: 16,
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: 11,
  fontWeight: 600,
  color: 'var(--text-secondary)',
  marginBottom: 6,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
};

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 12px',
  borderRadius: 'var(--radius-sm)',
  border: '1px solid var(--surface-border)',
  background: 'var(--surface-overlay)',
  color: 'var(--text-primary)',
  fontSize: 13,
  fontFamily: 'var(--font-body)',
  outline: 'none',
  boxSizing: 'border-box',
  transition: 'border-color var(--duration-normal) var(--ease-out)',
};

const buttonStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 16px',
  borderRadius: 'var(--radius-sm)',
  border: 'none',
  background: 'var(--accent)',
  color: 'var(--surface-base)',
  fontSize: 13,
  fontWeight: 600,
  fontFamily: 'var(--font-body)',
  cursor: 'pointer',
  transition: 'opacity var(--duration-normal) var(--ease-out)',
};

const errorStyle: React.CSSProperties = {
  padding: '8px 12px',
  borderRadius: 'var(--radius-sm)',
  background: 'rgba(248, 113, 113, 0.12)',
  border: '1px solid rgba(248, 113, 113, 0.25)',
  color: 'var(--status-failed)',
  fontSize: 12,
  marginBottom: 16,
};

const linkStyle: React.CSSProperties = {
  color: 'var(--accent)',
  textDecoration: 'none',
  cursor: 'pointer',
  fontSize: 13,
};

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface LoginPageProps {
  onSwitchToRegister: () => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function LoginPage({ onSwitchToRegister }: LoginPageProps) {
  const login = useAuthStore((s) => s.login);
  const isLoading = useAuthStore((s) => s.isLoading);
  const error = useAuthStore((s) => s.error);
  const clearError = useAuthStore((s) => s.clearError);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      try {
        await login(email, password);
      } catch {
        // Error is stored in authStore
      }
    },
    [email, password, login],
  );

  return (
    <div style={pageStyle}>
      <div style={cardStyle}>
        <div style={logoStyle}>Clotho</div>
        <div style={subtitleStyle}>Sign in to your workspace</div>

        {error && <div style={errorStyle}>{error}</div>}

        <form onSubmit={handleSubmit}>
          <div style={fieldGroup}>
            <label style={labelStyle}>Email</label>
            <input
              style={inputStyle}
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                clearError();
              }}
              placeholder="you@example.com"
              autoComplete="email"
              required
            />
          </div>

          <div style={fieldGroup}>
            <label style={labelStyle}>Password</label>
            <input
              style={inputStyle}
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                clearError();
              }}
              placeholder="Enter your password"
              autoComplete="current-password"
              required
            />
          </div>

          <button
            type="submit"
            style={{
              ...buttonStyle,
              opacity: isLoading ? 0.6 : 1,
            }}
            disabled={isLoading}
          >
            {isLoading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>

        <div
          style={{
            textAlign: 'center',
            marginTop: 20,
            fontSize: 13,
            color: 'var(--text-secondary)',
          }}
        >
          No account yet?{' '}
          <span style={linkStyle} onClick={onSwitchToRegister}>
            Create one
          </span>
        </div>
      </div>
    </div>
  );
}
