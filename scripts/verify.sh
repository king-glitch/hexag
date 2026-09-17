#!/usr/bin/env bash
# Runs verify from hexag framework to check architecture and naming rules.
#
# Usage: hexag verify [flags] [path]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HEXAG_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERIFY_BIN="$HEXAG_ROOT/bin/verify"

# Build verify binary if not present
if [ ! -x "$VERIFY_BIN" ]; then
  (cd "$HEXAG_ROOT" && go build -o "$VERIFY_BIN" ./framework/cmd/verify)
fi

exec "$VERIFY_BIN" "$@"
