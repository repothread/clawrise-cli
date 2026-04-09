#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)

cd "$REPO_ROOT"

if [ -z "${GOCACHE:-}" ]; then
  export GOCACHE="${TMPDIR:-/tmp}/clawrise-go-build"
fi

exec go test ./internal/plugin -run 'TestProjectSample'
