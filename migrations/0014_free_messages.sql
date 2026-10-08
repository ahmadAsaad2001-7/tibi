-- +goose Up

CREATE TABLE doctorposts_free_messages (
                                           id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                           doctor_profile_id BIGINT NOT NULL REFERENCES doctors_profiles(id) ON DELETE CASCADE,
                                           sender_name       TEXT NOT NULL CHECK (length(sender_name) BETWEEN 1 AND 200),
                                           sender_email      TEXT NOT NULL CHECK (length(sender_email) BETWEEN 3 AND 320),
                                           sender_phone      TEXT CHECK (sender_phone IS NULL OR length(sender_phone) BETWEEN 5 AND 30),
                                           content           TEXT NOT NULL CHECK (length(content) BETWEEN 1 AND 4000),
                                           is_replied_to     BOOLEAN NOT NULL DEFAULT false,
                                           replied_at        TIMESTAMPTZ,
                                           reply_content     TEXT,
                                           sender_ip         INET,
                                           created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
                                           updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX doctorposts_free_messages_doctor_idx
    ON doctorposts_free_messages (doctor_profile_id, created_at DESC);

-- Partial index for the doctor's "unreplied" inbox view.
CREATE INDEX doctorposts_free_messages_unreplied_idx
    ON doctorposts_free_messages (doctor_profile_id, created_at DESC)
    WHERE is_replied_to = false;

-- +goose Down

DROP TABLE doctorposts_free_messages;