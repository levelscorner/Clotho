import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent, within } from '@testing-library/react';
import ExecutionsPage from '../ExecutionsPage';

const listMock = vi.fn();
const retryMock = vi.fn();

vi.mock('../../lib/api', () => ({
  api: {
    executions: {
      list: (...args: unknown[]) => listMock(...args),
      retry: (...args: unknown[]) => retryMock(...args),
    },
  },
}));

// Stub executionStore — FailureDrawer pulls executionId + retryNode from it.
vi.mock('../../stores/executionStore', () => ({
  useExecutionStore: (selector: (s: unknown) => unknown) =>
    selector({ executionId: null, retryNode: vi.fn() }),
}));

const failedRow = {
  id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  pipeline_version_id: 'v1',
  status: 'failed',
  total_cost: 0.12,
  total_tokens: 200,
  error: 'auth failed',
  failure: {
    class: 'auth',
    stage: 'provider_call',
    retryable: false,
    message: 'Authentication failed',
    attempts: 1,
    at: '2026-04-17T12:00:00Z',
  },
  started_at: '2026-04-17T12:00:00Z',
  completed_at: '2026-04-17T12:00:01Z',
  created_at: '2026-04-17T12:00:00Z',
};

const okRow = {
  id: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
  pipeline_version_id: 'v1',
  status: 'completed',
  total_cost: 0.05,
  total_tokens: 50,
  started_at: '2026-04-17T12:00:00Z',
  completed_at: '2026-04-17T12:00:02Z',
  created_at: '2026-04-17T12:00:00Z',
};

describe('ExecutionsPage', () => {
  beforeEach(() => {
    listMock.mockReset();
    retryMock.mockReset();
  });

  it('defaults to the failed filter and fetches with status=failed', async () => {
    listMock.mockResolvedValue([]);
    render(<ExecutionsPage />);
    await waitFor(() => expect(listMock).toHaveBeenCalled());
    expect(listMock.mock.calls[0][0]).toMatchObject({ status: 'failed' });
  });

  it('renders the empty state when no rows return', async () => {
    listMock.mockResolvedValue([]);
    render(<ExecutionsPage />);
    expect(await screen.findByText(/No failed executions yet/)).toBeInTheDocument();
  });

  it('switching to "all" refetches without status filter', async () => {
    listMock.mockResolvedValue([]);
    render(<ExecutionsPage />);
    await waitFor(() => expect(listMock).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByRole('button', { name: 'all' }));
    await waitFor(() => expect(listMock).toHaveBeenCalledTimes(2));

    expect(listMock.mock.calls[1][0]).toMatchObject({ status: undefined });
  });

  it('renders a row per execution with status badge + Retry button', async () => {
    listMock.mockResolvedValue([failedRow, okRow]);

    fireEvent; // ensure import isn't tree-shaken in CI lint
    render(<ExecutionsPage />);
    await waitFor(() => expect(screen.getByRole('table')).toBeInTheDocument());

    const tbody = screen.getByRole('table').querySelector('tbody');
    expect(tbody).not.toBeNull();
    if (!tbody) return;
    const rows = within(tbody as HTMLElement).getAllByRole('row');
    expect(rows.length).toBe(2);

    // The failed row gets a "Why?" button + "Retry"; the OK row only Retry.
    expect(screen.getByText('Why?')).toBeInTheDocument();
    expect(screen.getAllByText('Retry').length).toBe(2);
  });

  it('clicking Retry calls api.executions.retry with the row id and refetches', async () => {
    listMock.mockResolvedValue([failedRow]);
    retryMock.mockResolvedValue({ id: 'newexec' });

    render(<ExecutionsPage />);
    await waitFor(() => expect(screen.getByText('Retry')).toBeInTheDocument());

    fireEvent.click(screen.getByText('Retry'));

    await waitFor(() => expect(retryMock).toHaveBeenCalledWith(failedRow.id));
    // Refetch after retry — list called twice (initial + post-retry).
    await waitFor(() => expect(listMock).toHaveBeenCalledTimes(2));
  });

  it('clicking Why? opens the FailureDrawer for that row', async () => {
    listMock.mockResolvedValue([failedRow]);

    render(<ExecutionsPage />);
    await waitFor(() => expect(screen.getByText('Why?')).toBeInTheDocument());
    fireEvent.click(screen.getByText('Why?'));

    // Drawer should now render with the auth failure shown.
    expect(screen.getByText('Authentication failed')).toBeInTheDocument();
  });

  it('surfaces a load error inline', async () => {
    listMock.mockRejectedValue(new Error('500 server error'));
    render(<ExecutionsPage />);
    await waitFor(() =>
      expect(screen.getByText(/500 server error/)).toBeInTheDocument(),
    );
  });
});
