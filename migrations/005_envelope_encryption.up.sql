-- Add envelope encryption columns to credentials and remove plaintext api_key.

ALTER TABLE credentials ADD COLUMN encrypted_value BYTEA;
ALTER TABLE credentials ADD COLUMN encrypted_dek BYTEA;
ALTER TABLE credentials ADD COLUMN nonce BYTEA;
ALTER TABLE credentials DROP COLUMN IF EXISTS api_key;
