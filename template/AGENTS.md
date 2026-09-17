# AGENTS.md

## Contract

* Run `/ponytail ultra` before work; keep it active.
* Be terse and implementation-focused.
* Use installed skills/MCP before improvising.
* Go 1.27+; module `{{MODULE_PATH}}`; strict hexagonal architecture across two Go modules.
* Framework: `[github.com/king-glitch/hexag](https://github.com/king-glitch/hexag)`; use a temporary local
  `go.mod replace` until published.
* Root `CLAUDE.md` imports this file; never remove/break that link.
* Run `make verify` before updating `MEMORY.md`. If it fails, fix the code immediately.
* Never commit. Stage completed task files; user controls commit scope/message.
* Check `MEMORY.md` to resume where left off; update `MEMORY.md` after completing each step in `docs/plans/` so next
  agent can seamlessly continue.
* If repository state or user instructions conflict with this file, stop and ask.

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
| **Dependencies**     | `s.deps.GetConnectionRepo()`                    | `s.deps.GetConnectionRepository()` (never abbreviate)        |
| **Dependencies**     | `s.deps.ConnectionRepo` (struct field access)   | `s.deps.GetConnectionRepository()` (getter method)           |
| **Tx Runner**        | Storing `txRunner` as a field on service struct | Call `s.GetTransactionRunner().Run(...)` via base service    |
| **Collections**      | Raw strings: `"users"`, `"user"`                | `ports.UserModel{}.CollectionName()`                         |
| **Collections**      | Plural names: `"subscriptions"`                 | Singular names: `"subscription"`                             |
| **Errors**           | Naked returns: `return err`                     | `return errors.Wrap(err, "context message")`                 |
| **Sentinels**        | `if err == ports.ErrNotFound`                   | `if errors.Is(err, ports.ErrNotFound)`                       |
| **Time Handling**    | Calling `time.Now()` in services/handlers       | Capture once: `at := hextransport.RequestTime(c)`            |
| **HTTP Queries**     | `c.Query("page")` or manual `strconv`           | `hextransport.BindQuery(c, &req)`                            |
| **Validation**       | Re-checking string length/enums in service      | Let HTTP validator tags handle transport validation          |
| **Verification**     | Skipping `make verify` check                    | Run `make verify` (must exit code 0 before updating MEMORY)  |

---

## Architecture

```text
HTTP/Mongo adapters -> project ports <- services
                                    <- core
services -> core + project ports

```

* Adapters implement interfaces from `internal/ports`.
* All cross-boundary behavior uses interfaces.
* Services/core depend inward; never on concrete adapters.
* Services never import HTTP, Fiber, Mongo implementation, or adapter packages.
* Core is pure and performs no I/O.
* Ports never import project core/services/adapters; they may import only stdlib, contract-required Mongo primitives
  (e.g. `bson.ObjectID`), and `hexag/framework/ports`.
* Keep one repository interface and implementation package per service, even across multiple collections.

---

## File Naming & Directory Layout

* **Default rule:** Single-word lowercase filenames. The directory provides the contextual scope.
* `internal/services/user/service.go` (NOT `user_service.go`)
* `internal/services/user/handler.go` (NOT `user_handler.go`)
* `internal/adapters/database/mongo/user/repository.go` (NOT `user_repository.go`)
* `internal/core/domain/user/rules.go` (NOT `user_rules.go`)


* **Sibling disambiguation:** Use kebab-case ONLY when multiple files share a directory and cannot be named cleanly by
  role.
* Allowed: `rule-billing.go` vs `rule-trial.go`
* Banned: `user-service.go`, `create-user-handler.go`, `user_repo.go`


* **Route segments:** kebab-case: `/api/v1/{service}/{resource-or-action}`.
* **Mongo collections:** Singular snake_case (`user`, `subscription_tier`).

---

## Naming & Structural Invariants

### 1. Service Structs & Injections

* The service's primary repository field MUST be named `repository`:

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

* Injected sibling services MUST be named by lowercase acronyms of their interface type:
* `ports.UserService` → `us`
* `ports.CreditService` → `cs`
* `ports.AuthenticationService` → `as`
* `ports.BotConnectionService` → `bcs`
* `ports.AaBbCcService` (3+ words) → `abcs`


* Injected services are never nil. Never write nil checks (`if s.us != nil`). Inject all dependencies unconditionally
  via constructor `NewService(...)`.
* A service must ONLY hold its own repository (`repository ports.<Entity>Repository`). Never inject another service's
  repository directly.
* Never use cyclic dependencies or setter injection (`func (s Service) WithAuth(...) Service`). Extract shared workflows
  into an orchestrator service.
* Local repository/service variables must use clean acronyms (`ur` for `UserRepository`), never abbreviations like
  `userRepo`.
* Never stutter package and function names: `user.NewService()`, not `user.NewUserService()`.

### 2. Dependency Getters

* Dependency getter methods MUST use full words. Shortened names or direct field accesses are strictly forbidden:
* `s.deps.GetConnectionRepository()` (REQUIRED)
* `s.deps.GetConnectionRepo()` (BANNED)
* `s.deps.ConnectionRepo` (BANNED)

### 3. Signatures & Multiline Formatting

* `context.Context` is always the first parameter.
* Immutable services use value receivers.
* If a signature, call site, or struct initialization does not fit on one line, format with **strictly one
  argument/parameter per line**, including function literals:

```go
result, err := s.repository.UpdateStatus(
ctx,
id,
ports.StatusActive,
at,
)

```

* Never group or column-align multiline arguments.

---

## Ownership

Framework-owned (`[github.com/king-glitch/hexag](https://github.com/king-glitch/hexag)`):

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

* Import framework symbols directly from their owning package. Never re-alias, pass-through wrap, copy, vendor, or
  locally regenerate them.
* Project owns `internal/ports/domain.go`, project interfaces/sentinels, core rules, services, repositories,
  handlers/routes, config, and domain DTO/mapping.

---

## Domain Models

All domain models live in `internal/ports/domain.go` and must:

* Be named `<Entity>Model`; no factories.
* Embed `modelbase.ModelBase` first; add `modelbase.WithUserID` when user-scoped.
* Use snake_case `json` and `bson` tags.
* Implement `CollectionName() string` returning a singular snake_case name.
* Implement `WithBase(hexports.ModelBase) <Entity>Model` via `m.ModelBase = m.ModelBase.WithBase(base)`.
* Implement `MarshalJSON()` via direct `modelbase.MarshalOmitBase(...)`.
* Represent polymorphism with small interfaces (e.g. `GetID`, `GetType`) and concrete implementations.
* Always obtain collection names via `ports.<Entity>Model{}.CollectionName()`. Never hardcode collection name strings in
  queries, migrations, or repositories.

---

## Rules & Enums

* Pure invariants, transitions, lookups, filtering, and calculations live in `internal/core/domain/<entity>/rules.go`.
* Call domain functions directly; never hide domain/framework functions behind trivial service wrappers.
* Define typed enums/constants in `internal/ports`; never use raw strings in domain or service code.

---

## HTTP Handlers

* One value-struct handler per service with `Register(fiber.Router)`.
* Handler methods MUST be exported with capitalized names: `(h <Service>Handler) <MethodName>(c fiber.Ctx) error`.
* Handlers only bind/validate, convert transport types, capture/pass `at`, call services, map response DTOs, and return
  framework responses; no business rules.
* Capture request time once via `at := hextransport.RequestTime(c)` and pass `at` through to services; never invoke
  `time.Now()`.
* Request DTOs may use `time.Time` and typed domain enums directly; never manually parse date/time strings
  (`time.Parse`) or cast raw strings in handlers.
* Bind request DTOs via `hextransport.Bind(c, &req)` or `hextransport.BindQuery(c, &req)`. Never manually parse with
  `c.Query()` or `strconv`.
* Parse pagination with `hextransport.ParsePaginationParamsContext(c, [defaultAmount])`.
* File layout order per endpoint (cluster together, never separate at file extremes):

1. `type <MethodName>Request struct`
2. `type <MethodName>Response struct`
3. `func (h <Service>Handler) <MethodName>(c fiber.Ctx) error`


* Handlers returning data return `hextransport.NewSuccessResponse(<MethodName>Response{...}).ToJSON(c)`.
* Handlers returning no data return `hextransport.NewSuccessResponse().ToJSON(c)` without arguments; never define empty
  response structs.
* Never hand-roll CORS, request logging, struct validation, or error handling. Build Fiber only via `hexhttpx.New`.

---

## Persistence, Security & Transactions

* Pass `at time.Time` from HTTP through service to repository.
* Repositories assign persisted IDs/base timestamps. Repository `Create` calls `hexmongo.GenerateBaseModel` immediately
  before `InsertOne`.
* Handlers and services never generate persisted IDs or timestamps.
* Hash bearer tokens with `hexcrypto.HashToken(token)` before any DB interaction; never store or query raw tokens.
* Repository mutations return `(bson.ObjectID, error)`.
* Repository reads return models by value: `(ports.<Entity>Model, error)`.
* Repositories return plain `error`; services return `*serviceerrors.ServiceError`.
* Multi-write operations use `s.GetTransactionRunner().Run(ctx, func(ctx context.Context) error { ... })`.
* Never pass `TransactionRunner` as a constructor argument or store it on service structs.
* Always pass the transaction callback's inner `ctx` to all inner repository calls. Wrap both callback errors and the
  outer transaction runner return.

---

## Error Handling

* Every layer adds context with `errors.Wrap(err, "action context")`; never return naked errors.
* Evaluate error checks in `if` initializers when the variable is not reused:

```go
if err := s.repository.Update(ctx, id, at); err != nil {
return serviceerrors.NewServiceErrorFromCause(errors.Wrap(err, "update failed"), "Failed to update record")
}

```

* Variable conventions: `err` for standard `error`; `serr` for `*serviceerrors.ServiceError`.
* Sentinel comparisons MUST use `errors.Is(err, ports.ErrX)`. Never use `==`, `!=`, or `switch err`.
* Define project sentinels in `internal/ports/errors.go`; register every sentinel in `init()` with
  `serviceerrors.RegisterSentinel`.
* For multiple settle/cleanup failure paths: named error return + one `defer` + one final error inspection. Never
  duplicate cleanup calls.

---

## Code Generation & Mocks

* `internal/ports/domain.go` must contain:

```go
//go:generate go run github.com/king-glitch/hexag/framework/cmd/mongogen -file=domain.go -out=../adapters/database/mongo/models -pkg=models
package main

```

* After model changes, run `go generate ./internal/ports/...` (or `make generate`).
* Run `make bruno` to generate Bruno API collection (`docs/bruno`) and API contract reference (`docs/API.md`) after
  adding or updating routes/handlers.
* Never manually write or edit generated files (`internal/adapters/database/mongo/models/*.go`, `docs/bruno/**/*.bru`,
  `docs/API.md`).
* Project mocks live in `.mockery.yml` targeting only `{{MODULE_PATH}}/internal/ports`. Run `make mockery`.
* Never edit mocks manually and never regenerate framework interfaces locally. Import framework mocks directly from
  `[github.com/king-glitch/hexag/framework/ports/mocks](https://github.com/king-glitch/hexag/framework/ports/mocks)`.

---

## Comments

* Default to none.
* Never restate code.
* Comment only hidden invariants, subtle compatibility workarounds, package-rule rationale, or caller obligations not
  expressible by types.

---

## Pre-Completion Checklist

Execute this checklist before reporting any task complete:

* [ ] **File Names:** Every file is single-word lowercase (`service.go`, `handler.go`, `repository.go`), with kebab-case
  reserved solely for sibling disambiguation.
* [ ] **Service Fields:** Primary repository is named `repository`. Sibling services are named using lowercase acronyms
  (`us`, `cs`, `bcs`).
* [ ] **Dependencies:** All getter methods use full names (`s.deps.GetConnectionRepository()`, never abbreviations).
* [ ] **Collections:** All collections use `ports.<Entity>Model{}.CollectionName()` (never hardcoded strings or plural
  names).
* [ ] **Error Wrapping:** Every error propagation uses `errors.Wrap`; all comparisons use `errors.Is`.
* [ ] **Security:** All token lookups/writes hash tokens with `hexcrypto.HashToken(token)` before DB operations.
* [ ] **Transactions:** Multi-write mutations run via `s.GetTransactionRunner().Run` using the inner callback context.
* [ ] **Generators:** Ran `make generate` and `make bruno` if domain models or HTTP routes were modified.
* [ ] **Hand-Edits:** Verified zero manual modifications to generated Mongo models or Bruno documents.
* [ ] **Verification:** Ran `make verify` (or `hexag verify`) and confirmed zero rule violations (exit code 0).
* [ ] **Memory:** Updated `MEMORY.md` with step progress, state, and next actions.
* [ ] **Git:** Staged all task files; did not commit.