-- Reverse of 001_initial_schema.up.sql
-- Drop indexes first, then tables in reverse dependency order

DROP INDEX IF EXISTS idx_executions_version;
DROP INDEX IF EXISTS idx_executions_tenant;
DROP INDEX IF EXISTS idx_pipelines_project;
DROP INDEX IF EXISTS idx_projects_tenant;
DROP INDEX IF EXISTS idx_job_queue_poll;
DROP INDEX IF EXISTS idx_step_results_execution;

DROP TABLE IF EXISTS job_queue;
DROP TABLE IF EXISTS step_results;
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS agent_presets;
DROP TABLE IF EXISTS pipeline_versions;
DROP TABLE IF EXISTS pipelines;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
