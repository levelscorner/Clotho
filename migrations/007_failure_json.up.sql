-- Structured failure payloads + OTel trace ID for the "tell-me-why-it-broke"
-- UX. The JSONB columns mirror domain.StepFailure (internal/domain/failure.go);
-- existing `error` text columns stay populated with a 1-line human summary
-- so older readers keep working while the FailureDrawer reads `failure_json`.

ALTER TABLE step_results ADD COLUMN failure_json JSONB;
ALTER TABLE executions ADD COLUMN failure_json JSONB;
ALTER TABLE executions ADD COLUMN trace_id TEXT;

-- Cheap index for "show me failed executions of class X" queries — used by
-- the upcoming /executions?status=failed&class=auth list view.
CREATE INDEX IF NOT EXISTS idx_executions_failure_class
    ON executions ((failure_json->>'class'))
    WHERE failure_json IS NOT NULL;
