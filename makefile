.PHONY: generate migrate-up migrate-down run lint test

generate:
	sqlc generate

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

run:
	go run ./cmd/api

lint:
	golangci-lint run

test:
	go test ./... -race -count=1