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
  background: '#121216',
  fontFamily: "'Inter', sans-serif",
};

const cardStyle: React.CSSProperties = {
  width: 380,
  padding: 32,
  background: '#1a1a20',
  borderRadius: 14,
  border: '1px solid #2e2e38',
};

const logoStyle: React.CSSProperties = {
  fontFamily: "'Sora', sans-serif",
  fontWeight: 700,
  fontSize: 28,
  color: '#e5a84b',
  letterSpacing: '-0.5px',
  textAlign: 'center',
  marginBottom: 8,
};

const subtitleStyle: React.CSSProperties = {
  fontSize: 13,
  color: '#8888a0',
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
  color: '#8888a0',
  marginBottom: 6,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
};

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 12px',
  borderRadius: 6,
  border: '1px solid #2e2e38',
  background: '#222228',
  color: '#ececf0',
  fontSize: 13,
  fontFamily: "'Inter', sans-serif",
  outline: 'none',
  boxSizing: 'border-box',
  transition: 'border-color 200ms cubic-bezier(0.16, 1, 0.3, 1)',
};

const buttonStyle: React.CSSProperties = {
  width: '100%',
  padding: '10px 16px',
  borderRadius: 6,
  border: 'none',
  background: '#e5a84b',
  color: '#121216',
  fontSize: 13,
  fontWeight: 600,
  fontFamily: "'Inter', sans-serif",
  cursor: 'pointer',
  transition: 'opacity 200ms cubic-bezier(0.16, 1, 0.3, 1)',
};

const errorStyle: React.CSSProperties = {
  padding: '8px 12px',
  borderRadius: 6,
  background: 'rgba(248, 113, 113, 0.12)',
  border: '1px solid rgba(248, 113, 113, 0.25)',
  color: '#f87171',
  fontSize: 12,
  marginBottom: 16,
};

const linkStyle: React.CSSProperties = {
  color: '#e5a84b',
  textDecoration: 'none',
  cursor: 'pointer',
  fontSize: 13,
};

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface RegisterPageProps {
  onSwitchToLogin: () => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function RegisterPage({ onSwitchToLogin }: RegisterPageProps) {
  const register = useAuthStore((s) => s.register);
  const isLoading = useAuthStore((s) => s.isLoading);
  const error = useAuthStore((s) => s.error);
  const clearError = useAuthStore((s) => s.clearError);

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [localError, setLocalError] = useState<string | null>(null);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setLocalError(null);

      if (password !== confirmPassword) {
        setLocalError('Passwords do not match');
        return;
      }

      if (password.length < 8) {
        setLocalError('Password must be at least 8 characters');
        return;
      }

      try {
        await register(email, password, name);
      } catch {
        // Error is stored in authStore
      }
    },
    [email, password, confirmPassword, name, register],
  );

  const displayError = localError ?? error;

  return (
    <div style={pageStyle}>
      <div style={cardStyle}>
        <div style={logoStyle}>Clotho</div>
        <div style={subtitleStyle}>Create your account</div>

        {displayError && <div style={errorStyle}>{displayError}</div>}

        <form onSubmit={handleSubmit}>
          <div style={fieldGroup}>
            <label style={labelStyle}>Name</label>
            <input
              style={inputStyle}
              type="text"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                clearError();
                setLocalError(null);
              }}
              placeholder="Your name"
              autoComplete="name"
              required
            />
          </div>

          <div style={fieldGroup}>
            <label style={labelStyle}>Email</label>
            <input
              style={inputStyle}
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                clearError();
                setLocalError(null);
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
                setLocalError(null);
              }}
              placeholder="At least 8 characters"
              autoComplete="new-password"
              required
            />
          </div>

          <div style={fieldGroup}>
            <label style={labelStyle}>Confirm Password</label>
            <input
              style={inputStyle}
              type="password"
              value={confirmPassword}
              onChange={(e) => {
                setConfirmPassword(e.target.value);
                clearError();
                setLocalError(null);
              }}
              placeholder="Re-enter your password"
              autoComplete="new-password"
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
            {isLoading ? 'Creating account...' : 'Create Account'}
          </button>
        </form>

        <div
          style={{
            textAlign: 'center',
            marginTop: 20,
            fontSize: 13,
            color: '#8888a0',
          }}
        >
          Already have an account?{' '}
          <span style={linkStyle} onClick={onSwitchToLogin}>
            Sign in
          </span>
        </div>
      </div>
    </div>
  );
}
