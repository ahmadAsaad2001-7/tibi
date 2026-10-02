.PHONY: generate migrate-up migrate-down migrate-status run lint test

# Recipes load .env themselves so make does not parse `?` in DATABASE_URL.
define load-env
set -a; [ -f .env ] && . ./.env; set +a
endef

generate:
	sqlc compile

migrate-up:
	@$(load-env); goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	@$(load-env); goose -dir migrations postgres "$$DATABASE_URL" down

migrate-status:
	@$(load-env); goose -dir migrations postgres "$$DATABASE_URL" status

run:
	go run ./cmd/api

lint:
	golangci-lint run

test:
	go test ./... -race -count=1
