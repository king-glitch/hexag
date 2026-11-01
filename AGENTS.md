# AGENTS.md

## Contract

- Run `/caveman ultra` and `/ponytail ultra` before work; keep both active every response until the user says otherwise.
- Be terse and implementation-focused.
- Use installed skills/MCP before improvising.
- Go 1.27+; module `github.com/king-glitch/hexag`; strict hexagonal architecture.
- Framework: `github.com/king-glitch/hexag`. In this repo, reference files below live under `template/`.
- Root `CLAUDE.md` imports this file; never remove/break that link.
- Run `make verify` before updating `MEMORY.md`. If it fails, fix the code immediately.
- After edits, commit changes, tag a new release incrementing the patch version by 1 (e.g. v0.0.1 -> v0.0.2 or v0.0.x -> v0.0.x+1), push commit and tag, create the GitHub release, and warm the Go proxy via 'GOPROXY=https://proxy.golang.org go list -m github.com/king-glitch/hexag@<version>'.
- Check `MEMORY.md` to resume where left off; update `MEMORY.md` after completing each step in `docs/plans/` so next agent can seamlessly continue.
- `template/AGENTS.md` ships to generated projects; keep its body in sync with this file.
- If repository state or user instructions conflict with this file, stop and ask.

---

## Operating Loop

Long sessions drift. Rules read at the start fade, and agents start to copy their own recent code or generic Go habits.
This loop prevents that. Follow it on every step, not once per session.

1. **Resume.** Read `MEMORY.md`, then the open plan in `docs/plans/`. After a context compaction, a resume, or every
   ~5 edited files, re-read this section and the Zero-Tolerance table before the next edit.
2. **Mirror a reference.** Before you create or rewrite a file, open its reference file (table below) and copy its
   shape: imports, aliases, struct fields, constructor, error handling, response style. Never write from memory or
   generic Go idioms.
3. **Edit small.** Change one layer or one feature slice per step.
4. **Verify every step.** Run `make verify` after each step, not only at the end. Fix every violation before the next
   edit. Never defer fixes. Claude Code (`.claude/settings.json`) and Antigravity (`.agents/hooks.json`) hooks call
   `hexag hook`, which runs it automatically; treat a hook failure as blocking. Install the CLI so hooks find it:
   `go install github.com/king-glitch/hexag/cmd/hexag@latest`.
5. **Check blind spots.** `make verify` cannot see some rules. Check them by hand before you mark a step done.
6. **Record.** Update `MEMORY.md` (step, state, next action) only after `make verify` exits 0.
7. **Trust order.** This file > `make verify` output > verified existing code > your memory of earlier turns. If code you
   wrote earlier in this session conflicts with this file, fix that code. Do not copy it.

### Reference Files

If an example file was renamed or replaced, mirror the oldest verified sibling in the same directory.

| Writing                       | Mirror                                                              |
|-------------------------------|---------------------------------------------------------------------|
| Domain model                  | `internal/ports/domain.go`                                          |
| Enum / sentinel               | `internal/ports/enum.go`, `internal/ports/errors.go`                |
| Port interface                | `internal/ports/service.go`, `internal/ports/repository.go`         |
| Service                       | `internal/services/example/service.go`                              |
| HTTP handler                  | `internal/adapters/endpoint/fiber/routes/example.go`                |
| Mongo repository / indexes    | `internal/adapters/database/mongo/repository/example/`              |
| Wiring                        | `commands/main.go`, `internal/adapters/endpoint/fiber/handler.go`   |

### Verifier Blind Spots (check by hand)

- Multiline signatures, calls, and struct literals: one argument per line.
- Handler file order per endpoint: Request, Response, method. Handlers hold no business rules.
- Transactions: inner callback `ctx` on every inner call; wrap callback errors and the outer `Run` return.
- Bearer tokens: `hexcrypto.HashToken(token)` before any DB read/write.
- Domain models: `CollectionName()`, `WithBase(...)`, `MarshalJSON()` via `modelbase.MarshalOmitBase(...)`.
- Every `errors.Wrap` message names the failed action.
- Generated files untouched; `make generate` / `make mockery` / `make bruno` ran when needed.
- No comments that restate code.

---

## ⛔ Zero-Tolerance Rules

Agents drift toward generic Go idioms. Each violation causes task rejection.

| Category             | ❌ FORBIDDEN                                                                 | ✅ MANDATORY                                                                  |
|----------------------|-----------------------------------------------------------------------------|------------------------------------------------------------------------------|
| **File Names**       | `user_service.go`, `create_user.go`, `user-handler.go`, `user_repo.go`      | `service.go`, `handler.go`, `repository.go`, `user.go` (single-word)         |
| **Primary Repo**     | `repo ports.UserRepository`, `userRepository ...`                           | `repository ports.UserRepository` (exact word)                               |
| **Service Fields**   | `userService`, `connectionService`, `AccountService` (any struct)           | `us`, `bcs`, `gas` (unexported lowercase acronym, every struct in `internal/`) |
| **Repo Fields**      | `ConnectionRepository ...` exported on a struct                             | Unexported field; external access via `GetConnectionRepository()`           |
| **Dependencies**     | `s.deps.GetConnectionRepo()`, `s.deps.ConnectionRepo`, `s.deps.ConnectionRepository` | `s.deps.GetConnectionRepository()` (full-word getter)              |
| **Nil Checks**       | `if s.us != nil`, `if t.brr != nil`                                         | None. Constructor injection guarantees non-nil                               |
| **Tx Runner**        | `txRunner` field or constructor argument                                    | `s.GetTransactionRunner().Run(...)` via base service                         |
| **Collections**      | `"users"`, `"user"`, `"subscriptions"`                                      | `ports.UserModel{}.CollectionName()` returning singular snake_case           |
| **Errors**           | `return err`, `_ = dep.Method(...)`, `_ = err`                              | `return errors.Wrap(err, "action context")`; handle every error              |
| **Violations**       | `serr.AddError(...)` as a statement, then `return serr`                     | `return serr.AddError(...).AddError(...)`; `serr = serr.AddError(...)` in loops |
| **Sentinels**        | `err == ports.ErrNotFound`, local `var errX = errors.New(...)`              | `errors.Is(err, ports.ErrX)`; sentinels only in `internal/ports/errors.go`   |
| **Constants**        | `const loadTimeout = ...` in adapter/service files                          | `internal/core/constant/` (untyped) or typed enums in `internal/ports/`      |
| **Time**             | `time.Now()` in a function that receives `time.Time`; 2+ `time.Now()` calls | Capture once (`at := hextransport.RequestTime(c)` or `at := time.Now()`), pass `at` |
| **Struct Fields**    | Exported fields on a struct that has `Get...()` methods                     | All fields unexported; `Get<Field>()` for external access                    |
| **HTTP Input**       | `c.Query("page")`, `strconv`, `time.Parse`                                  | `hextransport.Bind` / `hextransport.BindQuery` into typed DTOs               |
| **Validation**       | Re-checking length/enum in service; `validate:"oneof=..."`                  | Validator tags + typed enums (framework calls `IsValid()`)                   |
| **Enums**            | Typed enum without `IsValid() bool`, or missing a constant                  | `IsValid()` covers every declared constant                                   |
| **Verification**     | Skipping `make verify`, or running it only at the end                       | `make verify` exits 0 after every step                                       |

---

## Architecture

```text
HTTP/Mongo adapters -> project ports <- services
                                    <- core
services -> core + project ports
```

- Adapters implement interfaces from `internal/ports`. All cross-boundary behavior uses interfaces.
- Services/core depend inward. Services never import HTTP, Fiber, Mongo implementation, or adapter packages.
- Core is pure and performs no I/O.
- Ports import only stdlib, contract-required Mongo primitives (e.g. `bson.ObjectID`), and `hexag/framework/ports`.
- One repository interface and implementation package per service, even across multiple collections.

---

## Naming & Layout

### Files & Routes

- Single-word lowercase filenames; the directory gives scope (`internal/services/user/service.go`,
  `internal/adapters/database/mongo/user/repository.go`, `internal/core/domain/user/rules.go`).
- Kebab-case only to disambiguate siblings that a role name cannot separate: `rule-billing.go` vs `rule-trial.go`.
  Banned: `user-service.go`, `create-user-handler.go`.
- Routes: kebab-case `/api/v1/{service}/{resource-or-action}`. Mongo collections: singular snake_case.

### Services & Injection

```go
type Service struct {
    servicebase.ServiceBase
    repository ports.UserRepository
    us         ports.UserService
    cs         ports.CreditService
}
```

- Service fields use the lowercase acronym of the interface: `UserService` → `us`, `CreditService` → `cs`,
  `AuthenticationService` → `as`, `BotConnectionService` → `bcs`, `GameAccountService` → `gas`.
- On acronym collision use a 2-4 letter subsequence ending in `s` (`AuditService` → `aus` when `as` is taken).
- `NewService(...)` parameters use the same names: `repository`, `us`, `cs`.
- A service holds only its own repository. Never inject another service's repository.
- No cyclic dependencies, no setter injection (`WithAuth(...)`). Extract shared workflows into an orchestrator service.
- Local variables use clean acronyms (`ur` for `UserRepository`), never `userRepo`.
- Never stutter: `user.NewService()`, not `user.NewUserService()`.
- Structs used only inside their directory may have unexported fields freely. Cross-package access goes through a
  `ports` interface or full-word `Get...()` getters. Never access another package's struct fields directly.

### Signatures & Formatting

- `context.Context` is always the first parameter. Immutable services use value receivers.
- If a signature, call, or struct literal does not fit on one line, put exactly one argument per line, including
  function literals. Never group or column-align.

```go
result, err := s.repository.UpdateStatus(
    ctx,
    id,
    ports.StatusActive,
    at,
)
```

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

- Import framework symbols from their owning package. Never re-alias, wrap, copy, vendor, or regenerate them.
- Project owns `internal/ports/domain.go`, project interfaces/sentinels, core rules, services, repositories,
  handlers/routes, config, and domain DTO/mapping.

---

## Domain Models

All domain models live in `internal/ports/domain.go`:

- Name `<Entity>Model`; no factories.
- Embed `modelbase.ModelBase` first; add `modelbase.WithUserID` when user-scoped.
- snake_case `json` and `bson` tags.
- `CollectionName() string` returns a singular snake_case name.
- `WithBase(hexports.ModelBase) <Entity>Model` sets `m.ModelBase = m.ModelBase.WithBase(base)`.
- `MarshalJSON()` calls `modelbase.MarshalOmitBase(...)` directly.
- Polymorphism: small interfaces (`GetID`, `GetType`) with concrete implementations.

---

## Rules & Enums

- Pure invariants, transitions, lookups, filtering, and calculations live in `internal/core/domain/<entity>/rules.go`.
- Call domain/framework functions directly; never wrap them in trivial service methods.
- Typed enums live in `internal/ports/enum.go`; never use raw strings in domain or service code.
- Every typed enum implements `IsValid() bool` (`hexports.Validatable`) covering all declared constants:

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

- Request DTOs use typed enums directly (`Plan ports.PlanType`). The validator detects `IsValid()`; never add `oneof=`.

---

## HTTP Handlers

- One value-struct handler per service with `Register(fiber.Router)`.
- Exported methods: `func (h <Service>Handler) <MethodName>(c fiber.Ctx) error`.
- Handlers only bind, convert transport types, capture `at`, call services, map response DTOs, and return framework
  responses. No business rules.
- `at := hextransport.RequestTime(c)` once per request; never `time.Now()`.
- Bind via `hextransport.Bind(c, &req)` or `hextransport.BindQuery(c, &req)`. DTOs may use `time.Time` and typed enums.
- Pagination: `hextransport.ParsePaginationParamsContext(c, [defaultAmount])`.
- Keep each endpoint together: `<MethodName>Request`, `<MethodName>Response`, then the method.
- With data: `hextransport.NewSuccessResponse(<MethodName>Response{...}).ToJSON(c)`.
- Without data: `hextransport.NewSuccessResponse().ToJSON(c)`; never define empty response structs.
- Never hand-roll CORS, request logging, validation, or error handling. Build Fiber only via `hexhttpx.New`.

---

## Persistence, Security & Transactions

- Pass `at time.Time` from HTTP through service to repository.
- Repositories assign IDs/base timestamps: `Create` calls `GenerateBaseModel` immediately before `InsertOne`.
  Handlers and services never generate IDs or timestamps.
- Mutations return `(bson.ObjectID, error)`; reads return models by value `(ports.<Entity>Model, error)`.
- Repositories return plain `error`; services return `*serviceerrors.ServiceError`.
- Hash bearer tokens with `hexcrypto.HashToken(token)` before any DB interaction.
- Multi-write operations use `s.GetTransactionRunner().Run(ctx, func(ctx context.Context) error { ... })`, pass the
  inner `ctx` to every inner call, and wrap both callback errors and the outer return.

---

## Error Handling

- Use `if` initializers when the error variable is not reused:

```go
if err := s.repository.Update(ctx, id, at); err != nil {
    return serviceerrors.NewServiceErrorFromCause(errors.Wrap(err, "update failed"), "Failed to update record")
}
```

- Names: `err` for `error`, `serr` for `*serviceerrors.ServiceError`.
- Build field violations as one chain. `AddError` returns the receiver; never discard its result:

```go
return serviceerrors.NewServiceError(
    serviceerrors.ServiceErrorCodeValidation,
    errors.New("invalid profile"),
).AddError("email", "must be a valid email", nil).AddError("age", "must be at least 18", nil)
```

- Never compare errors with `==`, `!=`, or `switch err`.
- Register every sentinel in `internal/ports/errors.go` `init()` with `serviceerrors.RegisterSentinel`.
- Multiple settle/cleanup failure paths: named error return, one `defer`, one final inspection. Never duplicate cleanup.

---

## Code Generation & Mocks

- `internal/ports/domain.go` starts with:

```go
//go:generate go run github.com/king-glitch/hexag/framework/cmd/mongogen -file=domain.go -out=../adapters/database/mongo/models -pkg=models
package ports
```

- Model change: `make generate`. Port change: `make mockery`. Route/handler change: `make bruno`.
- Never edit generated files: `internal/adapters/database/mongo/models/*.go`, mocks, `docs/bruno/**/*.bru`, `docs/API.md`.
- Mocks target only project interfaces in `internal/ports` (`.mockery.yml`). Import framework mocks from
  `github.com/king-glitch/hexag/framework/ports/mocks`.

---

## Comments

- Default to none. Never restate code.
- Comment only hidden invariants, compatibility workarounds, package-rule rationale, or caller obligations that types
  cannot express.

---

## Pre-Completion Checklist

- [ ] `make verify` exits 0 (ran after every step, not only now).
- [ ] Every new file mirrors its reference file.
- [ ] Verifier blind spots checked by hand.
- [ ] Generators ran for changed models, ports, or routes; generated files untouched.
- [ ] `MEMORY.md` updated with step, state, and next action.
- [ ] Release: commit, bump patch tag, push commit/tag, create GitHub release, warm Go proxy.
