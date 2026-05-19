#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <destination-dir>" >&2
  exit 2
fi

dest="$1"
presto_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
registry_root="$presto_root/../template-registry"
templates_root="$registry_root/templates"
dest_templates="$dest/templates"

mkdir -p "$dest_templates"

for name in gongwen jiaoan-shicao jiaoan-jihua; do
  src="$templates_root/$name"
  if [ ! -f "$src/manifest.json" ]; then
    echo "ERROR: missing official template manifest: $src/manifest.json" >&2
    exit 1
  fi
  rm -rf "$dest_templates/$name"
  cp -R "$src" "$dest_templates/$name"
done

if [ ! -f "$registry_root/registry.json" ]; then
  echo "ERROR: missing production template registry snapshot: $registry_root/registry.json" >&2
  exit 1
fi
cp "$registry_root/registry.json" "$dest_templates/registry.json"

for name in gongwen jiaoan-shicao jiaoan-jihua; do
  test -f "$dest_templates/$name/manifest.json" || {
    echo "ERROR: portable template copy failed for $name" >&2
    exit 1
  }
done

echo "==> portable official templates copied to $dest_templates"
