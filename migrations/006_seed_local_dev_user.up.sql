-- Seed the local-dev user used by NO_AUTH=true auth bypass.
-- See internal/api/middleware/auth.go (localDevUserID) and
-- web/src/stores/authStore.ts (VITE_NO_AUTH branch).
--
-- The user is attached to the default tenant seeded in 002_seed_presets.up.sql
-- (00000000-0000-0000-0000-000000000001). A default project is created so the
-- unauthenticated UI has somewhere to land.

INSERT INTO users (id, tenant_id, email, name, password_hash, is_active, created_at)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'you@local',
    'Local Dev',
    '',
    true,
    NOW()
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO projects (id, tenant_id, name, description, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-0000000d0001',
    '00000000-0000-0000-0000-000000000001',
    'Local Dev Project',
    'Default project for unauthenticated local development.',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
