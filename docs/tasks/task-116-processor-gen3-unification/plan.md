# Processor Gen3 Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Converge every non-mock processor package under `services/` onto the Gen3 idiom (`Processor` interface + `ProcessorImpl` struct + `NewProcessor(...) Processor`), add a `mock/` package per converted package, and fix all CP-2 concrete-return signatures — strictly behavior-preserving.

**Architecture:** Recipe-driven mechanical refactor. Four conversion recipes (R1–R4) plus a mock template (R5) and a non-processor file-rename rule (R6) are defined once in this document; each task applies them to an exact file list, one service visit per task (per design §5), with green verification between commits. No logic moves; only declarations change.

**Tech Stack:** Go 1.x workspaces (`go.work`), logrus, GORM, `github.com/Chronicle20/atlas/libs/*`, docker buildx bake.

**Spec:** `docs/tasks/task-116-processor-gen3-unification/design.md` (PRD: `prd.md` in the same folder).

## Global Constraints

Copied from the design/PRD — every task's requirements implicitly include all of these:

1. **Behavior preservation is absolute.** No logic changes, no bug fixes (report pre-existing bugs in `inventory.md` under "Deferred findings", do not fix), no Kafka/REST/packet changes, no `libs/` changes. Diff confined to `services/` + `docs/tasks/task-116-processor-gen3-unification/`.
2. **Capture semantics are preserved, not normalized** (design §2.4). A package that resolves `tenant.MustFromContext(ctx)` (or constructs `producer.ProviderImpl(l)(ctx)`, or collaborator processors) per-call keeps per-call acquisition; construction-time capture is kept only where it already exists. Moving an acquisition point changes failure timing and is forbidden.
3. **Currying inside method bodies/signatures is preserved exactly** (design §2.5, §7.6). Buffered logic stays `Method(mb *message.Buffer) func(...)...`, side-effecting stays `MethodAndEmit(...)`, providers stay `XxxProvider(...) model.Provider[T]`. Only the outer `F(l)(ctx)` currying levels of Gen1 functions collapse into struct fields.
4. **Interface = complete exported receiver-method set** (design §2.1). Exported package-level functions *without* a receiver (`Extract`, builders, event providers, registry accessors used only internally) stay package functions.
5. **`NewProcessor` returns `Processor`, never `*ProcessorImpl`.** Extra deps follow `l, ctx` positionally: `NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor`.
6. **Conformance assertion `var _ Processor = (*ProcessorImpl)(nil)` lives in `processor.go`** (not a test file). Exception: the three existing assertions in `atlas-messages` `*_test.go` files stay where they are.
7. **Every converted package gains `mock/processor.go`** per recipe R5; existing mocks are kept (not rewritten to func-field shape), updated for interface changes, and gain `var _ <pkg>.Processor = (*ProcessorMock)(nil)`.
8. **Test files update in the same commit as their package** (they reference internal names; renames break them).
9. **Test setup uses the project's Builder pattern; no `*_testhelpers.go` files with test-only constructors.**
10. **No `// TODO`, stubs, or 501s in landed commits.**
11. **Verification cadence:** per commit — `go test -race ./...`, `go vet ./...`, `go build ./...` clean in the touched module. Per service visit (after its last commit) — `docker buildx bake atlas-<svc>` from the worktree root. Per phase end and at branch end — `tools/redis-key-guard.sh` from the repo root plus full inventory re-scan.
12. **Refactor test discipline (in place of test-first):** before converting a module, run its full test suite and record green; after converting, the same suite must be green with zero test-logic changes (mechanical rename/call-shape updates only). Characterization tests (where required, see R7) are written and committed green *before* the conversion they protect.
13. **Commit granularity:** one commit per service visit, except atlas-channel (one commit per package group, Tasks 19–23) and atlas-data (one commit per package, Tasks 32–35).
14. **All file paths in commits/docs are repo-relative.** Never write absolute home paths into committed files.
15. **`inventory.md` is the ground truth** (FR-8). Task 1 creates it; every task updates its rows; acceptance (Task 36) is evaluated against the final scan, not the counts in the PRD/design/this plan.

---

## Ground-truth inventory (point-in-time)

Scanned at plan time on branch `task-116-processor-gen3-unification` (fde55e232): 377 non-mock `processor.go` files. The live scan **supersedes the PRD §7 table** — it revealed work in services the PRD never listed (atlas-account, atlas-monsters, atlas-rates, atlas-messages, atlas-gachapons, atlas-npc-shops, atlas-character-factory, atlas-reactor-actions, atlas-channel `server`, atlas-messengers `invite`). Design §5's rule ("the FR-8 scan, not the prose, is authoritative"; FR-4 final paragraph: newly revealed files convert under whichever recipe matches) covers all of these. Task 1 re-runs the scan and commits it as `inventory.md`; the per-task file lists below are the plan-time snapshot.

**Classification counts (plan-time):** CP-2 concrete-return 20 files (18 with an existing interface = recipe R1; 2 atlas-messengers files have no interface and fold into R2); Gen2 `type Processor struct` 58 files (R2); `ProcessorImpl`-without-interface ("Gen2.5") 5 files (R2 minus the rename step); Gen1 no-type 50 files, of which 45 convert via R3, 2 are ctx-per-call REST clients (R4), and 3 are not processors at all (R6 file renames).

---

## Recipes

### R1 — CP-2 signature fix (interface exists, constructor returns `*ProcessorImpl`)

Worked example: `services/atlas-pets/atlas.com/pets/pet/processor.go`.

Before (`processor.go:96`):

```go
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *ProcessorImpl {
	p := &ProcessorImpl{ ... }
	p.Despawner = p.defaultDespawn
	return p
}
```

After — only the return type changes; the body still builds and returns `*ProcessorImpl` (it satisfies the interface):

```go
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{ ... }
	p.Despawner = p.defaultDespawn
	return p
}
```

Steps per file:

1. Change the `NewProcessor` return type to `Processor`.
2. **Option-pattern wrinkle:** if the interface (or impl) has a method returning the concrete type — e.g. atlas-pets `With(opts ...ProcessorOption) *ProcessorImpl` at `processor.go:39` — change its return type to `Processor` in both the interface and the impl. `ProcessorOption func(*ProcessorImpl)` and the `WithXxx` option constructors are unchanged (they configure the impl, they don't leak it to callers).
3. If any caller uses a method on the concrete type that is missing from the interface, add that method to the interface (declaration change only — the method was already de-facto public).
4. Update every caller that declares a var/struct field/param as `*<pkg>.ProcessorImpl` → `<pkg>.Processor`. Find them: `grep -rn "\*<pkg-import-name>\.ProcessorImpl" <module-dir> --include="*.go"`.
5. Add `var _ Processor = (*ProcessorImpl)(nil)` to `processor.go` if absent.
6. Mock: if `<pkg>/mock/processor.go` exists, update it for any interface change; otherwise create it per R5.

### R2 — Gen2 → Gen3 (concrete `Processor` struct, no interface)

Worked example: `services/atlas-channel/atlas.com/channel/macro/processor.go`.

Before (complete file body, imports elided):

```go
type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
}

func (p *Processor) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *Processor) Update(characterId uint32, macros []Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(macro2.EnvCommandTopic)(UpdateCommandProvider(characterId, macros))
}
```

After (same imports; method bodies byte-identical apart from the receiver type):

```go
type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	Update(characterId uint32, macros []Model) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) Update(characterId uint32, macros []Model) error {
	return producer.ProviderImpl(p.l)(p.ctx)(macro2.EnvCommandTopic)(UpdateCommandProvider(characterId, macros))
}
```

Steps per file:

1. Rename `type Processor struct` → `type ProcessorImpl struct`; all receivers follow (`(p *Processor)` → `(p *ProcessorImpl)`). **Gen2.5 files** (struct already named `ProcessorImpl`: atlas-messengers `character`/`messenger`, atlas-map-actions `script`, atlas-portal-actions `script`, atlas-reactor-actions `script`) skip this step.
2. Write `type Processor interface` containing every **exported** method with the (now-)`ProcessorImpl` receiver — exact signatures, in source order. Unexported receiver methods stay out of the interface.
3. Change `NewProcessor(...) *Processor` / `*ProcessorImpl` → `NewProcessor(...) Processor`. Same for any secondary constructors (`NewProcessorWith...`) if present.
4. Add `var _ Processor = (*ProcessorImpl)(nil)` after `NewProcessor`.
5. Update call sites in the module: vars/fields/params typed `*<pkg>.Processor` or `*<pkg>.ProcessorImpl` → `<pkg>.Processor`. Test files constructing `&Processor{...}` literals switch to `NewProcessor(...)`. Find them: `grep -rn "\*<pkg-import-name>\.Processor" <module-dir> --include="*.go"` (matches both old names).
6. Create `mock/processor.go` per R5.

### R3 — Gen1 → Gen3 (curried package functions, no type)

Worked example: `services/atlas-rates/atlas.com/rates/buffs/processor.go`.

Before (complete file):

```go
package buffs

import (
	"context"

	"github.com/sirupsen/logrus"
)

// GetActiveBuffs retrieves all active buffs for a character from atlas-buffs
func GetActiveBuffs(l logrus.FieldLogger) func(ctx context.Context) func(characterId uint32) ([]RestModel, error) {
	return func(ctx context.Context) func(characterId uint32) ([]RestModel, error) {
		return func(characterId uint32) ([]RestModel, error) {
			return requestBuffs(characterId)(l, ctx)
		}
	}
}
```

After (complete file):

```go
package buffs

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetActiveBuffs(characterId uint32) ([]RestModel, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetActiveBuffs retrieves all active buffs for a character from atlas-buffs
func (p *ProcessorImpl) GetActiveBuffs(characterId uint32) ([]RestModel, error) {
	return requestBuffs(characterId)(p.l, p.ctx)
}
```

Call-site change (e.g. `services/atlas-rates/atlas.com/rates/character/initializer.go:198`):

```go
// before
activeBuffs, err := buffs.GetActiveBuffs(l)(ctx)(characterId)
// after
activeBuffs, err := buffs.NewProcessor(l, ctx).GetActiveBuffs(characterId)
```

Steps per file:

1. Introduce `Processor` interface + `ProcessorImpl` struct + `NewProcessor(l, ctx /*, db where any converted function curries one in */) Processor` + conformance assertion.
2. Each **exported** curried function `F(l)(ctx)(args...) R` becomes interface method + impl method `F(args...) R`; only the `l`/`ctx` (and `db`) currying levels collapse — inner currying beyond those levels is preserved verbatim (Constraint 3). Non-curried Gen1 functions taking `(l, ctx, args...)` (e.g. atlas-account `ban.CheckBan`) become `F(args...)` the same way.
3. **Private** curried helpers may become unexported methods or stay package functions — bias toward methods when they use `l`/`ctx`; either way they never enter the interface.
4. Package-level registry/storage singletons (`GetModelRegistry()`, `NewStorage`) stay package-private helpers.
5. Update every call site: REST handlers, Kafka consumers, tickers, `main.go` construct `NewProcessor(l, ctx, ...)` at the top of the handler function (matching existing Gen3 wiring) and call methods. Find call sites: `grep -rn "<pkg-import-name>\.<ExportedFunc>" <module-dir> --include="*.go"` for each exported function.
6. **Higher-order call sites use method values** (design §3.3.4): `registerAllInDirectory(l, ctx, dir, npc.RegisterNpc(db))` becomes `registerAllInDirectory(l, ctx, dir, npc.NewProcessor(l, ctx, db).RegisterNpc)` — the passed value's type simplifies from `func(l)(ctx)(path) error`-curried to `func(path string) error`, and the higher-order function's parameter type simplifies to match. No logic moves.
7. Create `mock/processor.go` per R5.

### R4 — Rename-only conversion for ctx-per-call REST clients

Applies to exactly two packages (both are startup-wired REST clients whose methods take `ctx` per call; full Gen3 would change their lifecycle/failure timing — design §4.2):

- `services/atlas-configurations/atlas.com/configurations/data/processor.go` (`Client`/`ClientImpl`/`NewClient(l) *ClientImpl`, stateful map-based `FakeClient` mock)
- `services/atlas-character-factory/atlas.com/character-factory/data/processor.go` (same shape: `Client` interface at line 26, `ClientImpl`, `NewClient(l) *ClientImpl` at line 35)

Steps:

1. Rename `Client` → `Processor`, `ClientImpl` → `ProcessorImpl`, `NewClient` → `NewProcessor`; the constructor keeps its `(l logrus.FieldLogger)` parameter list (no `ctx` — sanctioned deviation) but the return type becomes `Processor`.
2. Methods keep their per-call `ctx context.Context` first parameter — unchanged bodies.
3. Add `var _ Processor = (*ProcessorImpl)(nil)`.
4. Existing fake (`FakeClient` in atlas-configurations) renames to `ProcessorMock`, **keeps its map-based design**, gains `var _ data.Processor = (*ProcessorMock)(nil)`. If atlas-character-factory has no mock, create one per R5 (func-field, methods taking `ctx`).
5. Update call sites (`grep -rn "data\.NewClient\|data\.Client\b\|data\.ClientImpl" <module-dir> --include="*.go"`; known: `atlas-configurations` `tenants/resource.go:87`, `templates/resource.go:122`).
6. Document both packages in `inventory.md` under "Sanctioned shape deviations": long-lived, wired-at-startup processor; ctx-per-call methods; `NewProcessor(l)` signature.

### R5 — Mock template

Exemplar: `services/atlas-notes/atlas.com/notes/note/mock/processor.go`. For a package `<pkg>` with import path `<module>/<pkg>`:

```go
package mock

import (
	"<module>/<pkg>"
)

type ProcessorMock struct {
	// one field per interface method, identical signature:
	GetByCharacterIdFunc func(characterId uint32) ([]<pkg>.Model, error)
	// ...
}

var _ <pkg>.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetByCharacterId(characterId uint32) ([]<pkg>.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return nil, nil
}
```

Rules:

- One `XxxFunc` field per interface method; each method delegates when the field is non-nil, else returns zero values.
- **Curried interface methods return fully-nested zero-value closures** when the func field is nil — copy the shape from `atlas-notes/note/mock/processor.go:26-39` (`Create` returns `func(uint32) func(...) ... { return note.Model{}, nil }` nested to full depth).
- `model.Provider[T]`-returning methods return `func() (T, error) { return T-zero, nil }` (i.e. `model.FixedProvider(T{})` shape — match whatever the nearest existing mock in the same service does; if none, use a literal closure returning the zero Model and nil).
- Place the conformance assertion in the mock file itself.

### R6 — Non-processor `processor.go` files: rename the file

Three plan-time-scanned `processor.go` files are not the Processor idiom at all. Consistent with design §7.5's "an exemption leaves a permanent asterisk" principle, rename the file (`git mv`, content unchanged) instead of exempting it:

| File | Content | New name |
|---|---|---|
| `services/atlas-messages/atlas.com/messages/command/processor.go` | two exported func-type decls (`Producer`, `Executor`), no functions | `types.go` |
| `services/atlas-gachapons/atlas.com/gachapons/test/processor.go` | test-fixture factories `CreateXxxProcessor(t *testing.T)` returning *other* packages' Processors | `fixtures.go` |
| `services/atlas-npc-shops/atlas.com/npc/test/processor.go` | same fixture-factory shape | `fixtures.go` |

Each rename is recorded in `inventory.md` with its justification. (The `test/` fixture packages are pre-existing; this task does not convert or delete them — that would be a behavior/test-infra change.)

### R7 — Characterization-test classification

During each visit, classify each converted package in `inventory.md`:

- **delegation shim** — exported surface is one-line registry/requester/producer delegation → no new tests required.
- **logic-bearing** — branching, math, state transitions, emission batching. If no existing test exercises that logic, write characterization tests against pre-conversion behavior **first** (commit them green before or with the conversion); they must pass unchanged post-conversion. Target the pure functions directly (they are package-internal, so the tests live in the same package). Known plan-time candidates are handled concretely in Task 31; any additional logic-bearing/untested packages discovered during a visit get the same treatment and an `inventory.md` row.

---

## Verification protocol (referenced by every task)

**V-commit** (before every commit, from the module dir — the directory containing the service's `go.mod`, e.g. `services/atlas-doors/atlas.com/doors`):

```bash
go build ./... && go vet ./... && go test -race ./...
```

Expected: zero errors, all tests pass.

**V-visit** (after a service's last commit, from the worktree root):

```bash
docker buildx bake atlas-<svc>
```

Expected: image builds successfully. (No `go.mod` changes anywhere in this task, but the bake step stays mandatory per CLAUDE.md / design §6.)

**V-phase** (at the end of phases A, B, C — from the worktree root):

```bash
tools/redis-key-guard.sh
```

Expected: clean. Then re-run the Task 1 inventory scan and update `inventory.md` counts; commit the doc update (`docs(task-116): inventory after phase <X>`).

**Acceptance greps** (Task 36, from the worktree root):

```bash
grep -rn "func NewProcessor(" services/ --include="*.go" | grep -v "_test.go" | grep -v "/mock/" | grep "\*ProcessorImpl"   # expect: no output
grep -rln "type Processor struct" services/ --include="*.go" | grep -v mock                                                  # expect: no output
```

---

## Phase 0 — Inventory

### Task 1: FR-8 inventory scan → `inventory.md`

**Files:**
- Create: `docs/tasks/task-116-processor-gen3-unification/inventory.md`

**Interfaces:**
- Produces: the ground-truth classification table every later task updates; the scan script (embedded in the doc) that Tasks 36 and each V-phase re-run.

- [ ] **Step 1: Run the classification scan** (from the worktree root):

```bash
# all non-mock processor.go files
find services -name "processor.go" -not -path "*/mock/*" | sort > /tmp/all.txt
# CP-2: NewProcessor returning *ProcessorImpl
grep -rn "func NewProcessor(" services/ --include="*.go" | grep -v "_test.go" | grep -v "/mock/" | grep "\*ProcessorImpl" | sed 's/:.*//' | sort
# Gen2: concrete Processor struct
grep -rln "type Processor struct" services/ --include="*.go" | grep -v mock | sort
# Gen2.5: ProcessorImpl without an interface anywhere in the package
for f in $(grep -rln "type ProcessorImpl struct" services/ --include="*.go" | grep -v mock); do d=$(dirname $f); grep -q "type Processor interface" $d/*.go 2>/dev/null || echo "$f"; done
# Gen1: processor.go with no Processor type at all in the file, and none in the package
for f in $(cat /tmp/all.txt); do d=$(dirname $f); grep -q "type Processor interface\|type Processor struct\|type ProcessorImpl struct" $d/*.go 2>/dev/null || echo "$f"; done
```

- [ ] **Step 2: Write `inventory.md`** with: (a) the scan script verbatim (so it is re-runnable); (b) one table row per non-Gen3-conforming file: `path | classification (Gen1/Gen2/Gen2.5/CP-2/R4-client/R6-rename) | recipe | task # | status (pending)`; (c) empty sections titled "Sanctioned shape deviations", "Characterization tests", "Deferred findings", "R6 file renames". Use the plan-time classification in this document as the expected result; investigate and re-classify any drift (main may have moved).

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-116-processor-gen3-unification/inventory.md
git commit -m "docs(task-116): FR-8 processor inventory baseline"
```

---

## Phase A — CP-2 signature fixes (recipe R1 throughout)

Every Phase A task follows the same step skeleton; it is written out fully in Task 2 and the later tasks list only their specifics (files, wrinkles, commit message). The skeleton *is* the task — do every step.

### Task 2: atlas-doors — 4 signature fixes

**Files:**
- Modify: `services/atlas-doors/atlas.com/doors/party/processor.go`, `services/atlas-doors/atlas.com/doors/door/processor.go`, `services/atlas-doors/atlas.com/doors/data/skill/processor.go`, `services/atlas-doors/atlas.com/doors/data/map/processor.go` (+ any `*ProcessorImpl`-typed callers found in Step 2)
- Create: `party/mock/processor.go`, `door/mock/processor.go`, `data/skill/mock/processor.go`, `data/map/mock/processor.go` (under the same module dir)

**Interfaces:**
- Produces: `party.NewProcessor(...) party.Processor` (and likewise for `door`, `data/skill`, `data/map`); `mock.ProcessorMock` per package.

- [ ] **Step 1: Record green baseline.** From `services/atlas-doors/atlas.com/doors`: run **V-commit**. Must pass before any edit (if it doesn't, STOP and report — the branch is broken upstream of this task).
- [ ] **Step 2: Enumerate concrete-type callers** for each package:

```bash
grep -rn "ProcessorImpl" services/atlas-doors/atlas.com/doors --include="*.go" | grep -v "/mock/" | grep -v "processor.go:"
```

- [ ] **Step 3: Apply R1** to each of the four `processor.go` files (return type → `Processor`; option-method returns → `Processor` if present; interface extended only if Step 2 revealed de-facto-public methods; assertion added if absent; callers from Step 2 retyped).
- [ ] **Step 4: Create the four mocks** per R5 (none of these packages has an existing mock).
- [ ] **Step 5: Run V-commit.** Fix any compile fallout (typically test files typed to the concrete type — update them in this commit).
- [ ] **Step 6: Run V-visit:** `docker buildx bake atlas-doors`.
- [ ] **Step 7: Update `inventory.md`** rows (status → done) — include in the same commit.
- [ ] **Step 8: Commit**

```bash
git add -A services/atlas-doors docs/tasks/task-116-processor-gen3-unification/inventory.md
git commit -m "refactor(atlas-doors): NewProcessor returns Processor interface; add mocks (CP-2)"
```

### Task 3: atlas-summons — 3 signature fixes

Same skeleton as Task 2. Module dir: `services/atlas-summons/atlas.com/summons`. Bake target: `atlas-summons`.

**Files:**
- Modify: `data/skill/processor.go`, `effectivestats/processor.go`, `inventory/processor.go` (+ callers)
- Create: `data/skill/mock/processor.go`, `effectivestats/mock/processor.go`, `inventory/mock/processor.go`

- [ ] Steps 1–8 as in Task 2. Commit: `refactor(atlas-summons): NewProcessor returns Processor interface; add mocks (CP-2)`

### Task 4: atlas-saga-orchestrator — 1 signature fix (existing mock)

Same skeleton. Module dir: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`. Bake target: `atlas-saga-orchestrator`.

**Files:**
- Modify: `validation/processor.go` (+ callers), `validation/mock/processor.go` (**existing mock** — audit it against the interface; update only if the interface is extended; add the conformance assertion if missing)

- [ ] Steps 1–8 as in Task 2 (Step 4 = audit existing mock instead of create). Commit: `refactor(atlas-saga-orchestrator): NewProcessor returns Processor interface (CP-2)`

### Task 5: atlas-pets — 1 signature fix (option-pattern wrinkle)

Same skeleton. Module dir: `services/atlas-pets/atlas.com/pets`. Bake target: `atlas-pets`.

**Files:**
- Modify: `pet/processor.go` (+ callers)
- Create: `pet/mock/processor.go`

Specifics: this is the R1 worked example. `Processor` interface at `pet/processor.go:38` declares `With(opts ...ProcessorOption) *ProcessorImpl` — change to `With(opts ...ProcessorOption) Processor` in interface and impl (R1 Step 2). `NewProcessor` at line 96 assigns `p.Despawner = p.defaultDespawn` before returning — body unchanged. The `pet` package is large; the mock will have many func fields — generate it methodically from the interface, top to bottom.

- [ ] Steps 1–8 as in Task 2. Commit: `refactor(atlas-pets): NewProcessor returns Processor interface; add mock (CP-2)`

### Task 6: atlas-npc-conversations — 1 signature fix (existing mock)

Same skeleton. Module dir: `services/atlas-npc-conversations/atlas.com/npc`. Bake target: `atlas-npc-conversations`.

**Files:**
- Modify: `validation/processor.go` (+ callers), `validation/mock/processor.go` (existing — audit as in Task 4)

- [ ] Steps 1–8 as in Task 2. Commit: `refactor(atlas-npc-conversations): NewProcessor returns Processor interface (CP-2)`

### Task 7: atlas-mounts — 1 signature fix

Same skeleton. Module dir: `services/atlas-mounts/atlas.com/mounts`. Bake target: `atlas-mounts`.

**Files:**
- Modify: `mount/processor.go` (+ callers)
- Create: `mount/mock/processor.go`

- [ ] Steps 1–8 as in Task 2. Commit: `refactor(atlas-mounts): NewProcessor returns Processor interface; add mock (CP-2)`
- [ ] **Phase A close-out:** run **V-phase** (redis-key-guard + inventory re-scan + `docs(task-116): inventory after phase A` commit).

---

## Phase B — Gen2/Gen2.5 extraction (+ co-located CP-2 and R4 renames)

Phase B tasks reuse the Task 2 step skeleton with R2 (and R1/R4 where noted) in Step 3, and Step 2's caller grep from R2 Step 5. Each visit does ALL of its service's work (design §5).

### Task 8: atlas-chairs — 1 Gen2 extraction

Module dir: `services/atlas-chairs/atlas.com/chairs`. Bake: `atlas-chairs`.

**Files:**
- Modify: `validation/processor.go` (+ callers)
- Create: `validation/mock/processor.go`

- [ ] Steps 1–8 (R2). Commit: `refactor(atlas-chairs): extract validation Processor interface (Gen2→Gen3)`

### Task 9: atlas-storage — 2 Gen2 extractions

Module dir: `services/atlas-storage/atlas.com/storage`. Bake: `atlas-storage`.

**Files:**
- Modify: `asset/processor.go`, `storage/processor.go` (+ callers; note `storage` and `asset` may reference each other — convert both, then fix call sites once)
- Create: `asset/mock/processor.go`, `storage/mock/processor.go`

- [ ] Steps 1–8 (R2). Commit: `refactor(atlas-storage): extract Processor interfaces (Gen2→Gen3)`

### Task 10: atlas-map-actions — 1 CP-2 fix + 1 Gen2.5 extraction

Module dir: `services/atlas-map-actions/atlas.com/map-actions`. Bake: `atlas-map-actions`.

**Files:**
- Modify: `validation/processor.go` (R1), `script/processor.go` (R2, skip rename — struct already `ProcessorImpl`) (+ callers)
- Create: `validation/mock/processor.go` (if absent), `script/mock/processor.go`

- [ ] Steps 1–8. Commit: `refactor(atlas-map-actions): Gen3 processor conformance for validation and script`

### Task 11: atlas-portal-actions — 1 CP-2 fix + 1 Gen2.5 extraction

Module dir: `services/atlas-portal-actions/atlas.com/portal`. Bake: `atlas-portal-actions`.

**Files:**
- Modify: `validation/processor.go` (R1), `script/processor.go` (R2, skip rename) (+ callers)
- Create: `validation/mock/processor.go` (if absent), `script/mock/processor.go`

- [ ] Steps 1–8. Commit: `refactor(atlas-portal-actions): Gen3 processor conformance for validation and script`

### Task 12: atlas-reactor-actions — 1 Gen2.5 extraction (service not in PRD; revealed by scan)

Module dir: `services/atlas-reactor-actions/atlas.com/reactor`. Bake: `atlas-reactor-actions`.

**Files:**
- Modify: `script/processor.go` (R2, skip rename) (+ callers)
- Create: `script/mock/processor.go`

- [ ] Steps 1–8. Commit: `refactor(atlas-reactor-actions): extract script Processor interface (Gen2.5→Gen3)`

### Task 13: atlas-messengers — 2 Gen2.5 extractions + 1 Gen1 conversion

Module dir: `services/atlas-messengers/atlas.com/messengers`. Bake: `atlas-messengers`.

**Files:**
- Modify: `character/processor.go` (R2 skip-rename; its `NewProcessor` currently returns `*ProcessorImpl` — fixed by R2 Step 3), `messenger/processor.go` (same), `invite/processor.go` (R3: single exported curried func `Create(l)(ctx)(transactionID, actorId, worldId, messengerId, targetId) error` → method `Create(transactionID uuid.UUID, actorId uint32, worldId world.Id, messengerId uint32, targetId uint32) error`) (+ callers)
- Create: `character/mock/processor.go`, `messenger/mock/processor.go`, `invite/mock/processor.go`

- [ ] Steps 1–8. Commit: `refactor(atlas-messengers): Gen3 processor conformance (character, messenger, invite)`

### Task 14: atlas-configurations — 3 Gen2 extractions + R4 rename

Module dir: `services/atlas-configurations/atlas.com/configurations`. Bake: `atlas-configurations`.

**Files:**
- Modify: `services/processor.go`, `templates/processor.go`, `tenants/processor.go` (all R2), `data/processor.go` (R4; known call sites `tenants/resource.go:87`, `templates/resource.go:122`), `data/mock/processor.go` (rename `FakeClient` → `ProcessorMock`, keep map-based design, add assertion; update its tests same-commit)
- Create: `services/mock/processor.go`, `templates/mock/processor.go`, `tenants/mock/processor.go`

- [ ] Steps 1–8 (R2 ×3, R4 ×1; record the R4 deviation in `inventory.md` "Sanctioned shape deviations"). Commit: `refactor(atlas-configurations): Gen3 processor conformance; rename data Client to Processor`

### Task 15: atlas-character-factory — R4 rename

Module dir: `services/atlas-character-factory/atlas.com/character-factory`. Bake: `atlas-character-factory`.

**Files:**
- Modify: `data/processor.go` (R4: `Client`→`Processor`, `ClientImpl`→`ProcessorImpl`, `NewClient(l) *ClientImpl`→`NewProcessor(l) Processor`; methods `GetSkillsByIds(ctx, ids)` / `GetItemById(ctx, id)` keep their `ctx` param) (+ callers via `grep -rn "data\.NewClient\|data\.Client" services/atlas-character-factory/atlas.com/character-factory --include="*.go"`)
- Create: `data/mock/processor.go` (R5, methods take `ctx`), if no existing fake; otherwise rename/audit the existing one per R4 Step 4.

- [ ] Steps 1–8 (record the R4 deviation in `inventory.md`). Commit: `refactor(atlas-character-factory): rename data Client to Processor (Gen3 naming)`

### Task 16: atlas-login — 1 CP-2 fix + 1 Gen2 extraction

Module dir: `services/atlas-login/atlas.com/login`. Bake: `atlas-login`.

**Files:**
- Modify: `inventory/processor.go` (R1), `guild/processor.go` (R2) (+ callers)
- Create: `inventory/mock/processor.go` (if absent), `guild/mock/processor.go`

- [ ] Steps 1–8. Commit: `refactor(atlas-login): Gen3 processor conformance (inventory, guild)`

### Task 17: atlas-consumables — 16 Gen2 extractions

Module dir: `services/atlas-consumables/atlas.com/consumables`. Bake: `atlas-consumables`.

**Files (all R2, + callers):**
- Modify: `cash/processor.go`, `character/buff/processor.go`, `character/processor.go`, `compartment/processor.go`, `consumable/processor.go`, `data/consumable/processor.go`, `data/equipable/processor.go`, `data/map/processor.go`, `equipable/processor.go`, `inventory/processor.go`, `map/character/processor.go`, `map/processor.go`, `monster/drop/position/processor.go`, `monster/processor.go`, `pet/processor.go`, `portal/processor.go`
- Create: a `mock/processor.go` beside each (16 mocks)

Specifics: `consumable/processor.go` is the service's core logic package — cross-package collaborator fields (e.g. the `consumable` processor holding `character`/`inventory` processors) retype from `*X.Processor` to `X.Processor` as part of R2 Step 5. Convert leaf packages (`data/*`, `monster/drop/position`) first within the single commit's work so intermediate builds stay meaningful.

- [ ] Steps 1–8. Commit: `refactor(atlas-consumables): extract Processor interfaces across 16 packages (Gen2→Gen3)`

### Task 18: atlas-inventory — 1 CP-2 fix + 9 Gen2 extractions

Module dir: `services/atlas-inventory/atlas.com/inventory`. Bake: `atlas-inventory`.

**Files:**
- Modify: `data/consumable/processor.go` (R1); R2: `asset/processor.go`, `compartment/processor.go`, `data/equipment/processor.go`, `data/equipment/slot/processor.go`, `data/equipment/statistics/processor.go`, `data/etc/processor.go`, `data/setup/processor.go`, `drop/processor.go`, `pet/processor.go` (+ callers)
- Create: 10 `mock/processor.go` files (one per package above, where absent)

Specifics: `compartment` ↔ `asset` are the heavily-coupled pair; convert `data/*` leaves first, then `asset`, then `compartment`, then `drop`/`pet`.

- [ ] Steps 1–8. Commit: `refactor(atlas-inventory): Gen3 processor conformance across 10 packages`

### Tasks 19–23: atlas-channel — one visit, five package-group commits

Module dir: `services/atlas-channel/atlas.com/channel`. Bake once, in Task 23. Each group task: Steps 1–5 + 7 of the Task 2 skeleton (V-commit per group; V-visit only in Task 23), one commit per group.

### Task 19: atlas-channel group 1 — `data/*`

**Files:**
- Modify: `data/portal/processor.go` (R1), `data/skill/processor.go` (R1), `data/cash/processor.go` (R2), `data/npc/processor.go` (R2) (+ callers)
- Create: mocks for all four packages (where absent)

- [ ] Steps per skeleton. Commit: `refactor(atlas-channel): Gen3 conformance for data/* packages`

### Task 20: atlas-channel group 2 — field/world

**Files (all R2 except noted, + callers):**
- Modify: `map/processor.go`, `movement/processor.go`, `portal/processor.go` (R1), `monster/processor.go`, `monster/information/processor.go`, `reactor/processor.go`, `drop/processor.go`, `weather/processor.go`
- Create: 8 mocks (where absent)

- [ ] Steps per skeleton. Commit: `refactor(atlas-channel): Gen3 conformance for field packages (map, movement, portal, monster, reactor, drop, weather)`

### Task 21: atlas-channel group 3 — social

**Files (all R2, + callers):**
- Modify: `party/processor.go`, `guild/processor.go`, `guild/thread/processor.go`, `messenger/processor.go`, `invite/processor.go`, `fame/processor.go`
- Create: 6 mocks

- [ ] Steps per skeleton. Commit: `refactor(atlas-channel): Gen3 conformance for social packages (party, guild, messenger, invite, fame)`

### Task 22: atlas-channel group 4 — character-adjacent

**Files (all R2, + callers):**
- Modify: `pet/processor.go`, `mount/processor.go`, `summon/processor.go`, `session/processor.go`, `macro/processor.go` (the R2 worked example), `food/processor.go`, `consumable/processor.go`
- Create: 7 mocks

Specifics: `session/processor.go` is wired from the socket layer — expect callers under `socket/` and `kafka/`; the R2 Step 5 grep covers them.

- [ ] Steps per skeleton. Commit: `refactor(atlas-channel): Gen3 conformance for character packages (pet, mount, summon, session, macro, food, consumable)`

### Task 23: atlas-channel group 5 — world/infra + service bake

**Files:**
- Modify: `merchant/processor.go` (R2), `party_quest/processor.go` (R2), `door/processor.go` (R2), `server/processor.go` (R3 — Gen1: package funcs `Register(t tenant.Model, ch channel.Model, ipAddress string, port int) Model` and `GetAll() []Model` become interface methods with identical parameter lists; `NewProcessor(l, ctx)` per Gen3 uniformity even though the methods don't use `l`/`ctx`) (+ callers — `server` has wide fan-in: `main.go:356`, `configuration/projection/loop.go`, `listener/`, and ~8 consumer `_test.go` files construct via `server.Register(tm, ch, "127.0.0.1", ...)` → `server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", ...)` or the test's existing `l`/`ctx` where in scope)
- Create: 4 mocks

- [ ] Steps per skeleton, then **V-visit**: `docker buildx bake atlas-channel`. Commit: `refactor(atlas-channel): Gen3 conformance for world packages (merchant, party_quest, door, server)`
- [ ] **Phase B close-out:** run **V-phase** (redis-key-guard + inventory re-scan + `docs(task-116): inventory after phase B` commit).

---

## Phase C — Gen1 modernization (recipe R3 throughout, except noted)

### Task 24: Non-processor file renames (R6) — 3 services

**Files:**
- Rename: `services/atlas-messages/atlas.com/messages/command/processor.go` → `command/types.go`; `services/atlas-gachapons/atlas.com/gachapons/test/processor.go` → `test/fixtures.go`; `services/atlas-npc-shops/atlas.com/npc/test/processor.go` → `test/fixtures.go`
- Modify: `docs/tasks/task-116-processor-gen3-unification/inventory.md` ("R6 file renames" section, with the justification from the R6 table)

- [ ] **Step 1:** `git mv` each file (content untouched). **Step 2:** V-commit in each of the three module dirs (`services/atlas-messages/atlas.com/messages`, `services/atlas-gachapons/atlas.com/gachapons`, `services/atlas-npc-shops/atlas.com/npc`). **Step 3:** V-visit for all three (`docker buildx bake atlas-messages atlas-gachapons atlas-npc-shops`). **Step 4:** Commit: `refactor(task-116): rename non-processor processor.go files (messages, gachapons, npc-shops)`

### Task 25: atlas-account — 1 Gen1 conversion

Module dir: `services/atlas-account/atlas.com/account`. Bake: `atlas-account`.

**Files:**
- Modify: `ban/processor.go` (R3; note the function is *not* curried — `CheckBan(l logrus.FieldLogger, ctx context.Context, ip string, hwid string, accountId uint32) (CheckRestModel, error)` becomes method `CheckBan(ip string, hwid string, accountId uint32) (CheckRestModel, error)`) (+ callers via `grep -rn "ban\.CheckBan" services/atlas-account/atlas.com/account --include="*.go"`)
- Create: `ban/mock/processor.go`

- [ ] Steps 1–8 of the Task 2 skeleton with R3. Commit: `refactor(atlas-account): Gen3 ban processor (Gen1→Gen3)`

### Task 26: atlas-portals — 2 Gen1 conversions

Module dir: `services/atlas-portals/atlas.com/portals`. Bake: `atlas-portals`.

**Files:**
- Modify: `character/processor.go`, `portal/processor.go` (R3) (+ callers)
- Create: `character/mock/processor.go`, `portal/mock/processor.go`

- [ ] Steps 1–8 (R3). Commit: `refactor(atlas-portals): Gen3 processors (Gen1→Gen3)`

### Task 27: atlas-reactors — 2 Gen1 conversions

Module dir: `services/atlas-reactors/atlas.com/reactors`. Bake: `atlas-reactors`.

**Files:**
- Modify: `reactor/data/processor.go`, `reactor/processor.go` (R3; convert `reactor/data` first — `reactor` likely calls it) (+ callers)
- Create: `reactor/data/mock/processor.go`, `reactor/mock/processor.go`

- [ ] Steps 1–8 (R3). Commit: `refactor(atlas-reactors): Gen3 processors (Gen1→Gen3)`

### Task 28: atlas-asset-expiration — 5 Gen1 conversions

Module dir: `services/atlas-asset-expiration/atlas.com/asset-expiration`. Bake: `atlas-asset-expiration`.

**Files:**
- Modify: `cashshop/processor.go`, `character/processor.go`, `data/processor.go`, `inventory/processor.go`, `storage/processor.go` (R3) (+ callers — this service is ticker-driven; expect wiring in `main.go`/ticker setup)
- Create: 5 `mock/processor.go` files

- [ ] Steps 1–8 (R3). Commit: `refactor(atlas-asset-expiration): Gen3 processors (Gen1→Gen3)`

### Task 29: atlas-monsters — 4 Gen1 conversions (service not in PRD; revealed by scan)

Module dir: `services/atlas-monsters/atlas.com/monsters`. Bake: `atlas-monsters`.

**Files:**
- Modify: `map/processor.go` (single exported provider fn `CharacterIdsInFieldProvider(l)(ctx)(f) model.Provider[[]uint32]` → method `CharacterIdsInFieldProvider(f field.Model) model.Provider[[]uint32]`), `monster/drop/processor.go`, `monster/information/processor.go`, `monster/mobskill/processor.go` (R3) (+ callers — the `monster` package here is already Gen3 and is the main consumer)
- Create: 4 `mock/processor.go` files

- [ ] Steps 1–8 (R3). Commit: `refactor(atlas-monsters): Gen3 processors for map/drop/information/mobskill (Gen1→Gen3)`

### Task 30: atlas-rates — 5 Gen1 conversions (service not in PRD; revealed by scan)

Module dir: `services/atlas-rates/atlas.com/rates`. Bake: `atlas-rates`.

**Files:**
- Modify: `buffs/processor.go` (the R3 worked example — apply it verbatim), `data/cash/processor.go`, `data/equipment/processor.go`, `inventory/processor.go`, `session/processor.go` (R3) (+ callers; known: `character/initializer.go:198`)
- Create: 5 `mock/processor.go` files

- [ ] Steps 1–8 (R3). Commit: `refactor(atlas-rates): Gen3 processors (Gen1→Gen3)`

### Task 31: atlas-monster-death — characterization tests + 5 Gen1 + 1 Gen2

Module dir: `services/atlas-monster-death/atlas.com/monster`. Bake: `atlas-monster-death`. Design §4.3 moved this whole service to Phase C; it is the flagship FR-6.3 characterization-test target.

**Files:**
- Create (first): `monster/processor_test.go` additions (or new test file if none exists in package `monster`), covering the pure functions below
- Modify: `party/processor.go`, `monster/processor.go`, `character/processor.go`, `monster/drop/processor.go`, `monster/drop/position/processor.go` (all R3), `data/equipment/statistics/processor.go` (R2) (+ callers)
- Create: 6 `mock/processor.go` files

**Interfaces:**
- Consumes: `drop.NewBuilder().SetItemId(...).SetChance(...).SetMinimumQuantity(...).SetMaximumQuantity(...).SetQuestId(...).Build()` (exists in `monster/drop/builder.go`)

- [ ] **Step 1: Record green baseline** (V-commit in the module dir).
- [ ] **Step 2: Write characterization tests** in package `monster` (file `monster/characterization_test.go`) against pre-conversion behavior. These target package-internal pure functions, so they compile against the *current* Gen1 file and must not reference `l`/`ctx`:

```go
package monster

import (
	"math"
	"testing"

	"atlas-monster/monster/drop"
)

func TestCalculateExperienceStandardDeviationThreshold_Uniform(t *testing.T) {
	// identical ratios → variance 0 → threshold == mean
	got := calculateExperienceStandardDeviationThreshold([]float64{0.25, 0.25, 0.25, 0.25}, 4)
	if math.Abs(got-0.25) > 1e-9 {
		t.Fatalf("expected 0.25, got %f", got)
	}
}

func TestCalculateExperienceStandardDeviationThreshold_Skewed(t *testing.T) {
	// ratios {0.7,0.1,0.1,0.1}, totalEntries 4: mean=0.25,
	// var=((0.45)^2+3*(0.15)^2)/4=0.0675, threshold=0.25+sqrt(0.0675)
	got := calculateExperienceStandardDeviationThreshold([]float64{0.7, 0.1, 0.1, 0.1}, 4)
	want := 0.25 + math.Sqrt(0.0675)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("expected %f, got %f", want, got)
	}
}

func TestIsWhiteExperienceGain(t *testing.T) {
	ratios := map[uint32]float64{1: 0.6, 2: 0.2}
	if !isWhiteExperienceGain(1, ratios, 0.5) {
		t.Fatal("ratio above threshold should be white gain")
	}
	if isWhiteExperienceGain(2, ratios, 0.5) {
		t.Fatal("ratio below threshold should not be white gain")
	}
	if isWhiteExperienceGain(99, ratios, 0.5) {
		t.Fatal("absent character should not be white gain")
	}
}

func TestGetSuccessfulDrops_DeterministicEdges(t *testing.T) {
	// evaluateSuccess: rand.Int31n(999999) < min(chance*rate, MaxInt32)
	certain, _ := drop.NewBuilder().SetItemId(1000).SetChance(999999).SetMinimumQuantity(1).SetMaximumQuantity(1).Build()
	never, _ := drop.NewBuilder().SetItemId(2000).SetChance(0).SetMinimumQuantity(1).SetMaximumQuantity(1).Build()
	for i := 0; i < 100; i++ {
		got := getSuccessfulDrops([]drop.Model{certain, never}, 1.0)
		if len(got) != 1 || got[0].ItemId() != 1000 {
			t.Fatalf("iteration %d: expected exactly the certain drop, got %d entries", i, len(got))
		}
	}
}
```

  And in package `drop` (file `monster/drop/characterization_test.go`) — `getRandomStat` bounds and `isEquipment`:

```go
package drop

import (
	"math"
	"testing"
)

func TestGetRandomStat_ZeroStaysZero(t *testing.T) {
	for i := 0; i < 100; i++ {
		if got := getRandomStat(0, 5); got != 0 {
			t.Fatalf("zero stat must stay zero, got %d", got)
		}
	}
}

func TestGetRandomStat_Bounds(t *testing.T) {
	// maxRange = min(ceil(default*0.1), max); result in [default-maxRange, default+maxRange]
	for i := 0; i < 1000; i++ {
		def, maxSpread := uint16(100), uint16(5)
		maxRange := math.Min(math.Ceil(float64(def)*0.1), float64(maxSpread)) // = 5
		got := getRandomStat(def, maxSpread)
		if float64(got) < float64(def)-maxRange || float64(got) > float64(def)+maxRange {
			t.Fatalf("stat %d outside [%f,%f]", got, float64(def)-maxRange, float64(def)+maxRange)
		}
	}
}

func TestIsEquipment(t *testing.T) {
	if !isEquipment(1302000) {
		t.Fatal("1302000 is equipment")
	}
	if isEquipment(2000000) {
		t.Fatal("2000000 is not equipment")
	}
}
```

  Adjust the import path in the first file to the module's real name (check the `module` line in `services/atlas-monster-death/atlas.com/monster/go.mod` — project memory: module names are short, e.g. `atlas-transports`, so verify rather than assume `atlas-monster`).
- [ ] **Step 3: Run the new tests, expect PASS** (they characterize existing behavior): `go test -race ./monster/... -run 'Characteriz|StandardDeviation|WhiteExperience|SuccessfulDrops|RandomStat|IsEquipment' -v` — all PASS. Record the two packages in `inventory.md` "Characterization tests".
- [ ] **Step 4: Commit the tests alone:** `test(atlas-monster-death): characterization tests for drop and experience math`
- [ ] **Step 5: Apply R3** to the five Gen1 files and R2 to `data/equipment/statistics/processor.go`. Conversion order: `data/equipment/statistics`, `party`, `character`, `monster/drop/position`, `monster/drop`, `monster` (dependency order — `monster` consumes `drop`, which consumes `position` and `statistics`). The characterization tests' target functions become unexported methods or stay package functions per R3 Step 3 — if they become methods, update the test call sites mechanically in the same commit (`getSuccessfulDrops(...)` → `(&ProcessorImpl{}).getSuccessfulDrops(...)` is NOT acceptable; keep pure math as package functions so the tests stay untouched — that is the R3 Step 3 bias for functions that do NOT use `l`/`ctx`).
- [ ] **Step 6: Create the 6 mocks (R5).**
- [ ] **Step 7: V-commit — including the Step 2 tests passing unchanged.** Then V-visit: `docker buildx bake atlas-monster-death`.
- [ ] **Step 8: Update `inventory.md`; commit:** `refactor(atlas-monster-death): Gen3 processors across all packages (Gen1/Gen2→Gen3)`

### Tasks 32–35: atlas-data — one visit, package-by-package commits

Module dir: `services/atlas-data/atlas.com/data`. Bake once, in Task 35. **One commit per package** (FR-4 mandate: green build between packages). Each package: R3 + mock + V-commit + commit `refactor(atlas-data): Gen3 <pkg> processor (Gen1→Gen3)`. Callers inside the leaf package's own REST `resource.go`/`rest.go` update with it; callers in the `data` orchestrator package and `data/workers` are deferred to Task 35 — until then those call sites still compile because the old exported curried functions are only *removed* in the same commit that rewires their callers. Where a leaf package's function is consumed by `data`/`workers` (Step 6 grep hit outside the leaf), rewire that call site in the same package commit — "defer to Task 35" applies only to `data`'s own conversion, not to leaving broken references.

### Task 32: atlas-data leaves, group 1

**Files (R3 + mock each, one commit per package):**
- Modify+Create mocks for: `cash/processor.go`, `commodity/processor.go`, `consumable/processor.go`, `etc/processor.go`, `setup/processor.go`, `pet/processor.go`

- [ ] Per package: R3 → mock → V-commit → commit.

### Task 33: atlas-data leaves, group 2

**Files (R3 + mock each, one commit per package):**
- Modify+Create mocks for: `job/processor.go`, `map/processor.go`, `mobskill/processor.go`, `monster/processor.go`, `npc/processor.go`, `quest/processor.go`, `reactor/processor.go`, `skill/processor.go`

- [ ] Per package: R3 → mock → V-commit → commit. `npc` is the design's higher-order exemplar: `RegisterNpc(db)(l)(ctx)(path)` → `NewProcessor(l, ctx, db)` + method `RegisterNpc(path string) error`; its `data/workers/npc.go` call site (`registerAllInDirectory(l, ctx, dir, npc.RegisterNpc(db))`) rewires to `npc.NewProcessor(l, ctx, db).RegisterNpc` in this package's commit (R3 Step 6).

### Task 34: atlas-data leaves, group 3

**Files (R3 + mock each, one commit per package):**
- Modify+Create mocks for: `equipment/processor.go`, `characters/templates/processor.go`, `cosmetic/face/processor.go`, `cosmetic/hair/processor.go`

- [ ] Per package: R3 → mock → V-commit → commit.

### Task 35: atlas-data orchestrator + workers + service bake

**Files:**
- Modify: `data/processor.go` (the orchestrator package `data/data/` — R3; note it has existing tests `data/processor_test.go`, `runwz_test.go`, `status_test.go` which update same-commit), remaining higher-order plumbing in `data/workers/*.go` (parameter types simplify per R3 Step 6)
- Create: `data/mock/processor.go`

- [ ] R3 → mock → V-commit → **V-visit**: `docker buildx bake atlas-data` → commit: `refactor(atlas-data): Gen3 data orchestrator processor; simplify workers plumbing (Gen1→Gen3)`
- [ ] **Phase C close-out:** run **V-phase** (redis-key-guard + inventory re-scan + `docs(task-116): inventory after phase C` commit).

---

## Phase D — Final verification and review

### Task 36: Acceptance sweep

**Files:**
- Modify: `docs/tasks/task-116-processor-gen3-unification/inventory.md` (final scan, all statuses done, deviations/renames/characterization sections complete)

- [ ] **Step 1: Acceptance greps** (from the worktree root) — both must return no output:

```bash
grep -rn "func NewProcessor(" services/ --include="*.go" | grep -v "_test.go" | grep -v "/mock/" | grep "\*ProcessorImpl"
grep -rln "type Processor struct" services/ --include="*.go" | grep -v mock
```

- [ ] **Step 2: Full inventory re-scan** (Task 1 script). Every non-mock `processor.go`'s package declares `type Processor interface`; the only documented deviations are the two R4 clients and the three R6 renames (which are no longer `processor.go` files).
- [ ] **Step 3: Conformance-assertion sweep:** every converted package has the impl assertion, every mock the mock assertion:

```bash
for f in $(find services -name "processor.go" -not -path "*/mock/*"); do d=$(dirname $f); grep -qr "var _ Processor = (\*ProcessorImpl)(nil)" $d/*.go || echo "MISSING-ASSERT: $f"; done
```

  (Expected output: only pre-existing Gen3 packages that never got touched AND keep their assertion in a `_test.go` file — the three known `atlas-messages` cases per Global Constraint 6; anything else is a gap to fix now.)
- [ ] **Step 4: Full verification:** V-commit in EVERY touched module (script the loop over the module list from `inventory.md`), `tools/redis-key-guard.sh`, and `docker buildx bake` for every touched service (re-run even if run during the visit — this is the branch-end sweep; design §6).
- [ ] **Step 5: Diff-scope check:** `git diff --stat main...HEAD -- ':!services' ':!docs/tasks/task-116-processor-gen3-unification'` returns nothing (no changes outside the sanctioned areas).
- [ ] **Step 6: Commit:** `docs(task-116): final inventory and acceptance verification`

### Task 37: Code review

- [ ] Invoke `superpowers:requesting-code-review` — dispatches `backend-guidelines-reviewer` (Go changes) + `plan-adherence-reviewer` against this plan; findings land in `docs/tasks/task-116-processor-gen3-unification/audit.md`.
- [ ] Address findings (behavior-preserving fixes only; anything larger goes to "Deferred findings" in `inventory.md` and is raised to the user).
- [ ] STOP. Do not open a PR in this task — `superpowers:finishing-a-development-branch` runs after the user reviews the audit.

---

## Self-review record

- **Spec coverage:** design §2 (target pattern) → Recipes R1–R3 + Global Constraints 4–6; §3.1–3.4 → R1/R2/R3/R5; §4.1 (one PR, per-service commits) → Global Constraint 13 + task commits; §4.2 → R4/Task 14; §4.3 → Task 31; §4.4 → R3 applied uniformly incl. single-function packages (Tasks 29, 30); §5 (one visit per service, order) → task ordering; §6 (testing/verification) → Verification protocol + Constraint 12 + Task 31; scan-revealed extras (FR-4 final ¶) → Tasks 12, 15, 24, 25, 29, 30 and channel `server`/messengers `invite` folded into Tasks 23/13.
- **Known open risk:** the plan-time file lists may drift if main moves under the branch; Task 1's scan is authoritative and each task re-greps its callers, so drift surfaces as inventory rows, not silent misses.
- **Type consistency:** all constructors named `NewProcessor`, impls `ProcessorImpl`, mocks `ProcessorMock`, assertions `var _ Processor = (*ProcessorImpl)(nil)` / `var _ <pkg>.Processor = (*ProcessorMock)(nil)` throughout.
