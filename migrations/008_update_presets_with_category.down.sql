-- 008_update_presets_with_category.down.sql
--
-- Removes the `preset_category` key from every built-in preset's config
-- JSONB blob. Restores the pre-008 shape.

UPDATE agent_presets
SET config = config - 'preset_category'
WHERE is_built_in = true;
