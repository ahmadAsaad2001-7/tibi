-- +goose Up

CREATE TYPE file_scope AS ENUM
    ('ProfileImage', 'PostAttachment', 'MedicalAttachment');

CREATE TABLE platform_files (
                                id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                uploader_id     BIGINT NOT NULL REFERENCES identity_users(id) ON DELETE RESTRICT,
                                scope           file_scope NOT NULL,
                                object_key      TEXT NOT NULL UNIQUE,
                                original_name   TEXT NOT NULL,
                                content_type    TEXT NOT NULL,
                                size_bytes      BIGINT NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 20971520),
                                content_sha256  CHAR(64) NOT NULL,
                                created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
                                deleted_at      TIMESTAMPTZ
);

CREATE INDEX platform_files_uploader_idx
    ON platform_files (uploader_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX platform_files_scope_idx
    ON platform_files (scope, created_at DESC)
    WHERE deleted_at IS NULL;

-- Adopt the File module in Content's post attachments. The existing columns
-- stay for now; a later slice drops file_url once all readers use file_id.
ALTER TABLE content_post_attachments
    ADD COLUMN file_id BIGINT REFERENCES platform_files(id) ON DELETE RESTRICT;

ALTER TABLE content_post_attachments
    ALTER COLUMN file_url DROP NOT NULL;

CREATE INDEX content_post_attachments_file_idx
    ON content_post_attachments (file_id) WHERE file_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS content_post_attachments_file_idx;
ALTER TABLE content_post_attachments
    DROP COLUMN IF EXISTS file_id;
ALTER TABLE content_post_attachments
    ALTER COLUMN file_url SET NOT NULL;

DROP TABLE platform_files;
DROP TYPE file_scope;