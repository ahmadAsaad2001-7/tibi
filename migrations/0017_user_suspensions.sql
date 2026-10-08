-- +goose Up

ALTER TABLE admin_votes
    ADD COLUMN payload JSONB;

CREATE TABLE admin_user_suspensions (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    reason      TEXT NOT NULL CHECK (length(reason) BETWEEN 1 AND 500),
    from_ts     TIMESTAMPTZ NOT NULL DEFAULT now(),
    to_ts       TIMESTAMPTZ,
    lifted_at   TIMESTAMPTZ,
    lifted_by   BIGINT REFERENCES identity_users(id) ON DELETE SET NULL,
    lift_reason TEXT,
    vote_id     BIGINT REFERENCES admin_votes(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (to_ts IS NULL OR to_ts > from_ts),
    CHECK ((lifted_at IS NULL AND lifted_by IS NULL) OR (lifted_at IS NOT NULL AND lifted_by IS NOT NULL))
);

-- At most one active suspension per user. "Active" means not lifted and
-- (permanent or not yet expired).
CREATE UNIQUE INDEX admin_user_suspensions_one_active_idx
    ON admin_user_suspensions (user_id)
    WHERE lifted_at IS NULL;

CREATE INDEX admin_user_suspensions_active_lookup_idx
    ON admin_user_suspensions (user_id, to_ts)
    WHERE lifted_at IS NULL;

-- +goose Down

DROP TABLE admin_user_suspensions;
ALTER TABLE admin_votes DROP COLUMN payload;