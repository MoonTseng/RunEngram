#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT

printf '%s\n' \
  'TASKLINE_LISTEN=127.0.0.1:19991' \
  'TASKLINE_DB=/tmp/runengram-env-test.db' \
  > "$test_dir/.env"

unset TASKLINE_LISTEN TASKLINE_DB

# shellcheck source=scripts/lib/load-env.sh
source scripts/lib/load-env.sh
load_runengram_env "$test_dir/.env"

[[ "${TASKLINE_LISTEN}" == "127.0.0.1:19991" ]]
[[ "${TASKLINE_DB}" == "/tmp/runengram-env-test.db" ]]

echo "ok: start-local loads .env before resolving listen and storage config"
