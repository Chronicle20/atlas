# Backend Guideline Conformance Sweep — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close backend-dev-guidelines checklist rows DOM-01, DOM-04 and FILE-05 repo-wide, with an evidence-backed exemption list for the rows that cannot be satisfied.

**Architecture:** One throwaway AST codemod (`docs/tasks/task-263-backend-guideline-conformance/codemod/`, its own module, deliberately outside `go.work`) with four subcommands — `classify`, `rename`, `transform`, `relocate` — plus hand work for the cases it must skip. Every subcommand emits a per-package ledger of `APPLIED` / `SKIPPED(reason)` whose union must equal its input inventory. Three workstreams (W2 rename → W3 transform → W1 relocate) never mix in one commit.

**Tech Stack:** Go 1.27.0, `golang.org/x/tools/go/packages` v0.49.0 (present in the module cache at `~/go/pkg/mod/golang.org/x/tools@v0.49.0`), `go/ast`, `go/types`, `go/format`.

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md))

## Global Constraints

- **No behavior change (FR-17).** No `RestModel` JSON tag, no `GetName()` return value, no route registration, no Kafka topic, no struct field type, and no `Build()` validation rule may change. Identifier renames only; never a literal or a struct tag.
- **Excluded trees (FR-18).** The branch diff must touch no `.sql` migration, no `docker-compose*.yml`, no `.tpl`/template file, and nothing under `libs/atlas-packet/`.
- **One checklist row per commit** (PRD §8). Never mix W1, W2 and W3 in a single commit.
- **Each commit independently buildable.** Cut a per-service commit only after that module's `go build ./...` and `go test ./...` pass from its module root.
- **Codemod-first** (PRD §8, `docs/codemod-vs-agents.md`). The 100 FILE-05 moves and the 59 DOM-01 renames MUST be done by the scripted transformation, never by fanning out implementer agents at the same transformation. Only the 81 tier-B/C DOM-04 packages are agent work.
- **Codemod module is invisible to the gate (D6).** It has its own `go.mod`, is NOT added to `go.work`, and lives outside `services/` and `libs/` so `tools/verify.sh:182`'s module enumeration never sees it. Build and run it with `GOWORK=off`.
- **Transform bodies read unexported fields directly (D1).** No new accessors are minted anywhere in this task. Hand-written `Transform`s in packages that already have complete accessor sets (`atlas-ban`, `atlas-buddies`) are untouched.
- **Landing order is W2 → W3 → W1** (design §7). W1's `model.go` rewrites are the conflict generator against the nine in-flight worktrees, and W1 is re-derivable by re-running the codemod after a rebase.
- **Ledger completeness.** For every codemod subcommand, `APPLIED + SKIPPED == input inventory`, and every `SKIPPED` package must land either in a hand-work commit or in `exemptions.md`. A package that is skipped and then forgotten is the specific failure this task exists to end.
- Never commit or push to `main`. All work happens on branch `task-263-backend-guideline-conformance` inside the worktree at `.worktrees/task-263-backend-guideline-conformance`.

---

## Task 1: Codemod module skeleton and ledger

### Files

- `docs/tasks/task-263-backend-guideline-conformance/codemod/go.mod` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/main.go` — new file; subcommand dispatch
- `docs/tasks/task-263-backend-guideline-conformance/codemod/load.go` — new file; `packages.Load` front-end shared by all subcommands
- `docs/tasks/task-263-backend-guideline-conformance/codemod/report.go` — new file; the ledger
- `docs/tasks/task-263-backend-guideline-conformance/codemod/report_test.go` — new file
- `go.work` — read-only; confirm the codemod module is NOT listed
- `tools/verify.sh:181-186` — read-only; the `all_modules()` enumeration that must not see the codemod

Module root for all `go build`/`go test` in this task: `docs/tasks/task-263-backend-guideline-conformance/codemod`, run with `GOWORK=off` in the environment (the repo's `go.work` at the tree root would otherwise capture it).

**Interfaces**

- Produces, consumed by Tasks 2, 6, 11, 20:
  - `func LoadModules(roots []string) ([]*packages.Package, error)` in `load.go` — loads with `packages.NeedName | NeedFiles | NeedCompiledGoFiles | NeedSyntax | NeedTypes | NeedTypesInfo | NeedImports | NeedDeps`, one `packages.Config` per module root, `Dir` set to the module root and `Env` carrying `GOWORK=off` unset (i.e. the module's own `go.mod` governs).
  - `type Ledger struct` in `report.go` with methods `func NewLedger(name string, input []string) *Ledger`, `func (l *Ledger) Applied(pkg string)`, `func (l *Ledger) Skipped(pkg, reason string)`, `func (l *Ledger) WriteTo(path string) error`, `func (l *Ledger) Verify() error`.
  - `Verify()` returns a non-nil error unless every input entry was recorded exactly once as either applied or skipped; `WriteTo` emits one TSV line per entry: `<pkg>\tAPPLIED\t` or `<pkg>\tSKIPPED\t<reason>`.

- [ ] **Step 1: Write the failing test**

`TestLedger` in `report_test.go` — table-driven, three subtests, no external fixtures.

| subtest | input | calls | expect `Verify()` | expect `WriteTo` content |
|---|---|---|---|---|
| `all applied` | `["a","b"]` | `Applied("a")`, `Applied("b")` | `nil` | `"a\tAPPLIED\t\nb\tAPPLIED\t\n"` |
| `mixed` | `["a","b"]` | `Applied("a")`, `Skipped("b","non-flat body")` | `nil` | `"a\tAPPLIED\t\nb\tSKIPPED\tnon-flat body\n"` |
| `incomplete` | `["a","b"]` | `Applied("a")` | error containing `"1 unrecorded: b"` | — |
| `unknown package` | `["a"]` | `Applied("z")` | error containing `"not in input: z"` | — |
| `double record` | `["a"]` | `Applied("a")`, `Skipped("a","x")` | error containing `"recorded twice: a"` | — |

Output lines are sorted by package path. `WriteTo` writes to a `t.TempDir()` path and the test reads it back with `os.ReadFile`.

- [ ] **Step 2: Run the test and confirm it fails**

```
GOWORK=off go test ./... -run TestLedger -v
```
from `docs/tasks/task-263-backend-guideline-conformance/codemod`. Expected: build failure, `undefined: NewLedger`.

- [ ] **Step 3: Write `go.mod`**

```
module atlas-task-263-codemod

go 1.27.0

require golang.org/x/tools v0.49.0
```

Then `GOWORK=off go mod tidy`. The version is pinned to the copy already in the module cache (`~/go/pkg/mod/golang.org/x/tools@v0.49.0`) so the build works without a network fetch.

- [ ] **Step 4: Implement `report.go`**

`Ledger` holds `name string`, `input map[string]bool`, and `entries map[string]entry` where `entry` is `{applied bool; reason string}`. `Applied`/`Skipped` record into `entries` and are no-ops on state; all validation happens in `Verify()`, which checks three conditions in this order: every recorded key is in `input` (`"ledger %s: not in input: %s"`), no key recorded twice (`"ledger %s: recorded twice: %s"`), and every input key recorded (`"ledger %s: %d unrecorded: %s"` with the missing keys joined by `", "` in sorted order).

- [ ] **Step 5: Implement `load.go` and `main.go`**

`load.go` exposes `LoadModules` as specified in Interfaces. It returns an error if any returned package has `len(pkg.Errors) > 0`, wrapping the first error — a codemod must never operate on a package that does not type-check.

`main.go` dispatches on `os.Args[1]`, printing usage and exiting 2 on an unrecognised subcommand. **Register only the subcommands that exist.** At this task that set is empty, so every invocation hits the usage path; Tasks 2, 6, 10 and 19 each register their subcommand as they land it. Do not pre-register a name whose handler does not yet do its job — an unregistered subcommand produces a clear usage error, a registered one that returns an error is a stub.

- [ ] **Step 6: Run the tests and confirm they pass**

```
GOWORK=off go test ./... -v
GOWORK=off go vet ./...
```
Expected: PASS.

- [ ] **Step 7: Confirm the gate cannot see the module**

```
find services libs -name go.mod | grep -c task-263
```
Run from the worktree root. Expected: `0`. Also confirm `grep -c 'task-263' go.work` prints `0`.

- [ ] **Step 8: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/codemod
git commit -m "chore(task-263): add conformance codemod skeleton and ledger"
```

---

## Task 2: `classify` subcommand — re-derive the work inventories

The design's tier counts (§2.1, §2.4, §2.5) came from classifier scripts that were not persisted. This task rebuilds them as a codemod subcommand so every later task reads a derived inventory rather than a number copied out of prose.

### Files

- `docs/tasks/task-263-backend-guideline-conformance/codemod/classify.go` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/classify_test.go` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/main.go` — wire the `classify` subcommand
- `docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-has-model.txt` — read-only input (185 lines)
- `docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-other.txt` — read-only input (69 lines)
- `docs/tasks/task-263-backend-guideline-conformance/inventory-file05-builders.txt` — read-only input (100 lines)

Module root: `docs/tasks/task-263-backend-guideline-conformance/codemod` (`GOWORK=off`).

**Interfaces**

- Produces three TSV files in the task folder, consumed by Tasks 6–24:
  - `classify-dom04.tsv` — `<pkgDir>\t<tier>\t<evidence>` where `<tier>` is one of `A`, `B1`, `B2`, `C`, `NO-RESTMODEL`.
  - `classify-dom01-fr15.tsv` — `<pkgDir>\t<class>\t<evidence>` where `<class>` is one of `RENAME`, `SIBLING-EXEMPT`, `NO-MODEL-GO`.
  - `classify-file05.tsv` — `<pkgDir>\t<srcFile>\t<builderType>\t<disposition>` where `<disposition>` is one of `RELOCATE`, `HAS-BUILDER-GO`, `ENTITY-BUILDER`, `EXCLUDED-TREE`.
- Produces `func ClassifyDOM04(pkg *packages.Package) (tier string, evidence string)` in `classify.go`.

### Classification rules

**DOM-04 tier**, evaluated in this order against the package loaded by `LoadModules`:

| tier | condition | evidence string |
|---|---|---|
| `NO-RESTMODEL` | the package declares no `type RestModel` | `"no type RestModel; wire types: <comma-separated names ending in RestModel>"` |
| `C` | no `func Extract*` anywhere in the package | `"no Extract in package"` |
| `B1` | two or more `func Extract*` declarations in the package | `"<n> Extract funcs: <names>"` |
| `B2` | exactly one `Extract`, but it is declared outside `rest.go`, OR its body is not a single `return <&?>Model{...}, nil` with a flat `*ast.CompositeLit` | `"Extract at <file>:<line>: <reason>"` |
| `A` | exactly one `Extract` in `rest.go` with a flat composite-literal body | `"flat literal, <n> fields"` |

"Flat" means the body is exactly one `*ast.ReturnStmt` with two results, the second being the identifier `nil`, and the first being a `*ast.CompositeLit` (optionally behind `&`) whose every element is a `*ast.KeyValueExpr` whose value is either an `*ast.SelectorExpr` on the parameter or a single-argument `*ast.CallExpr` wrapping such a selector. Any `for`, `if`, `switch`, `range`, or intermediate assignment in the function body disqualifies it.

**DOM-01 FR-15 class**, for each entry of `inventory-dom01-other.txt`:

| class | condition | evidence string |
|---|---|---|
| `NO-MODEL-GO` | the package directory has no `model.go` | `"no model.go; DOM-01 trigger does not fire"` |
| `RENAME` | the package declares exactly one `New*Builder` constructor and its return type's `Build()` returns the package's `Model` | `"sole builder over Model at <file>:<line>"` |
| `SIBLING-EXEMPT` | otherwise | `"siblings: <comma-separated New*Builder names with file:line>"` |

**FILE-05 disposition**, for each entry of `inventory-file05-builders.txt`:

| disposition | condition |
|---|---|
| `EXCLUDED-TREE` | path starts with `libs/atlas-packet/` (FR-18) |
| `ENTITY-BUILDER` | source file is named `entity_builder.go` (D5) |
| `HAS-BUILDER-GO` | the package directory already contains `builder.go` |
| `RELOCATE` | otherwise |

`HAS-BUILDER-GO` is an append target, not an exemption — those builders still move (FR-9). The disposition exists so Task 20 knows to append rather than create.

- [ ] **Step 1: Write the failing test**

`TestClassifyDOM04` in `classify_test.go` — table-driven over in-memory sources parsed and type-checked with `packages.Load` against a `t.TempDir()` throwaway module. Setup shape: write a `go.mod` (`module fixture\n\ngo 1.27.0\n`) plus the named `.go` files into the temp dir, then `LoadModules([]string{dir})`.

| case | fixture files | expect tier |
|---|---|---|
| `flat literal` | `model.go` with `type Model struct{ a uint32; b string }`; `rest.go` with `type RestModel struct{ A uint32; B string }` and `func Extract(rm RestModel) (Model, error) { return Model{a: rm.A, b: rm.B}, nil }` | `A` |
| `flat literal with conversion` | same, but `Extract` body `return Model{a: uint32(rm.A), b: rm.B}, nil` | `A` |
| `pointer literal` | `Extract` body `return &Model{a: rm.A}, nil` returning `(*Model, error)` | `A` |
| `has control flow` | `Extract` body contains `if rm.A == 0 { return Model{}, errors.New("x") }` before the return | `B2` |
| `builder chain` | `Extract` body `return NewBuilder(rm.A).SetB(rm.B).Build()` | `B2` |
| `Extract outside rest.go` | flat `Extract` declared in `converter.go`, `rest.go` holds only `RestModel` | `B2` |
| `two Extracts` | `rest.go` declares `Extract` and `ExtractCard`, both flat | `B1` |
| `no Extract` | `rest.go` declares `RestModel` only | `C` |
| `no RestModel` | `rest.go` declares `CardRestModel` and `func ExtractCard(rm CardRestModel) (Card, error)`, no `RestModel` | `NO-RESTMODEL` |

Assert both the tier and that the evidence string is non-empty.

- [ ] **Step 2: Run the test and confirm it fails**

```
GOWORK=off go test ./... -run TestClassifyDOM04 -v
```
Expected: FAIL, `undefined: ClassifyDOM04`.

- [ ] **Step 3: Implement `classify.go` and wire the subcommand**

`classify` takes `-repo <path>` (the worktree root) and `-out <dir>` (the task folder), reads the three input inventories, resolves each `rest.go` path to its package directory with `filepath.Dir`, loads every distinct module root once, and writes the three TSVs. It records a `Ledger` per inventory so a path that cannot be loaded shows up as `SKIPPED(load error: …)` rather than vanishing.

- [ ] **Step 4: Run the tests and confirm they pass**

```
GOWORK=off go test ./... -v
```
Expected: PASS.

- [ ] **Step 5: Run `classify` over the real tree and record the counts**

From the worktree root:

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod classify \
  -repo . -out docs/tasks/task-263-backend-guideline-conformance
cut -f2 docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv | sort | uniq -c
cut -f2 docs/tasks/task-263-backend-guideline-conformance/classify-dom01-fr15.tsv | sort | uniq -c
cut -f4 docs/tasks/task-263-backend-guideline-conformance/classify-file05.tsv | sort | uniq -c
```

The design's §2 expectation is DOM-04 `A≈104 B1≈37 B2≈37 C≈7` with 14 `NO-RESTMODEL`, FR-15 `RENAME=9 SIBLING-EXEMPT=55 NO-MODEL-GO=5`, and FILE-05 `RELOCATE≈89 HAS-BUILDER-GO=6 ENTITY-BUILDER=4 EXCLUDED-TREE=1`. **Paste the three verbatim `uniq -c` outputs into `docs/tasks/task-263-backend-guideline-conformance/progress.md` under a `## Task 2 — derived classification` heading.** If a count differs from the design by more than ±3, stop and report it — the design's sizing depends on it and a large divergence is a finding, not a rounding difference.

Sum checks that must hold exactly: DOM-04 TSV has 185 rows; FR-15 TSV has 69 rows; FILE-05 TSV has 100 rows.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/codemod \
        docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
        docs/tasks/task-263-backend-guideline-conformance/classify-dom01-fr15.tsv \
        docs/tasks/task-263-backend-guideline-conformance/classify-file05.tsv \
        docs/tasks/task-263-backend-guideline-conformance/progress.md
git commit -m "chore(task-263): derive conformance work classification"
```

---

## Task 3: W2 — merge the six alias constructor pairs

Six `atlas-channel` packages declare both `NewModelBuilder` and a `NewBuilder` that is a one-line delegation carrying the comment "alias for NewModelBuilder for backward compatibility". D3 merges them: inline `NewModelBuilder`'s body into `NewBuilder`, delete `NewModelBuilder`, repoint call sites, drop the now-false comment. Signatures are identical in all six, so this is behavior-preserving by inspection.

### Files

- `services/atlas-channel/atlas.com/channel/channel/builder.go:32` — alias comment; `NewBuilder()` / `NewModelBuilder()`
- `services/atlas-channel/atlas.com/channel/note/builder.go:25` — same shape
- `services/atlas-channel/atlas.com/channel/world/builder.go:24` — same shape
- `services/atlas-channel/atlas.com/channel/compartment/builder.go:33` — `(id, characterId, it, capacity)`
- `services/atlas-channel/atlas.com/channel/inventory/builder.go:18-29` — `NewModelBuilder(characterId uint32)` at :19, alias `NewBuilder` at :27
- `services/atlas-channel/atlas.com/channel/transport/route/builder.go:30-44` — `NewModelBuilder(name string)` at :31, alias `NewBuilder` at :42

Module root: `services/atlas-channel/atlas.com/channel`.

Do NOT touch `channel/asset` (Task 4) or `atlas-character/character` (Task 5) — their signatures differ and they are handled separately.

**Interfaces**

- Produces: in each of the six packages, exactly one constructor named `NewBuilder` with the pre-existing signature. `NewModelBuilder` no longer exists in these six packages. The `modelBuilder` type name is NOT changed here — that is the codemod's job in Task 7.

- [ ] **Step 1: Capture the call-site baseline**

```
grep -rn --include='*.go' 'NewModelBuilder(' services/atlas-channel/atlas.com/channel/channel services/atlas-channel/atlas.com/channel/note services/atlas-channel/atlas.com/channel/world services/atlas-channel/atlas.com/channel/compartment services/atlas-channel/atlas.com/channel/inventory services/atlas-channel/atlas.com/channel/transport/route | wc -l
```
Record the number. Also run the repo-wide form to catch cross-package callers:
```
grep -rn --include='*.go' -e 'channel\.NewModelBuilder(' -e 'note\.NewModelBuilder(' -e 'world\.NewModelBuilder(' -e 'compartment\.NewModelBuilder(' -e 'inventory\.NewModelBuilder(' -e 'route\.NewModelBuilder(' services libs
```

- [ ] **Step 2: Merge each pair**

For each of the six files: replace the `NewModelBuilder` declaration and the aliasing `NewBuilder` declaration with a single `NewBuilder` carrying `NewModelBuilder`'s original body and a plain doc comment. For `inventory/builder.go` the result is:

```go
// NewBuilder creates a new builder instance
func NewBuilder(characterId uint32) *modelBuilder {
	return &modelBuilder{
		characterId:  characterId,
		compartments: make(map[inventory.Type]compartment.Model),
	}
}
```

Note `inventory/builder.go:31-35` also has `BuilderSupplier` which already calls `NewBuilder` — leave it unchanged. Do not alter `CloneModel`, `FoldCompartment`, or any setter.

- [ ] **Step 3: Repoint every call site found in Step 1**

Rewrite each `NewModelBuilder(` occurrence in these six packages (and any cross-package caller found in Step 1) to `NewBuilder(`. Test files included (FR-12).

- [ ] **Step 4: Verify no `NewModelBuilder` remains in the six packages**

```
grep -rn --include='*.go' 'NewModelBuilder' services/atlas-channel/atlas.com/channel/channel services/atlas-channel/atlas.com/channel/note services/atlas-channel/atlas.com/channel/world services/atlas-channel/atlas.com/channel/compartment services/atlas-channel/atlas.com/channel/inventory services/atlas-channel/atlas.com/channel/transport/route
```
Expected: no output.

- [ ] **Step 5: Build and test**

```
go build ./... && go vet ./... && go test ./...
```
from `services/atlas-channel/atlas.com/channel`. Expected: PASS. `services/atlas-channel/atlas.com/channel/inventory/builder_test.go` and `services/atlas-channel/atlas.com/channel/transport/route/builder_test.go` exercise these constructors and must pass unmodified except for the call-site rename.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel
git commit -m "refactor(atlas-channel): merge NewModelBuilder aliases into NewBuilder"
```

---

## Task 4: W2 — disambiguate `channel/asset`'s two constructors

`channel/asset` is the one package of the eight where the two constructors genuinely differ in arity, so a merge would change a call site's semantics. Per D3: keep `NewBuilder(compartmentId, templateId)` unchanged and rename `NewModelBuilder(id, compartmentId, templateId)` to `NewBuilderWithId`.

### Files

- `services/atlas-channel/atlas.com/channel/asset/builder.go:119` — `NewBuilder(compartmentId uuid.UUID, templateId uint32) *ModelBuilder`; leave unchanged
- `services/atlas-channel/atlas.com/channel/asset/builder.go:126-127` — `NewModelBuilder(id uint32, compartmentId uuid.UUID, templateId uint32) *ModelBuilder`; rename to `NewBuilderWithId`

Module root: `services/atlas-channel/atlas.com/channel`.

The `ModelBuilder` type in this package is NOT renamed here — Task 7's codemod renames it to `Builder`, at which point both constructors return `*Builder`.

**Interfaces**

- Produces: `func NewBuilderWithId(id uint32, compartmentId uuid.UUID, templateId uint32) *ModelBuilder` in package `asset`. `NewBuilder`'s signature is unchanged.

- [ ] **Step 1: Enumerate the call sites**

```
grep -rn --include='*.go' 'asset\.NewModelBuilder(\|[^.]NewModelBuilder(' services/atlas-channel/atlas.com/channel/asset services/atlas-channel/atlas.com/channel
```
Record every hit that resolves to package `asset`.

- [ ] **Step 2: Rename the declaration and its call sites**

```go
// NewBuilderWithId creates a new builder with an explicit ID
func NewBuilderWithId(id uint32, compartmentId uuid.UUID, templateId uint32) *ModelBuilder {
	return &ModelBuilder{
		id:            id,
		compartmentId: compartmentId,
		templateId:    templateId,
	}
}
```

- [ ] **Step 3: Verify**

```
grep -rn --include='*.go' 'NewModelBuilder' services/atlas-channel/atlas.com/channel/asset
```
Expected: no output.

- [ ] **Step 4: Build and test**

```
go build ./... && go vet ./... && go test ./...
```
from `services/atlas-channel/atlas.com/channel`. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/asset
git commit -m "refactor(atlas-channel): rename asset NewModelBuilder to NewBuilderWithId"
```

---

## Task 5: W2 — disambiguate `atlas-character/character`'s two builders

**This package is not in the design's §2.3 table.** The design lists seven collisions, all in `atlas-channel`; the tree has eight. `services/atlas-character/atlas.com/character/character` declares two genuinely distinct builders:

- `builder.go:91` — `func NewBuilder(c BuilderConfiguration, accountId uint32, worldId world.Id, name string, skinColor byte, gender byte, hair uint32, face uint32) *Builder`, the character-creation builder that applies `BuilderConfiguration` defaults.
- `model.go:242` — `func NewModelBuilder() *modelBuilder`, the zero-value model-reconstruction builder used by the Kafka consumer and the REST layer, with a companion `CloneModel(m Model) *modelBuilder` at `model.go:246`.

They are not interchangeable and `NewBuilder` is already taken by the conformant creation builder, so FR-12's rename cannot apply. Two consequences, both recorded in `exemptions.md` in Task 25:

1. **The constructor is renamed `NewEmptyBuilder`,** satisfying FR-16's "zero `NewModelBuilder` in the sweep" without colliding.
2. **The `modelBuilder` type is exempt from FR-13.** Renaming it to `builder` would leave `Builder` and `builder` in one package differing only in case. Evidence: `builder.go:91` declares `type Builder`.

88 call sites in `services/atlas-character/atlas.com/character` reference `NewModelBuilder()`, the majority in `_test.go` files, which FR-12 explicitly includes.

### Files

- `services/atlas-character/atlas.com/character/character/model.go:242` — rename `NewModelBuilder` to `NewEmptyBuilder`; leave `type modelBuilder` and `CloneModel` unchanged
- `services/atlas-character/atlas.com/character/character/builder.go:91` — read-only; the `Builder` that already holds the `NewBuilder` name
- `services/atlas-character/atlas.com/character/character/provider.go:60` — call site
- `services/atlas-character/atlas.com/character/kafka/consumer/character/consumer.go:371` — call site
- plus the remaining ~85 call sites across `services/atlas-character/atlas.com/character`, enumerated in Step 1

Module root: `services/atlas-character/atlas.com/character`.

**Interfaces**

- Produces: `func NewEmptyBuilder() *modelBuilder` in package `character`. `NewBuilder` and `type Builder` are unchanged; `type modelBuilder` keeps its name.

- [ ] **Step 1: Enumerate and count the call sites**

```
grep -rn --include='*.go' 'NewModelBuilder(' services/atlas-character | wc -l
```
Expected at branch point: `88`. If the count differs, list the hits before proceeding.

- [ ] **Step 2: Rename the declaration**

```go
// NewEmptyBuilder creates a zero-valued builder for reconstructing a Model.
// The creation-flow builder is NewBuilder in builder.go; the two are distinct.
func NewEmptyBuilder() *modelBuilder {
	return &modelBuilder{}
}
```

- [ ] **Step 3: Rewrite every call site**

Rewrite `NewModelBuilder()` → `NewEmptyBuilder()` and `character.NewModelBuilder()` → `character.NewEmptyBuilder()` across `services/atlas-character`, test files included. This is a repository-mechanical sweep over one service, so a scripted `sed` over the enumerated file list is appropriate here.

- [ ] **Step 4: Verify**

```
grep -rn --include='*.go' 'NewModelBuilder' services/atlas-character
```
Expected: no output.

- [ ] **Step 5: Build and test**

```
go build ./... && go vet ./... && go test ./...
```
from `services/atlas-character/atlas.com/character`. Expected: PASS. `services/atlas-character/atlas.com/character/character/rest_test.go:53,102`, `hp_mp_gain_test.go:55`, `kafka_integration_test.go`, `patch_integration_test.go`, `meso_outbox_test.go`, `pending_change/*_test.go` and `kafka/consumer/character/pending_change_applier_test.go:81` all exercise the renamed constructor.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character
git commit -m "refactor(atlas-character): rename NewModelBuilder to NewEmptyBuilder"
```

---

## Task 6: W2 — `rename` subcommand

Type-aware rename driven by `types.Object` identity, not by name. After Tasks 3–5 every remaining target package has exactly one constructor to rename.

### Files

- `docs/tasks/task-263-backend-guideline-conformance/codemod/rename.go` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/rename_test.go` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/main.go` — wire the `rename` subcommand
- `docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-newmodelbuilder.txt` — read-only input

Module root: `docs/tasks/task-263-backend-guideline-conformance/codemod` (`GOWORK=off`).

**Interfaces**

- Produces `func Rename(pkgs []*packages.Package, targets []Target) (*Ledger, error)` in `rename.go`, where `type Target struct { PkgDir string; From, To string }`.
- The rename set applied per target package, from FR-12/FR-13/FR-14:

| from | to | note |
|---|---|---|
| `NewModelBuilder` | `NewBuilder` | FR-12 |
| `ModelBuilder` | `Builder` | FR-13, exported stays exported |
| `modelBuilder` | `builder` | FR-13, unexported stays unexported |
| `CloneModelBuilder` | `CloneBuilder` | FR-14 |

`CloneModel` is NOT renamed — it is a different function (it takes a `Model`, not a builder) and FR-14 names only `CloneModelBuilder`.

### Algorithm

1. Load every module under `services/` and `libs/` (the same enumeration `tools/verify.sh:181-186` uses: `find services libs -name go.mod`), so the rewrite reaches the 743 cross-package call sites and the 35 qualified type references.
2. For each target package, resolve the `*types.TypeName` and `*types.Func` objects for the four names above in that package's scope. A name that does not resolve in that package is simply absent (e.g. most packages have no `CloneModelBuilder`) and is not an error.
3. Walk every `*ast.File` of every loaded package. For each `*ast.Ident`, look up `pkg.TypesInfo.Uses[ident]` and `pkg.TypesInfo.Defs[ident]`; if the resulting object is identical (`==`) to a resolved target object, set `ident.Name` to the replacement.
4. **Collision guard (R2).** Before applying `modelBuilder` → `builder` in a package, check `pkg.Types.Scope().Lookup("builder")` and every function scope in the package's files for an existing object named `builder`. If one exists, record `SKIPPED(builder identifier already in scope)` for the whole package and apply none of its four renames.
5. Write each modified file back with `format.Node` into a `bytes.Buffer`, then `os.WriteFile` with the file's existing permissions. Preserve existing line endings — read the original bytes first and, if they contain `\r\n`, re-apply CRLF before writing.
6. Emit the ledger.

The rename touches `*ast.Ident` nodes only. It never touches a `*ast.BasicLit`, so no struct tag, no JSON tag, and no string literal can change (R4, FR-17).

- [ ] **Step 1: Write the failing test**

`TestRename` in `rename_test.go` — table-driven over throwaway modules built in `t.TempDir()`, setup shape copied from `TestClassifyDOM04` (Task 2, `classify_test.go`).

| case | fixture | expect |
|---|---|---|
| `exported type and constructor` | `builder.go` with `type ModelBuilder struct{ a uint32 }`, `func NewModelBuilder() *ModelBuilder { return &ModelBuilder{} }`, `func (b *ModelBuilder) SetA(v uint32) *ModelBuilder { b.a = v; return b }` | file contains `type Builder struct`, `func NewBuilder() *Builder`, `func (b *Builder) SetA(v uint32) *Builder`; contains no `ModelBuilder` |
| `unexported type` | same with `modelBuilder` / `NewModelBuilder` returning `*modelBuilder` | contains `type builder struct`, `func NewBuilder() *builder`; contains no `modelBuilder` |
| `CloneModelBuilder` | adds `func CloneModelBuilder(b *ModelBuilder) *ModelBuilder { return b }` | contains `func CloneBuilder(b *Builder) *Builder` |
| `CloneModel untouched` | adds `func CloneModel(m Model) *ModelBuilder { return &ModelBuilder{} }` and `type Model struct{}` | still contains `func CloneModel(m Model) *Builder` — the name `CloneModel` is unchanged |
| `cross-package call site` | second package `caller` importing the first and calling `fixture.NewModelBuilder()` and declaring `var x *fixture.ModelBuilder` | `caller`'s file contains `fixture.NewBuilder()` and `*fixture.Builder` |
| `collision guard` | package declares `type modelBuilder struct{}` and a function `func f() { builder := 1; _ = builder }` | ledger records `SKIPPED` with reason containing `builder identifier already in scope`; the file is byte-identical to its input |
| `string literal untouched` | `type ModelBuilder struct{ a uint32 \`json:"modelBuilder"\` }` plus `const s = "NewModelBuilder"` | the tag still reads `json:"modelBuilder"` and `s` still reads `"NewModelBuilder"` |
| `ledger completeness` | two input packages, one applied one skipped | `ledger.Verify()` returns `nil` |

- [ ] **Step 2: Run the test and confirm it fails**

```
GOWORK=off go test ./... -run TestRename -v
```
Expected: FAIL, `undefined: Rename`.

- [ ] **Step 3: Implement `rename.go` and wire the subcommand**

`rename` takes `-repo <path>`, `-inventory <path>` (defaults to `inventory-dom01-newmodelbuilder.txt`), `-ledger <path>`, and `-only <comma-separated service prefixes>` so Tasks 7–9 can apply it one service batch at a time and cut per-service commits. A `-dry-run` flag prints the ledger without writing files.

- [ ] **Step 4: Run the tests and confirm they pass**

```
GOWORK=off go test ./... -v
GOWORK=off go vet ./...
```
Expected: PASS.

- [ ] **Step 5: Dry-run against the real tree**

From the worktree root:
```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod rename \
  -repo . -dry-run -ledger /dev/stdout
```
Confirm the ledger's line count equals the remaining `NewModelBuilder` declaration count:
```
grep -rn --include='*.go' '^func NewModelBuilder(' services libs | grep -v _test.go | wc -l
```
Expected after Tasks 3–5: `52` (59 minus the 6 merged in Task 3 and the 1 in Task 4; Task 5's package was never in `inventory-dom01-newmodelbuilder.txt`'s 59 — verify this and report the actual number). Report any `SKIPPED` entries; they become hand work.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/codemod
git commit -m "chore(task-263): add type-aware builder rename to codemod"
```

---

## Task 7: W2 — apply the rename to `atlas-channel`

`atlas-channel` carries 19 of the 59 `NewModelBuilder` declarations, the single largest cluster, and is the module most likely to expose a cross-package call-site bug. It gets its own commit.

### Files

- `services/atlas-channel/atlas.com/channel/**` — rewritten by the codemod; the exact file set is the codemod's ledger output
- `docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv` — new file

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Apply**

From the worktree root:
```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod rename \
  -repo . -only services/atlas-channel \
  -ledger docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv
```

- [ ] **Step 2: Check the ledger**

```
cut -f2 docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv | sort | uniq -c
```
Expected: `APPLIED` for every entry. Any `SKIPPED` must be resolved by hand in this same task before committing, and its reason recorded in `progress.md`.

- [ ] **Step 3: Confirm no `NewModelBuilder` survives in the service**

```
grep -rn --include='*.go' 'NewModelBuilder\|ModelBuilder' services/atlas-channel
```
Expected: no output. (`ModelBuilder` is gone too — FR-13.)

- [ ] **Step 4: Build and test**

```
go build ./... && go vet ./... && go test ./...
```
from `services/atlas-channel/atlas.com/channel`. Expected: PASS.

- [ ] **Step 5: Confirm no literal changed (FR-17)**

```
git diff -U0 -- services/atlas-channel | grep '^[+-]' | grep -E '"|`' | grep -v 'ModelBuilder|modelBuilder'
```
Expected: no output. Any hit is a literal the rename touched and must be reverted.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv
git commit -m "refactor(atlas-channel): rename ModelBuilder to Builder"
```

---

## Task 8: W2 — apply the rename to the remaining services

The other 33 declarations spread across 23 services, one to six each. Same mechanical change repeated, so it batches into one task with one commit per service.

### Files

The service list, from `docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-newmodelbuilder.txt`, minus `atlas-channel` (Task 7):

- `services/atlas-query-aggregator` (6) — e.g. `.../query-aggregator/character/model.go:405`, `.../skill/model.go:99`, `.../marriage/model.go:70`, `.../party/model.go:38`, `.../quest/model.go:234`, `.../party_quest/model.go:54`
- `services/atlas-pets` (5) — `.../pets/data/cash/builder.go:11`, `.../pets/pet/exclude/builder.go:10`, `.../pets/pet/builder.go:28`, `.../pets/data/pet/model.go:65`, `.../pets/character/model.go:327`
- `services/atlas-monsters` (3), `services/atlas-world` (2), `services/atlas-tenants` (2), `services/atlas-skills` (2), `services/atlas-quest` (2), `services/atlas-monster-book` (2), `services/atlas-consumables` (2)
- `services/atlas-storage`, `services/atlas-rps`, `services/atlas-reactors`, `services/atlas-npc-shops`, `services/atlas-mounts`, `services/atlas-messages`, `services/atlas-maps`, `services/atlas-keys`, `services/atlas-inventory`, `services/atlas-expressions`, `services/atlas-drops`, `services/atlas-data`, `services/atlas-cashshop` (1 each)
- `services/atlas-character` — 1 entry in the inventory; **check whether it is the `character/character` package already handled in Task 5.** If so it is already done; if it is a different package, apply the rename normally.
- `docs/tasks/task-263-backend-guideline-conformance/ledger-rename-rest.tsv` — new file

- [ ] **Step 1: Apply per service and commit per service**

For each service `S` in the list above, from the worktree root:

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod rename \
  -repo . -only services/<S> \
  -ledger docs/tasks/task-263-backend-guideline-conformance/ledger-rename-rest.tsv -append
```

then, from that service's module root (`services/<S>/atlas.com/<name>`, as listed in `plan-context.sh`'s "Module roots"):

```
go build ./... && go vet ./... && go test ./...
```

and only on PASS:

```bash
git add services/<S>
git commit -m "refactor(<S>): rename ModelBuilder to Builder"
```

A cross-service caller pulled in by the rename (a `libs/` consumer, or another service importing the renamed package) goes in the same commit as the package that owns the symbol.

- [ ] **Step 2: Confirm FR-16 repo-wide**

```
grep -rn --include='*.go' 'NewModelBuilder' services libs
```
Expected: no output. This is acceptance criterion 1.

Also confirm the type rename is complete:
```
grep -rn --include='*.go' 'ModelBuilder' services libs
```
Expected: only `libs/atlas-packet/model/skill_usage_info.go` (`SkillUsageInfoBuilder` is not a `ModelBuilder`, so expect no output here either — if `libs/atlas-packet` appears, revert it, FR-18).

- [ ] **Step 3: Check the combined ledger**

```
cut -f2 docs/tasks/task-263-backend-guideline-conformance/ledger-rename-rest.tsv | sort | uniq -c
wc -l docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv docs/tasks/task-263-backend-guideline-conformance/ledger-rename-rest.tsv
```
The two ledgers' combined line count must equal the number of declarations the codemod was given in Task 6 Step 5. Record the verbatim output in `progress.md`.

- [ ] **Step 4: Commit the ledger**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/ledger-rename-rest.tsv \
        docs/tasks/task-263-backend-guideline-conformance/progress.md
git commit -m "chore(task-263): record builder rename ledger"
```

---

## Task 9: W2 — FR-15 triage and the nine sole-builder renames

`classify-dom01-fr15.tsv` (Task 2) partitions the 69 `other New<X>Builder` constructors. Only the `RENAME` class changes code; the other two classes become `exemptions.md` entries in Task 25.

### Files

- `docs/tasks/task-263-backend-guideline-conformance/classify-dom01-fr15.tsv` — new file at plan time; produced by Task 2, read-only input here, 69 rows
- The ~9 packages whose row reads `RENAME` — the exact list is the TSV; each gets its `New<X>Builder`, `<X>Builder`, and any `Clone<X>Builder` renamed to `NewBuilder`, `Builder`, `CloneBuilder`
- `docs/tasks/task-263-backend-guideline-conformance/ledger-rename-fr15.tsv` — new file

- [ ] **Step 1: Read the RENAME rows**

```
awk -F'\t' '$2=="RENAME"' docs/tasks/task-263-backend-guideline-conformance/classify-dom01-fr15.tsv
```
Expected: 9 rows per D4. If the count differs, record the actual number in `progress.md` and proceed with the derived list — the TSV is authoritative over the design's prose.

- [ ] **Step 2: Apply the rename per package**

The `rename` subcommand takes `-targets <tsv>` where the TSV supplies explicit `pkgDir\tfrom\tto` triples, so the FR-15 renames reuse Task 6's machinery with a different symbol list. Build that triple file from the `RENAME` rows: for a package whose sole builder is `NewProposalBuilder`/`ProposalBuilder`, the triples are `<pkgDir>\tNewProposalBuilder\tNewBuilder` and `<pkgDir>\tProposalBuilder\tBuilder`, plus `<pkgDir>\tCloneProposalBuilder\tCloneBuilder` where that function exists.

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod rename \
  -repo . -targets docs/tasks/task-263-backend-guideline-conformance/fr15-targets.tsv \
  -ledger docs/tasks/task-263-backend-guideline-conformance/ledger-rename-fr15.tsv
```

- [ ] **Step 3: Build and test each affected module**

From each affected module root: `go build ./... && go vet ./... && go test ./...`. Expected: PASS.

- [ ] **Step 4: Commit per service**

```bash
git add services/<S>
git commit -m "refactor(<S>): rename sole builder to NewBuilder"
```

- [ ] **Step 5: Commit the ledger and triple file**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/ledger-rename-fr15.tsv \
        docs/tasks/task-263-backend-guideline-conformance/fr15-targets.tsv
git commit -m "chore(task-263): record FR-15 builder rename ledger"
```

---

## Task 9a: W2 — the out-of-inventory `ModelBuilder` type declarations (FR-13 scope gap)

Added mid-execution by user decision (2026-08-26), recorded in `progress.md` under
"USER DECISION — FR-13 scope gap". Tasks 6-8 drove the rename from
`inventory-dom01-newmodelbuilder.txt`, which lists `NewModelBuilder` **constructor**
declarations. FR-13's `ModelBuilder`/`modelBuilder` → `Builder`/`builder` **type** rename has a
wider footprint: packages whose builder is constructed by a `Clone`-based or otherwise
differently-named function declare the type but were never in the inventory, so Tasks 6-8 never
reached them. This task closes that gap so the repo-wide FR-13 check can actually pass.

The codemod needs no new rename logic: `renamePairs` (`codemod/rename.go:44-49`) already carries
`ModelBuilder`→`Builder` and `modelBuilder`→`builder`. The only thing Tasks 6-8 lacked was a way
to name a package that has no `NewModelBuilder` — which Task 9's `-targets` flag supplies. A
targets row with an **empty stem column** falls back to `renamePairs`, which is exactly what this
task wants.

### Files

- `docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-modelbuilder-type.txt` —
  new file, written by the controller at Task 9 time; 39 `grep -n` lines, read-only input here
- `docs/tasks/task-263-backend-guideline-conformance/dom01-type-targets.tsv` — new file
- `docs/tasks/task-263-backend-guideline-conformance/ledger-rename-type.tsv` — new file
- The 16 services holding those 39 declarations: `atlas-cashshop` (6), `atlas-npc-shops` (4),
  `atlas-character` (4), `atlas-query-aggregator` (3), `atlas-pets` (3), `atlas-merchant` (3),
  `atlas-login` (3), `atlas-inventory` (3), `atlas-consumables` (3), and one each in
  `atlas-summons`, `atlas-storage`, `atlas-quest`, `atlas-monsters`, `atlas-drops`,
  `atlas-doors`, `atlas-channel`
- `docs/tasks/task-263-backend-guideline-conformance/progress.md` — appended

Patterns to copy: Task 7's `atlas-channel` commit (`fdcac2dbd`) for the per-service commit shape
and the by-hand test-name / doc-comment fixup precedent; Task 8's per-service temp ledger +
merge approach (the plan's `-append` flag does not exist).

- [ ] **Step 1: Re-derive and confirm the inventory**

```
grep -rn --include='*.go' -E 'type (M|m)odelBuilder' services libs | grep -v 'libs/atlas-packet/'
```
Expected: 39 lines — 32 `type ModelBuilder`, 7 `type modelBuilder` — matching
`inventory-dom01-modelbuilder-type.txt`. If the count differs (Task 9 may have moved one),
record the actual number in `progress.md` and proceed with the live derivation, which is
authoritative.

- [ ] **Step 2: Build the targets file**

One row per **package directory** (dedupe — several declarations share a package), stem column
empty so `renameImpl` falls back to `renamePairs`:

```
services/atlas-cashshop/atlas.com/cashshop/asset	
services/atlas-cashshop/atlas.com/cashshop/character/inventory	
…
```

- [ ] **Step 3: Dry-run the whole set first**

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod rename \
  -repo . -targets docs/tasks/task-263-backend-guideline-conformance/dom01-type-targets.tsv \
  -ledger /tmp/ledger-type-dry.tsv -dry-run
```
Read the ledger before applying. SKIPPED rows are expected and are the point of the dry run:
`services/atlas-quest/atlas.com/quest/quest` will SKIP on the known R2 collision
(`quest/processor.go:851,921`), and `services/atlas-character/atlas.com/character/character`
carries a Task 5 exemption — its `type modelBuilder` and `CloneModel` were deliberately left
alone. **Record every SKIP and its reason; do not force any of them.** They become
`exemptions.md` rows in Task 25 or hand work, whichever the reason warrants.

- [ ] **Step 4: Apply per service, one service per commit**

Run the codemod restricted to one service at a time (`-only services/<S>`), then from that
module root `go build ./... && go vet ./... && go test ./...` — expected PASS — then:

```bash
git add services/<S>
git commit -m "refactor(<S>): rename ModelBuilder type to Builder"
```

Bring stale `TestModelBuilder_*` test-function names and doc-comment text up to date by hand in
the same commit, per the Task 4/7 precedent. Never `git add -A` or `git add .`.

- [ ] **Step 5: Merge the per-service ledgers and commit the artifacts**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/ledger-rename-type.tsv \
        docs/tasks/task-263-backend-guideline-conformance/dom01-type-targets.tsv \
        docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-modelbuilder-type.txt
git commit -m "chore(task-263): record DOM-01 type rename ledger"
```

- [ ] **Step 6: Re-run the repo-wide check and record the residue**

Re-run Step 1's grep. Expected residue: only the SKIPs recorded in Step 3 — the
`atlas-quest/quest` collision and the `atlas-character/character` Task 5 exemption. Paste the
verbatim output into `progress.md` under `## Task 9a — DOM-01 type rename`. Any path in the
residue that is not an already-recorded SKIP is a forgotten package; go back and close it.

---

## Task 10: W3 — `transform` subcommand

Inverts a tier-A `Extract` into a `Transform` and generates the round-trip test. Tier A only; anything else is skipped to hand work.

### Files

- `docs/tasks/task-263-backend-guideline-conformance/codemod/transform.go` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/transform_test.go` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/main.go` — wire the `transform` subcommand
- `docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv` — new file at plan time; produced by Task 2, read-only input here
- `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest.go:40-42` — read-only; the canonical tier-A `Extract`
- `services/atlas-ban/atlas.com/ban/ban/rest.go:36-47` — read-only; the house `Transform` shape
- `services/atlas-ban/atlas.com/ban/ban/rest_test.go:10-45` — read-only; the house `TestTransform` shape

Module root: `docs/tasks/task-263-backend-guideline-conformance/codemod` (`GOWORK=off`).

**Interfaces**

- Produces `func GenerateTransform(pkg *packages.Package) (restGoAddition, testGoAddition string, err error)` in `transform.go`.

### Algorithm

1. Locate the package's single `func Extract(rm RestModel) (Model, error)` (or `(*Model, error)`) and require the tier-A flat-literal shape from Task 2's `ClassifyDOM04`.
2. Invert each `*ast.KeyValueExpr`. Key `id` with value `rm.Id` becomes key `Id` with value `m.id`. Where the value is a conversion `T(rm.X)`, emit `U(m.x)` where `U` is the `RestModel` field's declared type read from `pkg.TypesInfo.TypeOf` — **never guessed**. A `RestModel` field that `Extract` does not read (e.g. `consumable.RestModel.Id`, which `Extract` at `rest.go:40` ignores) is not emitted; the generated `Transform` is the exact inverse of `Extract`, no more.
3. Append the generated `func Transform(m Model) (RestModel, error)` to `rest.go`.
4. Generate the round-trip test (below) into `rest_test.go`, creating the file with the package clause if absent.
5. **Type-check before writing.** Re-run `packages.Load` on the modified package in a scratch copy. If it does not type-check, discard both additions and record `SKIPPED(generated code does not type-check: <first error>)`. The codemod never writes code it has not checked. This is what makes the 8 conversion cases safe.

### Generated test

Per §6, with the model built through the package's builder where one is usable:

```go
func TestTransformRoundTrip(t *testing.T) {
	m, err := NewBuilder().SetMonsterBook(true).SetMonsterId(100100).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, m)
	}
}
```

Construction rule, evaluated per package:

- Use `NewBuilder(...).Set<Field>(v)....Build()` when the package declares `NewBuilder` **and** every field `Extract` maps has a corresponding `Set<Field>` setter on the builder type, resolved from `TypesInfo`. (This is CLAUDE.md's builder-pattern rule and §6's preference, and it is the dependency that puts W3 after W2.)
- Otherwise construct `Model{...}` directly with a composite literal — legal in-package and consistent with D1. `consumable` is such a package: it declares no builder at all.
- Never emit a `*_testhelpers.go` file.

Field values are derived from the field's type and **must be distinct and non-zero** — this is the only thing that catches R1, a field mapped to the wrong same-typed sibling:

| type | value sequence |
|---|---|
| any integer type | `11`, `22`, `33`, `44`, … (`n*11` for the n-th integer field) |
| `string` | `"field1"`, `"field2"`, … |
| `bool` | first `bool` field `true`; a second `bool` field makes the values indistinguishable, so the package is **skipped to hand work** with reason `two or more bool fields` |
| `float32`/`float64` | `1.5`, `2.5`, … |
| `uuid.UUID` | `uuid.MustParse("00000000-0000-0000-0000-0000000000<nn>")` with `nn` the field index |
| `time.Time` | `time.Unix(1700000000+n, 0).UTC()` |
| any other type (named, slice, pointer, map, struct) | **skip to hand work**, reason `unsupported field type <type>` |

A package where any field's value cannot be derived is skipped whole — never given a partially-zero-valued test, which would pass while `Transform` silently drops the zero-valued field.

- [ ] **Step 1: Write the failing test**

`TestGenerateTransform` in `transform_test.go` — table-driven over throwaway modules in `t.TempDir()`, setup shape copied from `classify_test.go` (Task 2).

| case | fixture `Extract` | expect in `restGoAddition` | expect skip |
|---|---|---|---|
| `plain fields` | `return Model{a: rm.A, b: rm.B}, nil` with `a uint32`/`A uint32`, `b string`/`B string` | `func Transform(m Model) (RestModel, error)` returning `RestModel{A: m.a, B: m.b}` | — |
| `conversion` | `return Model{t: BanType(rm.T)}, nil` with `T byte`, `t BanType` | `RestModel{T: byte(m.t)}` | — |
| `unmapped RestModel field` | `RestModel` has `Id uint32`, `Extract` ignores it | generated `Transform` contains no `Id:` key | — |
| `pointer return` | `return &Model{a: rm.A}, nil` | `Transform(m Model) (RestModel, error)` reading `m.a` | — |
| `two bools` | `return Model{x: rm.X, y: rm.Y}, nil`, both `bool` | — | skip, reason `two or more bool fields` |
| `slice field` | `return Model{ids: rm.Ids}, nil` with `ids []uint32` | — | skip, reason contains `unsupported field type` |
| `type-check failure` | fixture forced to produce an ill-typed body via a deliberately mismatched conversion | — | skip, reason contains `does not type-check`; `rest.go` byte-identical to input |

And `TestGenerateRoundTripTest`:

| case | fixture | expect in `testGoAddition` |
|---|---|---|
| `builder available` | package declares `NewBuilder()` with `SetA`/`SetB` covering both fields | `NewBuilder().SetA(11).SetB("field2").Build()` |
| `no builder` | package declares no `New*Builder` | `m := Model{a: 11, b: "field2"}` and no `err` from construction |
| `builder missing a setter` | `NewBuilder()` has `SetA` but no `SetB` | composite-literal form, not the builder form |
| `always` | any | `reflect.DeepEqual(got, m)` and the function name `TestTransformRoundTrip` |

- [ ] **Step 2: Run the test and confirm it fails**

```
GOWORK=off go test ./... -run 'TestGenerateTransform|TestGenerateRoundTripTest' -v
```
Expected: FAIL, `undefined: GenerateTransform`.

- [ ] **Step 3: Implement `transform.go` and wire the subcommand**

`transform` takes `-repo`, `-classification <classify-dom04.tsv>`, `-ledger`, `-only <service prefix>`, and `-dry-run`. It operates only on rows whose tier is `A`.

- [ ] **Step 4: Run the tests and confirm they pass**

```
GOWORK=off go test ./... -v
GOWORK=off go vet ./...
```
Expected: PASS.

- [ ] **Step 5: Dry-run against the real tree**

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod transform \
  -repo . -classification docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
  -dry-run -ledger /dev/stdout
```
Record the `APPLIED`/`SKIPPED` split in `progress.md`. Every `SKIPPED` package joins the tier-B/C hand-work list of Tasks 13–18.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/codemod
git commit -m "chore(task-263): add Transform generation to codemod"
```

---

## Task 11: W3 — apply `transform` to `atlas-channel` tier-A packages

`atlas-channel` holds 50 of the 185 `has-model` packages, the largest cluster. Own commit.

### Files

- `services/atlas-channel/atlas.com/channel/**/rest.go` — `Transform` appended by the codemod
- `services/atlas-channel/atlas.com/channel/**/rest_test.go` — `TestTransformRoundTrip` added; several are new files
- `docs/tasks/task-263-backend-guideline-conformance/ledger-transform-channel.tsv` — new file

Module root: `services/atlas-channel/atlas.com/channel`.

- [ ] **Step 1: Apply**

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod transform \
  -repo . -classification docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
  -only services/atlas-channel \
  -ledger docs/tasks/task-263-backend-guideline-conformance/ledger-transform-channel.tsv
```

- [ ] **Step 2: Run the generated tests and confirm they pass**

```
go test ./... -run TestTransformRoundTrip -v
```
from `services/atlas-channel/atlas.com/channel`. Expected: PASS for every generated test. **A failure here is R1 firing** — the codemod mapped a field to the wrong same-typed sibling. Fix the generator, not the generated file, and re-run.

- [ ] **Step 3: Full module gate**

```
go build ./... && go vet ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 4: Confirm no behavior change (FR-17)**

```
git diff -U0 -- services/atlas-channel | grep '^-' | grep -v '^---'
```
Expected: no output at all. `transform` only appends; it must delete nothing.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel docs/tasks/task-263-backend-guideline-conformance/ledger-transform-channel.tsv
git commit -m "feat(atlas-channel): add Transform and round-trip tests"
```

---

## Task 12: W3 — apply `transform` to the remaining tier-A services

Same mechanical change, one commit per service.

### Files

Services with tier-A rows in `classify-dom04.tsv`, minus `atlas-channel`. From `inventory-dom04-has-model.txt`'s per-service distribution, the candidates are:

- `services/atlas-messages` (12), `services/atlas-consumables` (10), `services/atlas-monster-death` (8), `services/atlas-inventory` (7), `services/atlas-character` (7), `services/atlas-npc-shops` (6)
- `services/atlas-saga-orchestrator`, `services/atlas-reactors`, `services/atlas-query-aggregator`, `services/atlas-monsters`, `services/atlas-maps` (5 each)
- `services/atlas-trades`, `services/atlas-summons`, `services/atlas-storage`, `services/atlas-pets`, `services/atlas-npc-conversations`, `services/atlas-login`, `services/atlas-doors`, `services/atlas-cashshop` (4 each)
- `services/atlas-party-quests`, `services/atlas-mini-games`, `services/atlas-guilds`, `services/atlas-drops`, `services/atlas-dragons`, `services/atlas-chairs`, `services/atlas-ban` (2 each)
- `services/atlas-transports`, `services/atlas-tenants`, `services/atlas-rankings`, `services/atlas-portals`, `services/atlas-parties`, `services/atlas-mts`, `services/atlas-monster-book`, `services/atlas-messengers`, `services/atlas-merchant`, `services/atlas-marriages`, `services/atlas-kites`, `services/atlas-fame`, `services/atlas-data`, `services/atlas-buddies` (1 each)
- `docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest.tsv` — new file

The authoritative list is `awk -F'\t' '$2=="A"' classify-dom04.tsv`; the table above is the expected shape, not the input.

- [ ] **Step 1: Apply and commit per service**

For each service `S`:

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod transform \
  -repo . -classification docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
  -only services/<S> \
  -ledger docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest.tsv -append
```

then from `services/<S>/atlas.com/<name>`:

```
go test ./... -run TestTransformRoundTrip -v
go build ./... && go vet ./... && go test ./...
```

and only on PASS:

```bash
git add services/<S>
git commit -m "feat(<S>): add Transform and round-trip tests"
```

- [ ] **Step 2: Verify the ledger is complete**

```
cat docs/tasks/task-263-backend-guideline-conformance/ledger-transform-channel.tsv docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest.tsv | cut -f2 | sort | uniq -c
awk -F'\t' '$2=="A"' docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv | wc -l
```
The two must sum to the tier-A count. Record verbatim in `progress.md`.

- [ ] **Step 3: Write the hand-work list**

```
awk -F'\t' '$2!="A"' docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv > docs/tasks/task-263-backend-guideline-conformance/handwork-dom04.tsv
cat docs/tasks/task-263-backend-guideline-conformance/ledger-transform-*.tsv | awk -F'\t' '$2=="SKIPPED"' >> docs/tasks/task-263-backend-guideline-conformance/handwork-dom04.tsv
wc -l docs/tasks/task-263-backend-guideline-conformance/handwork-dom04.tsv
```
This file is the input to Tasks 13–18. Expected ≈81 rows plus any tier-A skips.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/ledger-transform-rest.tsv \
        docs/tasks/task-263-backend-guideline-conformance/handwork-dom04.tsv \
        docs/tasks/task-263-backend-guideline-conformance/progress.md
git commit -m "chore(task-263): record Transform ledger and hand-work list"
```

---

## Task 13: W3 hand work — the four packages named in #1498 (FR-7)

These are the packages the original issue named, and three of the four are `NO-RESTMODEL` (D2), so they exercise every hand-work shape at once. Doing them first establishes the pattern the remaining hand-work tasks copy.

### Files

- `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go:17-100` — declares `type Model` at :17 and five sibling wire types (`EquipmentRestModel`:34, `ConsumableRestModel`:52, `SetupRestModel`:70, plus Etc and Cash). No `RestModel`. Needs `TransformEquipment`, `TransformConsumable`, `TransformSetup`, `TransformEtc`, `TransformCash` (FR-3), preferably over a generic helper mirroring the existing `extract[R]`.
- `services/atlas-channel/atlas.com/channel/data/tradeability/rest_test.go` — new file
- `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go` (117 lines) — same five-compartment shape
- `services/atlas-inventory/atlas.com/inventory/data/tradeability/rest_test.go` — new file
- `services/atlas-channel/atlas.com/channel/monsterbook/rest.go:102,107` — `ExtractCard(rm CardRestModel) (Card, error)` at :102 and `Extract(rm CollectionRestModel) (Collection, error)` at :107. Needs `TransformCard(Card) (CardRestModel, error)` and `Transform(Collection) (CollectionRestModel, error)` per FR-2.
- `services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go` — existing file, add the round-trip tests
- `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest.go:40-42` — tier A; if Task 12 already generated its `Transform`, verify it and move on
- `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest_test.go` — existing file

Patterns to copy: `services/atlas-ban/atlas.com/ban/ban/rest.go:36-47` (`Transform` shape) and `services/atlas-ban/atlas.com/ban/ban/rest_test.go:10-45` (`TestTransform` shape). Module roots: `services/atlas-channel/atlas.com/channel`, `services/atlas-inventory/atlas.com/inventory`, `services/atlas-monster-book/atlas.com/monster-book`.

- [ ] **Step 1: Write the failing tests**

Three test functions, one per package needing new work. Each is the round-trip from §6, built through the package's builder where one exists and through a composite literal otherwise (`data/tradeability`'s `Model` is constructed by `NewModel(tradeBlock bool, tradeAvailable int32, only bool)` at `rest.go:30` — use it).

`TestTransformRoundTrip` in `services/atlas-channel/atlas.com/channel/data/tradeability/rest_test.go` — table-driven over the five compartment types:

| case | input `Model` | transform fn | extract fn |
|---|---|---|---|
| equipment | `NewModel(true, 77, false)` | `TransformEquipment` | the existing equipment `extract` path |
| consumable | `NewModel(false, 88, true)` | `TransformConsumable` | consumable `extract` |
| setup | `NewModel(true, 99, true)` | `TransformSetup` | setup `extract` |
| etc | `NewModel(false, 111, false)` | `TransformEtc` | etc `extract` |
| cash | `NewModel(true, 222, true)` | `TransformCash` | cash `extract` |

Each case asserts `reflect.DeepEqual(Extract<X>(Transform<X>(m)), m)`. Every field is distinct and non-zero across cases so a cross-compartment mix-up shows up.

`TestTransformRoundTrip` in `.../monsterbook/rest_test.go` — two subtests:

| subtest | input | assertion |
|---|---|---|
| `card` | a `Card` with every field set to a distinct non-zero value | `ExtractCard(TransformCard(c)) == c` via `reflect.DeepEqual` |
| `collection` | a `Collection` holding two distinct `Card`s | `Extract(Transform(col)) == col` via `reflect.DeepEqual` |

`TestTransformRoundTrip` in `.../monster-book/data/consumable/rest_test.go` — single case, `Model{monsterBook: true, monsterId: 100100}` (the package has no builder), asserting `Extract(Transform(m)) == m`. Note `RestModel.Id` at `rest.go:13` is not mapped by `Extract` at `rest.go:41`, so `Transform` must not emit it and the round trip is over the two mapped fields only.

Read the actual field lists from each package's `Model`/`Card`/`Collection` declaration before writing the values — do not invent field names.

- [ ] **Step 2: Run the tests and confirm they fail**

From each module root: `go test ./<pkg>/... -run TestTransformRoundTrip -v`. Expected: FAIL, `undefined: TransformEquipment` (and the equivalents).

- [ ] **Step 3: Implement the `Transform` functions**

Each is the exact inverse of the package's existing `Extract`, reading unexported fields directly per D1. For `data/tradeability`, mirror the existing generic `extract[R]` with a `transform[R]` helper so the five named functions are one line each (FR-3 states this form is preferred).

- [ ] **Step 4: Run the tests and confirm they pass**

Same commands. Expected: PASS.

- [ ] **Step 5: Full module gate for each of the three modules**

```
go build ./... && go vet ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 6: Commit per service**

```bash
git add services/atlas-channel/atlas.com/channel/data/tradeability services/atlas-channel/atlas.com/channel/monsterbook
git commit -m "feat(atlas-channel): add Transform for tradeability and monsterbook"
git add services/atlas-inventory
git commit -m "feat(atlas-inventory): add Transform for tradeability"
git add services/atlas-monster-book
git commit -m "feat(atlas-monster-book): add Transform round-trip test for consumable"
```

---

## Task 14: W3 hand work — the remaining `NO-RESTMODEL` packages (D2)

`classify-dom04.tsv`'s `NO-RESTMODEL` rows are the 14 packages of design §8.2 where `func Transform(` cannot be written because no `type RestModel` exists. Three are done in Task 13. The rest get `Transform`-prefixed named variants per FR-2/FR-3 **and** an `exemptions.md` entry recorded in Task 25.

### Files

The remaining 11 of the design's §8.2 list (regenerate with `awk -F'\t' '$2=="NO-RESTMODEL"' classify-dom04.tsv` — that list is authoritative):

- `services/atlas-cashshop/atlas.com/cashshop/rewardpool/rest.go` (35 lines; `RewardRestModel` at :7)
- `services/atlas-dragons/atlas.com/dragons/dragon/rest.go` (57 lines)
- `services/atlas-drops/atlas.com/drops/data/foothold/rest.go` (72 lines)
- `services/atlas-messengers/atlas.com/messengers/character/rest.go` (106 lines)
- `services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go` (980 lines)
- `services/atlas-parties/atlas.com/parties/character/rest.go` (107 lines)
- `services/atlas-pets/atlas.com/pets/data/position/rest.go` (62 lines)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/rates/rest.go` (31 lines)
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/rest.go` (188 lines)
- `services/atlas-summons/atlas.com/summons/summon/rest.go` (45 lines)
- `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` (1093 lines; has `rest_test.go`)

Each package also needs its `rest_test.go` — new files except `atlas-tenants/configuration`, which has one.

Patterns to copy: `services/atlas-ban/atlas.com/ban/ban/rest.go:36-47` and `:91` (`TransformCheck(m *Model) CheckRestModel` — the house form for a named variant over a secondary wire type).

Module roots are the per-service roots listed in `plan-context.sh`'s "Module roots" section.

**Scope note for `npc/conversation` (resolves PRD open question 2 per design §8.2):** FR-1 applies only to the types this package's `rest.go` already declares a `RestModel` for and that `Extract` already round-trips. A builder-backed domain type with no wire representation gets no `Transform`. `services/atlas-npc-conversations/atlas.com/npc/conversation/model.go` (2430 lines) declares 20+ builder-backed types; do **not** give them `Transform` functions.

- [ ] **Step 1: For each package, enumerate its `Extract*` functions**

```
grep -n '^func Extract' services/<path>/rest.go
```
Every `Extract<X>(rm <X>RestModel) (<X>, error)` gets a matching `Transform<X>(<X>) (<X>RestModel, error)`; a bare `Extract(rm <Y>RestModel) (<Y>, error)` gets `Transform`. This is FR-2's "mirror the existing `Extract` naming" rule, applied mechanically.

- [ ] **Step 2: Write the failing round-trip test per package**

One `TestTransformRoundTrip` per package, with a subtest per `Extract*`/`Transform*` pair. Each subtest constructs the domain value through the package's builder where one exists (`NewBuilder(...).Set<Field>(v)....Build()`) and through a composite literal otherwise, with **every field distinct and non-zero**, then asserts `reflect.DeepEqual(Extract<X>(Transform<X>(v)), v)`.

Read each package's domain type declaration and enumerate its fields before writing the case; do not invent field names or values.

- [ ] **Step 3: Run and confirm the tests fail**

From each module root: `go test ./<pkg>/... -run TestTransformRoundTrip -v`. Expected: FAIL with `undefined: Transform<X>`.

- [ ] **Step 4: Implement each `Transform<X>`**

Exact inverse of the paired `Extract<X>`, reading unexported fields directly (D1). Mint no accessors.

- [ ] **Step 5: Run the tests and the full module gate**

```
go test ./<pkg>/... -run TestTransformRoundTrip -v
go build ./... && go vet ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 6: Record the exemption evidence**

For each package, append to `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md` a line of the form used by design §8.1:

```
services/atlas-summons/atlas.com/summons/summon — no `type RestModel`; wire types: <names>; Extract at rest.go:<line>; provided: <Transform names>
```

Task 25 turns these into `exemptions.md` entries. Do not skip a package here — an unrecorded package is the exact failure mode this task exists to end.

- [ ] **Step 7: Commit per service**

```bash
git add services/<S>
git commit -m "feat(<S>): add Transform variants for <pkg>"
```

---

## Task 15: W3 hand work — tier B1 packages (multiple `Extract*` in one package)

`awk -F'\t' '$2=="B1"' classify-dom04.tsv` — ≈37 packages, each declaring two or more `Extract*` functions over distinct types. Per-package judgment; not codemod work.

### Files

The authoritative list is the `B1` rows of `docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv`. Each row names a package directory; the files are that package's `rest.go` and `rest_test.go` (new file where absent).

Patterns to copy: `services/atlas-ban/atlas.com/ban/ban/rest.go:36` (`Transform`) and `:91` (`TransformCheck`, the named-variant form), with the test shape at `services/atlas-ban/atlas.com/ban/ban/rest_test.go:10-45`.

**Because this list spans ~15 services and the transformation needs per-package judgment, this is the part of the task where fanning out `atlas-implementer` agents is correct** (design §5, `docs/codemod-vs-agents.md`). Split the B1 rows into batches of at most **one service each** and dispatch one agent per batch, with `model: sonnet`. The 100 FILE-05 moves and the 59 DOM-01 renames must NOT be agent work.

- [ ] **Step 1: Partition the B1 rows by service and record the batches**

```
awk -F'\t' '$2=="B1"{print $1}' docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
  | sed -E 's#^(services/[^/]+).*#\1#' | sort | uniq -c
```
Write the resulting batch list into `progress.md`.

- [ ] **Step 2: Per batch, write the failing tests**

One `TestTransformRoundTrip` per package with one subtest per `Extract*`/`Transform*` pair, exactly as in Task 14 Step 2: domain value built through the package's builder where available, every field distinct and non-zero, assertion `reflect.DeepEqual(Extract<X>(Transform<X>(v)), v)`.

- [ ] **Step 3: Run and confirm the tests fail**

From the batch's module root: `go test ./... -run TestTransformRoundTrip -v`. Expected: FAIL, `undefined: Transform<X>`.

- [ ] **Step 4: Implement each `Transform<X>` as the exact inverse of its `Extract<X>`**

Naming follows FR-2: `Transform` for the type whose `Extract` is bare, `Transform<X>` mirroring `Extract<X>`. Read unexported fields directly (D1); mint no accessors.

- [ ] **Step 5: Run the tests and the full module gate per batch**

```
go test ./... -run TestTransformRoundTrip -v
go build ./... && go vet ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 6: Commit per service**

```bash
git add services/<S>
git commit -m "feat(<S>): add Transform and round-trip tests"
```

- [ ] **Step 7: Confirm every B1 row is now closed**

```
awk -F'\t' '$2=="B1"{print $1}' docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv \
  | while IFS= read -r d; do grep -Lq '' /dev/null; grep -q '^func Transform' "$d/rest.go" || echo "MISSING: $d"; done
```
Expected: no `MISSING` lines.

---

## Task 16: W3 hand work — tier B2 packages (non-flat or displaced `Extract`)

`awk -F'\t' '$2=="B2"' classify-dom04.tsv` — ≈37 packages with exactly one `Extract` whose body has control flow, an intermediate assignment, or a builder chain, or which is declared outside `rest.go`.

### Files

The authoritative list is the `B2` rows of `docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv`, each with an evidence string naming the `Extract`'s `file:line` and the disqualifying reason. Files per package: its `rest.go` (add `Transform`) and `rest_test.go` (new file where absent).

Patterns to copy: `services/atlas-ban/atlas.com/ban/ban/rest.go:36-62` — this is itself a B2 shape (`Extract` at :49 is a builder chain, `Transform` at :36 is its hand-written inverse using accessors). Its `rest_test.go:10-45` is the test shape.

Same fan-out rule as Task 15: one `atlas-implementer` per service batch, `model: sonnet`.

**`Transform` goes in `rest.go` even when `Extract` lives elsewhere.** DOM-04's rule (`audit-checklist.md:97`) places `Transform` in `rest.go`; a displaced `Extract` is a separate pre-existing deviation and moving it is out of scope (PRD §2 non-goals).

- [ ] **Step 1: Partition the B2 rows by service and record the batches**

```
awk -F'\t' '$2=="B2"{print $1"\t"$3}' docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv
```
Write the batch list into `progress.md`.

- [ ] **Step 2: Per package, write the failing round-trip test**

`TestTransformRoundTrip`, single case per package: construct the `Model` through the package's builder with every field distinct and non-zero, assert `reflect.DeepEqual(Extract(Transform(m)), m)`.

**Where `Extract` can return an error on some inputs** (the common B2 reason — a validating builder chain), the test's `Model` must be a value `Extract` accepts, and the test additionally asserts `err == nil` from `Extract`. Do not add a new validation rule to satisfy the test; FR-17 forbids changing a `Build()` validation rule.

**Where `Extract` is genuinely lossy** — it drops a `Model` field that no `RestModel` field carries — the round trip cannot hold. Record the package in `handwork-notes.md` with the `file:line` of the dropped field and assert only over the fields that do round-trip, naming them explicitly in the test. Do NOT add a `RestModel` field to close the gap; that is an API surface change and PRD §5 forbids it.

- [ ] **Step 3: Run and confirm the tests fail**

From each module root: `go test ./... -run TestTransformRoundTrip -v`. Expected: FAIL, `undefined: Transform`.

- [ ] **Step 4: Implement each `Transform` in `rest.go`**

The inverse of the package's `Extract`, reading unexported fields directly (D1). Where `Extract`'s builder chain applies a default that no `Model` field records, `Transform` simply does not emit it.

- [ ] **Step 5: Run the tests and the full module gate per batch**

```
go test ./... -run TestTransformRoundTrip -v
go build ./... && go vet ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 6: Commit per service**

```bash
git add services/<S>
git commit -m "feat(<S>): add Transform and round-trip tests"
```

---

## Task 17: W3 hand work — tier C packages (no `Extract` at all)

`awk -F'\t' '$2=="C"' classify-dom04.tsv` — ≈7 packages that declare both a `Model` and a `RestModel` but no converter in either direction. `Transform` is derived from the two struct declarations rather than inverted from an `Extract`.

### Files

The authoritative list is the `C` rows of `docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv`. Per package: `rest.go` and `rest_test.go` (new file).

Patterns to copy: `services/atlas-ban/atlas.com/ban/ban/rest.go:36-47`.

- [ ] **Step 1: For each package, pair the fields**

Read the `Model` and `RestModel` declarations and pair each exported `RestModel` field with the `Model` field whose name matches case-insensitively. **A `RestModel` field with no `Model` counterpart, or vice versa, is a finding — record it in `handwork-notes.md` with both `file:line`s and omit it from `Transform` rather than inventing a mapping.**

- [ ] **Step 2: Write the failing round-trip test**

Tier C has no `Extract` to round-trip against, so the assertion is field-by-field on the produced `RestModel` — the one place in this task where field-by-field is correct, because there is no inverse to compose with.

`TestTransform` per package, single case:

| input | assertion |
|---|---|
| `Model` with every field set to a distinct non-zero value, built through the package's builder where one exists | one `if rm.<Field> != <expected>` check per mapped field, with the message `"<Field> mismatch. Expected %v, got %v"` — the shape at `services/atlas-ban/atlas.com/ban/ban/rest_test.go:25-45` |

- [ ] **Step 3: Run and confirm the tests fail**

`go test ./... -run TestTransform -v` from each module root. Expected: FAIL, `undefined: Transform`.

- [ ] **Step 4: Implement `Transform`**

```go
func Transform(m Model) (RestModel, error) {
	return RestModel{
		// one line per paired field, reading m's unexported field directly
	}, nil
}
```

Return `nil` for the error unless a field conversion can genuinely fail (FR-1).

- [ ] **Step 5: Run the tests and the full module gate**

```
go test ./... -run TestTransform -v
go build ./... && go vet ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 6: Commit per service**

```bash
git add services/<S>
git commit -m "feat(<S>): add Transform and test"
```

---

## Task 18: W3 — close out DOM-04 and confirm the ledger

### Files

- `docs/tasks/task-263-backend-guideline-conformance/handwork-dom04.tsv` — new file at plan time; produced by Task 12, read-only input here
- `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md` — new file at plan time; accumulated by Tasks 14, 16, 17, read-only here
- `docs/tasks/task-263-backend-guideline-conformance/progress.md` — new file at plan time; first written in Task 2, appended here

- [ ] **Step 1: Confirm every `has-model` package now has a `Transform`**

```
bash docs/tasks/task-263-backend-guideline-conformance/sweep.sh
bash docs/tasks/task-263-backend-guideline-conformance/split-by-model.sh
wc -l docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-has-model.txt
```
Expected: only the `NO-RESTMODEL` packages (design §8.2, ≈14) remain, because `sweep.sh` greps the literal `^func Transform(` which those packages cannot host by construction (D2, accepted residue R3). Every other `has-model` row must be gone.

- [ ] **Step 2: Cross-check the residue against `handwork-notes.md`**

Every path still in `inventory-dom04-has-model.txt` must appear in `handwork-notes.md`. Any path in neither the closed set nor the notes is a forgotten package — go back and close it.

- [ ] **Step 3: Record the evidence**

Paste the verbatim `wc -l` output and the residual path list into `progress.md` under `## Task 18 — DOM-04 closeout`.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance
git commit -m "chore(task-263): record DOM-04 closeout evidence"
```

---

## Task 19: W1 — `relocate` subcommand

Moves a builder's declaration set from its current file into `builder.go` in the same package. Intra-package, byte-identical at the declaration level.

### Files

- `docs/tasks/task-263-backend-guideline-conformance/codemod/relocate.go` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/relocate_test.go` — new file
- `docs/tasks/task-263-backend-guideline-conformance/codemod/main.go` — wire the `relocate` subcommand
- `docs/tasks/task-263-backend-guideline-conformance/classify-file05.tsv` — new file at plan time; produced by Task 2, read-only input here

Module root: `docs/tasks/task-263-backend-guideline-conformance/codemod` (`GOWORK=off`).

**Interfaces**

- Produces `func Relocate(pkg *packages.Package, srcFile, builderType string) (newSrc, newBuilderGo []byte, err error)` in `relocate.go`.

### Algorithm

1. Collect the builder's declaration set from the source file's AST: the `type <X>Builder struct` declaration, every `func New<X>Builder`/`NewBuilder` returning `*<X>Builder` (or `<X>Builder`), every `func` with a `*<X>Builder` or `<X>Builder` receiver, and every free function returning `*<X>Builder` whose name begins with `Clone`.
2. Move those declarations verbatim into `builder.go` in the same package — creating it with the package clause if absent, appending if present (the 6 `HAS-BUILDER-GO` packages).
3. Recompute both files' import sets from the moved declarations' actual references via `TypesInfo`, **not** by copying the source file's import block. Format both with `format.Source`.
4. Preserve the source file's line endings on both outputs.

**Skip conditions** (→ hand work, recorded in the ledger):

| reason | condition |
|---|---|
| `receiver shared with non-builder decl` | a moved method's receiver type is also the receiver of a declaration the collector did not select |
| `source file would become empty` | removing the set leaves the source file with only a package clause and imports |
| `reference_data.go excluded` | source file is `reference_data.go` (FR-11, hand-split in Task 22) |
| `entity builder` | disposition is `ENTITY-BUILDER` (D5, never moved) |
| `excluded tree` | disposition is `EXCLUDED-TREE` (FR-18) |
| `multiple builders in file` | the source file declares more than one `<X>Builder` type — batch them in one move or hand them off; `conversation/model.go`'s 20 go to Task 23 |

- [ ] **Step 1: Write the failing test**

`TestRelocate` in `relocate_test.go` — table-driven over throwaway modules in `t.TempDir()`, setup shape copied from `classify_test.go` (Task 2).

| case | fixture `model.go` | expect `builder.go` | expect `model.go` |
|---|---|---|---|
| `basic move` | `type Model struct{ a uint32 }`, `type ModelBuilder struct{ a uint32 }`, `func NewModelBuilder() *ModelBuilder`, `func (b *ModelBuilder) SetA(v uint32) *ModelBuilder`, `func (b *ModelBuilder) Build() (Model, error)` | contains all four builder decls and `package fixture` | contains `type Model` only; no `Builder` identifier |
| `Clone function moves` | adds `func CloneModel(m Model) *ModelBuilder` | contains `CloneModel` | does not contain `CloneModel` |
| `imports recomputed` | builder uses `uuid.UUID`, `Model` uses only `uint32`; `model.go` imports both `uuid` and `time` | `builder.go` imports `uuid`, not `time` | `model.go` imports neither `uuid` nor `time` if unused |
| `appends to existing builder.go` | package already has `builder.go` with `type OtherBuilder struct{}` | contains both `OtherBuilder` and `ModelBuilder`, one package clause | as `basic move` |
| `source would be empty` | `model.go` declares only the builder set | ledger records `SKIPPED` reason `source file would become empty`; both files byte-identical to input | — |
| `shared receiver` | a `func (b *ModelBuilder) NotABuilderMethod()` that the package's `resource.go` requires stay in `model.go` — model it as a method whose receiver is also used by a non-selected decl | ledger records `SKIPPED` reason `receiver shared with non-builder decl` | — |
| `CRLF preserved` | `model.go` written with `\r\n` line endings | both outputs use `\r\n` | both outputs use `\r\n` |

- [ ] **Step 2: Run the test and confirm it fails**

```
GOWORK=off go test ./... -run TestRelocate -v
```
Expected: FAIL, `undefined: Relocate`.

- [ ] **Step 3: Implement `relocate.go` and wire the subcommand**

`relocate` takes `-repo`, `-classification <classify-file05.tsv>`, `-ledger`, `-only <service prefix>`, `-dry-run`. It processes only `RELOCATE` and `HAS-BUILDER-GO` rows.

- [ ] **Step 4: Run the tests and confirm they pass**

```
GOWORK=off go test ./... -v
GOWORK=off go vet ./...
```
Expected: PASS.

- [ ] **Step 5: Dry-run against the real tree**

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod relocate \
  -repo . -classification docs/tasks/task-263-backend-guideline-conformance/classify-file05.tsv \
  -dry-run -ledger /dev/stdout
```
Record the `APPLIED`/`SKIPPED` split in `progress.md`. Design §5 expects ≈65 applied and ≈7 hand.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/codemod
git commit -m "chore(task-263): add builder relocation to codemod"
```

---

## Task 20: W1 — apply `relocate`, first half

The 72 FILE-05 packages, minus the four special cases (Tasks 22, 23) and the two `exemptions.md` dispositions. Split across two tasks so each stays inside the implementer budget.

### Files

Services from `classify-file05.tsv`'s `RELOCATE`/`HAS-BUILDER-GO` rows, first batch:

- `services/atlas-query-aggregator` (10 declarations)
- `services/atlas-login` (6)
- `services/atlas-pets` (5)
- `services/atlas-maps` (4)
- `services/atlas-consumables` (4)
- `services/atlas-npc-shops` (3)
- `libs/atlas-script-core` (3)
- `docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-a.tsv` — new file

Module roots: `services/atlas-query-aggregator/atlas.com/query-aggregator`, `services/atlas-login/atlas.com/login`, `services/atlas-pets/atlas.com/pets`, `services/atlas-maps/atlas.com/maps`, `services/atlas-consumables/atlas.com/consumables`, `services/atlas-npc-shops/atlas.com/npc-shops`, `libs/atlas-script-core`.

- [ ] **Step 1: Apply per service**

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod relocate \
  -repo . -classification docs/tasks/task-263-backend-guideline-conformance/classify-file05.tsv \
  -only <path> \
  -ledger docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-a.tsv -append
```

- [ ] **Step 2: Confirm the move is a pure relocation**

```
git diff -M --stat -- <path>
git diff -M -- <path> | grep '^[+-]' | grep -v '^[+-][+-]' | grep -vE '^\+package |^\+$|^\+import|^\+\t"|^\+\)|^-import|^-\t"|^-\)'
```
Every remaining line must be a moved declaration appearing once as `-` and once as `+`. **No signature, field name, or body text may differ between the two.** Any asymmetry is a defect — revert and fix the codemod.

- [ ] **Step 3: Build and test per module**

```
go build ./... && go vet ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 4: Commit per service**

```bash
git add <path>
git commit -m "refactor(<S>): move builders into builder.go"
```

---

## Task 21: W1 — apply `relocate`, second half

### Files

Remaining `RELOCATE`/`HAS-BUILDER-GO` rows:

- `services/atlas-quest` (3, excluding `entity_builder.go` — see D5)
- `services/atlas-tenants` (2, excluding `entity_builder.go`)
- `services/atlas-channel` (2)
- `libs/atlas-constants` (2)
- `libs/atlas-database` (1)
- `services/atlas-storage`, `services/atlas-saga-orchestrator`, `services/atlas-notes`, `services/atlas-messages`, `services/atlas-marriages`, `services/atlas-inventory`, `services/atlas-families`, `services/atlas-drops`, `services/atlas-doors`, `services/atlas-character-factory`, `services/atlas-character` (1 each)
- `services/atlas-npc-conversations` — all except `conversation/model.go` (Task 23): `conversation/item/model.go` (1), `conversation/npc/model.go` (1), `conversation/quest/model.go` (2), `conversation/recipe/model.go` (1), `saved_location/model.go` (1), `validation/model.go` (1)
- `services/atlas-cashshop` — all except `reference_data.go` (Task 22): 9 of the 16
- `docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-b.tsv` — new file

- [ ] **Step 1: Apply per service**

Same command as Task 20 Step 1, with `-ledger .../ledger-relocate-b.tsv`.

- [ ] **Step 2: Confirm pure relocation**

Same two `git diff -M` checks as Task 20 Step 2, per service.

- [ ] **Step 3: Build and test per module**

```
go build ./... && go vet ./... && go test ./...
```
Expected: PASS.

- [ ] **Step 4: Commit per service**

```bash
git add <path>
git commit -m "refactor(<S>): move builders into builder.go"
```

- [ ] **Step 5: Confirm ledger completeness across both halves**

```
cat docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-a.tsv docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-b.tsv | cut -f2 | sort | uniq -c
```
`APPLIED + SKIPPED` must equal the count of `RELOCATE` + `HAS-BUILDER-GO` rows in `classify-file05.tsv`. Record verbatim in `progress.md`.

- [ ] **Step 6: Commit the ledgers**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-*.tsv \
        docs/tasks/task-263-backend-guideline-conformance/progress.md
git commit -m "chore(task-263): record builder relocation ledger"
```

---

## Task 22: W1 hand work — split `reference_data.go` (FR-11)

Seven reference-data builders sit in a 1038-line file alongside their models. FR-11 requires the builders move to `builder.go` and the models stay.

### Files

- `services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go` — 1038 lines; remove the seven builder declaration sets:
  - `:73` `type EquipableReferenceDataBuilder struct`
  - `:397` `type CashEquipableReferenceDataBuilder struct`
  - `:684` `type ConsumableReferenceDataBuilder struct`
  - `:738` `type SetupReferenceDataBuilder struct`
  - `:785` `type EtcReferenceDataBuilder struct`
  - `:842` `type CashReferenceDataBuilder struct`
  - `:930` `type PetReferenceDataBuilder struct`
- `services/atlas-cashshop/atlas.com/cashshop/asset/builder.go` — new file; receives all seven sets
- `services/atlas-cashshop/atlas.com/cashshop/asset/karma_roundtrip_test.go` — read-only; existing test that must still pass

Module root: `services/atlas-cashshop/atlas.com/cashshop`.

Each declaration set is the `type <X>Builder struct`, its `New<X>Builder` constructor, every method with a `*<X>Builder` receiver, and any `Clone*` returning `*<X>Builder`. Move them verbatim. The `EquipableReferenceData` model and its accessors (`reference_data.go:9-67`) stay.

**FR-15 interaction:** these seven are siblings over distinct types in one package, so per D4 they keep their `New<Type>Builder` names and are recorded as `SIBLING-EXEMPT` in `exemptions.md` (Task 25). Do not rename them.

- [ ] **Step 1: Enumerate each builder's declaration set**

```
grep -n '^type .*ReferenceDataBuilder struct\|^func New.*ReferenceDataBuilder(\|^func (b \*.*ReferenceDataBuilder)' services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go
```
Record the line ranges. Every listed line must end up in `builder.go`.

- [ ] **Step 2: Create `builder.go` and move the sets**

Move verbatim. Recompute the import block of both files from what each actually references; do not copy `reference_data.go`'s import block wholesale.

- [ ] **Step 3: Confirm the move is byte-identical at the declaration level**

```
git diff -M -- services/atlas-cashshop/atlas.com/cashshop/asset | grep '^[+-]' | grep -v '^[+-][+-]' | grep -vE '^\+package |^\+$|^[+-]import|^[+-]\t"|^[+-]\)'
```
Every remaining line must appear once as `-` and once as `+` with identical text.

- [ ] **Step 4: Confirm no builder remains in `reference_data.go`**

```
grep -n 'Builder' services/atlas-cashshop/atlas.com/cashshop/asset/reference_data.go
```
Expected: no output.

- [ ] **Step 5: Build and test**

```
go build ./... && go vet ./... && go test ./...
```
from `services/atlas-cashshop/atlas.com/cashshop`. Expected: PASS, including `karma_roundtrip_test.go`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-cashshop/atlas.com/cashshop/asset
git commit -m "refactor(atlas-cashshop): move reference data builders into builder.go"
```

---

## Task 23: W1 hand work — split `conversation/model.go`'s 20 builders

The single largest FILE-05 concentration: 20 `type <X>Builder struct` declarations in one 2430-line file.

### Files

- `services/atlas-npc-conversations/atlas.com/npc/conversation/model.go` — 2430 lines; remove the 20 builder declaration sets, leave every domain type and accessor
- `services/atlas-npc-conversations/atlas.com/npc/conversation/builder.go` — new file; receives all 20 sets
- `services/atlas-npc-conversations/atlas.com/npc/conversation/model_json.go` — read-only; check it does not reference a moved declaration in a way the import recomputation would break
- `services/atlas-npc-conversations/atlas.com/npc/conversation/context_replacer_test.go`, `firstjob_scripts_test.go` — read-only; existing tests that must still pass

Module root: `services/atlas-npc-conversations/atlas.com/npc`.

Do NOT add `Transform` functions to these 20 types — design §8.2 resolves PRD open question 2 the other way: a builder-backed domain type with no wire representation gets no `Transform`.

**FR-15:** these 20 are siblings over distinct types in one package, so per D4 they keep their `New<Type>Builder` names and are recorded `SIBLING-EXEMPT` in Task 25. Do not rename them.

- [ ] **Step 1: Enumerate the 20 declaration sets**

```
grep -n '^type [A-Za-z0-9_]*Builder struct' services/atlas-npc-conversations/atlas.com/npc/conversation/model.go
```
Expected: 20 lines. For each type `<X>Builder`, collect its constructor, its methods, and any `Clone*` returning `*<X>Builder`:
```
grep -n '^func New[A-Za-z0-9_]*Builder(\|^func (b \*[A-Za-z0-9_]*Builder)\|^func Clone[A-Za-z0-9_]*(' services/atlas-npc-conversations/atlas.com/npc/conversation/model.go
```

- [ ] **Step 2: Create `builder.go` and move all 20 sets**

Move verbatim. Recompute both files' import blocks from actual references.

- [ ] **Step 3: Confirm the move is byte-identical at the declaration level**

```
git diff -M -- services/atlas-npc-conversations/atlas.com/npc/conversation | grep '^[+-]' | grep -v '^[+-][+-]' | grep -vE '^\+package |^\+$|^[+-]import|^[+-]\t"|^[+-]\)'
```
Every remaining line must appear once as `-` and once as `+` with identical text.

- [ ] **Step 4: Confirm no builder remains in `model.go`**

```
grep -c 'Builder' services/atlas-npc-conversations/atlas.com/npc/conversation/model.go
```
Expected: `0`.

- [ ] **Step 5: Build and test**

```
go build ./... && go vet ./... && go test ./...
```
from `services/atlas-npc-conversations/atlas.com/npc`. Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-npc-conversations/atlas.com/npc/conversation
git commit -m "refactor(atlas-npc-conversations): move conversation builders into builder.go"
```

---

## Task 24: W1 — confirm FILE-05 is closed

### Files

- `docs/tasks/task-263-backend-guideline-conformance/inventory-file05-builders.txt` — regenerated
- `docs/tasks/task-263-backend-guideline-conformance/progress.md` — new file at plan time; first written in Task 2, appended here

- [ ] **Step 1: Regenerate the inventory**

```
bash docs/tasks/task-263-backend-guideline-conformance/sweep.sh
cat docs/tasks/task-263-backend-guideline-conformance/inventory-file05-builders.txt
```

- [ ] **Step 2: Confirm the residue is exactly the five documented dispositions**

The only surviving entries must be:

- `libs/atlas-packet/model/skill_usage_info.go:143` — `EXCLUDED-TREE` (FR-18, PRD §7)
- `services/atlas-quest/atlas.com/quest/quest/entity_builder.go:10` — `ENTITY-BUILDER` (D5)
- `services/atlas-quest/atlas.com/quest/quest/progress/entity_builder.go:5` — `ENTITY-BUILDER` (D5)
- `services/atlas-tenants/atlas.com/tenants/configuration/entity_builder.go:9` — `ENTITY-BUILDER` (D5)
- `services/atlas-tenants/atlas.com/tenants/tenant/entity_builder.go:5` — `ENTITY-BUILDER` (D5)

Anything else is a forgotten package. Cross-check against the relocate ledgers before proceeding.

- [ ] **Step 3: Record the evidence**

Paste the verbatim inventory contents into `progress.md` under `## Task 24 — FILE-05 closeout`.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance
git commit -m "chore(task-263): record FILE-05 closeout evidence"
```

---

## Task 25: Write `exemptions.md`

The branch's only durable conformance artifact (D7). Every entry carries `file:line` evidence — acceptance criterion 4 forbids "out of scope" without it.

### Files

- `docs/tasks/task-263-backend-guideline-conformance/exemptions.md` — new file
- `docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv` — new file at plan time; produced by Task 2, read-only source for the `NO-RESTMODEL` section
- `docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-no-model.txt` — read-only source for the no-`Model` section (176 rows)
- `docs/tasks/task-263-backend-guideline-conformance/classify-dom01-fr15.tsv` — new file at plan time; produced by Task 2, read-only source for the sibling-builder and trigger-not-fired sections
- `docs/tasks/task-263-backend-guideline-conformance/classify-file05.tsv` — new file at plan time; produced by Task 2, read-only source for the entity-builder and excluded-tree sections
- `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md` — new file at plan time; accumulated by Tasks 14, 16, 17, read-only here
- `services/atlas-data/atlas.com/data/monster/` — read-only; the no-`Model` evidence cluster (`reader.go` produces `RestModel` directly, `resource.go:163` marshals `server.MarshalResponse[RestModel]`)

### Sections and their sources

| section | count | source | evidence per entry |
|---|---|---|---|
| `## DOM-04 — no domain Model in the package` | 176 | `inventory-dom04-no-model.txt` | path list grouped by service, with **one shared evidence paragraph per service cluster** (FR-5's `atlas-data` write-up is the model) — not 176 individual write-ups |
| `## DOM-04 — no type RestModel in the package` | 14 | `classify-dom04.tsv` `NO-RESTMODEL` rows + `handwork-notes.md` | per package: the `type Model` declaration's `file:line`, each `Extract*`'s `file:line` and signature, the `Transform*` names provided instead, and the disposition sentence |
| `## DOM-01 — sibling builders over distinct types` | 55 | `classify-dom01-fr15.tsv` `SIBLING-EXEMPT` rows | the sibling `New<X>Builder` constructors with `file:line`, per D4 |
| `## DOM-01 — trigger not fired (no model.go)` | 5 | `classify-dom01-fr15.tsv` `NO-MODEL-GO` rows | the package path and the absence of `model.go`; disposition **N/A**, not exempt (`audit-checklist.md:25-44`) |
| `## DOM-01 — NewBuilder name already taken` | 2 | Tasks 4 and 5 | `channel/asset` (`builder.go:119` vs `:126`, arity differs → `NewBuilderWithId`) and `atlas-character/character` (`builder.go:91` `type Builder` → `NewEmptyBuilder`, and `modelBuilder` exempt from FR-13) |
| `## FILE-05 — entity builders` | 4 | `classify-file05.tsv` `ENTITY-BUILDER` rows | `quest/quest/entity_builder.go:10`, `quest/quest/progress/entity_builder.go:5`, `tenants/configuration/entity_builder.go:9`, `tenants/tenant/entity_builder.go:5`; each builds the GORM `entity`, not the domain `Model`, so FILE-06's "genuine single-purpose utility" applies (D5, satisfies FR-10) |
| `## FILE-05 — excluded tree` | 1 | `classify-file05.tsv` `EXCLUDED-TREE` row | `libs/atlas-packet/model/skill_usage_info.go:143` `type SkillUsageInfoBuilder struct`; excluded by FR-18 / PRD §7 |
| `## DOM-04 — lossy Extract` | as found | `handwork-notes.md` (Task 16) | any package where the round trip could not be asserted over every field, with the dropped field's `file:line` |

Entry format, from design §8.1:

```markdown
## DOM-04 — no `type RestModel` in the package

### services/atlas-channel/atlas.com/channel/monsterbook
- `model.go:11` declares `type Model`, but `rest.go` converts `Collection` and `Card`.
- `rest.go:102` `func ExtractCard(rm CardRestModel) (Card, error)`
- `rest.go:107` `func Extract(rm CollectionRestModel) (Collection, error)`
- Provided instead: `TransformCard`, `Transform` (over `Collection`).
- Disposition: **exempt** from the literal `func Transform(` detector; the rule
  at `audit-checklist.md:97` is satisfied by the named variants.
```

Use repo-relative paths throughout; never a literal home or absolute path.

- [ ] **Step 1: Generate the section skeletons from the TSVs**

Derive each section's package list from its source file rather than transcribing from the design's prose — the TSVs are authoritative.

- [ ] **Step 2: Fill in the `file:line` evidence**

For every entry, open the cited file and confirm the line number is correct at HEAD. An entry citing a stale line is worse than no entry.

- [ ] **Step 3: Verify no entry lacks evidence**

Grep the finished `exemptions.md` for the string `out of scope`. Expected: no hit,
or only hits accompanied by a `file:line` on the same entry — acceptance criterion 4
forbids the phrase standing alone.

Then confirm every `### ` heading in the file is followed, before the next `### `,
by at least one backtick-quoted `<path>.go:<line>`. Read the file and check; an
entry with no evidence line is a plan failure, not a formatting nit.

- [ ] **Step 4: Confirm the counts**

Each section's entry count must match the count in the table above (as re-derived, not as the design predicted). Record any divergence in `progress.md`.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/exemptions.md
git commit -m "chore(task-263): record guideline exemptions"
```

---

## Task 26: Verify behavior preservation (FR-17, FR-18)

### Files

- `docs/tasks/task-263-backend-guideline-conformance/progress.md` — new file at plan time; first written in Task 2, appended here

No code changes expected. Any finding here is a defect to fix before proceeding.

- [ ] **Step 1: FR-18 — excluded trees**

```
git diff --name-only main...HEAD | grep -E '\.sql$|docker-compose.*\.ya?ml$|\.tpl$|^libs/atlas-packet/'
```
Expected: no output. Any hit must be reverted.

- [ ] **Step 2: FR-17 — no JSON tag changed**

```
git diff -G'json:"' main...HEAD
```
Expected: only additions of whole new declarations (a generated `Transform` does not carry tags, so realistically no output). Read every hunk; a modified tag line is a defect.

- [ ] **Step 3: FR-17 — no `GetName()` return value changed**

```
git diff -U0 main...HEAD | grep -E '^[+-].*GetName\(\) string'
```
Every `-` line must have a matching `+` line with the identical return literal (a relocation), or there must be no output at all.

- [ ] **Step 4: FR-17 — no route registration or Kafka topic changed**

```
git diff -U0 main...HEAD -- '*/resource.go' '*/producer.go' '*/consumer.go' | grep '^[+-]' | grep -v '^[+-][+-]'
```
Expected: only builder relocations from W1. Read every hunk.

- [ ] **Step 5: FR-17 — no `Build()` validation rule changed**

```
git diff -U0 main...HEAD | grep -E '^[+-].*errors\.New\(|^[+-].*fmt\.Errorf\('
```
Every `-` line must have a matching `+` line with the identical message (a relocation). A changed or removed validation error is a defect.

- [ ] **Step 6: FR-17 — no struct field type changed**

```
git diff -U0 main...HEAD -- '*/model.go' '*/entity.go' | grep '^-' | grep -v '^---'
```
Every deletion must be a builder declaration moved to `builder.go` by W1. A deleted `Model` field is a defect.

- [ ] **Step 7: Record the evidence**

Paste each command and its verbatim output into `progress.md` under `## Task 26 — FR-17/FR-18 verification`.

- [ ] **Step 8: Commit**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/progress.md
git commit -m "chore(task-263): record behavior-preservation verification"
```

---

## Task 27: Final gate, scaffolding removal, and review

### Files

- `docs/tasks/task-263-backend-guideline-conformance/progress.md` — new file at plan time; first written in Task 2, appended here before the scaffolding is deleted
- `docs/tasks/task-263-backend-guideline-conformance/sweep.sh` — deleted (D7)
- `docs/tasks/task-263-backend-guideline-conformance/split-by-model.sh` — deleted (D7)
- `docs/tasks/task-263-backend-guideline-conformance/codemod/` — new file at plan time; created by Task 1, deleted here (D6, D7)
- `docs/tasks/task-263-backend-guideline-conformance/inventory-*.txt` — deleted (D7)
- `docs/tasks/task-263-backend-guideline-conformance/classify-*.tsv`, `ledger-*.tsv`, `handwork-*.tsv`, `handwork-notes.md`, `fr15-targets.tsv` — deleted (D7)
- `docs/tasks/task-263-backend-guideline-conformance/exemptions.md` — new file at plan time; created by Task 25, **kept**; the branch's durable artifact
- `docs/tasks/task-263-backend-guideline-conformance/prd.md`, `design.md`, `plan.md`, `context.md`, `progress.md` — kept
- `tools/verify.sh` — read-only; not touched by this task (D7)

- [ ] **Step 1: Run the final sweep and record its verbatim output**

```
bash docs/tasks/task-263-backend-guideline-conformance/sweep.sh
bash docs/tasks/task-263-backend-guideline-conformance/split-by-model.sh
```

Paste both scripts' complete `wc -l` output into `progress.md` under `## Task 27 — final sweep evidence`. Per design §10 this run **is** acceptance criteria 1–3; the counts required are:

- `inventory-dom01-newmodelbuilder.txt`: **0**
- `inventory-file05-builders.txt`: **5**, and every one listed in `exemptions.md` (the 4 entity builders + `libs/atlas-packet`)
- `inventory-dom04-has-model.txt`: only packages listed in `exemptions.md`

If any count is wrong, stop — the row is not closed and the scaffolding must not be deleted.

- [ ] **Step 2: Commit the evidence before deleting anything**

```bash
git add docs/tasks/task-263-backend-guideline-conformance/progress.md \
        docs/tasks/task-263-backend-guideline-conformance/inventory-*.txt
git commit -m "chore(task-263): record final conformance sweep evidence"
```

- [ ] **Step 3: Run the flagless gate**

```
tools/verify.sh
```

Not `--quick`, not `--no-docker` — those skip the bake and `-race` and do not count (CLAUDE.md). Dispatch `atlas-verifier` for this rather than running it in a large context. Expected: exit 0. On failure, read `docs/verification.md` and fix; do not proceed.

- [ ] **Step 4: Code review**

Dispatch `backend-guidelines-reviewer` over the changed Go packages (`model: sonnet`) and `atlas-reviewer` over the commit range (`model: sonnet`), per `docs/review-protocol.md`. Acceptance criterion 7 requires DOM-01, DOM-04 and FILE-05 to come back PASS or N/A with no FAIL outside `exemptions.md`.

**Expect R3 to fire:** the reviewer follows the literal `func Transform(` grep and will re-raise DOM-04 on the 14 `NO-RESTMODEL` packages. That is the accepted, documented residue — point the reviewer at `exemptions.md`. Any *other* FAIL is real.

- [ ] **Step 5: Delete the scaffolding**

```bash
git rm -r docs/tasks/task-263-backend-guideline-conformance/codemod
git rm docs/tasks/task-263-backend-guideline-conformance/sweep.sh \
       docs/tasks/task-263-backend-guideline-conformance/split-by-model.sh
git rm docs/tasks/task-263-backend-guideline-conformance/inventory-*.txt \
       docs/tasks/task-263-backend-guideline-conformance/classify-*.tsv \
       docs/tasks/task-263-backend-guideline-conformance/ledger-*.tsv \
       docs/tasks/task-263-backend-guideline-conformance/handwork-dom04.tsv \
       docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md \
       docs/tasks/task-263-backend-guideline-conformance/fr15-targets.tsv
```

- [ ] **Step 6: Confirm nothing durable was deleted**

```
ls docs/tasks/task-263-backend-guideline-conformance/
```
Expected exactly: `context.md`, `design.md`, `exemptions.md`, `plan.md`, `prd.md`, `progress.md`.

- [ ] **Step 7: Confirm the gate still passes after the deletion**

```
git status --short
```
Expected: clean after commit. Then re-run `tools/verify.sh` (via `atlas-verifier`) — the deletion removes only files outside `services/` and `libs/`, so it must still exit 0.

- [ ] **Step 8: Commit**

```bash
git commit -m "chore(task-263): remove conformance scaffolding"
```

- [ ] **Step 9: Confirm the branch and worktree**

```
git rev-parse --show-toplevel
git branch --show-current
```
Expected: a path ending in `/.worktrees/task-263-backend-guideline-conformance`, and branch `task-263-backend-guideline-conformance`. If either is wrong, STOP and report BLOCKED.
