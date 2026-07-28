#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
source_skill="${repo_root}/skills/taskline-management/SKILL.md"
target_dir="${repo_root}/plugins/runengram/skills/taskline-management"

mkdir -p "${target_dir}"
cp "${source_skill}" "${target_dir}/SKILL.md"
echo "synced: ${target_dir}/SKILL.md"
