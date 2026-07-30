#!/usr/bin/env bash
# scripts/build.sh — produce all release artifacts under ./dist/
#
# Output:
#   dist/runengram-server  server binary (with embedded web UI)
#   dist/runengram         CLI binary
#   dist/taskline*         compatibility symlinks
#
# The frontend bundle is embedded into the server binary at compile time;
# vite is configured to write into ../server/web/dist so go:embed picks it up.
set -euo pipefail

cd "$(dirname "$0")/.."
mkdir -p dist

echo "[build] web (pnpm build → server/web/dist/)" >&2
( cd web && pnpm install --silent && pnpm build )

echo "[build] runengram-server" >&2
( cd server && go build -o ../dist/runengram-server ./cmd/runengram-server )

echo "[build] runengram (CLI)" >&2
( cd cli && go build -o ../dist/runengram . )

rm -f dist/taskline-server dist/taskline
ln -s runengram-server dist/taskline-server
ln -s runengram dist/taskline

echo "[build] done — dist/runengram-server  dist/runengram"
