import { Component, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

const containerStyle: React.CSSProperties = {
  width: '100vw',
  height: '100vh',
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  background: '#0f1117',
  color: '#e2e8f0',
  gap: 16,
};

const preStyle: React.CSSProperties = {
  padding: '12px 16px',
  borderRadius: 6,
  background: '#1a1c2e',
  border: '1px solid #334155',
  color: '#f87171',
  fontSize: 13,
  maxWidth: '80vw',
  overflow: 'auto',
};

const buttonStyle: React.CSSProperties = {
  padding: '8px 20px',
  borderRadius: 4,
  border: '1px solid #334155',
  background: '#1e2030',
  color: '#e2e8f0',
  fontSize: 13,
  cursor: 'pointer',
};

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={containerStyle}>
          <h2 style={{ margin: 0, fontSize: 20 }}>Something went wrong</h2>
          <pre style={preStyle}>{this.state.error?.message}</pre>
          <button
            style={buttonStyle}
            onClick={() => this.setState({ hasError: false, error: null })}
          >
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
