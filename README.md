# Tibi — Medical Consultation Platform

A Go REST API for a medical consultation platform: identity, doctors,
bookings, queue, payments (Kashier), files, clinical records, messaging,
notifications, and admin governance. Built in 18 slices; see
[`docs/slices/00-slice-log.md`](docs/slices/00-slice-log.md) and
[`docs/ROADMAP.md`](docs/ROADMAP.md).

## Prerequisites

- Go 1.26+
- PostgreSQL 16 (or `docker compose` for the full local stack)
- `goose` for migrations — `go install github.com/pressly/goose/v3/cmd/goose@v3.24.3`
- optional: Docker (compose stack, containerized builds, integration tests)

## Run locally

```bash
# 1. Create your env file
cp .env.example .env            # then edit DATABASE_URL, secrets, etc.

# 2. Start the local stack (Postgres + Minio + Mailhog)
docker compose up -d

# 3. Apply migrations
goose -dir migrations postgres "$DATABASE_URL" up

# 4. Run the API
make run                        # or: go run ./cmd/api
```

The server listens on `:8080` (see `.env`). Open the Scalar API reference at
http://localhost:8080/docs. Minio console: http://localhost:9001
(credentials `minio` / `minio123`). Mailhog: http://localhost:8025.

To run against your own Postgres without Docker, point `DATABASE_URL` at it
and leave `STORAGE_DRIVER=local` (files land in `./data/files`).

## Unit + integration tests

```bash
make test                       # unit tests, all packages
go test ./integration -tags=integration -count=1   # real Postgres via testcontainers
```

The integration suite starts one Postgres 16 container, applies every
migration, and runs the suite against it. Requires Docker.

## Lint

```bash
golangci-lint run
```

## Deploy

### Build

```bash
docker build --target api .        # runtime image < 30 MB, non-root, no shell
docker build --target retention .  # retention worker image
```

### Migrate (deployment step, never app startup — R31)

```bash
docker build --target migrate -t tibi-migrate .
docker run --rm -e DATABASE_URL="$DATABASE_URL" tibi-migrate up
```

### Seed admin (first deploy only — R31 complements 18f)

```bash
docker run --rm \
  -e DATABASE_URL="$DATABASE_URL" \
  -e ADMIN_EMAIL=admin@example.com \
  -e ADMIN_PASSWORD='a-strong-password' \
  medical-api:latest --seed
```

`--seed` is idempotent: it re-inserts the specialty catalog with
`ON CONFLICT DO NOTHING` and creates the admin only if the email is new.

### Run the API

```bash
docker run --rm -p 8080:8080 \
  -e APP_ENV=production \
  -e DATABASE_URL="$DATABASE_URL" \
  -e JWT_SECRET="$(openssl rand -base64 32)" \
  -e KASHIER_API_KEY="$KASHIER_API_KEY" \
  -e KASHIER_WEBHOOK_SECRET="$KASHIER_WEBHOOK_SECRET" \
  -e STORAGE_DRIVER=s3 -e S3_ENDPOINT="$S3_ENDPOINT" \
  -e S3_ACCESS_KEY="$S3_ACCESS_KEY" -e S3_SECRET_KEY="$S3_SECRET_KEY" \
  -e S3_BUCKET="$S3_BUCKET" \
  medical-api:latest
```

### Run the retention worker (opt-in)

```bash
docker run --rm \
  -e DATABASE_URL="$DATABASE_URL" \
  -e RETENTION_ENABLED=true \
  -e RETENTION_CLINICAL_YEARS=7 \
  medical-retention:latest
```

### Production checklist

The full pre-ship checklist lives in slice 18 (`docs/ROADMAP.md`). The
short version: validate `JWT_SECRET` is not the default, `KASHIER_API_KEY`
is not a test key, webhook signature verification is on, assigned admin
routes are `RoleAdmin`-gated, `/readyz` is polled, Postgres is backed up,
object storage has versioning, retention is enabled and logging, and the
Kashier dashboard points at the production webhook URL.

## Configuration

Every setting is read from the environment; `.env` is loaded automatically
in development (see `internal/platform/config`). The canonical, commented
list is in `.env.example`.

## Architecture

- One binary per concern: `cmd/api` serves HTTP + WebSockets; `cmd/retention`
  mutates data on a schedule. SQLC-generated repositories, pgx v5, chi,
  Prometheus metrics, Scalar docs.
- Module boundaries are enforced by `depguard` (see `golangci.yaml`); modules
  talk through package-level `contracts` interfaces.
- See [`docs/decisions/00-decisions.md`](docs/decisions/00-decisions.md) for
  the decision log (R1–R32).