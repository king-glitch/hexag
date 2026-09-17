# AGENTS.md

## Contract

- Run `/ponytail ultra` before work; keep it active.
- Be terse and implementation-focused.
- Use installed skills/MCP before improvising.
- Go 1.27+; module `github.com/king-glitch/hexag`; strict hexagonal architecture.
- Framework: `github.com/king-glitch/hexag`.
- Root `CLAUDE.md` imports this file; never remove/break that link.
- Run `make verify` before updating `MEMORY.md`. If it fails, fix the code immediately.
- After edits, commit changes, tag a new release incrementing the patch version by 1 (e.g. v0.0.1 -> v0.0.2 or v0.0.x -> v0.0.x+1), push commit and tag, create the GitHub release, and warm the Go proxy via 'GOPROXY=https://proxy.golang.org go list -m github.com/king-glitch/hexag@<version>'.
- Check `MEMORY.md` to resume where left off; update `MEMORY.md` after completing each step in `docs/plans/` so next agent can seamlessly continue.
- If repository state or user instructions conflict with this file, stop and ask.

---

## ⛔ Zero-Tolerance Rules

AI agents frequently drift toward generic Go idioms. The following violations will cause task rejection:

| Category             | ❌ FORBIDDEN (Do NOT do this)                   | ✅ MANDATORY (Do this instead)                               |
|----------------------|-------------------------------------------------|--------------------------------------------------------------|
| **File Names**       | `user_service.go`, `create_user.go`             | `service.go`, `user.go` (single-word lowercase)              |
| **File Names**       | `user-handler.go`, `user_repo.go`               | `handler.go`, `repository.go`                                |
| **Service Struct**   | `repo ports.UserRepository` or `userRepo ...`   | `repository ports.UserRepository` (exact word: `repository`) |
| **Sibling Services** | `userService ports.UserService`                 | `us ports.UserService` (strict lowercase acronym)            |
| **Sibling Services** | `creditService ports.CreditService`             | `cs ports.CreditService`                                     |
| **Handler Services** | `connectionService ports.BotConnectionService` | `bcs ports.BotConnectionService` (strict lowercase acronym)  |
| **Service Fields**   | `AccountService ...` (exported / full name)     | `gas ...` (unexported lowercase acronym across all structs)  |
| **Repo Fields**      | `ConnectionRepository ...` (exported on struct) | Unexported field or accessed via `GetConnectionRepository()` |
| **Dependencies**     | `s.deps.GetConnectionRepo()`                    | `s.deps.GetConnectionRepository()` (never abbreviate)        |
| **Dependencies**     | `s.deps.ConnectionRepo` (struct field access)   | `s.deps.GetConnectionRepository()` (getter method)           |
| **Dependencies**     | `s.deps.ConnectionRepository` (field access)    | `s.deps.GetConnectionRepository()` (getter method)           |
| **Dependencies**     | Nil checks: `if s.us != nil`, `if t.brr != nil` | Dependencies are guaranteed non-nil via constructor injection|
| **Tx Runner**        | Storing `txRunner` as a field on service struct | Call `s.GetTransactionRunner().Run(...)` via base service    |
| **Collections**      | Raw strings: `"users"`, `"user"`                | `ports.UserModel{}.CollectionName()`                         |
| **Collections**      | Plural names: `"subscriptions"`                 | Singular names: `"subscription"`                             |
| **Errors**           | Naked returns: `return err`                     | `return errors.Wrap(err, "context message")`                 |
| **Errors**           | Discarding errors: `_ = dep.Method(...)`, `_ = err` | Handle or return every error wrapped with `errors.Wrap`      |
| **Sentinels**        | `if err == ports.ErrNotFound` or defining local error `var errX = errors.New(...)` | Sentinel comparisons via `errors.Is(err, ports.ErrX)`; all sentinels defined in `internal/ports/errors.go` |
| **Constants**        | Defining constants in adapter or service files (`const loadTimeout = ...`) | Define constants in `internal/core/constant/` or typed enums in `internal/ports/` |
| **Time Handling**    | Calling `time.Now()` in functions that receive `time.Time`; multiple `time.Now()` in same function | Capture once at entry (`at := time.Now()` or `at := hextransport.RequestTime(c)`) and pass `at` through |
| **Struct Fields**    | Exported fields on structs that have `Get...()` getter methods | Make all fields unexported; add `Get<Field>()` for each field that needs external access |
| **HTTP Queries**     | `c.Query("page")` or manual `strconv`           | `hextransport.BindQuery(c, &req)`                            |
| **Validation**       | Re-checking string length/enums in service      | Let HTTP validator tags handle transport validation          |
| **Enums**            | Enums without `IsValid() bool`                  | All domain enums must implement `IsValid() bool`             |
| **Enum Validation**  | `validate:"oneof=active ..."`                   | Use typed enum directly; framework autodetects `IsValid()`   |
| **Verification**     | Skipping `make verify` check                    | Run `make verify` (must exit code 0 before updating MEMORY)  |

---

## Architecture

```text
HTTP/Mongo adapters -> project ports <- services
                                    <- core
services -> core + project ports
```

- Adapters implement interfaces from `internal/ports`.
- All cross-boundary behavior uses interfaces.
- Services/core depend inward; never on concrete adapters.
- Services never import HTTP, Fiber, Mongo implementation, or adapter packages.
- Core is pure and performs no I/O.
- Ports never import project core/services/adapters; they may import only stdlib, contract-required Mongo primitives
  (e.g. `bson.ObjectID`), and `hexag/framework/ports`.
- Keep one repository interface and implementation package per service, even across multiple collections.

---

## File Naming & Directory Layout

- **Default rule:** Single-word lowercase filenames. The directory provides the contextual scope.
  - `internal/services/user/service.go` (NOT `user_service.go`)
  - `internal/services/user/handler.go` (NOT `user_handler.go`)
  - `internal/adapters/database/mongo/user/repository.go` (NOT `user_repository.go`)
  - `internal/core/domain/user/rules.go` (NOT `user_rules.go`)

- **Sibling disambiguation:** Use kebab-case ONLY when multiple files share a directory and cannot be named cleanly by
  role.
  - Allowed: `rule-billing.go` vs `rule-trial.go`
  - Banned: `user-service.go`, `create-user-handler.go`, `user_repo.go`

- **Route segments:** kebab-case: `/api/v1/{service}/{resource-or-action}`.
- **Mongo collections:** Singular snake_case (`user`, `subscription_tier`).

---

## Naming & Structural Invariants

### 1. Service Structs & Injections

- The service's primary repository field MUST be named `repository`:

```go
// ❌ WRONG
type Service struct {
    servicebase.ServiceBase
    repo             ports.UserRepository
    userRepository   ports.UserRepository
}

// ✅ CORRECT
type Service struct {
    servicebase.ServiceBase
    repository ports.UserRepository
    us         ports.UserService
    cs         ports.CreditService
}
```

- Injected sibling services MUST be named by lowercase acronyms of their interface type:
  - `ports.UserService` → `us`
  - `ports.CreditService` → `cs`
  - `ports.AuthenticationService` → `as`
  - `ports.BotConnectionService` → `bcs`
  - `ports.GameAccountService` → `gas`
  - `ports.AaBbCcService` (3+ words) → `abcs`

- Across all structs in `internal/` (including handlers like `ConnectionHandler`, dependency structs like `Deps`, runners, supervisors), service fields MUST be unexported lowercase acronyms of their interface type (`cs`, `gas`, `us`, `bcs`, `es`). Never export service interface fields (`AccountService`, `CreditService`), never use full service names (`connectionService`, `userService`), and never access services directly by full names (`h.connectionService`, `s.deps.CreditService`).
- If a struct or dependency is used only within the same directory, unexported fields are fine; if it needs to be exported across package boundaries, it MUST be exposed through an interface in `internal/ports` or accessed via full-word getter methods (`GetConnectionRepository()`)—zero direct struct field access across package boundaries.
- Repository fields on dependency structs or other structs must be unexported (e.g. `connectionRepository` or `bcr`). Never access repository fields directly across boundaries; always expose and call getter methods (`GetConnectionRepository()`).
- In `NewService(...)` constructors, the primary repository parameter must be named `repository`, and sibling service parameters must use lowercase acronyms (`cs`, `us`, `as`).
- Injected services are never nil. Never write nil checks (`if s.us != nil`). Inject all dependencies unconditionally
  via constructor `NewService(...)`.
- A service must ONLY hold its own repository (`repository ports.<Entity>Repository`). Never inject another service's
  repository directly.
- Never use cyclic dependencies or setter injection (`func (s Service) WithAuth(...) Service`). Extract shared workflows
  into an orchestrator service.
- Local repository/service variables must use clean acronyms (`ur` for `UserRepository`), never abbreviations like
  `userRepo`.
- Never stutter package and function names: `user.NewService()`, not `user.NewUserService()`.

### 2. Dependency Getters

- Dependency getter methods MUST use full words starting with `Get`. Shortened names or direct field accesses are strictly forbidden:
  - `s.deps.GetConnectionRepository()` (REQUIRED)
  - `s.deps.GetConnectionRepo()` (BANNED)
  - `s.deps.ConnectionRepo` (BANNED)
  - `s.deps.ConnectionRepository` (BANNED)

### 3. Signatures & Multiline Formatting

- `context.Context` is always the first parameter.
- Immutable services use value receivers.
- If a signature, call site, or struct initialization does not fit on one line, format with **strictly one
  argument/parameter per line**, including function literals:

```go
result, err := s.repository.UpdateStatus(
    ctx,
    id,
    ports.StatusActive,
    at,
)
```

- Never group or column-align multiline arguments.

---

## Ownership

Framework-owned (`github.com/king-glitch/hexag`):

| Package                           | Owns                                                                                                                                                                  |
|-----------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `framework/ports`                 | Pure interfaces: `ModelBase`, `BaseSetter[T]`, `ServiceContext` (`GetLogger()`, `GetTransactionRunner()`), `ServiceBase`, `TransactionRunnerAdapter`, queue contracts |
| `framework/api/model/base`        | `ModelBase`, `WithUserID`, `MarshalOmitBase`, `GenerateBaseModel`                                                                                                     |
| `framework/api/service/base`      | `ServiceBase`, `NewBaseService`                                                                                                                                       |
| `framework/api/service/errors`    | `ServiceError`, `NewServiceError`, `NewServiceErrorFromCause`, `RegisterSentinel`, `ServiceErrorCode*`                                                                |
| `framework/api/shared/collection` | `PaginationParams`, `Collection[T]`                                                                                                                                   |
| `framework/api/shared/crypto`     | Password/token hashing (`HashPassword`, `VerifyPassword`, `HashToken`)                                                                                                |
| `framework/api/shared/env`        | Generic `env:"..."` loading (`LoadEnv`, `LoadConfigFromEnv`)                                                                                                          |
| `framework/api/http`              | Fiber, CORS, logging, validation, transport (`Bind`, `BindQuery`, `RequestTime`, `NewSuccessResponse`), global error handling                                         |
| `framework/mongo`                 | `Field[T]`, `NewField`, `GenerateBaseModel`, Mongo transaction runner                                                                                                 |
| `framework/queue`                 | Generic Mongo queue repository, queue service, executor, indexes                                                                                                      |
| `framework/cmd/mongogen`          | Mongo field generation                                                                                                                                                |
| `framework/cmd/brunogen`          | Bruno collection and API reference (`docs/API.md`) generation                                                                                                         |
| `framework/cmd/verify`            | Architecture, naming, and style rule verification (`hexag verify`, `make verify`)                                                                                     |
| `cmd/hexag`                       | Unified CLI (`hexag verify`, `hexag bruno`, `hexag mongo`, `hexag new`, `hexag update`)                                                                               |

- Import framework symbols directly from their owning package. Never re-alias, pass-through wrap, copy, vendor, or
  locally regenerate them.
- Project owns `internal/ports/domain.go`, project interfaces/sentinels, core rules, services, repositories,
  handlers/routes, config, and domain DTO/mapping.

---

## Domain Models

All domain models live in `internal/ports/domain.go` and must:

- Be named `<Entity>Model`; no factories.
- Embed `modelbase.ModelBase` first; add `modelbase.WithUserID` when user-scoped.
- Use snake_case `json` and `bson` tags.
- Implement `CollectionName() string` returning a singular snake_case name.
- Implement `WithBase(hexports.ModelBase) <Entity>Model` via `m.ModelBase = m.ModelBase.WithBase(base)`.
- Implement `MarshalJSON()` via direct `modelbase.MarshalOmitBase(...)`.
- Represent polymorphism with small interfaces (e.g. `GetID`, `GetType`) and concrete implementations.
- Always obtain collection names via `ports.<Entity>Model{}.CollectionName()`. Never hardcode collection name strings in
  queries, migrations, or repositories.

---

## Rules & Enums

- Pure invariants, transitions, lookups, filtering, and calculations live in `internal/core/domain/<entity>/rules.go`.
- Call domain functions directly; never hide domain/framework functions behind trivial service wrappers.
- Define typed enums/constants in `internal/ports`; never use raw strings in domain or service code.
- All typed domain enums defined in `internal/ports` MUST implement `IsValid() bool` (implementing `hexports.Validatable`) covering all declared constants. The verifier mechanically checks that no declared constants are missed:

```go
type PlanType string

const (
    PlanTypeFree    PlanType = "free"
    PlanTypePremium PlanType = "premium"
)

func (p PlanType) IsValid() bool {
    switch p {
    case PlanTypeFree, PlanTypePremium:
        return true
    default:
        return false
    }
}
```

- In HTTP request DTOs, use typed domain enums directly (e.g. `Plan PlanType ` + "`" + `json:"plan"` + "`" + `).
- Never write hardcoded `validate:"oneof=..."` tags for enum values. The framework validator automatically detects any field implementing `hexports.Validatable` and validates it via `IsValid()`.

---

## HTTP Handlers

- One value-struct handler per service with `Register(fiber.Router)`.
- Handler methods MUST be exported with capitalized names: `(h <Service>Handler) <MethodName>(c fiber.Ctx) error`.
- Handlers only bind/validate, convert transport types, capture/pass `at`, call services, map response DTOs, and return
  framework responses; no business rules.
- Capture request time once via `at := hextransport.RequestTime(c)` and pass `at` through to services; never invoke
  `time.Now()`.
- Request DTOs may use `time.Time` and typed domain enums directly; never manually parse date/time strings
  (`time.Parse`) or cast raw strings in handlers.
- Bind request DTOs via `hextransport.Bind(c, &req)` or `hextransport.BindQuery(c, &req)`. Never manually parse with
  `c.Query()` or `strconv`.
- Parse pagination with `hextransport.ParsePaginationParamsContext(c, [defaultAmount])`.
- File layout order per endpoint (cluster together, never separate at file extremes):
  1. `type <MethodName>Request struct`
  2. `type <MethodName>Response struct`
  3. `func (h <Service>Handler) <MethodName>(c fiber.Ctx) error`

- Handlers returning data return `hextransport.NewSuccessResponse(<MethodName>Response{...}).ToJSON(c)`.
- Handlers returning no data return `hextransport.NewSuccessResponse().ToJSON(c)` without arguments; never define empty
  response structs.
- Never hand-roll CORS, request logging, struct validation, or error handling. Build Fiber only via `hexhttpx.New`.

---

## Persistence, Security & Transactions

- Pass `at time.Time` from HTTP through service to repository.
- Repositories assign persisted IDs/base timestamps. Repository `Create` calls `hexmongo.GenerateBaseModel` immediately
  before `InsertOne`.
- Handlers and services never generate persisted IDs or timestamps.
- Hash bearer tokens with `hexcrypto.HashToken(token)` before any DB interaction; never store or query raw tokens.
- Repository mutations return `(bson.ObjectID, error)`.
- Repository reads return models by value: `(ports.<Entity>Model, error)`.
- Repositories return plain `error`; services return `*serviceerrors.ServiceError`.
- Multi-write operations use `s.GetTransactionRunner().Run(ctx, func(ctx context.Context) error { ... })`.
- Never pass `TransactionRunner` as a constructor argument or store it on service structs.
- Always pass the transaction callback's inner `ctx` to all inner repository calls. Wrap both callback errors and the
  outer transaction runner return.

---

## Error Handling

- Every layer adds context with `errors.Wrap(err, "action context")`; never return naked errors.
- Evaluate error checks in `if` initializers when the variable is not reused:

```go
if err := s.repository.Update(ctx, id, at); err != nil {
    return serviceerrors.NewServiceErrorFromCause(errors.Wrap(err, "update failed"), "Failed to update record")
}
```

- Variable conventions: `err` for standard `error`; `serr` for `*serviceerrors.ServiceError`.
- Sentinel comparisons MUST use `errors.Is(err, ports.ErrX)`. Never use `==`, `!=`, or `switch err`.
- Define project sentinels in `internal/ports/errors.go`; register every sentinel in `init()` with
  `serviceerrors.RegisterSentinel`.
- For multiple settle/cleanup failure paths: named error return + one `defer` + one final error inspection. Never
  duplicate cleanup calls.

---

## Code Generation & Mocks

- `internal/ports/domain.go` must contain:

```go
//go:generate go run github.com/king-glitch/hexag/framework/cmd/mongogen -file=domain.go -out=../adapters/database/mongo/models -pkg=models
package ports
```

- After model changes, run `go generate ./internal/ports/...` (or `make generate`).
- Run `make bruno` to generate Bruno API collection (`docs/bruno`) and API contract reference (`docs/API.md`) after
  adding or updating routes/handlers.
- Run `make verify` (or `hexag verify`) to verify strict adherence to architecture, naming, and style rules.
- Never manually write or edit generated files (`internal/adapters/database/mongo/models/*.go`, `docs/bruno/**/*.bru`,
  `docs/API.md`).
- Project mocks live in `.mockery.yml` targeting only project interfaces in `internal/ports`. Run `make mockery`.
- Never edit mocks manually and never regenerate framework interfaces locally. Import framework mocks directly from
  `github.com/king-glitch/hexag/framework/ports/mocks`.

---

## Comments

- Default to none.
- Never restate code.
- Comment only hidden invariants, subtle compatibility workarounds, package-rule rationale, or caller obligations not
  expressible by types.

---

## Pre-Completion Checklist

Execute this checklist before reporting any task complete:

- [ ] **Verification:** Ran `make verify` (or `hexag verify`) and confirmed zero rule violations (exit code 0).
- [ ] **File Names:** Every file is single-word lowercase (`service.go`, `handler.go`, `repository.go`), with kebab-case
  reserved solely for sibling disambiguation.
- [ ] **Service Fields:** Primary repository is named `repository`. Sibling services and handler service fields are named using lowercase acronyms (`us`, `cs`, `bcs`, `gas`, `es`) and are unexported across all structs in `internal/`.
- [ ] **Dependencies:** All getter methods use full names (`s.deps.GetConnectionRepository()`, never abbreviations or direct field accesses); zero nil checks on dependencies (`if s.us != nil`, `if t.brr != nil`).
- [ ] **Collections:** All collections use `ports.<Entity>Model{}.CollectionName()` (never hardcoded strings or plural
  names).
- [ ] **Error Wrapping:** Every error propagation uses `errors.Wrap`; all comparisons use `errors.Is`; zero discarded errors (`_ = dep.Method(...)`, `_ = err`).
- [ ] **Security:** All token lookups/writes hash tokens with `hexcrypto.HashToken(token)` before DB operations.
- [ ] **Transactions:** Multi-write mutations run via `s.GetTransactionRunner().Run` using the inner callback context.
- [ ] **Generators:** Ran `make generate` and `make bruno` if domain models or HTTP routes were modified.
- [ ] **Hand-Edits:** Verified zero manual modifications to generated Mongo models or Bruno documents.
- [ ] **Enums:** Every domain enum implements `IsValid() bool` (`hexports.Validatable`) covering ALL declared constants; zero hardcoded `oneof=` validator tags.
- [ ] **Constants & Sentinels:** All constants live in 'internal/core/constant/' or 'internal/ports/'; all sentinels live in 'internal/ports/errors.go' (zero local const or var err... = errors.New(...) in adapters/services).
- [ ] **Memory:** Updated `MEMORY.md` with step progress, state, and next actions.
- [ ] **Release:** Commit completed task files, tag release with bumped patch version (e.g. v0.0.x -> v0.0.x+1), push commit/tag, create GitHub release, and warm Go proxy (`GOPROXY=https://proxy.golang.org go list -m github.com/king-glitch/hexag@<version>`) so consumers can immediately update without proxy cache delays.
