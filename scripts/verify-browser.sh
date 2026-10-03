#!/bin/sh
# Exercises writes only in the dedicated finguard_test database.
set -eu
cd "$(dirname "$0")/.."
export DATABASE_URL='postgres://finguard_app:finguard_app@localhost:54329/finguard_test?sslmode=disable'
export MIGRATE_DATABASE_URL='postgres://finguard:finguard@localhost:54329/finguard_test?sslmode=disable'
export AI_PROVIDER=stub
export HTTP_ADDR=127.0.0.1:8081
export RATE_LIMIT_PER_MIN=1200
if curl -fsS http://127.0.0.1:8081/healthz >/dev/null 2>&1; then
  echo 'Port 8081 is occupied. Stop the existing test server before running this script.' >&2
  exit 1
fi
npm --prefix frontend run build
cp -R frontend/dist/. internal/web/dist/
mkdir -p bin
go build -o bin/finguard-e2e ./cmd/finguard
bin/finguard-e2e seed --reset
bin/finguard-e2e api > /tmp/finguard-browser-server.log 2>&1 &
server_pid=$!
trap 'kill "$server_pid" 2>/dev/null || true' EXIT INT TERM
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS http://127.0.0.1:8081/readyz >/dev/null 2>&1; then break; fi
  sleep 1
done
npm --prefix frontend run test:e2e
