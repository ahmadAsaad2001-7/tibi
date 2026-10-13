# syntax=docker/dockerfile:1.7
#
# Slice 18a — reproducible multi-stage image.
#   docker build --target api .        -> runtime API image (<30 MB, non-root)
#   docker build --target retention .  -> retention worker image
#   docker build --target migrate .    -> goose binary + migrations (CI/CD only)
#
# SD53: runtime images are gcr.io/distroless/static-debian12:nonroot (no
# shell, CA certs included). SD54: CGO_ENABLED=0 — pgx is pure Go.
# go.mod declares go 1.26, so the build stage cannot be go1.23-alpine.

# ---- build stage ----
FROM golang:1.26-alpine AS build

WORKDIR /src

# Cache dependencies separately from source.
# GOPROXY is the default module proxy; pinned go.sum keeps it reproducible.
COPY go.mod go.sum ./
RUN go mod download

# Build the two binaries. version is stamped for the API binary; retention
# does not need it.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w -X main.version=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build \
        -trimpath \
        -ldflags="-s -w" \
        -o /out/retention ./cmd/retention

# ---- migrate stage (has goose, used only in CI/CD) ----
# R31: migrations never run from the API binary. This image is the single
# migration runner: it reads DATABASE_URL and runs goose against /migrations.
FROM alpine:3.20 AS migrate

RUN apk add --no-cache ca-certificates
COPY --from=build /src/migrations /migrations
RUN wget -q -O /usr/local/bin/goose \
        https://github.com/pressly/goose/releases/download/v3.24.3/goose_linux_x86_64 && \
    chmod +x /usr/local/bin/goose

ENTRYPOINT ["/usr/local/bin/goose", "-dir", "/migrations", "postgres"]

# ---- api runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot AS api

COPY --from=build /out/api /api

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]

# ---- retention runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot AS retention

COPY --from=build /out/retention /retention

USER nonroot:nonroot
ENTRYPOINT ["/retention"]