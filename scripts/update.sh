#!/usr/bin/env bash
# Updates AI agent standards (AGENTS.md, CLAUDE.md) and optional framework
# template files in an existing hexag project.
#
# Usage: update.sh [dest-dir] [flags]
# Flags:
#   --mockery    Also update .mockery.yml
#   --all        Update AGENTS.md, CLAUDE.md, and .mockery.yml
#
# If dest-dir is omitted, defaults to current directory.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HEXAG_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_DIR="$HEXAG_ROOT/template"

DEST_DIR="."
UPDATE_MOCKERY=false

for arg in "$@"; do
  case "$arg" in
    --mockery)
      UPDATE_MOCKERY=true
      ;;
    --all)
      UPDATE_MOCKERY=true
      ;;
    -h|--help)
      echo "usage: hexag update [dest-dir] [--mockery] [--all]"
      echo ""
      echo "updates AGENTS.md and ensures CLAUDE.md links to it in a project."
      echo "options:"
      echo "  --mockery   also update .mockery.yml"
      echo "  --all       update all framework config files (.mockery.yml)"
      exit 0
      ;;
    *)
      if [[ "$arg" != -* ]]; then
        DEST_DIR="$arg"
      fi
      ;;
  esac
done

if [ ! -d "$DEST_DIR" ]; then
  echo "error: destination directory '$DEST_DIR' does not exist" >&2
  exit 1
fi

DEST_DIR="$(cd "$DEST_DIR" && pwd)"

if [ ! -f "$DEST_DIR/go.mod" ]; then
  echo "error: no go.mod found in $DEST_DIR" >&2
  exit 1
fi

MODULE_PATH="$(grep -E '^[[:space:]]*module[[:space:]]+' "$DEST_DIR/go.mod" | head -n 1 | awk '{print $2}')"
if [ -z "$MODULE_PATH" ]; then
  echo "error: failed to detect module path from $DEST_DIR/go.mod" >&2
  exit 1
fi

echo "target project: $DEST_DIR"
echo "module path:    $MODULE_PATH"

# 1. Update AGENTS.md from template/AGENTS.md
sed \
  -e "s|{{MODULE_PATH}}|$MODULE_PATH|g" \
  -e "s|{{HEXAG_PATH}}|$HEXAG_ROOT|g" \
  "$TEMPLATE_DIR/AGENTS.md" > "$DEST_DIR/AGENTS.md"
echo "✓ updated $DEST_DIR/AGENTS.md"

# 2. Update/ensure CLAUDE.md references @AGENTS.md
if [ ! -f "$DEST_DIR/CLAUDE.md" ]; then
  echo "@AGENTS.md" > "$DEST_DIR/CLAUDE.md"
  echo "✓ created $DEST_DIR/CLAUDE.md"
else
  if ! grep -q "@AGENTS.md" "$DEST_DIR/CLAUDE.md"; then
    tmp_claude="$(mktemp)"
    printf "@AGENTS.md\n\n%s\n" "$(cat "$DEST_DIR/CLAUDE.md")" > "$tmp_claude"
    mv "$tmp_claude" "$DEST_DIR/CLAUDE.md"
    echo "✓ prepended @AGENTS.md to $DEST_DIR/CLAUDE.md"
  else
    echo "✓ $DEST_DIR/CLAUDE.md already references @AGENTS.md"
  fi
fi

# 3. Optional: .mockery.yml
if [ "$UPDATE_MOCKERY" = true ]; then
  sed \
    -e "s|{{MODULE_PATH}}|$MODULE_PATH|g" \
    "$TEMPLATE_DIR/.mockery.yml" > "$DEST_DIR/.mockery.yml"
  echo "✓ updated $DEST_DIR/.mockery.yml"
fi
