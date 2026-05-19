#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

dist_dir="${1:-dist}"

fail() {
  echo "RELEASE_ASSET_MATRIX=FAIL: $*" >&2
  exit 1
}

[[ -d "${dist_dir}" ]] || fail "dist directory not found: ${dist_dir}"

require_match() {
  local label="$1"
  local pattern="$2"
  local allow_portable="${3:-yes}"
  shopt -s nullglob
  local matches=("${dist_dir}"/${pattern})
  shopt -u nullglob
  if [[ "${allow_portable}" == "no" ]]; then
    local filtered=()
    local asset
    for asset in "${matches[@]}"; do
      [[ "$(basename "${asset}")" == *-portable-* ]] && continue
      filtered+=("${asset}")
    done
    matches=("${filtered[@]}")
  fi
  if [[ "${#matches[@]}" -eq 0 ]]; then
    fail "missing ${label} release artifact matching ${pattern}"
  fi
}

require_match "default macOS" "Presto-*-macOS-*.dmg" "no"
require_match "default Windows installer" "Presto-*-windows-amd64-installer.exe" "no"
require_match "default Linux" "Presto-*-linux-amd64.tar.gz" "no"
require_match "portable macOS" "Presto-*-portable-macOS-*.dmg"
require_match "portable Windows ZIP" "Presto-*-portable-windows-amd64.zip"

if [[ "${ALLOW_PORTABLE_TAR_FALLBACK:-0}" == "1" ]]; then
  shopt -s nullglob
  portable_linux_appimages=("${dist_dir}"/Presto-*-portable-linux-amd64.AppImage)
  shopt -u nullglob
  if [[ "${#portable_linux_appimages[@]}" -eq 0 ]]; then
    require_match "portable Linux explicit tar fallback" "Presto-*-portable-linux-amd64.tar.gz"
  fi
else
  require_match "portable Linux AppImage" "Presto-*-portable-linux-amd64.AppImage"
fi

checksums="${dist_dir}/checksums.txt"
[[ -f "${checksums}" ]] || fail "missing checksums.txt in ${dist_dir}"

shopt -s nullglob
presto_assets=("${dist_dir}"/Presto-*)
shopt -u nullglob
[[ "${#presto_assets[@]}" -gt 0 ]] || fail "no Presto-* release assets found in ${dist_dir}"

allowed_asset() {
  local basename="$1"
  case "${basename}" in
    Presto-*-portable-macOS-*.dmg) return 0 ;;
    Presto-*-portable-windows-amd64.zip) return 0 ;;
    Presto-*-portable-linux-amd64.AppImage) return 0 ;;
    Presto-*-portable-linux-amd64.tar.gz) [[ "${ALLOW_PORTABLE_TAR_FALLBACK:-0}" == "1" ]] ;;
    Presto-*-macOS-*.dmg) [[ "${basename}" != *-portable-* ]] ;;
    Presto-*-windows-amd64-installer.exe) [[ "${basename}" != *-portable-* ]] ;;
    Presto-*-linux-amd64.tar.gz) [[ "${basename}" != *-portable-* ]] ;;
    *) return 1 ;;
  esac
}

for asset in "${presto_assets[@]}"; do
  [[ -f "${asset}" ]] || continue
  basename="$(basename "${asset}")"
  if ! allowed_asset "${basename}"; then
    fail "unexpected release artifact shape: ${basename}"
  fi
  count="$(awk -v name="${basename}" '$2 == name { count++ } END { print count + 0 }' "${checksums}")"
  if [[ "${count}" -ne 1 ]]; then
    fail "${basename} appears ${count} times in checksums.txt; expected exactly once"
  fi
done

while read -r _hash filename _rest; do
  [[ -n "${filename:-}" ]] || continue
  if [[ "${filename}" == Presto-* && ! -f "${dist_dir}/${filename}" ]]; then
    fail "checksums.txt references missing artifact: ${filename}"
  fi
done < "${checksums}"

echo "RELEASE_ASSET_MATRIX=PASS"
