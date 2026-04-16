DROP INDEX IF EXISTS idx_executions_failure_class;
ALTER TABLE executions DROP COLUMN IF EXISTS trace_id;
ALTER TABLE executions DROP COLUMN IF EXISTS failure_json;
ALTER TABLE step_results DROP COLUMN IF EXISTS failure_json;
