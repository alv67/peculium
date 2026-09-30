-- VaultLab migration 000019
-- Account lifecycle for admin governance: every user carries a status and
-- registrations are gated by a server-wide setting. Existing accounts become
-- active; the setting defaults to auto-approval so behavior is unchanged until
-- an admin opts in.
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'pending', 'disabled'));

CREATE TABLE server_settings (
    id INT PRIMARY KEY CHECK (id = 1),
    auto_approve_registrations BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO server_settings (id, auto_approve_registrations)
VALUES (1, TRUE)
ON CONFLICT (id) DO NOTHING;
