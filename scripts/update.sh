#!/usr/bin/env bash
# Updates AI agent standards (AGENTS.md, agent hooks) and optional framework
# template files in an existing hexag project.
#
# Usage: update.sh [dest-dir] [flags]
# Flags:
#   --mockery    Also update .mockery.yml
#   --makefile   Also update makefile
#   --all        Update AGENTS.md, agent hooks, .mockery.yml, and makefile
#
# If dest-dir is omitted, defaults to current directory.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HEXAG_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_DIR="$HEXAG_ROOT/template"

DEST_DIR="."
UPDATE_MOCKERY=false
UPDATE_MAKEFILE=false

for arg in "$@"; do
  case "$arg" in
    --mockery)
      UPDATE_MOCKERY=true
      ;;
    --makefile)
      UPDATE_MAKEFILE=true
      ;;
    --all)
      UPDATE_MOCKERY=true
      UPDATE_MAKEFILE=true
      ;;
    -h|--help)
      echo "usage: hexag update [dest-dir] [--mockery] [--makefile] [--all]"
      echo ""
      echo "updates AGENTS.md and installs missing agent hooks in a project."
      echo "options:"
      echo "  --mockery    also update .mockery.yml"
      echo "  --makefile   also update makefile"
      echo "  --all        update all framework config files (.mockery.yml, makefile)"
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

# 2. Agent hook settings: created only when missing so project customizations survive.
for rel in .claude/settings.json .agents/hooks.json; do
  if [ ! -e "$DEST_DIR/$rel" ]; then
    mkdir -p "$DEST_DIR/$(dirname "$rel")"
    cp "$TEMPLATE_DIR/$rel" "$DEST_DIR/$rel"
    echo "✓ created $DEST_DIR/$rel"
  fi
done

# 3. Optional: .mockery.yml
if [ "$UPDATE_MOCKERY" = true ]; then
  sed \
    -e "s|{{MODULE_PATH}}|$MODULE_PATH|g" \
    "$TEMPLATE_DIR/.mockery.yml" > "$DEST_DIR/.mockery.yml"
  echo "✓ updated $DEST_DIR/.mockery.yml"
fi

# 4. Optional: makefile
if [ "$UPDATE_MAKEFILE" = true ]; then
  cp "$TEMPLATE_DIR/makefile" "$DEST_DIR/makefile"
  echo "✓ updated $DEST_DIR/makefile"
fi

# 5. Initialize MEMORY.md if it doesn't exist
if [ ! -f "$DEST_DIR/MEMORY.md" ]; then
  cp "$TEMPLATE_DIR/MEMORY.md" "$DEST_DIR/MEMORY.md"
  echo "✓ created $DEST_DIR/MEMORY.md"
fi
