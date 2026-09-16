#!/usr/bin/env bash
# Runs brunogen from hexag framework to generate Bruno API collection and docs/API.md.
#
# Usage: hexag bruno [flags]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HEXAG_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BRUNOGEN_BIN="$HEXAG_ROOT/bin/brunogen"

# Build brunogen binary if not present
if [ ! -x "$BRUNOGEN_BIN" ]; then
  (cd "$HEXAG_ROOT" && go build -o "$BRUNOGEN_BIN" ./framework/cmd/brunogen)
fi

# If -api-md flag is not provided in args, default to docs/API.md
HAS_API_MD=false
for arg in "$@"; do
  if [[ "$arg" == "-api-md" ]] || [[ "$arg" == "--api-md" ]] || [[ "$arg" == -api-md=* ]]; then
    HAS_API_MD=true
    break
  fi
done

if [ "$HAS_API_MD" = false ]; then
  exec "$BRUNOGEN_BIN" -api-md docs/API.md "$@"
else
  exec "$BRUNOGEN_BIN" "$@"
fi
