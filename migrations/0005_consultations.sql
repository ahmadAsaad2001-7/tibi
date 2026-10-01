-- +goose Up

CREATE TYPE consultation_status AS ENUM
    ('Pending', 'InProgress', 'Confirmed', 'Completed', 'Cancelled', 'NoShow');
CREATE TYPE payment_status AS ENUM ('Pending', 'Succeeded', 'Failed', 'Refunded');
CREATE TYPE payment_channel AS ENUM ('Card', 'MobileWallet', 'InstaPay');

CREATE TABLE doctors_clinic_sessions (
                                         id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                         doctor_profile_id BIGINT NOT NULL REFERENCES doctors_profiles(id) ON DELETE RESTRICT,
                                         session_date      DATE NOT NULL,
                                         start_time        TIME NOT NULL,
                                         end_time          TIME NOT NULL,
                                         created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
                                         UNIQUE (doctor_profile_id, session_date, start_time)
);

CREATE INDEX doctors_clinic_sessions_doctor_date_idx
    ON doctors_clinic_sessions (doctor_profile_id, session_date);

CREATE TABLE consultations_consultations (
                                             id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                             patient_profile_id BIGINT NOT NULL REFERENCES patients_profiles(id) ON DELETE RESTRICT,
                                             doctor_profile_id  BIGINT NOT NULL REFERENCES doctors_profiles(id) ON DELETE RESTRICT,
                                             clinic_session_id  BIGINT REFERENCES doctors_clinic_sessions(id) ON DELETE SET NULL,
                                             scheduled_at       TIMESTAMPTZ NOT NULL,
                                             duration_minutes   INT NOT NULL,
                                             status             consultation_status NOT NULL DEFAULT 'Pending',
                                             is_urgent          BOOLEAN NOT NULL DEFAULT false,
                                             notes              TEXT,
                                             created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
                                             updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
                                             deleted_at         TIMESTAMPTZ
);

CREATE INDEX consultations_patient_idx
    ON consultations_consultations (patient_profile_id, scheduled_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX consultations_doctor_idx
    ON consultations_consultations (doctor_profile_id, scheduled_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX consultations_session_idx
    ON consultations_consultations (clinic_session_id)
    WHERE clinic_session_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX consultations_active_slot_idx
    ON consultations_consultations (doctor_profile_id, scheduled_at)
    WHERE deleted_at IS NULL
      AND status IN ('Pending', 'Confirmed', 'InProgress');

CREATE TABLE payments_payments (
                                   id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                   patient_profile_id BIGINT NOT NULL REFERENCES patients_profiles(id) ON DELETE RESTRICT,
                                   doctor_profile_id  BIGINT NOT NULL REFERENCES doctors_profiles(id) ON DELETE RESTRICT,
                                   consultation_id    BIGINT NOT NULL UNIQUE REFERENCES consultations_consultations(id) ON DELETE CASCADE,
                                   amount             NUMERIC(10,2) NOT NULL,
                                   currency           CHAR(3) NOT NULL,
                                   channel            payment_channel NOT NULL,
                                   transaction_id     TEXT UNIQUE,
                                   status             payment_status NOT NULL DEFAULT 'Pending',
                                   transaction_date   TIMESTAMPTZ,
                                   checkout_url       TEXT,
                                   created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
                                   updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
                                   deleted_at         TIMESTAMPTZ
);

CREATE INDEX payments_pending_idx
    ON payments_payments (created_at)
    WHERE status = 'Pending' AND deleted_at IS NULL;

-- +goose Down

DROP TABLE payments_payments;
DROP TABLE consultations_consultations;
DROP TABLE doctors_clinic_sessions;
DROP TYPE payment_channel;
DROP TYPE payment_status;
DROP TYPE consultation_status;