#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
plugin_root="${repo_root}/plugins/runengram"
installer="${plugin_root}/scripts/install-runengram.sh"
manifest="${plugin_root}/.codex-plugin/plugin.json"
test_root="$(mktemp -d "${TMPDIR:-/tmp}/runengram-installer-test.XXXXXX")"
trap 'rm -rf "${test_root}"' EXIT

manifest_version="$(
  sed -nE 's/^[[:space:]]*"version"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/p' "${manifest}" |
    head -n 1
)"
base_version="${manifest_version%%+*}"
expected_version="v${base_version}"

curl() {
  if [[ -n "${RUNENGRAM_TEST_ARCHIVE:-}" ]]; then
    local output=""
    local url="${*: -1}"
    local index=1
    while (( index <= $# )); do
      if [[ "${!index}" == "-o" ]]; then
        (( index += 1 ))
        output="${!index}"
        break
      fi
      (( index += 1 ))
    done
    case "${url}" in
      */runengram_*.tar.gz) cp "${RUNENGRAM_TEST_ARCHIVE}" "${output}" ;;
      */SHA256SUMS) cp "${RUNENGRAM_TEST_SUMS}" "${output}" ;;
      *) return 22 ;;
    esac
    return 0
  fi
  printf '%s\n' "$*" >> "${RUNENGRAM_TEST_CURL_CALLS}"
  return 22
}
export -f curl

run_probe() {
  local calls_file="$1"
  shift
  : > "${calls_file}"
  if env \
    HOME="${test_root}/home" \
    RUNENGRAM_TEST_CURL_CALLS="${calls_file}" \
    "$@" \
    bash "${installer}" >"${test_root}/output.log" 2>&1; then
    echo "installer probe unexpectedly succeeded" >&2
    exit 1
  fi
}

default_calls="${test_root}/default-curl-calls"
run_probe "${default_calls}" env
grep -q "/releases/download/${expected_version}/" "${default_calls}" || {
  echo "installer did not bind default download to plugin ${manifest_version}" >&2
  cat "${default_calls}" >&2
  exit 1
}

override_calls="${test_root}/override-curl-calls"
run_probe "${override_calls}" env RUNENGRAM_VERSION=v9.8.7
grep -q "/releases/download/v9.8.7/" "${override_calls}" || {
  echo "installer ignored RUNENGRAM_VERSION override" >&2
  cat "${override_calls}" >&2
  exit 1
}

fixture="${test_root}/fixture"
mkdir -p "${fixture}/bin"
for executable in runengram runengram-server; do
  printf '#!/usr/bin/env bash\nexit 0\n' > "${fixture}/bin/${executable}"
  chmod +x "${fixture}/bin/${executable}"
done
printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$*" >> "${RUNENGRAM_TEST_SERVICE_CALLS}"\n' \
  > "${fixture}/bin/runengram-service"
chmod +x "${fixture}/bin/runengram-service"
ln -s runengram "${fixture}/bin/taskline"

case "$(uname -s)" in
  Darwin) test_os="darwin" ;;
  Linux) test_os="linux" ;;
  *) echo "unsupported test operating system" >&2; exit 2 ;;
esac
case "$(uname -m)" in
  arm64|aarch64) test_arch="arm64" ;;
  x86_64|amd64) test_arch="amd64" ;;
  *) echo "unsupported test architecture" >&2; exit 2 ;;
esac
archive_name="runengram_${test_os}_${test_arch}.tar.gz"
archive_path="${test_root}/${archive_name}"
tar -czf "${archive_path}" -C "${fixture}" .
checksum="$(LC_ALL=C LANG=C shasum -a 256 "${archive_path}" | LC_ALL=C LANG=C awk '{print $1}')"
printf '%s  %s\n' "${checksum}" "${archive_name}" > "${test_root}/SHA256SUMS"

install_home="${test_root}/legacy-home"
mkdir -p "${install_home}/.local/bin"
printf '#!/usr/bin/env bash\necho "taskline dev (unknown)"\n' > "${install_home}/.local/bin/taskline"
chmod +x "${install_home}/.local/bin/taskline"
service_calls="${test_root}/service-calls"
: > "${service_calls}"
if ! HOME="${install_home}" \
  RUNENGRAM_HOME="${test_root}/installed" \
  RUNENGRAM_TEST_ARCHIVE="${archive_path}" \
  RUNENGRAM_TEST_SUMS="${test_root}/SHA256SUMS" \
  RUNENGRAM_TEST_SERVICE_CALLS="${service_calls}" \
    bash "${installer}" >"${test_root}/install-output.log" 2>&1; then
  echo "installer failed to migrate legacy taskline binary" >&2
  cat "${test_root}/install-output.log" >&2
  exit 1
fi
[[ -L "${install_home}/.local/bin/taskline" ]] || {
  echo "installer did not migrate legacy taskline binary" >&2
  cat "${test_root}/install-output.log" >&2
  exit 1
}
grep -qx 'restart' "${service_calls}" || {
  echo "installer did not restart service after migration" >&2
  exit 1
}

echo "ok: plugin runtime version follows ${manifest_version}"
