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
- Enforced zero direct struct field access to services and repositories across package boundaries; interface-based access via `ports` or full-word getter methods (`GetStateRepository()`, `GetConnectionRepository()`).
- Excluded test files (`_test.go`) from architectural verification checks so internal test fixtures and mock setups are unaffected.
- Handled private struct field injections: differentiating between internal receiver field accesses (renaming abbreviated `*Repo` fields) vs external dependency accesses (calling getter methods).
- Mechanically enforced zero-tolerance ban on error suppression: discarding dependency method returns (`_ = dep.Method(...)`, `_, _ = dep.Method(...)`) or discarding errors (`_ = err`, `_ = serr`) is forbidden.
- Mechanically enforced zero-tolerance ban on nil checks across injected dependencies in `internal/`: `if s.us != nil`, `if t.brr != nil`, etc. are forbidden as dependencies must be unconditionally injected via constructors.
- Broadened `time.Now()` ban across all packages in `internal/`: timestamps must be passed in via `at time.Time`.
- Supported disambiguated service acronyms: valid unexported 2-4 lowercase letter acronyms that are subsequences of the service interface name ending in 's' (e.g. `aus` for `ports.AuditService` when `as` is taken by `ports.AuthenticationService`, `sgs`/`sugs` for `ports.SuggestionService`, `bos` for `ports.BountyService`, `ss` for `ports.BotScriptService`) are accepted.

## Hard-won facts

- `framework/cmd/verify` checks Go AST nodes to mechanically enforce zero-tolerance hexagonal architecture rules (service struct field naming `repository`, sibling service lowercase acronyms across all structs including handlers, no `time.Now()` in `internal/`, zero error suppression `_ = dep.Method(...)` or `_ = err`, zero nil checks on injected dependencies, no `c.Query()`, no direct sentinel comparisons `err == ports.Err*`, single-word lowercase file names in `internal/`, capitalized full-word getter methods).
- Function/method calls returning error like `taskCtx.Err()` must not be confused with error sentinels in binary expressions (`isErrSentinel` excludes calls ending in `)` or `(...)`).
- Error variable names like `aerr`, `txErr` ending in `err` or `error` must be excluded from dependency acronym heuristics.
- Structs injecting multiple services that share the same uppercase-letter acronym (e.g. `AuthenticationService` and `AuditService` both sharing `as`) must disambiguate fields using clean 2-4 letter lowercase acronyms (e.g. `aus`, `sgs`); the verifier recognizes valid subsequences of the interface type name ending in `'s'`.
