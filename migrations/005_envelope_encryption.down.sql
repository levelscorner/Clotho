-- Reverse of 005_envelope_encryption.up.sql

ALTER TABLE credentials ADD COLUMN api_key TEXT NOT NULL DEFAULT '';
ALTER TABLE credentials DROP COLUMN IF EXISTS nonce;
ALTER TABLE credentials DROP COLUMN IF EXISTS encrypted_dek;
ALTER TABLE credentials DROP COLUMN IF EXISTS encrypted_value;
