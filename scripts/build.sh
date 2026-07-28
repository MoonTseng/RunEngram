#!/usr/bin/env bash
# scripts/build.sh — produce all release artifacts under ./dist/
#
# Output:
#   dist/taskline-server   server binary (with embedded web UI)
#   dist/taskline          CLI binary
#
# The frontend bundle is embedded into the server binary at compile time;
# vite is configured to write into ../server/web/dist so go:embed picks it up.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p dist

echo "[build] web (pnpm build → server/web/dist/)" >&2
( cd web && pnpm install --silent && pnpm build )

echo "[build] taskline-server" >&2
( cd server && go build -o ../dist/taskline-server ./cmd/taskline-server )

echo "[build] taskline (CLI)" >&2
( cd cli && go build -o ../dist/taskline . )

echo "[build] done — dist/taskline-server  dist/taskline"
