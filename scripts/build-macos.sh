#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! xcode-select -p >/dev/null 2>&1; then
  echo "Xcode Command Line Tools required. Install with: xcode-select --install" >&2
  exit 1
fi

mkdir -p bin

export CGO_ENABLED=1
export GOOS=darwin
export GOARCH="${GOARCH:-arm64}"
export MACOSX_DEPLOYMENT_TARGET="${MACOSX_DEPLOYMENT_TARGET:-12.3}"

go build -o bin/screenshare ./cmd/screenshare
echo "Built bin/screenshare (${GOOS}/${GOARCH})"
