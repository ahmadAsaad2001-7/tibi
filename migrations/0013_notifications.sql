-- +goose Up

CREATE TABLE communication_notifications (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT NOT NULL
                REFERENCES identity_users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    payload     JSONB,
    is_read     BOOLEAN NOT NULL DEFAULT false,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX communication_notifications_user_all_idx
    ON communication_notifications (user_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX communication_notifications_user_unread_idx
    ON communication_notifications (user_id, created_at DESC)
    WHERE is_read = false AND deleted_at IS NULL;

-- +goose Down

DROP TABLE communication_notifications;
