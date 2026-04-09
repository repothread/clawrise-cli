#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../../../../" && pwd)

cd "$REPO_ROOT"
exec go run ./cmd/clawrise-plugin-sample-audit
