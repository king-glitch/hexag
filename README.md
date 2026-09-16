# hexag

A modular, zero-magic hexagonal architecture framework for Go 1.26+ services backed by MongoDB and Fiber.

`hexag` provides domain-agnostic infrastructure across pure hexagonal layers. Downstream services depend inward on interfaces, while domain business logic remains 100% pure and decoupled from transport and storage adapters.

---

## Table of Contents

- [Architecture & Flow](#architecture--flow)
- [Package Ownership](#package-ownership)
- [Framework API Reference](#framework-api-reference)
  - [`framework/ports`](#frameworkports)
  - [`framework/api/model/base`](#frameworkapimodelbase)
  - [`framework/api/service/base`](#frameworkapiservicebase)
  - [`framework/api/service/errors`](#frameworkapiserviceerrors)
  - [`framework/api/shared/collection`](#frameworkapisharedcollection)
  - [`framework/api/shared/crypto`](#frameworkapisharedcrypto)
  - [`framework/api/shared/env`](#frameworkapisharedenv)
  - [`framework/api/http`](#frameworkapihttp)
  - [`framework/mongo`](#frameworkmongo)
  - [`framework/queue`](#frameworkqueue)
  - [`framework/cmd/mongogen`](#frameworkcmdmongogen)
  - [`framework/cmd/brunogen`](#frameworkcmdbrunogen)
- [CLI & Scaffolding Tooling](#cli--scaffolding-tooling)
- [End-to-End Feature Recipe](#end-to-end-feature-recipe)
- [AI Agent Coding Standards](#ai-agent-coding-standards)

---

## Architecture & Flow

```text
HTTP/Mongo adapters -> project ports <- services
                                    <- core
services -> core + project ports
```

```text
ServiceContext -> DB -> adapters -> services -> hexhttpx.New
-> bind/validate -> service -> pure rules -> repositories
-> base generation -> sentinel translation -> ServiceError
-> global HTTP error handler
```

- **Adapters**: Implement interfaces from `internal/ports`. Translate driver/network payloads into project domain types.
- **Project Ports**: Pure Go interfaces, project sentinels, and domain models. Never import adapters, services, or core.
- **Core**: Pure business invariants, domain calculations, transitions, and state validations. No I/O, no database or HTTP dependencies.
- **Services**: Orchestrate business workflows by coordinating pure core rules and domain ports. Services hold only their own repository and sibling service interfaces.

---

## Package Ownership

| Package | Owns |
|---|---|
| `framework/ports` | Pure interfaces: `ModelBase`, `BaseSetter[T]`, `ServiceContext`, `ServiceBase`, `TransactionRunnerAdapter`, Queue contracts |
| `framework/api/model/base` | `ModelBase`, `WithUserID`, `MarshalOmitBase`, `GenerateBaseModel` |
| `framework/api/service/base` | `ServiceBase`, `NewBaseService` |
| `framework/api/service/errors` | `ServiceError`, `NewServiceError`, `NewServiceErrorFromCause`, `RegisterSentinel`, `ServiceErrorCode*` |
| `framework/api/shared/collection` | `PaginationParams`, `Collection[T]`, `NewCollection`, `NewEmptyCollection` |
| `framework/api/shared/crypto` | Password and bearer token hashing (`HashPassword`, `VerifyPassword`, `HashToken`, `GenerateUserSessionToken`) |
| `framework/api/shared/env` | Generic `env:"..."` struct parsing (`LoadEnv`, `LoadConfigFromEnv`) |
| `framework/api/http` | Fiber setup (`hexhttpx.New`), CORS, request logger, validator, and transport (`Bind`, `BindQuery`, `RequestTime`, `NewSuccessResponse`) |
| `framework/mongo` | Query field builders (`Field[T]`, `NewField`), transaction runner (`Runner`, `NewRunner`) |
| `framework/queue` | MongoDB queue repository, queue service, item lifecycle, worker executor, and indexes |
| `framework/cmd/mongogen` | AST-based MongoDB type-safe query field generation |
| `framework/cmd/brunogen` | AST-based Bruno API client collection and Markdown API reference (`docs/API.md`) generation |

---

## Framework API Reference

### `framework/ports`

Provides domain-agnostic contracts implemented by adapters and services.

- `hexports.ModelBase`: Embedded struct interface (`GetID()`, `GetCreatedAt()`, `GetUpdatedAt()`, `GetDeletedAt()`).
- `hexports.ServiceContext`: Base service context providing `GetLogger() *zerolog.Logger` and `GetTransactionRunner() TransactionRunnerAdapter`.
- `hexports.ServiceBase`: Embedded service base providing access to `GetContext() ServiceContext` and `GetTransactionRunner() TransactionRunnerAdapter`.
- `hexports.TransactionRunnerAdapter`: Runner interface `Run(ctx context.Context, fn func(ctx context.Context) error) error`.

---

### `framework/api/model/base`

Base domain model structs and JSON serialization helpers.

```go
import modelbase "github.com/king-glitch/hexag/framework/api/model/base"

type ItemModel struct {
    modelbase.ModelBase  `bson:",inline" json:",inline"`
    modelbase.WithUserID `bson:",inline" json:",inline"` // Optional: for user-scoped entities

    Name string `bson:"name" json:"name"`
}

func (m ItemModel) CollectionName() string {
    return "item"
}

func (m ItemModel) WithBase(base hexports.ModelBase) ItemModel {
    m.ModelBase = m.ModelBase.WithBase(base)
    return m
}

func (m ItemModel) MarshalJSON() ([]byte, error) {
    type alias ItemModel
    return modelbase.MarshalOmitBase(m.ModelBase, alias(m))
}
```

---

### `framework/api/service/base`

Embedded base for all domain services.

```go
import (
    servicebase "github.com/king-glitch/hexag/framework/api/service/base"
    "github.com/king-glitch/hexag/framework/api/service/errors"
)

type Service struct {
    servicebase.ServiceBase
    repository ports.ItemRepository
    us         ports.UserService
}

func NewService(ctx ports.ServiceContext, repository ports.ItemRepository, us ports.UserService) ports.ItemService {
    return Service{
        ServiceBase: servicebase.NewBaseService(ctx),
        repository:  repository,
        us:         us,
    }
}
```

#### Multi-Write Transactions:
```go
func (s Service) Transfer(ctx context.Context, fromID, toID bson.ObjectID, amount int64) *serviceerrors.ServiceError {
    err := s.GetTransactionRunner().Run(ctx, func(txCtx context.Context) error {
        if _, err := s.repository.Deduct(txCtx, fromID, amount); err != nil {
            return errors.Wrap(err, "deduct failed")
        }
        if _, err := s.repository.Credit(txCtx, toID, amount); err != nil {
            return errors.Wrap(err, "credit failed")
        }
        return nil
    })
    if err != nil {
        return serviceerrors.NewServiceErrorFromCause(err, "transfer transaction failed")
    }
    return nil
}
```

---

### `framework/api/service/errors`

Structured service error and sentinel mapping.

```go
import serviceerrors "github.com/king-glitch/hexag/framework/api/service/errors"

// 1. Register project sentinels in internal/ports/errors.go
var ErrItemNotFound = errors.New("item not found")

func init() {
    serviceerrors.RegisterSentinel(ErrItemNotFound, serviceerrors.ServiceErrorCodeNotFound)
}

// 2. Wrap and return in service
func (s Service) Get(ctx context.Context, id bson.ObjectID) (ports.ItemModel, *serviceerrors.ServiceError) {
    item, err := s.repository.Get(ctx, id)
    if err != nil {
        return ports.ItemModel{}, serviceerrors.NewServiceErrorFromCause(err, "failed to get item")
    }
    return item, nil
}
```

---

### `framework/api/shared/collection`

Type-safe pagination parameters and paginated collections.

```go
import "github.com/king-glitch/hexag/framework/api/shared/collection"

// Create paginated collection
items := collection.NewCollection(itemList, totalCount, paginationParams)
```

---

### `framework/api/shared/crypto`

Secure hashing and token generation.

```go
import hexcrypto "github.com/king-glitch/hexag/framework/api/shared/crypto"

// Password hashing
hash, err := hexcrypto.HashPassword("user-secret")
err = hexcrypto.VerifyPassword(hash, "user-secret")

// Bearer token hashing (always hash tokens before DB queries/storage)
tokenHash := hexcrypto.HashToken(rawBearerToken)
```

---

### `framework/api/shared/env`

Automated environment variable loading into struct tags.

```go
import hexenv "github.com/king-glitch/hexag/framework/api/shared/env"

type Config struct {
    Port          string `env:"PORT,required"`
    MongoURI      string `env:"MONGO_URI,required"`
    MongoDatabase string `env:"MONGO_DATABASE,required"`
}

func LoadConfigFromEnv() (Config, error) {
    var cfg Config
    if err := hexenv.LoadEnv(&cfg); err != nil {
        return Config{}, err
    }
    return cfg, nil
}
```

---

### `framework/api/http`

Fiber application bootstrap and HTTP transport helpers.

```go
import (
    hexhttpx "github.com/king-glitch/hexag/framework/api/http"
    hextransport "github.com/king-glitch/hexag/framework/api/http/transport"
)

// App initialization
app := hexhttpx.New(serviceContext.Logger(), func(api fiber.Router) {
    itemHandler.Register(api.Group("/items"))
})

// Handler implementation
type CreateItemRequest struct {
    Name string `json:"name" validate:"required,min=3"`
}

type CreateItemResponse struct {
    ports.ItemModel
}

func (h ItemHandler) Create(c fiber.Ctx) error {
    at := hextransport.RequestTime(c)

    var req CreateItemRequest
    if err := hextransport.Bind(c, &req); err != nil {
        return errors.Wrap(err, "failed to bind request")
    }

    item, serr := h.service.Create(c.RequestCtx(), req.Name, at)
    if serr != nil {
        return serr
    }

    return hextransport.NewSuccessResponse(CreateItemResponse{ItemModel: item}).ToJSON(c)
}
```

---

### `framework/mongo`

Type-safe Mongo field queries and transaction runner.

```go
import (
    hexmongo "github.com/king-glitch/hexag/framework/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo"
)

// Initialize transaction runner in commands/main.go
client, _ := mongo.Connect(options.Client().ApplyURI(config.MongoURI))
txRunner := hexmongo.NewRunner(client)

// Model base generation inside Repository Create
func (r Repository) Create(ctx context.Context, m ports.ItemModel, at time.Time) (bson.ObjectID, error) {
    m = hexmongo.GenerateBaseModel(m, at)
    res, err := r.collection.InsertOne(ctx, m)
    if err != nil {
        return bson.NilObjectID, errors.Wrap(err, "failed to insert item")
    }
    return res.InsertedID.(bson.ObjectID), nil
}
```

---

### `framework/queue`

MongoDB-backed asynchronous queue processing.

```go
import (
    "github.com/king-glitch/hexag/framework/queue"
)

// 1. Ensure indexes
if err := queue.EnsureIndexes(ctx, db); err != nil {
    logger.Fatal().Err(err).Msg("failed to ensure queue indexes")
}

// 2. Initialize Queue Service
queueRepo := queue.NewRepository(db)
queueService := queue.NewService(queueRepo, txRunner)

// 3. Enqueue work
item := queue.NewItem("process-report", payload, time.Now())
if _, serr := queueService.Enqueue(ctx, item); serr != nil {
    return serr
}

// 4. Run Executor
executor := queue.NewExecutor(queueService, logger, map[queue.ActionType]queue.Transformer{
    "process-report": handleReportProcessing,
})
go executor.Start(ctx)
```

---

### `framework/cmd/mongogen`

Generates typed Mongo query field models directly from `internal/ports/domain.go`.

```go
// internal/ports/domain.go
//go:generate go run github.com/king-glitch/hexag/framework/cmd/mongogen -file=domain.go -out=../adapters/database/mongo/models -pkg=models
package ports
```

Run via:
```bash
make generate
```

---

### `framework/cmd/brunogen`

Generates Bruno API client `.bru` files and API Markdown contract reference `docs/API.md` directly by parsing Fiber route handlers and request/response DTOs.

Run via:
```bash
make bruno
```

---

## CLI & Scaffolding Tooling

The `hexag` CLI tool (`scripts/hexag` or symlinked `bin/hexag`) manages projects:

```bash
# 1. Scaffold a brand new project
hexag new github.com/my-org/my-service ~/dev/my-service my_db

# 2. Update AI agent standards (AGENTS.md, CLAUDE.md, MEMORY.md) in an existing project
hexag update ~/dev/my-service --all

# 3. Generate Bruno collections and docs/API.md
hexag bruno
```

---

## End-to-End Feature Recipe

When implementing a new entity (e.g. `Widget`), follow this sequence:

1. **Domain Model & Ports (`internal/ports/`)**:
   - Add `WidgetModel` in `internal/ports/domain.go`.
   - Add `WidgetRepository` in `internal/ports/repository.go`.
   - Add `WidgetService` in `internal/ports/service.go`.
   - Register any domain sentinels in `internal/ports/errors.go`.

2. **Code Generation & Mocks**:
   - Run `make generate` to generate Mongo fields in `internal/adapters/database/mongo/models/widget.go`.
   - Run `make mockery` to regenerate interface mocks in `internal/ports/mocks/mocks.go`.

3. **Core Business Rules (`internal/core/domain/widget/rules.go`)**:
   - Implement pure calculations, state transitions, and validation rules without I/O.

4. **Database Adapter (`internal/adapters/database/mongo/repository/widget/`)**:
   - Implement `repository.go` and `indexes.go` implementing `ports.WidgetRepository`.
   - Use `ports.WidgetModel{}.CollectionName()`.
   - Call `hexmongo.GenerateBaseModel(m, at)` on `Create`.

5. **Domain Service (`internal/services/widget/service.go`)**:
   - Embed `servicebase.ServiceBase`.
   - Hold only `repository ports.WidgetRepository` and injected sibling services (`us ports.UserService`).
   - Wrap errors with `errors.Wrap` and return `*serviceerrors.ServiceError`.

6. **HTTP Route Handler (`internal/adapters/endpoint/fiber/routes/widget.go`)**:
   - Define `<Action>Request` and `<Action>Response` DTOs above handler method.
   - Use `hextransport.Bind(c, &req)` and `hextransport.RequestTime(c)`.
   - Return `hextransport.NewSuccessResponse(res).ToJSON(c)`.

7. **Composition & Bruno Generation**:
   - Wire repository, service, and handler in `commands/main.go` and `internal/adapters/endpoint/fiber/handler.go`.
   - Run `make bruno` to generate Bruno tests and `docs/API.md`.
   - Run `go test -race ./...` to verify all tests pass.

---

## AI Agent Coding Standards

- **Agent Mode**: Keep `/ponytail ultra` active.
- **Handoff & Memory**: Check `MEMORY.md` at session start; update `MEMORY.md` after each milestone.
- **Interface-Based Dependencies**: Always use full-word getter methods (`s.deps.GetConnectionRepository().MarkStopped()`, never `s.deps.ConnectionRepo`).
- **Repository Naming**: Service primary repository fields must always be named `repository` (never `repo` or `<entity>Repo`).
- **Acronym Local Variables**: Use clean acronyms (`ur` for `UserRepository`, `us` for `UserService`, `cs` for `CreditService`).
- **Service Isolation**: Services depend only on their own repository; sibling domains are accessed strictly through their service interfaces.
- **Transactions**: Never inject `TransactionRunner` directly; use `s.GetContext().GetTransactionRunner().Run(...)`.
- **Collections**: Never hardcode collection name strings; always use `ports.<Entity>Model{}.CollectionName()`.
- **Generated Files**: Never hand-edit `models/*.go`, `mocks/*.go`, `docs/bruno/**/*.bru`, or `docs/API.md`.
