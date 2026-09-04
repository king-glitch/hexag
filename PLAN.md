# hexag — implementation plan

A shared Go module for the hexagonal-architecture Mongo backends currently
duplicated across `personal/px/backend`, `freelance/yanui-paradise/yanui-paradise.backend`
and `x10/idler/backend`.

## 0. Why this is a module, not a framework

There is nothing to invent. The duplication is ordinary Go code with no
project knowledge in it, so the delivery mechanism is the one Go already has:
an importable module. "Injectable like a plugin" is an import statement.
"Out easily" is deleting the import and running `go mod tidy`. No CLI, no
codegen-into-your-repo, no vendoring step, no plugin registry.

Only two things ship:

1. **`github.com/king-glitch/hexag/framework/...`** — the shared code, imported.
2. **A template directory** — the per-project files that cannot be imported
   (module wiring, `main.go`, env files), copied once at project creation.

## 1. Evidence

Normalised diff of the same file across the three projects (module path
rewritten to a placeholder before diffing):

| file | lines | px vs yanui | px vs idler |
|---|---|---|---|
| `internal/ports/collection.go` | 67 | identical | identical |
| `internal/adapters/endpoint/fiber/validator/base.go` | 93 | identical | identical |
| `commands/mongo-generator/internal/generator/templates.go` | 169 | identical | identical |
| `commands/mongo-generator/main.go` | 129 | identical | 6 |
| `internal/core/domain/context/env.go` | 18 | identical | 5 |
| `internal/adapters/endpoint/fiber/transport/response.go` | 25 | 5 | 4 |
| `internal/core/domain/context/context.go` | 27 | 5 | 4 |
| `commands/mongo-generator/internal/generator/generator.go` | 145 | 6 | 8 |
| `internal/adapters/endpoint/fiber/handler.go` | 82 | 16 | 44 |
| `commands/mongo-generator/internal/parser/parser.go` | 360 | 46 | 4 |
| `internal/ports/errors.go` | 175 | 84 | 184 |
| `internal/adapters/endpoint/fiber/transport/params.go` | 194 | 196 | 80 |
| `internal/services/queue/service.go` | 379 | 209 | 117 |
| `internal/adapters/database/mongo/repository/queue/repository.go` | 373 | 340 | 116 |

Roughly 1,900 lines of infrastructure per project, three copies, already
drifting. `px/internal/core/constant/constants.go` still carries idler's
`SteamAddItemPath`, `CharacterFirstLevel` and `StorageStashMaxBoxes` — dead
constants that arrived by copy-paste and were never removed. That is the cost
this module removes.

## 2. Repository and module layout

Git root is `personal/hexag/`, published as the public repo
`github.com/king-glitch/hexag`. One `go.mod` at the root, so version tags stay
plain (`v0.1.0`) instead of the `framework/v0.1.0` sub-path tagging scheme a
nested module would force.

```
personal/hexag/
  go.mod                      module github.com/king-glitch/hexag
  PLAN.md
  AGENTS.md                   the shared coding standards
  framework/
    ports/                    interfaces and shared types, depends on nothing
    mongo/                    Field[T], transaction runner, queue repository
    queue/                    queue service
    httpx/                    fiber error handler, transport, validator, middleware
    crypto/                   token and password hashing
    cmd/mongogen/             the model-field generator
  template/                   files copied into a new project
  scripts/new.sh              copy template, rewrite module path
```

Import paths read `github.com/king-glitch/hexag/framework/ports`.

## 3. What moves into the framework

Only code with zero knowledge of any specific project.

**`framework/ports`** — `Collection[T]`, `NewCollection`, `NewEmptyCollection`,
`PaginationParams`; `ServiceError`, `ServiceErrorCode`, `Violation`,
`DefaultStatusCode`, `NewServiceErrorFromCause`; `Model`, `ModelBase`,
`BaseSetter[T]`, `WithUserID`, and an exported `MarshalOmitBase` (today it is
the unexported `marshalOmitBase`, and every project model calls it); the queue
contract — `QueueModel`, `QueueItem`, `Item`, `NewItem`, `DequeueOption`,
`QueueItemStatus`, `Priority`, `ActionType`, `Transformer`, `QueueExecutor`,
`QueueRepository`, `QueueService`; `TransactionRunner`; and `ServiceContext` /
`ServiceBase` / `NewBaseService`.

`ServiceContext` narrows to `Logger() *zerolog.Logger`. It does **not** carry
`Config()`. Every project's `Config` is different (px has
`APP_MAIL_INBOUND_DOMAIN`, idler has Steam keys), and the only framework code
that touches the context is the queue service, which uses the logger and
nothing else. Each project declares its own:

```go
type ServiceContext interface {
    hexports.ServiceContext
    Config() Config
}
```

Sentinel-to-code mapping stays extensible: the framework owns
`ServiceErrorCodeFor` plus a `RegisterSentinel(err, code)` call, and each
project registers its own `ports.Err*` values in an `init`. Without that, the
framework would have to know about `ErrBankSlipUnreadable`, which is exactly
the coupling being removed.

**`framework/mongo`** — the generic `Field[T]` with `Eq`/`Ne`/`Gt`/`In`/`Asc`/…,
which is currently *generated* into `models/field.go` in all three projects even
though it never varies; the `TransactionRunner` implementation; and the queue
repository, whose 373 lines are the same three-way-drifted file everywhere.

**`framework/queue`** — the queue service. Its constants (`QueueDequeueLimit`,
`QueueMaxAttempts`, `QueueRetryBaseBackoff`, `QueueExecutorInterval`) move with
it, as functional options with the current values as defaults, so a project can
override without editing a shared constants file.

**`framework/httpx`** — the fiber error handler, `transport.Response`,
`NewSuccessResponse`, generic `transport.Bind` and `transport.BindQuery` for
struct-driven request and query parsing, `transport.ParsePaginationParamsContext`,
the server time middleware and `transport.RequestTime` helper, the `transport.Param*`
/ `ParseDateRange` / pagination helpers, the struct validator with its `enum` rule
and message table, the zerolog request-logging middleware, and the bearer-token
authentication middleware with `AuthenticatedUserID`.

`httpx` exposes a `New(cfg)` returning `*fiber.App` with CORS, logger, error
handler and validator already wired, plus a `Mount(prefix, ...)` for route
groups. Route handlers themselves stay in each project — they are business
surface, not infrastructure.

**`framework/crypto`** — `GenerateUserSessionToken`, `HashToken`,
`HashPassword`, `ComparePassword`, and `GenerateBaseModel[T]`.

**`framework/cmd/mongogen`** — the mongo-generator, unchanged in behaviour but
with one simplification: it stops emitting `field.go`, because `Field[T]` now
lives in `framework/mongo`. Generated model files import it. Projects invoke it
without copying a single file:

```go
//go:generate go run github.com/king-glitch/hexag/framework/cmd/mongogen -file=domain.go -out=../adapters/database/mongo/models -pkg=models
```

That alone deletes ~800 lines of vendored generator from every project.

**`framework/cmd/brunogen`** — turns a project's Fiber routes into a Bruno API
client collection (`.bru` files mirroring the URL tree) plus an optional
`API.md` contract reference, by `go/ast`-parsing `routes/*.go` and the
composition root's `handler.Register(api.Group("/x"))` wiring — same
approach as `mongogen`, no compilation or reflection. Regeneration wipes and
rebuilds every route folder so removed/renamed routes don't leave stale
files behind, but never overwrites an existing `bruno.json` or
`environments/local.bru` (a developer's real token lives there):

```go
//go:generate go run github.com/king-glitch/hexag/framework/cmd/brunogen -routes=routes -composition=handler.go -ports=../../../ports -out=../../../../docs/bruno -api-md=../../../../../frontend/API.md
```

## 4. What stays per-project

Everything with business meaning: `commands/main.go` wiring,
`internal/ports/domain.go` / `service.go` / `repository.go` / `enum.go` and the
project's own `Err*` sentinels, `internal/core/domain/**` rules,
`internal/services/**`, the Mongo repositories and their `EnsureIndexes`,
`internal/adapters/endpoint/fiber/routes/**`, and the ops files — `.env`,
`Dockerfile`, `makefile`, `.mockery.yml`, `docs/bruno`.

User and authentication are a deliberate judgement call: they are identical in
all three projects, but they are the seam where projects diverge first (px
already added a mail-inbox token flow off the back of user identity). They stay
per-project for v0.1. If a fourth project copies them unchanged, they move into
`framework/auth` then — not before.

## 5. Scaffolding a new project

`scripts/new.sh <module-path> <dest-dir>` copies `template/`, rewrites the
module path, and runs `go mod tidy`. About fifteen lines of shell. The
alternative, `gonew` from `golang.org/x/tools`, does the same import rewriting
but resolves templates through the module proxy, which would require publishing
`template/` as a nested module with `template/v0.1.0`-style tags. The script
works offline and needs no tagging ceremony; if the template later becomes
worth versioning independently, switching to `gonew` is a drop-in change.

`template/` contains the skeleton only: `go.mod` with the hexag dependency,
`commands/main.go` wiring one health route, `internal/ports/` with an empty
`domain.go` carrying the `go:generate` line and a `Config` with the port and
Mongo variables, an empty services and routes tree, `.env.example`, `makefile`,
`.mockery.yml`, `Dockerfile`, and a `CLAUDE.md` that imports the framework's
`AGENTS.md`.

## 6. Phases

**Phase 1 — extract.** Create the module, move the shared code in, take `px` as
the base version wherever the three copies differ (it is the newest), reconcile
the drift by hand. Every package must compile with `go build ./...` and the
existing table tests carried over must pass.

**Phase 2 — template and script.** Build `template/` and `scripts/new.sh`.
Verify by scaffolding into a throwaway directory and building it.

**Phase 3 — duolingo-bot.** Scaffold `personal/duolingo-bot` with module
`github.com/king-glitch/duolingo-bot`, flat layout, no `backend/` subdirectory.
Wire Mongo, the generator, and one real domain model end to end so the
generator path is proven.

**Phase 4 — publish.** Push `github.com/king-glitch/hexag` public, tag `v0.1.0`.
Until then, projects consume it through a `replace` directive in `go.work`,
which keeps iteration fast and stays out of committed `go.mod` files.

**Phase 5 — migrate the existing three, one at a time, later.** Delete the
duplicated files, add the import, run the tests. This is not a prerequisite for
anything above; the three projects keep working untouched until each is
individually worth the churn. Expected deletion is roughly 1,900 lines per
project.

## 7. Removing it

Delete the `hexag` imports, run `go mod tidy`, and copy back whichever packages
the project still needs. There is no runtime registration, no init-time magic,
and no generated code tying a project to the module — the one generator output
(`models/*.go`) imports `framework/mongo` for `Field[T]`, and re-adding a local
`field.go` restores independence. That is the whole exit path.
