# Roadmap

All slices delivered. The system is a deployable medical consultation
platform: an HTTP API, a WebSocket hub, object storage, payments, and an
opt-in retention worker.

| Slice | Delivered |
|-------|-----------|
| 1  | Register + Login + Refresh + Logout |
| 2  | Doctor profile lifecycle |
| 3  | Admin verification votes |
| 4  | Schedules + availability |
| 5  | Public doctor search + detail |
| 6  | Booking + ClinicSession |
| 7  | Content (doctor posts) |
| 8  | Kashier integration + booking race fix |
| 9  | Queue subsystem |
| 10 | WebSocket hub |
| 11 | Files module |
| 12 | Clinical records + prescriptions |
| 12a–c | Files adoption follow-ups |
| 13 | Messages + notifications |
| 14 | Free consultation messages |
| 15 | Password reset + email verification |
| 16 | Persistent rate limiting + observability |
| 17 | Admin user management (suspensions) |
| 18 | Production hardening (Dockerfile, compose, CI, integration tests, retention worker, seed flag) |

## What slice 18 proves

- Deployable: one `docker build --target api .` produces a < 30 MB image
  running as non-root with no shell.
- Reproducible: `-trimpath` and locked dependencies.
- CI verifies lint, build, unit + integration tests, and that generated
  sqlc code is committed and up to date.
- Integration tests run against real Postgres (testcontainers).
- Retention is a policy, not a promise: opt-in, scheduled, two-phase
  soft/hard delete.
- Admin bootstrap is a one-time `--seed` flag; no admin password is ever
  committed to a migration.

## Known gaps (explicitly out of slice 18)

- Load testing (k6/Locust) — the queue subsystem has never been load tested.
- Chaos engineering / fault injection.
- A compliance audit of the retention policy (7-year placeholder, SD58).
- Kashier live-mode signature verification.
- A real monitoring stack (Grafana, alerting, on-call).
- "Suspended user cannot log in" — the suspension module exists (slice 17)
  but the login gate is not yet wired into the login service.