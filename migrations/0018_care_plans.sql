-- +goose Up

CREATE TYPE care_plan_category AS ENUM
    ('Medication', 'Vitals', 'Lifestyle', 'FollowUp', 'Other');
CREATE TYPE care_plan_frequency AS ENUM
    ('Daily', 'EveryOtherDay', 'Weekly', 'Once');

CREATE TABLE care_plan_protocols (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    patient_profile_id  BIGINT NOT NULL REFERENCES patients_profiles(id) ON DELETE RESTRICT,
    doctor_profile_id   BIGINT NOT NULL REFERENCES doctors_profiles(id)   ON DELETE RESTRICT,
    consultation_id     BIGINT REFERENCES consultations_consultations(id) ON DELETE SET NULL,
    title               TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    description         TEXT CHECK (description IS NULL OR length(description) <= 1000),
    category            care_plan_category NOT NULL,
    frequency           care_plan_frequency NOT NULL,
    start_date          DATE NOT NULL,
    end_date            DATE,
    reminder_time       TIME,
    cancelled_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,
    CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX care_plan_protocols_patient_active_idx
    ON care_plan_protocols (patient_profile_id, start_date DESC)
    WHERE deleted_at IS NULL AND cancelled_at IS NULL;

CREATE INDEX care_plan_protocols_doctor_idx
    ON care_plan_protocols (doctor_profile_id, start_date DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE care_plan_completions (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    protocol_id         BIGINT NOT NULL
                        REFERENCES care_plan_protocols(id) ON DELETE CASCADE,
    scheduled_date      DATE NOT NULL,
    marked_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    marked_by_user_id   BIGINT NOT NULL REFERENCES identity_users(id) ON DELETE RESTRICT,
    note                TEXT CHECK (note IS NULL OR length(note) <= 500),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (protocol_id, scheduled_date)
);

CREATE INDEX care_plan_completions_protocol_idx
    ON care_plan_completions (protocol_id, scheduled_date DESC);

-- +goose Down

DROP TABLE care_plan_completions;
DROP TABLE care_plan_protocols;
DROP TYPE care_plan_frequency;
DROP TYPE care_plan_category;
