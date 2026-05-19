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

required_patterns=(
  "Presto-*-macOS-*.dmg"
  "Presto-*-windows-*-installer.exe"
  "Presto-*-linux-*.tar.gz"
  "Presto-*-portable-macOS-*.dmg"
  "Presto-*-portable-windows-*"
  "Presto-*-portable-linux-*"
)

for pattern in "${required_patterns[@]}"; do
  shopt -s nullglob
  matches=("${dist_dir}"/${pattern})
  shopt -u nullglob
  if [[ "${#matches[@]}" -eq 0 ]]; then
    fail "missing release artifact matching ${pattern}"
  fi
done

checksums="${dist_dir}/checksums.txt"
[[ -f "${checksums}" ]] || fail "missing checksums.txt in ${dist_dir}"

shopt -s nullglob
presto_assets=("${dist_dir}"/Presto-*)
shopt -u nullglob
[[ "${#presto_assets[@]}" -gt 0 ]] || fail "no Presto-* release assets found in ${dist_dir}"

for asset in "${presto_assets[@]}"; do
  [[ -f "${asset}" ]] || continue
  basename="$(basename "${asset}")"
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
