# Slice log

A log is not a changelog; it is a pointer to why and when design-relevant
decisions were made. R-prefixed revisions live in
`docs/decisions/00-decisions.md`; S- and D-prefixed decisions (SD/S) live
in the code comments where they apply.

| Slice | What shipped | Relevant revisions |
|-------|--------------|--------------------|
| 1  | Identity: register, login, refresh, logout, Argon2id, JWT access + opaque refresh tokens | — |
| 2  | Doctor profile lifecycle: create, update, verification status, specialties | — |
| 3  | Admin verification votes: propose, list, cast, expire | R23 (vote model adjustments) |
| 4  | Weekly schedules + exceptions; availability days | — |
| 5  | Public doctor search + detail with rating aggregations | — |
| 6  | Booking + ClinicSession with a race fix (partial-index uniqueness) | R10 (session state lives on queue) later |
| 7  | Doctor posts (content), publish/unpublish, feed, per-doctor list | — |
| 8  | Kashier checkout, webhook, pending payments + booking race fix | — |
| 9  | Queue subsystem: queue windows, check-in, call, start, complete, skip, no-show, ETA | R10 |
| 10 | WebSocket hub with consultation + queue membership checks | — |
| 11 | Files module: upload, presigned downloads, metadata, delete; LocalFS + S3 drivers | R15 (access checker registry, slice 12) |
| 12 | Clinical records + prescriptions + attachments | R15, R16 |
| 12a–c | Files-adoption follow-ups: post attachments, profile images, access scope rules | R15 |
| 13 | Messages + notifications (per-conversation, unread counts, marks) | — |
| 14 | Free consultation messages (doctorposts) with IP/email rate limits | R26 (limiter signature) |
| 15 | Password reset + email verification, token hashing, revoke-all on reset | R24, R25 |
| 16 | Persistent (Postgres) rate limiting + observability (query tracer, Prometheus) | R26, R27 |
| 17 | Admin user management: suspensions, propose/approve/cast, lift, lists | — |
| 18 | Production hardening: Dockerfile, compose, CI, testcontainers integration tests, retention worker, seed flag | R31, R32 |

## Repository layout after slice 18

```
cmd/api         HTTP API binary            (also runs `--seed`)
cmd/retention   retention worker binary
internal/*      module packages (identity, doctors, admin, queue, clinical, …)
internal/platform/*  config, database, auth, httpx, docs, email, ratelimit, ws, …
internal/testutil    integration-test harness (testcontainers + goose as a library)
integration/    the slice-18 integration test suite (one Postgres container)
migrations/     goose migrations (applied by CI/CD, never by the API binary)
```

## Slice 18 verification notes

- `scripts/endpoint_smoke.sh` — 38-check endpoint sweep (health, auth,
  admin, role boundaries, notifications, password reset, webhook signature,
  token rotation). Run against a live dev server; it registers its own
  patient and uses the `--seed` admin.
- The first smoke run caught a production bug: `GET /api/v1/auth/me`
  returned 500 for **every patient**. `GetMe` LEFT JOINs
  `doctors_profiles`, whose `NOT NULL` `consultation_fee` /
  `average_rating` columns are NULL on patient rows; sqlc types them as
  plain `string`, so pgx failed scanning. Fixed in
  `internal/identity/me/query.sql` with `CASE WHEN d.id IS NULL`, guarded
  by regression tests `TestMe_*` in `integration/me_test.go`. Other
  LEFT-JOIN queries were audited — none share the trap (`listvotes`
  already used COALESCE; the clinical/wschecker joins only feed WHERE
  guards; `getpatientqueue` scans into nullable types).