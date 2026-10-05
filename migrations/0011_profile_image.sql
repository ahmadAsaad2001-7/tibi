-- +goose Up

ALTER TABLE identity_users
    ADD COLUMN profile_image_file_id BIGINT
        REFERENCES platform_files(id) ON DELETE SET NULL;

CREATE INDEX identity_users_profile_image_idx
    ON identity_users (profile_image_file_id)
    WHERE profile_image_file_id IS NOT NULL;

-- +goose Down

DROP INDEX identity_users_profile_image_idx;
ALTER TABLE identity_users DROP COLUMN profile_image_file_id;