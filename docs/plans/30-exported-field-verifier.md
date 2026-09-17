# Plan 30: Generic Exported-Field Verifier Rule

## Problem

The `runtime.Deps` struct in `aqw/backend` has a **mixed accessor pattern**:
some fields are unexported with `Get...()` getters (correct), while others
are exported for direct field access (breaks encapsulation). This violates
hexagonal architecture — consumers reach past the boundary and couple to the
concrete struct layout.

```go
type Deps struct {
    gas         ports.GameAccountService   // ✅ unexported + GetAccountService()
    LoginClient ports.GameLoginClient      // ❌ exported, no getter, direct access
    EventBus    *Bus                       // ❌ exported, no getter
}
```

The current verifier has **no rule** for this. We need one that is **generic
and dynamic** — no directory flags, no file-type checks, no interface-name
matching. Pure structural analysis of the AST.

---

## Audit Summary (aqw/backend)

| Category | Count | Exported Fields? | Getters? | Mixed? |
|---|---|---|---|---|
| `runtime.Deps` | 1 struct | 10 exported | 7 getters | **YES — only violation** |
| Services (`internal/services/`) | 10 structs | 0 | ✅ via ServiceBase | No |
| Handlers (`internal/adapters/endpoint/`) | 14 structs | 0 | N/A | No |
| Repositories (`internal/adapters/database/`) | all | 0 | N/A | No |
| HTTP DTOs (Request/Response) | ~50 structs | all exported | 0 | No |
| Domain models (`core/domain/game/`) | ~55 structs | all exported | 0 | No |
| Core args (`HuntArgs`, `Options`, etc.) | ~12 structs | all exported | 0 | No |
| Config (`rawConfig`) | 1 struct | all exported | 0 | No |
| `EnhancementPlan` | 1 struct | 8 exported | 0 | No |

**Key insight**: The rule "struct has getters → no exported fields" catches
exactly 1 struct (`Deps`) — the real violation. It naturally excludes DTOs,
models, args structs, and config (none of which have getters).

---

## Rule Definition

### Rule: Struct Accessor Consistency

> **If a struct has any `Get...()` methods, ALL its fields must be unexported.**
>
> A struct that provides getter methods has chosen the accessor pattern.
> Exported fields on such a struct are inconsistent — they bypass the getters
> and allow direct field access, breaking encapsulation.

**Detection is 100% structural:**
1. Does the struct have a method named `Get<X>()` with a return value? → It uses the accessor pattern.
2. Does the struct have any exported (capitalized) fields? → Violation.

**No directory, filename, or type-name checks needed.**

### What This Catches

| Pattern | Verdict | Why |
|---|---|---|
| Struct with getters + exported fields | ❌ Violation | Mixed access, breaks encapsulation |
| Struct with getters + all unexported fields | ✅ OK | Consistent accessor pattern |
| Struct with no getters + all exported fields | ✅ OK | Pure DTO/model/args, no encapsulation claimed |
| Struct with no getters + all unexported fields | ✅ OK | Package-private struct, no issue |

### What This Does NOT Catch (by design)

- Request/Response DTOs with exported fields (no getters → OK)
- Domain models like `game.Player` (no getters → OK)
- Config structs with env tags (no getters → OK)
- Args/options structs (no getters → OK)

---

## Implementation Plan

### Step 1: Collect struct fields and getter methods per file

In `checkAST`, after the existing `ast.Inspect` loop, add a **post-file
analysis phase**:

```
// Two maps, keyed by struct type name (within the current file):
structFields   map[string][]fieldInfo    // all fields of each struct
structGetters  map[string][]string       // Get...() method names per receiver type
```

**Collect structs** — in the existing `*ast.TypeSpec` case (or a new pass):
```go
if st, ok := ts.Type.(*ast.StructType); ok {
    for each field in st.Fields.List:
        record {name, exported, pos, typeStr}
}
```

**Collect getter methods** — in the existing `*ast.FuncDecl` case (or a new pass):
```go
if fn.Recv != nil && strings.HasPrefix(fn.Name.Name, "Get") && fn.Name.Name != "Get" {
    recvType := formatTypeExpr(fn.Recv.List[0].Type)  // strip pointer
    structGetters[recvType] = append(structGetters[recvType], fn.Name.Name)
}
```

### Step 2: Cross-reference and emit violations

After the file inspection is complete:

```go
for typeName, fields := range structFields {
    getters := structGetters[typeName]
    if len(getters) == 0 {
        continue  // No getters → pure data struct, skip
    }
    for _, f := range fields {
        if f.exported {
            addViolation(
                f.pos,
                "Struct Accessor Consistency",
                "Structs with getter methods must not have exported fields; "+
                    "all fields should be unexported and accessed via getters.",
                fmt.Sprintf(
                    "Exported field '%s' on struct '%s' which has %d getter method(s). "+
                    "Direct field access bypasses the accessor pattern.",
                    f.name, typeName, len(getters),
                ),
                fmt.Sprintf(
                    "Make field '%s' unexported and add a getter method 'Get%s()' "+
                    "if it needs to be accessed outside the struct.",
                    f.name, f.name,
                ),
            )
        }
    }
}
```

### Step 3: Handle cross-file methods (package-level)

Getter methods may be in a **different file** than the struct definition
(common pattern: struct in `runner.go`, getters in `deps.go`).

**Option A — Single-file only (simpler, catches most cases):**
Only correlate structs and methods within the same file. The `Deps` struct
and its getters are in the same file (`runner.go`), so this works for the
primary violation.

**Option B — Package-level collection (robust):**
Change the verifier to do a two-pass approach per package:
1. **Pass 1**: Parse all files in the package, collect `structFields` and
   `structGetters` maps at the package level.
2. **Pass 2**: Cross-reference and emit violations.

This requires refactoring `checkAST` from per-file to per-package. The
current `VerifyPath` → `filepath.Walk` → `checkFile` flow would need to
group files by package first.

**Recommendation**: Start with **Option A** (single-file). If it misses real
cases, upgrade to Option B in a follow-up.

### Step 4: Wire into checkAST

The new check fits naturally at the **end of `checkAST`**, after the existing
`ast.Inspect` loop. The two maps are populated during the inspection and
cross-referenced after.

```go
func (v *Verifier) checkAST(filePath string, f *ast.File) {
    // ... existing flags, imports, callFunMap ...

    structFields := make(map[string][]fieldInfo)
    structGetters := make(map[string]bool)

    ast.Inspect(f, func(n ast.Node) bool {
        switch node := n.(type) {
        // ... existing cases ...

        case *ast.TypeSpec:
            // existing: v.checkTypeSpec(...)
            // NEW: collect struct fields
            collectStructFields(node, structFields)

        case *ast.FuncDecl:
            // existing: v.checkFuncDecl(...)
            // NEW: collect getter methods
            collectGetterMethods(node, structGetters)
        }
        return true
    })

    // NEW: Post-inspection cross-reference
    if isInternal {
        v.checkStructAccessorConsistency(filePath, structFields, structGetters)
    }
}
```

### Step 5: Update AGENTS.md zero-tolerance table

Add a new row:

```
| **Struct Fields**    | Exported fields on structs with getter methods   | Make all fields unexported; add `Get...()` for each         |
```

### Step 6: Add tests

```go
func TestVerifier_StructAccessorConsistency(t *testing.T) {
    t.Run("struct with getters and exported fields is violation", ...)
    t.Run("struct with getters and all unexported fields is OK", ...)
    t.Run("struct with no getters and exported fields is OK", ...)
    t.Run("embedded fields are excluded from check", ...)
}
```

**Test case 1 — violation:**
```go
package runtime

type Deps struct {
    LoginClient SomeInterface  // exported → violation
    gas         SomeService    // unexported → OK
}

func (d Deps) GetAccountService() SomeService { return d.gas }
```
Expected: 1 violation on `LoginClient`.

**Test case 2 — clean struct with getters:**
```go
type Deps struct {
    gas    SomeService
    logger Logger
}

func (d Deps) GetAccountService() SomeService { return d.gas }
func (d Deps) GetLogger() Logger { return d.logger }
```
Expected: 0 violations.

**Test case 3 — pure data struct (no getters):**
```go
type HuntArgs struct {
    Map     string
    Monster string
    Item    string
}
```
Expected: 0 violations.

**Test case 4 — embedded struct fields excluded:**
```go
type Service struct {
    servicebase.ServiceBase  // embedded, exported type name but not a field
    repository ports.UserRepository
}
```
Expected: 0 violations (embedded fields are not "named exported fields").

### Step 7: Verify against aqw/backend

Run `go run ./cmd/hexag verify -dir ../aqw/backend` and confirm:
- `runtime.Deps` produces 10 violations (one per exported field)
- No false positives on DTOs, models, args, config, etc.
- Total violation count is manageable

---

## Edge Cases & Exclusions

| Case | Handling |
|---|---|
| Embedded struct fields (`servicebase.ServiceBase`) | Skip — embedded fields have no explicit name; `len(field.Names) == 0` |
| Getters returning error only (`GetError()`) | Still counts as getter — struct has accessor pattern |
| Methods named `Get` exactly (no suffix) | Exclude — not an accessor pattern (`Get()` is too generic) |
| Getter on pointer receiver (`func (d *Deps) Get...()`) | Still counts — strip `*` from receiver type |
| Struct in `internal/ports/` | Current `isInternal` flag plus `isPortsDir` exclusion handles this — ports models legitimately have exported fields |
| Struct in generated code (`mongo/models/`) | Already excluded by existing `_generated` / `models` path checks |
| `init()` or package-level functions | Ignored — only `FuncDecl` with receivers matter |

---

## Expected aqw/backend Impact

**New violations (10 — all on `runtime.Deps`):**
1. `LoginClient ports.GameLoginClient`
2. `BankClient ports.GameBankClient`
3. `Dialer ports.GameDialer`
4. `Handshaker ports.GameHandshaker`
5. `ScriptRegistry ports.ScriptRegistry`
6. `ClientVersion string`
7. `HeartbeatEvery time.Duration`
8. `EventBus *Bus`
9. `OnPartyJoined func(...)`
10. `LookupPlayer func(...)`

**Zero false positives:**
- 0 on HTTP DTOs (no getters)
- 0 on domain models (no getters)
- 0 on args/options structs (no getters)
- 0 on config structs (no getters)
- 0 on services/handlers (no exported fields)

---

## Future Extensions (out of scope for this plan)

1. **Package-level correlation**: If getter methods are in different files
   from the struct definition, extend to package-level analysis.
2. **Ports interface requirement**: Verify that structs with getters used
   cross-package have a corresponding interface in `internal/ports/`.
3. **Setter injection ban generalization**: Currently only checks `With...`
   on Service structs. Could generalize to any struct with getters.

---

## Checklist

- [ ] Implement `collectStructFields()` helper
- [ ] Implement `collectGetterMethods()` helper
- [ ] Implement `checkStructAccessorConsistency()` method on Verifier
- [ ] Wire into `checkAST` after the inspect loop
- [ ] Add tests (4 cases minimum)
- [ ] Update AGENTS.md and template/AGENTS.md zero-tolerance table
- [ ] Verify against aqw/backend: 10 new violations on `runtime.Deps`, 0 false positives
- [ ] Run `make verify && make test`
- [ ] Bump version, commit, tag, push, release, warm proxy
- [ ] Update MEMORY.md
