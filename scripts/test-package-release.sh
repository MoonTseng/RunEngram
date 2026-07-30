#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
test_output="$(mktemp -d "${TMPDIR:-/tmp}/runengram-release-test.XXXXXX")"
unpacked="${test_output}/unpacked"
trap 'rm -rf "${test_output}"' EXIT

case "$(uname -s)" in
  Darwin) target_os="darwin" ;;
  Linux) target_os="linux" ;;
  *) echo "unsupported test operating system: $(uname -s)" >&2; exit 2 ;;
esac
case "$(uname -m)" in
  arm64|aarch64) target_arch="arm64" ;;
  x86_64|amd64) target_arch="amd64" ;;
  *) echo "unsupported test architecture: $(uname -m)" >&2; exit 2 ;;
esac

RUNENGRAM_SKIP_WEB=1 \
RUNENGRAM_OUTPUT_DIR="${test_output}" \
GOOS="${target_os}" \
GOARCH="${target_arch}" \
  "${repo_root}/scripts/package-release.sh" v0.0.0-test >/dev/null

archive="${test_output}/runengram_${target_os}_${target_arch}.tar.gz"
[[ -f "${archive}" ]] || { echo "missing release archive: ${archive}" >&2; exit 1; }
mkdir -p "${unpacked}"
tar -xzf "${archive}" -C "${unpacked}"

for executable in runengram runengram-server runengram-service taskline taskline-server; do
  [[ -x "${unpacked}/bin/${executable}" ]] || {
    echo "missing executable: bin/${executable}" >&2
    exit 1
  }
done
for document in LICENSE README.md README.zh-CN.md; do
  [[ -f "${unpacked}/${document}" ]] || {
    echo "missing document: ${document}" >&2
    exit 1
  }
done

version_output="$("${unpacked}/bin/runengram" version)"
[[ "${version_output}" == runengram\ v0.0.0-test* ]] || {
  echo "unexpected runengram version: ${version_output}" >&2
  exit 1
}

echo "ok: ${archive}"
