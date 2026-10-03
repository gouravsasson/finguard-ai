.PHONY: db test unit integration api seed build ui demo

export DATABASE_URL ?= postgres://finguard_app:finguard_app@localhost:54329/finguard?sslmode=disable
export MIGRATE_DATABASE_URL ?= postgres://finguard:finguard@localhost:54329/finguard?sslmode=disable
export TEST_DATABASE_URL ?= postgres://finguard_app:finguard_app@localhost:54329/finguard_test?sslmode=disable

db:
	docker compose up -d postgres

test:
	go test ./...

integration:
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' MIGRATE_DATABASE_URL='$(MIGRATE_DATABASE_URL)' go test ./internal/integration -count=1

api:
	go run ./cmd/finguard api

seed:
	go run ./cmd/finguard seed --reset

ui:
	cd frontend && npm install && npm run build
	rm -rf internal/web/dist
	mkdir -p internal/web/dist
	cp -R frontend/dist/. internal/web/dist/

build: ui
	go build -o bin/finguard ./cmd/finguard

demo: db
	@echo "waiting for postgres"
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do docker compose exec -T postgres pg_isready -U finguard -d finguard && break; sleep 1; done
	go run ./cmd/finguard seed --reset
	go run ./cmd/finguard api
