-- 008_update_presets_with_category.up.sql
--
-- Backfills the `preset_category` dispatch key inside each built-in preset's
-- AgentNodeConfig JSONB blob. The frontend reads this key to render
-- specialized AgentNode personalities:
--
--   "script"   → Script Writer, Story Writer
--   "crafter"  → any "* Prompt Crafter" preset
--   (absent)   → generic AgentNode rendering
--
-- Only system (built-in) presets are touched. User-created presets keep their
-- existing config untouched.

-- Script personality: Script Writer, Story Writer
UPDATE agent_presets
SET config = jsonb_set(config, '{preset_category}', '"script"'::jsonb, true)
WHERE is_built_in = true
  AND name IN ('Script Writer', 'Story Writer');

-- Crafter personality: any preset whose name ends with "Prompt Crafter"
-- (currently "Image Prompt Crafter"; future Video/Audio Prompt Crafter
-- presets will match automatically if seeded with that naming convention).
UPDATE agent_presets
SET config = jsonb_set(config, '{preset_category}', '"crafter"'::jsonb, true)
WHERE is_built_in = true
  AND name LIKE '%Prompt Crafter';
