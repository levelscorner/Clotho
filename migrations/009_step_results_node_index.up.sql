-- Index to speed up the two hot step_result reads: ListByExecution ordered
-- by start time, and per-node lookups during re-run. Without it, ListBy
-- Execution is a seq-scan on a growing table.
CREATE INDEX IF NOT EXISTS idx_step_results_execution_node
    ON step_results (execution_id, node_id);
