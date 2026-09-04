# AGENTS.md

## Contract

- Run `/caveman ultra` before work; keep it active.
- Be terse and implementation-focused.
- Use installed skills/MCP before improvising.
- Go 1.26+; module `{{MODULE_PATH}}`; strict hexagonal architecture across two Go modules.
- Framework: `github.com/king-glitch/hexag`; use a temporary local `go.mod replace` until published.
- Root `CLAUDE.md` imports this file; never remove/break that link.
- Never commit. Stage completed task files; user controls commit scope/message.
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
| `framework/ports`        | `ServiceError`, `Collection[T]`, `PaginationParams`, `ModelBase`, `BaseSetter[T]`, `WithUserID`, queue contracts, `TransactionRunner`, base `ServiceContext` |
| `framework/mongo`        | `Field[T]`, `NewField`, `GenerateBaseModel`, Mongo transaction runner                                                                                        |
| `framework/mongo/queue`  | Generic Mongo queue repository                                                                                                                               |
| `framework/queue`        | Generic queue service                                                                                                                                        |
| `framework/httpx`        | Fiber, CORS, logging, validation, global error handling                                                                                                      |
| `framework/crypto`       | Password/token hashing                                                                                                                                       |
| `framework/env`          | Generic `env:"..."` loading                                                                                                                                  |
| `framework/cmd/mongogen` | Mongo field generation                                                                                                                                       |

- Domain-agnostic infrastructure belongs in `hexag`.
- Project owns `internal/ports/domain.go`, project interfaces/sentinels, core rules, services, repositories,
  handlers/routes, config, and domain DTO/mapping.
- Import framework symbols directly. Never re-alias, pass-through wrap, copy, vendor, or locally regenerate them.
- The only runtime registration allowed is project sentinel registration.
- Removing `hexag` must require only deleting imports/usages, removing `replace`, and running `go mod tidy`.

## Naming/Layout

- Prefer single-word filenames; directory supplies context.
- Use kebab-case filenames only to distinguish unavoidable siblings.
- Avoid stuttering: `user.NewService()`, not `user.NewUserService()`.
- Use explicit ports where ambiguous: `ports.UserService`, `ports.UserRepository`.
- Route segments are kebab-case: `/api/v1/{service}/{resource-or-action}`.
- Mongo collection names are singular snake_case.

## Models

All domain models live in `internal/ports/domain.go` and must:

- Be named `<Entity>Model`; no factories.
- Embed `hexports.ModelBase` first; add `hexports.WithUserID` when user-scoped.
- Use snake_case `json` and `bson` tags.
- Implement `CollectionName() string` with a singular name.
- Implement `WithBase(hexports.ModelBase) <Entity>Model`.
- Implement `MarshalJSON()` via direct `hexports.MarshalOmitBase(...)`.
- Represent polymorphism with small interfaces (e.g. `GetID`, `GetType`) and concrete implementations.

## Rules/Enums

- Pure invariants, transitions, lookups, filtering, and calculations live in `internal/core/domain/<entity>/rules.go`.
- Call domain functions directly; never hide domain/framework functions behind trivial service wrappers.
- Define typed enums/constants in `internal/ports`; never use raw strings in domain/service code.
- HTTP DTOs may use validated `string` with `oneof=...`; convert to typed enums before domain/service behavior.

## Persistence/Security

- Pass `at time.Time` from HTTP through service to repository.
- Repositories assign persisted IDs/base timestamps.
- Repository `Create` calls `hexmongo.GenerateBaseModel` immediately before `InsertOne`.
- Handlers/services never generate persisted IDs/timestamps.
- Hash bearer tokens with `hexcrypto.HashToken(token)` before any DB interaction; never store raw tokens.
- Repository mutations return `(bson.ObjectID, error)`.
- Repository reads return models by value: `(ports.<Entity>Model, error)`.
- Repositories return plain `error`; services return `*hexports.ServiceError`.

## Signatures/Formatting

- `context.Context` is always the first parameter.
- Immutable services use value receivers.
- If a signature/call does not fit one line, use one parameter/argument per line, including function literals.
- Never group or column-align multiline arguments.

## Errors

- Every layer adds context with `errors.Wrap`; never `return err` or directly return a failing repository/service call.
- Check error-only calls, `*ServiceError`-only calls, or calls with unused non-error results in an `if` initializer.
- Ordinary declarations remain valid when returned values must be reused/mutated.
- Use `err` for `error`; `serr` for concrete `*hexports.ServiceError`.
- Compare sentinels only with `errors.Is(err, ports.ErrX)`; never `==`, `!=`, or `switch err`.
- Prefer guards/`continue` over avoidable nesting.
- For multiple settle-worthy failure paths: named error return + one `defer` + one final-error inspection; never
  duplicate `_ = settle(...)`.
- Define project `Err*` in `internal/ports/errors.go`; register every sentinel in `init()` with
  `hexports.RegisterSentinel`.
- Framework code never owns project sentinels.
- Build service errors with `hexports.NewServiceErrorFromCause(err, message)`.
- `ServiceError` owns `Message` (root cause), `Code`, `Stack` (full wrapping chain), and `Violations`.
- HTTP status comes only from `serr.Code.DefaultStatusCode()`; handlers never map it manually.

## Services

- Project `ports.ServiceContext` embeds `hexports.ServiceContext` for logging and adds project `Config()`.
- Concrete services embed `hexports.ServiceBase`, constructed with `hexports.NewBaseService(ctx)`.
- Services depend only on ports, core rules, and direct framework abstractions.

## HTTP

- One value-struct handler per service with `Register(fiber.Router)`.
- All handler methods must be exported with capitalized names: `(h <Service>Handler) <MethodName>(c fiber.Ctx) error`.
- Handlers only bind/validate, convert transport types, capture/pass `at`, call services, map response DTOs, and return
  framework responses; no business rules.
- Capture request time once via `at := hextransport.RequestTime(c)` and pass `at` through to services; never invoke `time.Now()` multiple times.
- Every request input (body or query params) uses a dedicated named request DTO: `type <MethodName>Request struct`.
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

## Mongo/Transactions

- Mongo adapters may only generate base models during create, query, map records, translate driver errors to project
  sentinels, and wrap repository errors; no business rules.
- Every multi-write operation uses `hexports.TransactionRunner.Run`, with runner from `hexmongo.NewRunner(...)`.
- Always pass the transaction callback’s inner `ctx` to repository calls.
- Wrap both callback failures and the outer transaction failure.

## Pagination

- Use `hexports.PaginationParams` and `hexports.Collection[T]` directly across ports.
- Convert transport pagination before service calls.
- Never create local aliases/wrappers.

## Mocks

- `.mockery.yml` targets only `{{MODULE_PATH}}/internal/ports`.
- Generate project mocks with `mockery`; use testify `.EXPECT()`.
- Never hand-edit `internal/ports/mocks/mocks.go`.
- Never regenerate framework interfaces locally.
- Import framework mocks directly from `github.com/king-glitch/hexag/framework/ports/mocks`.
- Framework mocks own `TransactionRunner`, `QueueRepository`, and `QueueService`.

## Generation

`internal/ports/domain.go` must contain:

```go
//go:generate go run github.com/king-glitch/hexag/framework/cmd/mongogen -file=domain.go -out=../adapters/database/mongo/models -pkg=models
package main
```

- After model changes run `go generate ./internal/ports/...`.
- Never hand-edit `internal/adapters/database/mongo/models/*.go`.
- Never vendor/copy the generator.
- A new model is incomplete until all model requirements above are implemented and generation runs.

## Comments

- Default to none.
- Comment only hidden invariants, subtle compatibility workarounds, package-rule rationale, or caller obligations not
  expressible by types.
- Never restate code.

## Runtime Flow

```text
ServiceContext -> DB -> adapters -> services -> hexhttpx.New
-> bind/validate -> service -> pure rules -> repositories
-> base generation -> sentinel translation -> ServiceError
-> global HTTP error handler
```

## Completion Gate

- Format; generate after model changes; run relevant tests and full suite when practical.
- Run `go mod tidy` after dependency changes.
- Verify generated Mongo files were not hand-edited.
- Verify no framework aliases/wrappers or locally regenerated framework mocks.
- Verify every error hop wraps and every sentinel comparison uses `errors.Is`.
- Verify multi-writes use `TransactionRunner` and callback `ctx`.
- Verify handlers/repositories contain no business logic.
- Verify kebab-case routes, singular collections, and token hashing before DB access.
- Stage only completed task files; never commit.