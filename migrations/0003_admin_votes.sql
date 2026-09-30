-- +goose Up

CREATE TYPE vote_status AS ENUM ('Open', 'Resolved', 'Expired');
CREATE TYPE vote_choice AS ENUM ('For', 'Against');

CREATE TABLE admin_votes (
                             id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                             action_type    TEXT NOT NULL
                                 CHECK (action_type IN ('VerifyDoctor', 'UnverifyDoctor', 'BanUser')),
                             target_user_id BIGINT NOT NULL REFERENCES identity_users(id) ON DELETE RESTRICT,
                             status         vote_status NOT NULL DEFAULT 'Open',
                             required_votes INT NOT NULL CHECK (required_votes >= 1),
                             votes_for      INT NOT NULL DEFAULT 0,
                             votes_against  INT NOT NULL DEFAULT 0,
                             expires_at     TIMESTAMPTZ NOT NULL,
                             resolved_at    TIMESTAMPTZ,
                             created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
                             updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
                             deleted_at     TIMESTAMPTZ
);

-- Prevent two open votes for the same action on the same target.
CREATE UNIQUE INDEX admin_votes_open_unique_idx
    ON admin_votes (action_type, target_user_id)
    WHERE status = 'Open' AND deleted_at IS NULL;

-- The admin queue only ever reads open votes, ordered by expiry.
CREATE INDEX admin_votes_open_expiry_idx
    ON admin_votes (expires_at)
    WHERE status = 'Open' AND deleted_at IS NULL;

CREATE TABLE admin_vote_participants (
                                         id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                         admin_vote_id BIGINT NOT NULL REFERENCES admin_votes(id) ON DELETE CASCADE,
                                         admin_user_id BIGINT NOT NULL REFERENCES identity_users(id) ON DELETE RESTRICT,
                                         vote          vote_choice NOT NULL,
                                         voted_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
                                         UNIQUE (admin_vote_id, admin_user_id)
);

CREATE INDEX admin_vote_participants_vote_idx
    ON admin_vote_participants (admin_vote_id);

-- +goose Down

DROP TABLE admin_vote_participants;
DROP TABLE admin_votes;
DROP TYPE vote_choice;
DROP TYPE vote_status;