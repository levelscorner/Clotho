-- Update all 7 built-in presets from openai/gpt-4o to gemini/gemini-2.0-flash
-- This makes the app work out-of-the-box with a free Google AI Studio key.

UPDATE agent_presets
SET config = jsonb_set(
    jsonb_set(config, '{provider}', '"gemini"'),
    '{model}', '"gemini-2.0-flash"'
)
WHERE is_built_in = true
  AND config->>'provider' = 'openai'
  AND config->>'model' = 'gpt-4o';
