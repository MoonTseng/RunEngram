#!/usr/bin/env bash

load_runengram_env() {
    local env_file="${1:-.env}"
    [[ -f "$env_file" ]] || return 0

    set -a
    # shellcheck disable=SC1090
    source "$env_file"
    set +a
}
