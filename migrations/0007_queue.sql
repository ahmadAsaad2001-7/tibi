-- +goose Up

CREATE TYPE queue_session_status AS ENUM ('Scheduled', 'Open', 'Closed', 'Cancelled');
CREATE TYPE queue_status AS ENUM
    ('Waiting', 'Called', 'InProgress', 'Completed', 'Skipped', 'NoShow', 'Cancelled');

CREATE TABLE queue_windows (
    id                     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    clinic_session_id      BIGINT NOT NULL UNIQUE
                           REFERENCES doctors_clinic_sessions(id) ON DELETE CASCADE,
    status                 queue_session_status NOT NULL DEFAULT 'Scheduled',
    next_queue_number      INT NOT NULL DEFAULT 1,
    current_queue_entry_id BIGINT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE queue_entries (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    consultation_id    BIGINT NOT NULL UNIQUE
                       REFERENCES consultations_consultations(id) ON DELETE RESTRICT,
    queue_window_id    BIGINT NOT NULL REFERENCES queue_windows(id) ON DELETE CASCADE,
    clinic_session_id  BIGINT NOT NULL REFERENCES doctors_clinic_sessions(id) ON DELETE CASCADE,
    patient_profile_id BIGINT NOT NULL REFERENCES patients_profiles(id) ON DELETE RESTRICT,
    queue_number       INT NOT NULL,
    is_priority        BOOLEAN NOT NULL DEFAULT false,
    status             queue_status NOT NULL DEFAULT 'Waiting',
    checked_in_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    called_at          TIMESTAMPTZ,
    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    skip_reason        TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (queue_window_id, queue_number)
);

-- Circular FK: queue_windows.current_queue_entry_id -> queue_entries.id.
-- Added after both tables exist.
ALTER TABLE queue_windows
    ADD CONSTRAINT queue_windows_current_entry_fk
    FOREIGN KEY (current_queue_entry_id) REFERENCES queue_entries(id) ON DELETE SET NULL;

-- The doctor-facing ordered list. Partial: only active entries.
CREATE INDEX queue_entries_window_active_idx
    ON queue_entries (queue_window_id, is_priority DESC, queue_number ASC)
    WHERE status IN ('Waiting', 'Called', 'Skipped');

CREATE INDEX queue_entries_window_status_idx
    ON queue_entries (queue_window_id, status);

CREATE INDEX queue_entries_session_idx
    ON queue_entries (clinic_session_id);

-- +goose Down

DROP TABLE queue_entries;
DROP TABLE queue_windows;
DROP TYPE queue_status;
DROP TYPE queue_session_status;