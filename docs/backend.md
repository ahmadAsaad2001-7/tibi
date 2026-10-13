# Tibi Backend — Go API + Retention Worker

> Medical Consultation Platform API. One repo, two binaries (`cmd/api`, `cmd/retention`), Postgres 16, SQLC + pgx v5, chi, WebSockets, S3/Minio storage, Kashier payments, Prometheus metrics, Scalar OpenAPI docs.

Related docs: [`../README.md`](../README.md) · [`ROADMAP.md`](ROADMAP.md) · [`slices/00-slice-log.md`](slices/00-slice-log.md) · [`decisions/00-decisions.md`](decisions/00-decisions.md) · Frontend companion [`frontend.md`](frontend.md)

---
## 1. Quick start

```bash
cp .env.example .env            # edit DATABASE_URL, JWT_SECRET, Kashier keys
docker compose up -d            # postgres:5432, minio:9000/9001, mailhog:1025/8025
goose -dir migrations postgres "$DATABASE_URL" up
make run                        # or: go run ./cmd/api  -> :8080
```

| Entry point | URL |
|---|---|
| API base | `http://localhost:8080/api/v1` |
| Scalar API reference (OpenAPI in-binary) | `http://localhost:8080/docs` |
| Health / readiness / metrics | `/health`, `/healthz`, `/readyz`, `/metrics` |
| WS hub | `ws://localhost:8080/hubs/consultations` |
| Minio console / Mailhog UI | `:9001` (minio/minio123) · `:8025` |

```bash
make test                       # go test ./... -race -count=1 (unit)
go test ./integration -tags=integration -count=1   # real Postgres via testcontainers, needs Docker
golangci-lint run               # depguard + full lint (see golangci.yaml)
make generate                   # sqlc generate (CI fails if diff is uncommitted)
scripts/endpoint_smoke.sh       # 38-check live endpoint sweep (needs running server)
```

## 2. Tech stack

| Layer | Choice |
|---|---|
| Language | Go 1.26, `CGO_ENABLED=0`, `-trimpath` reproducible builds |
| HTTP router | `go-chi/chi/v5` + `internal/platform/httpx` (JSON, errors, trace, recover, request log) |
| DB | Postgres 16, `jackc/pgx/v5` pool, `pressly/goose` migrations, `sqlc` (pgx/v5 codegen, `v1.29.0` pinned in CI) |
| Auth | Argon2id (`alexedwards/argon2id`) + JWT access (`golang-jwt/v5`, 15m) + opaque refresh tokens (DB, SHA-256 hash, 720h) |
| Realtime | `coder/websocket` hub (`internal/platform/ws`) |
| Storage | `minio-go/v7` S3 driver + LocalFS dev driver (`./data/files`, HMAC-signed URLs) |
| Payments | Kashier (`KASHIER_API_KEY/URL/WEBHOOK_SECRET`, HMAC webhook verify) |
| Email | `logger` (stdout, dev) / `smtp` (Mailhog `:1025` locally) |
| Observability | `prometheus/client_golang` (`/metrics` + `metrics.HTTPMiddleware`), pgx query tracer (DEBUG per query, WARN >100ms), `httpx.Trace` request IDs |
| Config | `caarlos0/env/v11` + `joho/godotenv` (`.env` auto-loaded except production) |
| CI | `.github/workflows/ci.yml`: lint+sqlc-diff, build, unit+integration tests, docker build `api`+`retention` |

## 3. Repository layout

```
cmd/api            HTTP+WS binary (also --seed bootstrap). All wiring lives in main.go.
cmd/retention      scheduled retention worker binary (own config subset).
internal/
  identity/        register, login, refresh, logout, me, password-reset (15), email-verify (15), setprofileimage
  patients/        createprofile (patient profile creation on register)
  doctors/         profile CRUD, specialties, schedules, exceptions, availability, search, getdoctor, sessions
  consultations/   book, listmy, getbyid, cancel, markconfirmed/completed, checkininfo, clinicalcontext, wschecker
  queue/           checkin, call, start, complete, skip, noshow, closesession, getsessionqueue, getpatientqueue, eta, markcancelled
  doctorposts/     content posts CRUD + publish/unpublish + feed + freemessages (submit/listinbox/reply)
  files/           upload, get, getmetadata, delete, accesschecker (scope registry), api facade
  clinical/        upsertrecord, getrecord, add/deleteattachment, upsert/getprescription, scope rules
  communication/   send/listmessages, notifications (create/list/markread/markallread/unreadcount/notify)
  payments/        createpending, kashier client + webhook
  admin/           proposevote, listvotes, castvote, expirevotes, listpendingdoctors, suspensions (propose/create/lift/list), listusers
  storage/         LocalFS + S3 drivers behind one interface
  retention/       two-phase soft/hard delete worker
  platform/        auth, config, database, docs (OpenAPI), email, httpx, metrics, ratelimit, seed, trace, ws, observer
  testutil/        testcontainers Postgres harness (shared container, TruncateAll)
migrations/        0001..0017 goose files (applied by CI/CD migrate image, NEVER by the API — R31)
sqlc.yaml          ~40 per-module codegen blocks (migrations/ + sqlc/catalog.sql -> internal/<mod>/.../db)
integration/       login/register/me suites over real Postgres (build tag `integration`)
scripts/           endpoint_smoke.sh (38 checks)
docs/              ROADMAP, slices/00-slice-log, decisions/00-decisions, backend.md (this), frontend.md
```

**Module convention (enforced by `depguard` in `golangci.yaml`):** each bounded context owns `service.go` (business logic) + `handler.go` (HTTP) + `query.sql` + `db/` (generated) + `contracts/api.go` facades. Modules talk only through `contracts` interfaces — e.g. Clinical never imports Consultations tables, it calls `consultations.ClinicalContext`. Files never imports Clinical; Clinical registers a `ScopeRule` into `files/accesschecker` (R15, inversion of control).
**Module convention (enforced by `depguard` in `golangci.yaml`):** each bounded context owns `service.go` (business logic) + `handler.go` (HTTP) + `query.sql` + `db/` (generated) + `contracts/api.go` facades. Modules talk only through `contracts` interfaces — e.g. Clinical never imports Consultations tables, it calls `consultations.ClinicalContext`. Files never imports Clinical; Clinical registers a `ScopeRule` into `files/accesschecker` (R15, inversion of control).

## 4. Binaries and lifecycle

**`cmd/api` (~640 lines):** `Load()` then `Validate()` (fails closed; production rejects default JWT, test Kashier key, `local` storage, `logger` email, localhost base URL) then `database.Connect` (pgx pool + tracer) then wire hasher/issuer/refreshStore/Kashier/rate-limit factory/mailer then ~45 services then chi router then `ListenAndServe` with graceful 15s shutdown (stops Postgres limiter goroutines first). `--seed` applies the specialty catalog idempotently plus creates admin from `ADMIN_EMAIL`/`ADMIN_PASSWORD`, then exits without serving (first-deploy bootstrap, slice 18f).

**`cmd/retention` (68 lines):** loads `LoadRetention()` (only `DATABASE_URL` plus `RETENTION_*`, no JWT/Kashier needed), exits quietly when `RETENTION_ENABLED=false`, else `RunOnce` at startup plus every `RETENTION_INTERVAL` (default 24h). Two-phase lifecycle (R32): soft-delete clinical rows after `RETENTION_CLINICAL_YEARS` (default 7, placeholder pending legal review) past consultation completion, hard-delete one year later, in batches of `RETENTION_BATCH_SIZE` (>=100).

## 5. HTTP API route map (`/api/v1`, from `cmd/api/main.go`)

Auth: `Authorization: Bearer <jwt>` via `auth.RequireAuth(issuer)`; roles via `auth.RequireRole(...)`. Backend roles: `Patient`, `Doctor`, `PendingDoctor`, `Admin` (no `Receptionist` in Go — that role lives only in the Next.js prototype; the Go queue surface is Doctor plus Admin).

| Method and path | Auth | Handler / service |
|---|---|---|
| `POST /auth/register` | public | `register` — Argon2id, creates identity plus role profile, requests email verification |
| `POST /auth/login` | public | `login` — Argon2id verify, issues JWT plus refresh (note: suspended-login gate still unwired, roadmap gap) |
| `POST /auth/refresh` | refresh token | `refresh` — rotation |
| `POST /auth/logout` | auth | `logout` — revokes one refresh token |
| `GET /auth/me` | auth | `me` (plus `filesAPI` for profile image URL) |
| `POST /auth/forgot-password` | public, IP plus email limited | `requestpasswordreset` — mails `${APP_BASE_URL}/reset?token=` link |
| `POST /auth/reset-password` | public, token in body | `confirmpasswordreset` — consumes token, updates password, revokes ALL refresh (R25) |
| `POST /auth/verify-email/request` | auth | `requestemailverification` |
| `POST /auth/verify-email/confirm` | public, token in body | `confirmemailverification` — idempotent `email_verified_at` set (R24) |
| `POST /auth/webhooks/kashier` | HMAC signature | `payments/kashier/webhook` — settles pending payment, confirms booking |
| `GET /specialties` | public | `doctors/listspecialties` |
| `GET /doctors`, `GET /doctors/{id}` | public | `doctors/search`, `doctors/getdoctor` (filters, rating aggregates, image URLs) |
| `GET /doctors/{doctorId}/posts` | public | `doctorposts/listbydoctor` |
| `PATCH /profile/doctor`, `POST /profile/doctor/submit-for-verification` | Doctor,PendingDoctor | `doctors/updateprofile`, `submitforverification` |
| `PATCH /profile/image` | auth | `identity/setprofileimage` (plus `filesAPI`) |
| `GET /doctor-availability/{doctorProfileId}/days`, `.../hours` | public | `listavailabilitydays`, `listslots` |
| `GET|PUT /doctor-schedule/`, `POST /doctor-schedule/exceptions`, `DELETE /doctor-schedule/exceptions/{id}` | Doctor,PendingDoctor | `getschedule`, `putschedule`, `addexception`, `deleteexception` |
| `GET /doctor-posts/`, `GET /doctor-posts/{id}` | public | `listfeed`, `getpost` |
| `POST /doctor-posts/`, `PATCH|DELETE /doctor-posts/{id}`, `POST /doctor-posts/{id}/publish`, `POST .../unpublish` | Doctor | `createpost` (plus `filesAPI`), `updatepost`, `deletepost`, `publishpost`, `unpublishpost` |
| `POST /doctor-posts/`, `PATCH|DELETE /doctor-posts/{id}`, `POST /doctor-posts/{id}/publish`, `POST .../unpublish` | Doctor | `createpost` (plus `filesAPI`), `updatepost`, `deletepost`, `publishpost`, `unpublishpost` |
| `POST /consultations/` | Patient | `consultations/book` — availability check plus partial-index race guard plus pending Kashier payment |
| `POST /consultations/{id}/check-in` | Patient | `queue/checkin` — lazily creates QueueWindow, assigns queue number, broadcasts on hub |
| `GET /consultations/` | auth | `listmy` — scoped to caller |
| `GET /consultations/{id}`, `GET /consultations/{id}/queue` | auth | `getbyid`, `getpatientqueue` (plus ETA) |
| `POST /consultations/{id}/cancel` | auth | `cancel` via `markcancelled` (hub plus queue refresh) |
| `PUT /consultations/{id}/medical-record`, `POST .../attachments`, `DELETE .../attachments/{attachmentId}`, `PUT .../prescription` | Doctor | `upsertrecord`, `addattachment`, `deleteattachment`, `upsertprescription` — gated: treating doctor AND status InProgress/Completed (R16, else 422) |
| `GET /consultations/{id}/medical-record`, `GET .../prescription` | Doctor plus Patient | `getrecord` (fresh signed file URLs), `getprescription` — ownership enforced in SQL |
| `GET|POST /consultations/{id}/messages` | auth participant | `listmessages`, `sendmessage` (plus notification fan-out) |
| `GET /notifications`, `POST /notifications/{id}/read`, `POST /notifications/read-all`, `GET /notifications/unread-count` | auth | `listnotifications`, `markread`, `markallread`, `unreadcount` |
| `POST /doctors/{doctorId}/free-messages` | public, IP 3/h plus email 2/h | `freemessages/submitmessage` — public contact form |
| `GET /doctors/me/free-messages`, `POST /free-messages/{id}/reply` | Doctor | `listinbox`, `replymessage` (sends email via mailer) |
| `POST /files`, `GET /files/{id}`, `GET /files/{id}/metadata`, `DELETE /files/{id}` | auth | `upload`, `get`, `getmetadata`, `delete` plus `accesschecker` registry (uploader always allowed; owners register ScopeRules, R15) |
| `GET /clinic-sessions/{id}/queue`, `POST /clinic-sessions/{id}/close` | Doctor,Admin | `getsessionqueue`, `closesession` |
| `POST /queue/{entryId}/call`, `/start`, `/complete`, `/skip`, `/no-show` | Doctor,Admin | `call`, `start`, `complete` (also completes consultation), `skip`, `noshow` — all broadcast plus notify |
| `POST /admin/doctors/{userId}/propose-verification`, `.../propose-unverification` | Admin | `proposevote` |
| `GET /admin/votes`, `POST /admin/votes/{voteId}/cast` | Admin | `listvotes`, `castvote` (approval flips verification or triggers suspension via createsuspension) |
| `GET /admin/doctors/pending` | Admin | `listpendingdoctors` |

Top-level (outside `/api/v1`): `GET /health`, `GET /healthz`, `GET /readyz` (DB ping), `GET /metrics` (Prometheus), `/docs` (Scalar), `GET /hubs/consultations` (JWT WebSocket hub, membership-checked). Timeouts: 10s read-header, 30s read/write, 60s idle. Dev-only `GET /files/{key...}` serves LocalFS bytes with `?expires&sig=` — disabled in production.

## 6. Domain notes

- Identity: `identity_users` (email UNIQUE, role enum, `profile_image_url`, `email_verified_at` NULL plus partial index, soft-delete) plus `identity_refresh_tokens` (hash UNIQUE, revoked_at) plus hashed one-time tokens for reset/verify (consume-once). Register creates the role profile in-transaction. Reset consumes token, updates password, and revokes ALL sessions atomically (R25).
- Doctors and schedules: profiles, specialty catalog plus join, weekly schedules plus exceptions, immutable `doctors_clinic_sessions` rows; availability equals schedule minus exceptions; search has specialty/text/fee/sort filters with rating aggregates.
- Consultations and payments: booking checks availability then inserts under a partial-unique index (double-book race fix); creates pending Kashier payment; signed webhook settles it and confirms. Status flow Pending, Confirmed, Waiting, InProgress, Completed plus Cancelled/NoShow; `ClinicalContext` is the single authority others consult.
- Queue: `queue_windows` (one per session, status, next number, current entry) owns runtime state — never the session row (R10). Every mutation recomputes ETA and publishes to the hub plus a patient notification.
- Files: rows carry owner, scope, scope_id, storage key; bytes on LocalFS (`./data/files`) or S3/Minio (`medical-files`); downloads are presigned. Scope owners register `ScopeRule`s at startup (`clinical/scope`, `identity/scope`); uploader bypass always applies (R15).
- Clinical: one record per consultation (vitals, complaint, diagnosis, allergies, meds, conditions, notes), prescriptions, record attachments adopted through files. Writes need treating doctor plus InProgress/Completed (R16, else 422); reads resolve file IDs to fresh signed URLs.
- Messaging: per-consultation participant messages, notifications with unread counts, fan-out from booking/queue/message events; free-messages are the public rate-limited contact form plus doctor inbox plus email reply.
- Admin: verification votes propose/list/cast/expire (background goroutine from main; majority flips `verified`); suspensions propose/create/lift/list with `activecheck` at sensitive gates.
- Limits and observability: fixed-window `ratelimit.Config{Limit, Window}` with `memory` (dev) or `postgres` (shared) via `RATE_LIMIT_DRIVER`; `Allow(ctx,key)` fails open on error (R26). Buckets: free-message IP 3/h plus email 2/h, reset IP 3/h plus email 3/h, verify 3/h. Always-on pgx tracer logs DEBUG per query (WARN above 100ms) and feeds Prometheus; `httpx.Trace` propagates request IDs.
- Limits and observability: fixed-window `ratelimit.Config{Limit, Window}` with `memory` (dev) or `postgres` (shared) via `RATE_LIMIT_DRIVER`; `Allow(ctx,key)` fails open on error (R26). Buckets: free-message IP 3/h plus email 2/h, reset IP 3/h plus email 3/h, verify 3/h. Always-on pgx tracer logs DEBUG per query (WARN above 100ms) and feeds Prometheus; `httpx.Trace` propagates request IDs.

## 7. Data, config, deploy

Migrations `0001..0017` (identity, doctors, admin votes, schedules, consultations, content, queue, files, clinical, content-files adoption, profile image, messages, notifications, free messages, reset plus verify, rate limits, suspensions). They run ONLY via the `migrate` Docker stage, the CI goose step, or local `make migrate-up` — never from `main` (R31). SQLC (`sqlc.yaml`, one block per query file, `migrations/` plus `sqlc/catalog.sql` into `internal/<mod>/.../db`, pgx/v5): regenerate with `make generate`; CI asserts zero diff (`sqlc v1.29.0` pinned). Slice-18 lesson: NULLable LEFT JOIN columns need pointer emits or `CASE/COALESCE` guards (the `me` 500).

Config (`internal/platform/config`, `.env` auto-loaded outside production, canonical list in `.env.example`): `APP_ENV`, `HTTP_PORT=8080`, `DATABASE_URL*`, `JWT_SECRET*` plus `JWT_ACCESS_TTL=15m` / `JWT_REFRESH_TTL=720h`, `KASHIER_API_KEY*/URL/WEBHOOK_SECRET*`, `WS_ORIGIN_PATTERNS=*` (pin in prod), `STORAGE_DRIVER=local|s3` plus `STORAGE_LOCAL_*` / `S3_*` (S3 needs endpoint, keys, bucket), `APP_BASE_URL` (reset/verify links), `EMAIL_DRIVER=logger|smtp` plus `SMTP_*`, `RATE_LIMIT_DRIVER=memory|postgres`, `RETENTION_ENABLED=false` plus `RETENTION_CLINICAL_YEARS=7` / `BATCH_SIZE=1000` / `INTERVAL=24h`. `Validate()` fails closed (prod rejects defaults); retention validates only its subset.

Dockerfile: `build` (golang:1.26-alpine, cached modules, two `CGO_ENABLED=0 -trimpath` binaries, API version stamped) then `migrate` (alpine plus goose v3.24.3, `ENTRYPOINT goose -dir /migrations postgres`) then `api`/`retention` (distroless nonroot, no shell, under 30MB). Compose: `postgres:16` (healthcheck, pgdata), `minio` (:9000/:9001), `mailhog` (:1025/:8025), `retention` demo (`INTERVAL=1m`). CI (`.github/workflows/ci.yml`): lint job (Postgres service, goose up from clean DB, `sqlc generate`, `git diff --exit-code`, golangci-lint with depguard) then build, unit (`-race`) plus integration (`-tags=integration -race`, testcontainers), then docker builds of both images.

Deploy sequence: build images, run `migrate` image with `DATABASE_URL`, first-deploy `--seed` with `ADMIN_EMAIL/PASSWORD`, run `api` with real JWT/Kashier/S3/SMTP/`APP_BASE_URL`, opt-in `retention` with `RETENTION_ENABLED=true`. Prod checklist: live Kashier keys plus dashboard webhook URL, `/readyz` polling, Postgres backups, versioned storage, retention logging.

## 8. Tests and known gaps

Unit tests colocated (`service_test.go`/`handler_test.go`: castvote, proposevote, listspecialties, ...); integration (`integration/*_test.go`, one shared container in `TestMain`, `TruncateAll` per test); live sweep (`scripts/endpoint_smoke.sh`, 38 checks incl. role boundaries, webhook signature, token rotation). OpenAPI embedded at `/docs` (`internal/platform/docs/openapi.yaml`).

Known gaps (ROADMAP slice 18): queue never load-tested; no chaos/fault injection; 7-year retention unaudited; Kashier live-signature unverified; no Grafana/alerting; suspended users can still log in (module exists, gate unwired).

## 9. Key files

`cmd/api/main.go` (wiring plus route table), `cmd/retention/main.go`, `internal/platform/{config,database,auth,httpx,ws,email,ratelimit,metrics,docs,seed}`, `internal/platform/docs/openapi.yaml`, `migrations/0001..0017`, `sqlc.yaml`, `docker-compose.yml`, `Dockerfile`, `.env.example`, `makefile`, `golangci.yaml`, `integration/` plus `internal/testutil/`, `scripts/endpoint_smoke.sh`, `docs/ROADMAP.md`, `docs/slices/00-slice-log.md`, `docs/decisions/00-decisions.md`.
