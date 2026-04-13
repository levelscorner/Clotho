-- 007_add_preset_category.down.sql
--
-- No schema changes to revert — preset_category lives inside AgentNodeConfig
-- JSONB (see 007 up migration). See 008 down for data rollback.
SELECT 1;
