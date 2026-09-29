-- +goose Up

CREATE TYPE user_role AS ENUM ('Patient', 'Doctor', 'PendingDoctor', 'Admin');

CREATE TABLE identity_users (
                                id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                email             TEXT NOT NULL UNIQUE,
                                password_hash     TEXT NOT NULL,
                                role              user_role NOT NULL,
                                profile_image_url TEXT,
                                created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
                                updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
                                deleted_at        TIMESTAMPTZ
);

CREATE INDEX identity_users_role_active_idx
    ON identity_users (role) WHERE deleted_at IS NULL;

CREATE TABLE identity_refresh_tokens (
                                         id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                         user_id    BIGINT NOT NULL REFERENCES identity_users(id) ON DELETE CASCADE,
                                         token_hash TEXT NOT NULL UNIQUE,
                                         expires_at TIMESTAMPTZ NOT NULL,
                                         revoked_at TIMESTAMPTZ,
                                         created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX identity_refresh_tokens_user_active_idx
    ON identity_refresh_tokens (user_id) WHERE revoked_at IS NULL;

CREATE TABLE patients_profiles (
                                   id                      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                   user_id                 BIGINT NOT NULL UNIQUE REFERENCES identity_users(id) ON DELETE RESTRICT,
                                   full_name               TEXT NOT NULL,
                                   phone_number            TEXT NOT NULL,
                                   date_of_birth           DATE,
                                   insurance_provider      TEXT,
                                   insurance_policy_number TEXT,
                                   created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
                                   updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
                                   deleted_at              TIMESTAMPTZ
);

-- +goose Down

DROP TABLE patients_profiles;
DROP TABLE identity_refresh_tokens;
DROP TABLE identity_users;
DROP TYPE user_role;