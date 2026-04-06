-- Revert built-in presets back to openai/gpt-4o

UPDATE agent_presets
SET config = jsonb_set(
    jsonb_set(config, '{provider}', '"openai"'),
    '{model}', '"gpt-4o"'
)
WHERE is_built_in = true
  AND config->>'provider' = 'gemini'
  AND config->>'model' = 'gemini-2.0-flash';
