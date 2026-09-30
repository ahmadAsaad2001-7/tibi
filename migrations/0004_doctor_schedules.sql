-- +goose Up

CREATE TYPE schedule_exception_type AS ENUM ('Closed', 'OpenEarly', 'OpenLate', 'ModifiedHours');

CREATE TABLE doctors_weekly_schedules (
                                          id                    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                          doctor_profile_id     BIGINT NOT NULL REFERENCES doctors_profiles(id) ON DELETE CASCADE,
                                          day_of_week           SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
                                          start_time            TIME NOT NULL,
                                          end_time              TIME NOT NULL,
                                          slot_duration_minutes INT NOT NULL CHECK (slot_duration_minutes BETWEEN 5 AND 240),
                                          is_active             BOOLEAN NOT NULL DEFAULT true,
                                          created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
                                          updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
                                          CHECK (start_time < end_time)
);

CREATE INDEX doctors_weekly_schedules_doctor_idx
    ON doctors_weekly_schedules (doctor_profile_id, day_of_week)
    WHERE is_active;

CREATE TABLE doctors_schedule_exceptions (
                                             id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                                             doctor_profile_id BIGINT NOT NULL REFERENCES doctors_profiles(id) ON DELETE CASCADE,
                                             exception_date    DATE NOT NULL,
                                             from_time         TIME NOT NULL,
                                             to_time           TIME NOT NULL,
                                             type              schedule_exception_type NOT NULL,
                                             reason            TEXT,
                                             created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
                                             updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
                                             CHECK (from_time < to_time)
);

CREATE INDEX doctors_schedule_exceptions_doctor_date_idx
    ON doctors_schedule_exceptions (doctor_profile_id, exception_date);

-- +goose Down

DROP TABLE doctors_schedule_exceptions;
DROP TABLE doctors_weekly_schedules;
DROP TYPE schedule_exception_type;