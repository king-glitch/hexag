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
- Dynamic per-function `time.Now()` check (no directory flags): if function receives `time.Time` param, any `time.Now()` is a violation; if no time param, multiple `time.Now()` calls are a violation (capture once as `at := time.Now()` and reuse).
- Struct accessor consistency (v0.0.30): if a struct has any `Get...()` method (no params, returns value), ALL its fields must be unexported. Pure data structs (no getters) are unaffected. Embedded/anonymous fields excluded. Catches `runtime.Deps` pattern with 10 exported fields + getter methods.
- Supported disambiguated service acronyms: valid unexported 2-4 lowercase letter acronyms that are subsequences of the service interface name ending in 's' (e.g. `aus` for `ports.AuditService` when `as` is taken by `ports.AuthenticationService`, `sgs`/`sugs` for `ports.SuggestionService`, `bos` for `ports.BountyService`, `ss` for `ports.BotScriptService`) are accepted.
- Constants & Sentinels location (v0.0.32): All constants must be defined in the constant folder ('internal/core/constant/') or typed enums in 'internal/ports/'. Defining local constants (e.g. const loadTimeout = ...) in adapter or service files is forbidden. Sentinel errors must be defined globally in 'internal/ports/errors.go'; defining local error variables (e.g. var errNeedsRelogin = errors.New(...)) outside ports is forbidden.
- Typed Enums vs Constants (v0.0.33): Typed enums (e.g. `type EventKind string`, `const EventFoo EventKind = "..."`) defined outside `internal/ports/` are strictly classified under `Enums` and must be moved to `internal/ports/enum.go` (implementing `IsValid() bool`), not to `constant/`. Only primitive/untyped constants belong in `internal/core/constant/`.
- Bruno URL Path Environment Variables (v0.0.34): `brunogen` generates environment variables for path parameters (e.g. `:id` -> `{{ID}}`, `:sector_id` -> `{{SECTOR_ID}}`, `:stage_id` -> `{{STAGE_ID}}`) in `params:path`, collects all path variables across routes, creates/initializes them in `environments/local.bru`, and seamlessly synchronizes missing variables into all existing environment `.bru` files without overwriting user-configured values or secrets.
- Path-Prefixed Bruno Environment Variables (v0.0.35): `brunogen` prefixes path parameter environment variables with their preceding static path segments (skipping dynamic segments, deduplicating resource names, e.g. `/templates/sectors/:sector_id` -> `TEMPLATES_SECTOR_ID`, `/templates/sectors/:sector_id/stages/:stage_id` -> `TEMPLATES_SECTORS_STAGE_ID`), and falls back to `ROOT_` if there is no preceding static path (e.g. `/:id` -> `ROOT_ID`), properly scoping each parameter to its endpoint hierarchy.
- Sibling Services & Sentinel False Positive Resolution (v0.0.37): Differentiate type expressions (`*ast.Field.Type`, `*ast.TypeSpec.Type`, etc.) from struct field accesses; scope `isSiblingServiceType` and `isRepositoryType` strictly to domain interfaces from `ports` (excluding external client SDK types such as `*sheets.Service` or `git.Repository`); disallow bare `Service` as domain service entity; exempt nil comparisons (`bin.X != nil`) and struct error fields (`.Error`) from sentinel comparisons (`body.Response.Error != nil`).
- ServiceError Wrapped Error Chain Preservation (v0.0.38): Fixed `ServiceError.Error()` returning only `e.Message` instead of `e.cause.Error()` / `e.Stack`, which caused intermediate wrapped context (e.g. `serr.Wrap("failed to reserve expedition rewards")`) to be dropped when `serr` was returned as an `error` into transaction runners or error-wrapping helpers (`errors.Wrap(err, "failed to run transaction")`). Added `Cause() error` to `ServiceError` so `github.com/pkg/errors.Cause` can recursively unwrap `ServiceError` down to the root sentinel, and updated `NewServiceError` / `ServiceErrorCodeFor` to preserve existing `StatusCode`, `Violations`, and `Code` across wraps.

## Hard-won facts

- `ServiceError.Error()` must return `e.cause.Error()` (or `e.Stack`) rather than `e.Message` so that callers wrapping `*ServiceError` as a standard Go `error` (e.g. via `errors.Wrap` in transaction runners) do not truncate intermediate wrapped context.
- `ServiceError` must implement `Cause() error` in addition to `Unwrap() error` so that `github.com/pkg/errors.Cause` can unwrap `ServiceError` instances to their root sentinel instead of terminating early.

- `framework/cmd/verify` checks Go AST nodes to mechanically enforce zero-tolerance hexagonal architecture rules (service struct field naming `repository`, sibling service lowercase acronyms across all structs including handlers, no `time.Now()` in `internal/`, zero error suppression `_ = dep.Method(...)` or `_ = err`, zero nil checks on injected dependencies, no `c.Query()`, no direct sentinel comparisons `err == ports.Err*`, single-word lowercase file names in `internal/`, capitalized full-word getter methods).
- Type expressions in struct fields, function signatures, and type definitions (e.g. `s *sheets.Service`) must never be inspected as struct field accesses.
- In hexagonal architecture, sibling domain services and repositories are strictly defined in `ports` (`ports.<Entity>Service`, `ports.<Entity>Repository`); external packages (e.g. `sheets.Service`) must not be evaluated as domain sibling services.
- Binary expressions comparing expressions to `nil` (`x != nil`, `body.Response.Error != nil`) are nil checks, never sentinel error comparisons.
- Function/method calls returning error like `taskCtx.Err()` must not be confused with error sentinels in binary expressions (`isErrSentinel` excludes calls ending in `)` or `(...)`).
- Error variable names like `aerr`, `txErr` ending in `err` or `error` must be excluded from dependency acronym heuristics.
- Structs injecting multiple services that share the same uppercase-letter acronym (e.g. `AuthenticationService` and `AuditService` both sharing `as`) must disambiguate fields using clean 2-4 letter lowercase acronyms (e.g. `aus`, `sgs`); the verifier recognizes valid subsequences of the interface type name ending in `'s'`.

