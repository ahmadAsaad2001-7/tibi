-- sqlc-only catalog hints. Goose never applies this file.
-- xmin is a Postgres system column, so sqlc cannot see it from migrations.
ALTER TABLE doctors_profiles ADD COLUMN xmin xid;
ALTER TABLE admin_votes ADD COLUMN xmin xid;
ALTER TABLE consultations_consultations ADD COLUMN xmin xid;
ALTER TABLE queue_windows ADD COLUMN xmin xid;
ALTER TABLE queue_entries ADD COLUMN xmin xid;
ALTER TABLE clinical_medical_records ADD COLUMN xmin xid;
ALTER TABLE clinical_prescriptions ADD COLUMN xmin xid;
ALTER TABLE doctorposts_free_messages ADD COLUMN xmin xid;
ALTER TABLE identity_password_resets ADD COLUMN xmin xid;
ALTER TABLE identity_email_verifications ADD COLUMN xmin xid;
ALTER TABLE admin_user_suspensions ADD COLUMN xmin xid;
