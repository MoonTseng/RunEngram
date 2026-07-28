#!/usr/bin/env bash
set -euo pipefail

repository="${RUNENGRAM_REPOSITORY:-MoonTseng/RunEngram}"
version="${RUNENGRAM_VERSION:-latest}"
install_root="${RUNENGRAM_HOME:-${HOME}/.local/share/runengram}"
bin_dir="${HOME}/.local/bin"

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) echo "Unsupported operating system: $(uname -s)" >&2; exit 2 ;;
esac

case "$(uname -m)" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64) arch="amd64" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 2 ;;
esac

archive="runengram_${os}_${arch}.tar.gz"
if [[ "${version}" == "latest" ]]; then
  release_base="https://github.com/${repository}/releases/latest/download"
  install_id="latest"
else
  [[ "${version}" =~ ^[A-Za-z0-9._-]+$ ]] || {
    echo "Invalid RUNENGRAM_VERSION: ${version}" >&2
    exit 2
  }
  release_base="https://github.com/${repository}/releases/download/${version}"
  install_id="${version}"
fi

command -v curl >/dev/null 2>&1 || { echo "curl is required." >&2; exit 2; }
command -v tar >/dev/null 2>&1 || { echo "tar is required." >&2; exit 2; }

temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/runengram-install.XXXXXX")"
trap 'rm -rf "${temp_dir}"' EXIT

echo "[runengram] downloading ${archive}"
curl -fL --retry 3 -o "${temp_dir}/${archive}" "${release_base}/${archive}"
curl -fL --retry 3 -o "${temp_dir}/SHA256SUMS" "${release_base}/SHA256SUMS"

checksum_line="$(grep -E "[[:space:]]${archive}$" "${temp_dir}/SHA256SUMS" || true)"
[[ -n "${checksum_line}" ]] || {
  echo "Checksum entry missing for ${archive}." >&2
  exit 1
}
expected="$(printf '%s\n' "${checksum_line}" | awk '{print $1}')"
if command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "${temp_dir}/${archive}" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${temp_dir}/${archive}" | awk '{print $1}')"
else
  echo "shasum or sha256sum is required." >&2
  exit 2
fi
[[ "${actual}" == "${expected}" ]] || {
  echo "Checksum mismatch for ${archive}." >&2
  exit 1
}

mkdir -p "${temp_dir}/unpacked"
tar -xzf "${temp_dir}/${archive}" -C "${temp_dir}/unpacked"
for required in bin/taskline bin/taskline-server bin/runengram; do
  [[ -x "${temp_dir}/unpacked/${required}" ]] || {
    echo "Release archive missing executable ${required}." >&2
    exit 1
  }
done

version_dir="${install_root}/versions/${install_id}"
mkdir -p "${install_root}/versions" "${bin_dir}"
rm -rf "${version_dir}"
mkdir -p "${version_dir}"
cp -R "${temp_dir}/unpacked/." "${version_dir}/"
ln -sfn "${version_dir}" "${install_root}/current"

link_binary() {
  local name="$1"
  local source="${install_root}/current/bin/${name}"
  local target="${bin_dir}/${name}"
  if [[ -e "${target}" && ! -L "${target}" ]]; then
    echo "Refusing to overwrite non-symlink: ${target}" >&2
    exit 1
  fi
  ln -sfn "${source}" "${target}"
}

link_binary taskline
link_binary runengram
"${bin_dir}/runengram" restart

echo "[runengram] installed ${version} for ${os}/${arch}"
echo "[runengram] UI: http://127.0.0.1:8787"
case ":${PATH}:" in
  *":${bin_dir}:"*) ;;
  *) echo "[runengram] add ${bin_dir} to PATH." ;;
esac
