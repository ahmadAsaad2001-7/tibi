-- +goose Up

ALTER TABLE identity_users
    ADD COLUMN email_verified_at TIMESTAMPTZ;

CREATE INDEX identity_users_unverified_idx
    ON identity_users (id)
    WHERE email_verified_at IS NULL AND deleted_at IS NULL;

CREATE TABLE identity_password_resets (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    token_hash  CHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX identity_password_resets_user_active_idx
    ON identity_password_resets (user_id, created_at DESC)
    WHERE consumed_at IS NULL;

CREATE INDEX identity_password_resets_expiry_idx
    ON identity_password_resets (expires_at)
    WHERE consumed_at IS NULL;

CREATE TABLE identity_email_verifications (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
    token_hash  CHAR(64) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX identity_email_verifications_user_active_idx
    ON identity_email_verifications (user_id, created_at DESC)
    WHERE consumed_at IS NULL;

-- +goose Down

DROP TABLE identity_email_verifications;
DROP TABLE identity_password_resets;
DROP INDEX identity_users_unverified_idx;
ALTER TABLE identity_users DROP COLUMN email_verified_at;