-- +goose Up

CREATE TYPE post_type AS ENUM
    ('HealthTip', 'PatientEducation', 'ClinicNews', 'Publication');

CREATE TABLE content_posts (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doctor_profile_id BIGINT NOT NULL REFERENCES doctors_profiles(id) ON DELETE RESTRICT,
    title             TEXT NOT NULL,
    content           TEXT NOT NULL,
    excerpt           TEXT,
    type              post_type NOT NULL,
    cover_image_url   TEXT,
    view_count        INT NOT NULL DEFAULT 0,
    like_count        INT NOT NULL DEFAULT 0,
    is_published      BOOLEAN NOT NULL DEFAULT false,
    is_featured       BOOLEAN NOT NULL DEFAULT false,
    published_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX content_posts_feed_idx
    ON content_posts (published_at DESC, id DESC)
    WHERE is_published AND deleted_at IS NULL;

CREATE INDEX content_posts_doctor_idx
    ON content_posts (doctor_profile_id, published_at DESC)
    WHERE is_published AND deleted_at IS NULL;

CREATE INDEX content_posts_featured_idx
    ON content_posts (published_at DESC)
    WHERE is_published AND is_featured AND deleted_at IS NULL;

CREATE TABLE content_post_attachments (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    doctor_post_id BIGINT NOT NULL REFERENCES content_posts(id) ON DELETE CASCADE,
    file_url       TEXT NOT NULL,
    file_type      TEXT NOT NULL
);

-- +goose Down

DROP TABLE content_post_attachments;
DROP TABLE content_posts;
DROP TYPE post_type;