-- +goose Up

CREATE TABLE clinical_medical_records (
                                          id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                          consultation_id     BIGINT NOT NULL UNIQUE
                                              REFERENCES consultations_consultations(id) ON DELETE RESTRICT,
                                          patient_profile_id  BIGINT NOT NULL REFERENCES patients_profiles(id) ON DELETE RESTRICT,
                                          doctor_profile_id   BIGINT NOT NULL REFERENCES doctors_profiles(id)  ON DELETE RESTRICT,
                                          allergies           TEXT,
                                          current_medications TEXT,
                                          past_conditions     TEXT,
                                          doctor_notes        TEXT,
                                          created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
                                          updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
                                          deleted_at          TIMESTAMPTZ
);

CREATE INDEX clinical_medical_records_patient_idx
    ON clinical_medical_records (patient_profile_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE clinical_medical_attachments (
                                              id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                              medical_record_id BIGINT NOT NULL
                                                  REFERENCES clinical_medical_records(id) ON DELETE CASCADE,
                                              file_id           BIGINT NOT NULL REFERENCES platform_files(id) ON DELETE RESTRICT,
                                              label             TEXT,
                                              created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
                                              updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
                                              deleted_at        TIMESTAMPTZ
);

CREATE INDEX clinical_medical_attachments_record_idx
    ON clinical_medical_attachments (medical_record_id) WHERE deleted_at IS NULL;

CREATE TABLE clinical_prescriptions (
                                        id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                        consultation_id     BIGINT NOT NULL UNIQUE
                                            REFERENCES consultations_consultations(id) ON DELETE RESTRICT,
                                        patient_profile_id  BIGINT NOT NULL REFERENCES patients_profiles(id) ON DELETE RESTRICT,
                                        doctor_profile_id   BIGINT NOT NULL REFERENCES doctors_profiles(id)  ON DELETE RESTRICT,
                                        issued_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
                                        created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
                                        updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
                                        deleted_at          TIMESTAMPTZ
);

CREATE INDEX clinical_prescriptions_patient_idx
    ON clinical_prescriptions (patient_profile_id, issued_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE clinical_prescribed_medications (
                                                 id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                                 prescription_id BIGINT NOT NULL
                                                     REFERENCES clinical_prescriptions(id) ON DELETE CASCADE,
                                                 medication_name TEXT NOT NULL,
                                                 dosage          TEXT NOT NULL,
                                                 frequency       TEXT NOT NULL,
                                                 duration_days   INT  NOT NULL CHECK (duration_days > 0 AND duration_days <= 365),
                                                 notes           TEXT,
                                                 created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
                                                 updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
                                                 deleted_at      TIMESTAMPTZ
);

CREATE INDEX clinical_prescribed_medications_prescription_idx
    ON clinical_prescribed_medications (prescription_id) WHERE deleted_at IS NULL;

-- +goose Down

DROP TABLE clinical_prescribed_medications;
DROP TABLE clinical_prescriptions;
DROP TABLE clinical_medical_attachments;
DROP TABLE clinical_medical_records;