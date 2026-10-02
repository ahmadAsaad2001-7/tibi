-- sqlc-only catalog hints. Goose never applies this file.
-- xmin is a Postgres system column, so sqlc cannot see it from migrations.
ALTER TABLE doctors_profiles ADD COLUMN xmin xid;
ALTER TABLE admin_votes ADD COLUMN xmin xid;
ALTER TABLE consultations_consultations ADD COLUMN xmin xid;
ALTER TABLE queue_windows ADD COLUMN xmin xid;
ALTER TABLE queue_entries ADD COLUMN xmin xid;
