#!/usr/bin/env bash
# Scaffolds a new hexag-based project: copies template/ (including
# AGENTS.md), substitutes the module path and db name, wires a local
# replace to this hexag checkout, writes CLAUDE.md, then tidies and
# generates the first model.
#
# Usage: new.sh <module-path> <dest-dir> [db-name]
# Example: new.sh github.com/king-glitch/foo ~/dev/foo foo

set -euo pipefail

MODULE_PATH="${1:?usage: new.sh <module-path> <dest-dir> [db-name]}"
DEST_DIR="${2:?usage: new.sh <module-path> <dest-dir> [db-name]}"
DB_NAME="${3:-$(basename "$MODULE_PATH" | tr '-' '_')}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HEXAG_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_DIR="$HEXAG_ROOT/template"

# IDE/OS artifacts (.idea, .vscode, .DS_Store, ...) aren't project content —
# ignore them when deciding whether $DEST_DIR is "empty enough" to scaffold
# into, so they don't have to be manually moved aside first.
IGNORE_ENTRIES=(.idea .vscode .DS_Store .git .fleet)

if [ -e "$DEST_DIR" ]; then
  real_entries=()
  for entry in "$DEST_DIR"/* "$DEST_DIR"/.[!.]* "$DEST_DIR"/..?*; do
    [ -e "$entry" ] || continue
    name="$(basename "$entry")"
    skip=false
    for ignored in "${IGNORE_ENTRIES[@]}"; do
      [ "$name" = "$ignored" ] && skip=true && break
    done
    $skip || real_entries+=("$name")
  done

  if [ "${#real_entries[@]}" -gt 0 ]; then
    echo "error: $DEST_DIR already has project content: ${real_entries[*]}" >&2
    exit 1
  fi
fi

mkdir -p "$DEST_DIR"

# Copy every template file, including AGENTS.md — same {{MODULE_PATH}} pass
# handles it, so structure/naming/style stay identical across projects.
(cd "$TEMPLATE_DIR" && find . -type f) | while read -r rel; do
  mkdir -p "$DEST_DIR/$(dirname "$rel")"
  sed \
    -e "s|{{MODULE_PATH}}|$MODULE_PATH|g" \
    -e "s|{{HEXAG_PATH}}|$HEXAG_ROOT|g" \
    -e "s|{{DB_NAME}}|$DB_NAME|g" \
    "$TEMPLATE_DIR/$rel" > "$DEST_DIR/$rel"
done

echo "scaffolded $DEST_DIR"

printf "@AGENTS.md\n\nRun \`make verify\` before updating \`MEMORY.md\`. If it fails, fix the code immediately.\n" > "$DEST_DIR/CLAUDE.md"

echo "cp $DEST_DIR/.env.example $DEST_DIR/.env"
cp "$DEST_DIR/.env.example" "$DEST_DIR/.env"

(
  cd "$DEST_DIR"
  echo "go mod download"
  go mod download
  echo "go generate ./internal/ports/..."
  (cd internal/ports && go generate ./...)
  echo "go mod tidy"
  go mod tidy
)

cat <<MSG

done. next:
  cd $DEST_DIR
  go build ./...          # verify it compiles
  # edit internal/ports/domain.go — rename/replace ExampleModel with your first entity
  # make generate         (re-run after any domain.go change)
  # make mockery          (re-run after any internal/ports changes)
  # make bruno            (re-run after route/handler changes)
MSG
