.PHONY: db-up db-down sqlc run test fmt

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate

run:
	go run ./cmd/web

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal
