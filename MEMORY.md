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
- Enforced uniform unexported lowercase acronyms across all structs in `internal/` (services, handlers, runners, deps).
- Enforced zero direct struct field access to services and repositories; interface-based access via `ports` or full-word getter methods.
- Enforced enum completeness in `IsValid()` and forbid `oneof=` validation tags.
- Exempted test mock fixtures from selector rule checks and enforced kebab-case for disambiguated test files.

## Hard-won facts

- `framework/cmd/verify` checks Go AST nodes to mechanically enforce zero-tolerance hexagonal architecture rules (service struct field naming `repository`, sibling service lowercase acronyms across all structs including handlers, no `time.Now()` in services/handlers, no `c.Query()`, no direct sentinel comparisons `err == ports.Err*`, single-word lowercase file names in `internal/`, test file kebab-case naming, full-word getter methods).
