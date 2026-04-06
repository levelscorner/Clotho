import { useEffect } from 'react';
import { useExecutionStore } from '../stores/executionStore';

/**
 * Connects to the SSE stream for a given execution and tears down on unmount
 * or when the executionId changes.
 */
export function useSSE(executionId: string | null): void {
  const connectSSE = useExecutionStore((s) => s.connectSSE);
  const disconnectSSE = useExecutionStore((s) => s.disconnectSSE);

  useEffect(() => {
    if (!executionId) return;

    connectSSE(executionId);

    return () => {
      disconnectSSE();
    };
  }, [executionId, connectSSE, disconnectSSE]);
}
