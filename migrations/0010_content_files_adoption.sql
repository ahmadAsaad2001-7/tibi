-- +goose Up

-- file_id was added by 0008. file_url and file_type are legacy.
-- Ensure at least one reference is set, and allow file_type to be null.
ALTER TABLE content_post_attachments
    ALTER COLUMN file_type DROP NOT NULL,
    ADD CONSTRAINT content_post_attachments_file_or_url
        CHECK (file_id IS NOT NULL OR file_url IS NOT NULL);

-- +goose Down

ALTER TABLE content_post_attachments
DROP CONSTRAINT content_post_attachments_file_or_url,
    ALTER COLUMN file_type SET NOT NULL;