#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

fail() {
  echo "PORTABLE_OFFLINE_STATIC_AUDIT=FAIL: $*" >&2
  exit 1
}

require_file() {
  local file="$1"
  [[ -f "${file}" ]] || fail "missing required file: ${file}"
}

require_grep() {
  local pattern="$1"
  local file="$2"
  local description="$3"
  require_file "${file}"
  grep -Fq -- "${pattern}" "${file}" || fail "missing ${description}: ${pattern} in ${file}"
}

require_capability_false() {
  local field="$1"
  # Static evidence targets include strings such as "OnlineRegistry: false".
  require_file "cmd/presto-desktop/channel.go"
  grep -Eq -- "${field}:[[:space:]]*false" "cmd/presto-desktop/channel.go" || fail "missing portable capability ${field}=false in cmd/presto-desktop/channel.go"
}

require_backend_gate() {
  local field="$1"
  shift
  local found=0
  local file
  for file in "$@"; do
    require_file "${file}"
    if grep -Fq -- "${field}" "${file}"; then
      found=1
      break
    fi
  done
  [[ "${found}" -eq 1 ]] || fail "missing backend gate check for ${field} in: $*"
}

require_frontend_gate() {
  local field="$1"
  shift
  local found=0
  local file
  for file in "$@"; do
    require_file "${file}"
    if grep -Fq -- "${field}" "${file}"; then
      found=1
      break
    fi
  done
  [[ "${found}" -eq 1 ]] || fail "missing frontend capability gate for ${field} in: $*"
}

portable_false_capabilities=(
  OnlineRegistry
  OnlineTemplateStore
  OnlineSkillStore
  TemplateAutoUpdate
  FirstLaunchBootstrap
  AppUpdateCheck
  ExternalBrowserLinks
)

for capability in "${portable_false_capabilities[@]}"; do
  require_capability_false "${capability}"
done

require_backend_gate "OnlineRegistry" "cmd/presto-desktop/main.go"
require_backend_gate "FirstLaunchBootstrap" "cmd/presto-desktop/main.go" "cmd/presto-desktop/template.go"
require_backend_gate "TemplateAutoUpdate" "cmd/presto-desktop/main.go" "cmd/presto-desktop/template.go"
require_backend_gate "AppUpdateCheck" "cmd/presto-desktop/main.go" "cmd/presto-desktop/updater.go"

require_frontend_gate "onlineRegistry" "frontend/src/lib/config/channel.ts" "frontend/src/lib/stores/registry.svelte.ts"
require_frontend_gate "onlineTemplateStore" "frontend/src/lib/config/channel.ts" "frontend/src/routes/+layout.svelte" "frontend/src/routes/settings/+page.svelte" "frontend/src/routes/store-templates/+page.svelte"
require_frontend_gate "onlineSkillStore" "frontend/src/lib/config/channel.ts" "frontend/src/routes/+layout.svelte" "frontend/src/routes/settings/+page.svelte" "frontend/src/routes/store-skills/+page.svelte"
require_frontend_gate "appUpdateCheck" "frontend/src/lib/config/channel.ts" "frontend/src/routes/+layout.svelte" "frontend/src/routes/settings/+page.svelte"

if grep -REn 'portable|dist-.*portable|_frontend-embed-portable' Makefile .github/workflows/release.yml \
  | grep -F 'VITE_PRESTO_CHANNEL=slim' >/dev/null; then
  fail "portable packaging command uses VITE_PRESTO_CHANNEL=slim"
fi

if grep -REn 'fonts\.googleapis\.com|fonts\.gstatic\.com|@import url\(['\''"]https://' frontend/src >/dev/null; then
  fail "frontend source contains remote font or CSS imports"
fi

if grep -Fq 'Presto-Homepage/releases' frontend/src/routes/settings/+page.svelte; then
  fail "settings update fallback still points at Presto-Homepage releases"
fi

echo "PORTABLE_OFFLINE_STATIC_AUDIT=PASS"
