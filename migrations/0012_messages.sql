-- +goose Up

CREATE TABLE communication_messages (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    consultation_id BIGINT NOT NULL
                    REFERENCES consultations_consultations(id) ON DELETE RESTRICT,
    sender_user_id  BIGINT NOT NULL
                    REFERENCES identity_users(id) ON DELETE RESTRICT,
    content         TEXT NOT NULL
                    CHECK (length(content) > 0 AND length(content) <= 4000),
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX communication_messages_consultation_idx
    ON communication_messages (consultation_id, sent_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX communication_messages_unread_idx
    ON communication_messages (consultation_id, sender_user_id)
    WHERE read_at IS NULL AND deleted_at IS NULL;

-- +goose Down

DROP TABLE communication_messages;
