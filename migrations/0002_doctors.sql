-- +goose Up

CREATE TYPE verification_status AS ENUM ('NotSubmitted', 'PendingReview', 'Verified', 'Rejected');

CREATE TABLE doctors_specialties (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                TEXT NOT NULL UNIQUE,
    parent_specialty_id BIGINT REFERENCES doctors_specialties(id) ON DELETE RESTRICT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE doctors_profiles (
    id                            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id                       BIGINT NOT NULL UNIQUE REFERENCES identity_users(id) ON DELETE RESTRICT,
    full_name                     TEXT NOT NULL,
    bio                           TEXT NOT NULL DEFAULT '',
    consultation_fee              NUMERIC(10,2) NOT NULL DEFAULT 0,
    currency                      CHAR(3) NOT NULL DEFAULT 'EGP',
    clinic_name                   TEXT NOT NULL DEFAULT '',
    clinic_address                TEXT NOT NULL DEFAULT '',
    medical_license_number        TEXT,
    verification_status           verification_status NOT NULL DEFAULT 'NotSubmitted',
    verification_rejection_reason TEXT,
    submitted_at                  TIMESTAMPTZ,
    average_rating                NUMERIC(3,2) NOT NULL DEFAULT 0,
    rating_count                  INT NOT NULL DEFAULT 0,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at                    TIMESTAMPTZ
);

-- Partial index: the public doctor search only ever wants verified doctors.
CREATE INDEX doctors_profiles_verified_idx
    ON doctors_profiles (id)
    WHERE deleted_at IS NULL AND verification_status = 'Verified';

CREATE INDEX doctors_profiles_pending_idx
    ON doctors_profiles (submitted_at)
    WHERE deleted_at IS NULL AND verification_status = 'PendingReview';

CREATE TABLE doctors_profile_specialties (
    doctor_profile_id BIGINT NOT NULL REFERENCES doctors_profiles(id)   ON DELETE CASCADE,
    specialty_id      BIGINT NOT NULL REFERENCES doctors_specialties(id) ON DELETE RESTRICT,
    PRIMARY KEY (doctor_profile_id, specialty_id)
);

CREATE INDEX doctors_profile_specialties_specialty_idx
    ON doctors_profile_specialties (specialty_id);

-- Seed: minimal specialty catalog. Expand later.
INSERT INTO doctors_specialties (name, parent_specialty_id) VALUES
    ('General Practice', NULL),
    ('Cardiology',       NULL),
    ('Dermatology',      NULL),
    ('Pediatrics',       NULL),
    ('Psychiatry',       NULL),
    ('Orthopedics',      NULL),
    ('Internal Medicine',NULL),
    ('Neurology',        NULL);

INSERT INTO doctors_specialties (name, parent_specialty_id)
SELECT 'Interventional Cardiology', id FROM doctors_specialties WHERE name = 'Cardiology';

INSERT INTO doctors_specialties (name, parent_specialty_id)
SELECT 'Pediatric Cardiology', id FROM doctors_specialties WHERE name = 'Cardiology';

-- +goose Down

DROP TABLE doctors_profile_specialties;
DROP TABLE doctors_profiles;
DROP TABLE doctors_specialties;
DROP TYPE verification_status;