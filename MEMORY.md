# Project Memory

Hand-off state for the next agent. Rules: `AGENTS.md`.

## Environment

- Backend gate: `go test ./...` and `make verify` (or `hexag verify`).
- Verification tool: `framework/cmd/verify` mechanically checks architecture rules, naming conventions, service injection, sentinel comparisons, and collection names.

## Status

- Added `framework/cmd/verify` Go AST verification tool.
- Added `scripts/verify.sh` and `verify` subcommand to `scripts/hexag`.
- Added `verify` target to root `makefile` and `template/makefile`.
- Updated `AGENTS.md` (root and template) and `CLAUDE.md` to require `make verify` before updating `MEMORY.md`.

## Hard-won facts

- `framework/cmd/verify` checks Go AST nodes to mechanically enforce zero-tolerance hexagonal architecture rules (service struct field naming `repository`, sibling service lowercase acronyms, no `time.Now()` in services/handlers, no `c.Query()`, no direct sentinel comparisons `err == ports.Err*`, single-word lowercase file names in `internal/`).
