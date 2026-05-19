#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <AppDir> <output.AppImage>" >&2
  exit 2
fi

appdir="$1"
output="$2"

require_file() {
  if [ ! -f "$1" ]; then
    echo "ERROR: missing required AppDir file: $1" >&2
    exit 1
  fi
}

require_dir() {
  if [ ! -d "$1" ]; then
    echo "ERROR: missing required AppDir directory: $1" >&2
    exit 1
  fi
}

require_file "$appdir/AppRun"
require_file "$appdir/presto.desktop"
require_file "$appdir/usr/bin/Presto"
require_file "$appdir/usr/bin/typst"
require_file "$appdir/usr/bin/tinymist"
require_dir "$appdir/usr/share/presto/templates"
require_file "$appdir/usr/share/presto/templates/registry.json"
require_dir "$appdir/usr/share/presto/templates/gongwen"
require_dir "$appdir/usr/share/presto/templates/jiaoan-shicao"
require_dir "$appdir/usr/share/presto/templates/jiaoan-jihua"

mkdir -p "$(dirname "$output")"

if command -v appimagetool >/dev/null 2>&1; then
  appimagetool "$appdir" "$output"
  echo "==> $output"
  exit 0
fi

if [ "${ALLOW_PORTABLE_TAR_FALLBACK:-}" = "1" ]; then
  fallback="${output%.AppImage}.tar.gz"
  tar -C "$(dirname "$appdir")" -czf "$fallback" "$(basename "$appdir")"
  echo "WARNING: appimagetool unavailable; emitted explicit portable tar fallback: $fallback" >&2
  exit 0
fi

echo "ERROR: appimagetool not found. Install appimagetool or set ALLOW_PORTABLE_TAR_FALLBACK=1 to emit a documented tar fallback." >&2
exit 1
