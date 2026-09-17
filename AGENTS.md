# AGENTS.md

## Contract

- Run `/ponytail ultra` before work; keep it active.
- Be terse and implementation-focused.
- Use installed skills/MCP before improvising.
- Go 1.27+; module `github.com/king-glitch/hexag`; strict hexagonal architecture.
- Framework: `github.com/king-glitch/hexag`.
- Root `CLAUDE.md` imports this file; never remove/break that link.
- After edits, commit changes and tag a new release incrementing the patch version by 1 (e.g. v0.0.1 -> v0.0.2 or v0.0.x -> v0.0.x+1) for Go package consumers; push commit and tag.
- Check `MEMORY.md` to resume where left off; update `MEMORY.md` after completing each step in `docs/plans/` so next agent can seamlessly continue.
- If repository state or user instructions conflict with this file, stop and ask.

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

## Ownership

Framework-owned:

| Package                  | Owns                                                                                                                                                         |
|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `framework/ports`        | Pure interfaces: `ModelBase`, `BaseSetter[T]`, `ServiceContext` (provides `GetLogger()`, `GetTransactionRunner()`), `ServiceBase`, `TransactionRunnerAdapter`, queue contracts |
| `framework/api/model/base` | `ModelBase`, `WithUserID`, `MarshalOmitBase`, `GenerateBaseModel`                                                                                         |
| `framework/api/service/base` | `ServiceBase`, `NewBaseService`                                                                                                                           |
| `framework/api/service/errors` | `ServiceError`, `NewServiceError`, `NewServiceErrorFromCause`, `RegisterSentinel`, `ServiceErrorCode*`                                                     |
| `framework/api/shared/collection` | `PaginationParams`, `Collection[T]`                                                                                                                  |
| `framework/api/shared/crypto` | Password/token hashing (`HashPassword`, `VerifyPassword`, `HashToken`)                                                                                      |
| `framework/api/shared/env` | Generic `env:"..."` loading (`LoadEnv`, `LoadConfigFromEnv`)                                                                                                 |
| `framework/api/http`     | Fiber, CORS, logging, validation, transport (`Bind`, `BindQuery`, `RequestTime`, `NewSuccessResponse`), global error handling                              |
| `framework/mongo`        | `Field[T]`, `NewField`, `GenerateBaseModel`, Mongo transaction runner                                                                                        |
| `framework/queue`        | Generic Mongo queue repository, queue service, executor, indexes                                                                                             |
| `framework/cmd/mongogen` | Mongo field generation                                                                                                                                       |
| `framework/cmd/brunogen` | Bruno collection and API reference (`docs/API.md`) generation                                                                                                |

- Domain-agnostic infrastructure belongs in `hexag`.
- Import framework symbols directly from their owning package. Never re-alias, pass-through wrap, copy, vendor, or locally regenerate them.
- The only runtime registration allowed is project sentinel registration.

## Naming/Layout

- Prefer single-word filenames; directory supplies context.
- Use kebab-case filenames only to distinguish unavoidable siblings.
- Avoid stuttering: `user.NewService()`, not `user.NewUserService()`.
- Use explicit ports where ambiguous: `ports.UserService`, `ports.UserRepository`.
- Route segments are kebab-case: `/api/v1/{service}/{resource-or-action}`.
- Mongo collection names are singular snake_case.
- Interface-based dependencies use full-word getter methods: `s.deps.GetConnectionRepository().MarkStopped()`, never direct struct fields like `s.deps.ConnectionRepo`.
- Never use shortened abbreviations in getter methods (e.g. `GetConnectionRepository()`, not `GetConnectionRepo()`).
- In service structs, the service's primary repository field MUST be named `repository` (e.g. `repository ports.SubscriptionRepository`), never `repo`, `subscriptionRepo`, or `<entity>Repo`.
- Injected sibling services MUST be named by lowercase acronyms of their type name: `us ports.UserService` (not `userService`), `cs ports.CreditService` (not `creditService`), `abcs ports.AaBbCcService` (3+ words).
- When referencing repository/service variables or arguments locally, prefer clean acronyms (e.g. `ur` for `UserRepository`) over clumsy abbreviations like `userRepo`.

## Models

All domain models live in `internal/ports/domain.go` and must:

- Be named `<Entity>Model`; no factories.
- Embed `modelbase.ModelBase` first; add `modelbase.WithUserID` when user-scoped.
- Use snake_case `json` and `bson` tags.
- Implement `CollectionName() string` with a singular name.
- Implement `WithBase(hexports.ModelBase) <Entity>Model` via `m.ModelBase = m.ModelBase.WithBase(base)`.
- Implement `MarshalJSON()` via direct `modelbase.MarshalOmitBase(...)`.
- Represent polymorphism with small interfaces (e.g. `GetID`, `GetType`) and concrete implementations.
- Always obtain collection names via `ports.<Entity>Model{}.CollectionName()`; never hardcode collection name strings in repositories, indexes, migrations, or queries.

## Rules/Enums

- Pure invariants, transitions, lookups, filtering, and calculations live in `internal/core/domain/<entity>/rules.go`.
- Call domain functions directly; never hide domain/framework functions behind trivial service wrappers.
- Define typed enums/constants in `internal/ports`; never use raw strings in domain/service code.
- HTTP request DTOs should use typed domain enums and `time.Time` directly when supported by binding; no manual string-to-enum casting or date parsing in handlers.

## Persistence/Security

- Pass `at time.Time` from HTTP through service to repository.
- Repositories assign persisted IDs/base timestamps.
- Repository `Create` calls `hexmongo.GenerateBaseModel` immediately before `InsertOne`.
- Handlers/services never generate persisted IDs/timestamps.
- Hash bearer tokens with `hexcrypto.HashToken(token)` before any DB interaction; never store raw tokens.
- Repository mutations return `(bson.ObjectID, error)`.
- Repository reads return models by value: `(ports.<Entity>Model, error)`.
- Repositories return plain `error`; services return `*serviceerrors.ServiceError`.

## Signatures/Formatting

- `context.Context` is always the first parameter.
- Immutable services use value receivers.
- If a signature/call does not fit one line, use one parameter/argument per line, including function literals.
- Never group or column-align multiline arguments.

## Errors

- Every layer adds context with `errors.Wrap`; never `return err` or directly return a failing repository/service call.
- Check error-only calls, `*ServiceError`-only calls, or calls with unused non-error results in an `if` initializer.
- Ordinary declarations remain valid when returned values must be reused/mutated.
- Use `err` for `error`; `serr` for concrete `*serviceerrors.ServiceError`.
- Compare sentinels only with `errors.Is(err, ports.ErrX)`; never `==`, `!=`, or `switch err`.
- Prefer guards/`continue` over avoidable nesting.
- For multiple settle-worthy failure paths: named error return + one `defer` + one final-error inspection; never
  duplicate `_ = settle(...)`.
- Define project `Err*` in `internal/ports/errors.go`; register every sentinel in `init()` with
  `serviceerrors.RegisterSentinel`.
- Framework code never owns project sentinels.
- Build service errors with `serviceerrors.NewServiceErrorFromCause(err, message)`.
- `ServiceError` owns `Message` (root cause), `Code`, `Stack` (full wrapping chain), and `Violations`.
- HTTP status comes only from `serr.Code.DefaultStatusCode()`; handlers never map it manually.

## Services

- Project `ports.ServiceContext` embeds `hexports.ServiceContext` (which provides `GetLogger()` and `GetTransactionRunner()`) and adds project `Config()`.
- Concrete services embed `servicebase.ServiceBase`, constructed with `servicebase.NewBaseService(ctx)`.
- Services depend only on ports, core rules, and direct framework abstractions.
- A service must ONLY hold its own repository (`repository ports.<Entity>Repository`).
- A service must NEVER inject another service's repository directly. Sibling domains must be accessed strictly through their service interface.
- Injected sibling services use lowercase acronym field names: `us ports.UserService`, `cs ports.CreditService`, `abcs ports.AaBbCcService` (3+ words).
- Injected service interfaces are NEVER nil. If a service needs a dependency, it must be required in the constructor and stored on the struct. Never write nil guards like `if s.us != nil`.
- Cyclic dependencies between services are strictly prohibited. Never use setter injection workarounds like `WithAuthenticationService(...)` or `WithBotConnectionService(...)`. If two services need shared logic or state verification (e.g. suspension/ban checks, status lookups), extract that capability into a separate domain service (e.g. `BanService`) or orchestrator and inject it into both.
- Services enforce domain invariants and state transitions only; never duplicate input validations already enforced by HTTP request binding tags (`required`, formats, min/max, enums).
- Bad service struct:
```go
type Service struct {
    servicebase.ServiceBase
    ctx              ports.ServiceContext
    subscriptionRepo ports.SubscriptionRepository
    userRepo         ports.UserRepository
    creditService    ports.CreditService
    txRunner         hexports.TransactionRunner
}
```
- Good service struct:
```go
type Service struct {
    servicebase.ServiceBase
    repository ports.SubscriptionRepository
    us         ports.UserService
    cs         ports.CreditService
}
```
- Never inject `TransactionRunner` as a service parameter or struct field; access it via `s.GetContext().GetTransactionRunner().Run(...)` (or `s.GetTransactionRunner().Run(...)`).

## HTTP

- One value-struct handler per service with `Register(fiber.Router)`.
- All handler methods must be exported with capitalized names: `(h <Service>Handler) <MethodName>(c fiber.Ctx) error`.
- Handlers only bind/validate, convert transport types, capture/pass `at`, call services, map response DTOs, and return
  framework responses; no business rules.
- Capture request time once via `at := hextransport.RequestTime(c)` and pass `at` through to services; never invoke `time.Now()` multiple times.
- Every request input (body or query params) uses a dedicated named request DTO: `type <MethodName>Request struct`.
- Request DTOs may use `time.Time` and typed domain enums directly; never manually parse date/time strings (`time.Parse`) or cast raw strings in handlers when binding handles them directly.
- Bind request DTOs via `hextransport.Bind(c, &req)` or `hextransport.BindQuery(c, &req)`; never manually parse query/body parameters with ad-hoc `c.Query()` or `strconv` calls.
- Parse pagination with `hextransport.ParsePaginationParamsContext(c, [defaultAmount])`.
- Every success with data uses a dedicated named response DTO: `type <MethodName>Response struct`.
- Declare request and response DTO structs immediately above their corresponding handler method (in order: `<MethodName>Request`, `<MethodName>Response`, then the method itself) to keep code cohesive and readable; never cluster DTOs at the top or bottom of the file.
- Handlers returning data return `hextransport.NewSuccessResponse(<MethodName>Response{...}).ToJSON(c)`.
- Handlers returning no data return `hextransport.NewSuccessResponse().ToJSON(c)` without arguments; never define or return empty struct envelopes like `EmptyResponse{}`.
- Use framework `hextransport.Response` envelope: `{"data":{},"errors":{}}`; never define a local envelope.
- Build Fiber only via:

```go
hexhttpx.New(
	serviceContext.Logger(),
	func(api fiber.Router) {
		// Register routes.
	},
)
```

- Never hand-roll CORS, request logging, struct validation, or global error handling.
- Bruno requests (`docs/bruno`) and API contract reference (`docs/API.md`) are generated; never write or update them manually.

## Mongo/Transactions

- Mongo adapters may only generate base models during create, query, map records, translate driver errors to project
  sentinels, and wrap repository errors; no business rules.
- Always use `ports.<Entity>Model{}.CollectionName()` to specify collections; never hardcode collection strings.
- `TransactionRunner` is managed by `ServiceContext`: runner is created in `main` via `hexmongo.NewRunner(client)` and passed to `domaincontext.New(config, logger, txRunner)`.
- Multi-write operations use `s.GetContext().GetTransactionRunner().Run(ctx, func(ctx context.Context) error { ... })` (or `s.GetTransactionRunner().Run(...)`).
- Never pass `TransactionRunner` directly through service constructors or store it on service structs.
- Always pass the transaction callback’s inner `ctx` to repository calls.
- Wrap both callback failures and the outer transaction failure.

## Pagination

- Use `collection.PaginationParams` and `collection.Collection[T]` directly across ports.
- Convert transport pagination before service calls.
- Never create local aliases/wrappers.

## Mocks

- `.mockery.yml` targets only `github.com/king-glitch/hexag/framework/ports`.
- Generate project mocks with `mockery` (or `make mockery`); use testify `.EXPECT()`.
- Never hand-edit `internal/ports/mocks/mocks.go`.
- Never regenerate framework interfaces locally.
- Import framework mocks directly from `github.com/king-glitch/hexag/framework/ports/mocks`.
- Framework mocks own `TransactionRunnerAdapter`, `QueueRepository`, and `QueueService`.

## Generation

- Run `make bruno` to generate Bruno API collection (`docs/bruno`) and API contract reference (`docs/API.md`) after adding or updating routes/handlers.
- Never manually write or edit Bruno collections (`docs/bruno/**/*.bru`) or `docs/API.md`; they must always be generated via `brunogen`.

## Memory

- `MEMORY.md` tracks hand-off state, environment, plans status, and hard-won domain/runtime facts for subsequent agents.
- Check `MEMORY.md` at session start before picking up work.
- Update `MEMORY.md` after completing each step in `docs/plans/` or finishing tasks so the next agent can seamlessly continue.

## Comments

- Default to none.
- Comment only hidden invariants, subtle compatibility workarounds, package-rule rationale, or caller obligations not
  expressible by types.
- Never restate code.

## Completion Gate

- Format; generate after model changes; run relevant tests and full suite when practical.
- Run `go mod tidy` after dependency changes.
- Verify generated Mongo files were not hand-edited.
- Verify Bruno collection and `docs/API.md` were generated via `make bruno` and not manually edited.
- Verify no framework aliases/wrappers or locally regenerated framework mocks.
- Verify every error hop wraps and every sentinel comparison uses `errors.Is`.
- Verify multi-writes use `s.GetContext().GetTransactionRunner().Run` and callback `ctx`.
- Verify services hold only their own `repository` and inject sibling services via acronyms (`us`, `cs`, `abcs`), never foreign repositories.
- Verify all injected services are unconditionally passed to constructors and never nil-checked (`if s.svc != nil`).
- Verify no cyclic dependencies between services and no setter injection workarounds (`With...`).
- Verify services do not duplicate input validation already handled by HTTP transport tags.
- Verify request DTOs bind `time.Time` and typed enums directly without manual parsing in handlers.
- Verify dependencies use full-word getter methods (`GetConnectionRepository()`), never bare fields or shortened getter names.
- Verify collection names use `ports.<Entity>Model{}.CollectionName()` and are never hardcoded strings.
- Verify handlers/repositories contain no business logic.
- Verify kebab-case routes, singular collections, and token hashing before DB access.
- Update `MEMORY.md` with step progress, decisions, and handoff notes for next agents.
- Commit completed task files and tag release with bumped patch version (e.g. v0.0.x -> v0.0.x+1); push commits and tags so Go package consumers can immediately update.
