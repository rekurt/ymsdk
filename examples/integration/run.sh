#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${YM_TOKEN:-}" ]]; then
  echo "YM_TOKEN is required" >&2
  exit 1
fi

echo "Running ymsdk integration example..."
go run . "$@"
