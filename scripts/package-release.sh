#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
version="${1:-}"
target_os="${GOOS:-$(go env GOOS)}"
target_arch="${GOARCH:-$(go env GOARCH)}"
output_dir="${RUNENGRAM_OUTPUT_DIR:-${repo_root}/dist/release}"
commit="${RUNENGRAM_COMMIT:-$(git -C "${repo_root}" rev-parse --short=12 HEAD)}"

[[ -n "${version}" ]] || {
  echo "usage: scripts/package-release.sh <version>" >&2
  exit 2
}
[[ "${version}" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$ ]] || {
  echo "invalid version: ${version}" >&2
  exit 2
}
case "${target_os}" in
  darwin|linux) ;;
  *) echo "unsupported GOOS: ${target_os}" >&2; exit 2 ;;
esac
case "${target_arch}" in
  amd64|arm64) ;;
  *) echo "unsupported GOARCH: ${target_arch}" >&2; exit 2 ;;
esac

if [[ "${RUNENGRAM_SKIP_WEB:-0}" != "1" ]]; then
  echo "[package] web" >&2
  ( cd "${repo_root}/web" && pnpm install --frozen-lockfile && pnpm build )
elif [[ ! -f "${repo_root}/server/web/dist/index.html" ]]; then
  echo "RUNENGRAM_SKIP_WEB=1 requires server/web/dist/index.html" >&2
  exit 2
fi

staging_dir="$(mktemp -d "${TMPDIR:-/tmp}/runengram-package.XXXXXX")"
trap 'rm -rf "${staging_dir}"' EXIT
mkdir -p "${staging_dir}/bin" "${output_dir}"

echo "[package] taskline-server ${target_os}/${target_arch}" >&2
(
  cd "${repo_root}/server"
  CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" \
    go build -trimpath -o "${staging_dir}/bin/taskline-server" ./cmd/taskline-server
)

echo "[package] taskline ${target_os}/${target_arch}" >&2
(
  cd "${repo_root}/cli"
  CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" \
    go build -trimpath \
      -ldflags "-s -w -X main.Version=${version} -X main.Commit=${commit}" \
      -o "${staging_dir}/bin/taskline" .
)

cp "${repo_root}/plugins/runengram/scripts/runengram-service.sh" "${staging_dir}/bin/runengram"
chmod +x "${staging_dir}/bin/taskline-server" "${staging_dir}/bin/taskline" "${staging_dir}/bin/runengram"
cp "${repo_root}/LICENSE" "${staging_dir}/LICENSE"
cp "${repo_root}/README.md" "${staging_dir}/README.md"
cp "${repo_root}/README.zh-CN.md" "${staging_dir}/README.zh-CN.md"

archive="${output_dir}/runengram_${target_os}_${target_arch}.tar.gz"
rm -f "${archive}"
tar -czf "${archive}" -C "${staging_dir}" .
echo "${archive}"
