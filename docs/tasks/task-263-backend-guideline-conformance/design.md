# Backend Guideline Conformance Sweep — Design

Version: v1
Status: Approved
Created: 2026-08-26
PRD: [prd.md](prd.md)
Branch point: `eaa5ce6f7` (counts re-derived at `b74006ae3`)

---

## 1. What this design settles

The PRD establishes *what* must be true: DOM-01, DOM-04 and FILE-05 pass
repo-wide, with an evidence-backed exemption list for the rows that cannot be
satisfied. It leaves four questions open (§9) and, on three points, it asserts a
population from a grep that turns out not to survive contact with the code.

This document does three things:

1. Re-derives the change population with finer instruments than `sweep.sh`, so
   the plan phase partitions real work rather than grep hits (§2).
2. Records eight decisions (D1–D8), four of which resolve a PRD open question
   and three of which correct a PRD assumption (§3).
3. Specifies the execution architecture — one AST codemod, three workstreams,
   a commit partition, and a test strategy (§4–§7).

It introduces no requirement the PRD does not already carry, except where §3
marks a decision as **amends the PRD**.

---

## 2. Evidence base

Every count below was derived from the tree at `b74006ae3` by classifier
scripts run during this design session. They refine, and in three places
correct, the PRD's §4 tables.

### 2.1 DOM-04 — the 185 `has-model` packages are not one population

`sweep.sh` splits DOM-04 by "does the package serve a resource" and
`split-by-model.sh` splits it by "does a `type Model` exist". Neither predicts
how much work a package needs. The predictive axis is **the shape of the
package's existing `Extract`**, because `Transform` is its inverse (PRD FR-1).

| Tier | Count | Shape | Treatment |
|---|---|---|---|
| **A** | 104 | Exactly one `Extract` in `rest.go`, body is a flat composite literal (no `for`, `if`, `switch`, `range`, or intermediate assignment) | AST codemod inverts it |
| **B1** | 37 | Two or more `Extract*` functions in the package | Per-package judgment |
| **B2** | 37 | One `Extract`, but the body is non-flat, or `Extract` lives outside `rest.go` | Per-package judgment |
| **C** | 7 | No `Extract` anywhere in the package | Derive from the two struct declarations |

Within tier A:

- 96 of 104 are pure field↔field with no conversion call; 8 wrap a field in a
  conversion (`byte(...)`, a named type) whose inverse the codemod must emit.
- 88 of 104 construct `Model` from **unexported** fields
  (`Model{id: rm.Id, …}`) rather than from a constructor.
- 33 of 104 have **no accessor** for at least one of those fields, so an
  accessor-based `Transform` would require minting new exported methods.
- 1 of 104 involves a pointer or slice field.

### 2.2 DOM-04 — the `Transform` name is not always available

The PRD's FR-7 states the four files named in #1498 "all fall in the
`has-model` population, so none is exempt". That is true of the
`split-by-model.sh` grep and false of the code.

**14 of the 185 `has-model` packages declare no `type RestModel`.** The
`Model` type that made them `has-model` is not the type their `rest.go`
converts. Examples:

- `services/atlas-channel/atlas.com/channel/monsterbook/` — `model.go:11`
  declares `type Model`, but `rest.go` declares `CollectionRestModel` and
  `CardRestModel` and extracts to `Collection` (`rest.go:107`) and `Card`
  (`rest.go:102`). There is no `RestModel`.
- `services/atlas-channel/atlas.com/channel/data/tradeability/` — `rest.go:17`
  declares `type Model`, and `rest.go` declares five sibling wire types
  (`EquipmentRestModel:34`, `ConsumableRestModel:52`, `SetupRestModel:70`, and
  two more). There is no `RestModel` and no principled primary among the five.

The full 14 are enumerated in §8.2. For these, `func Transform(` — the literal
grep at `file-responsibilities.md:193` — cannot be written, because the
signature `Transform(Model) (RestModel, error)` names a type that does not
exist.

### 2.3 DOM-01 — rename volume and the seven collisions

| Fact | Count |
|---|---|
| `func NewModelBuilder(` declarations (non-test) | 59 |
| Cross-package `X.NewModelBuilder(` call sites | 743 |
| `CloneModelBuilder` references | 31 |
| Qualified `X.ModelBuilder` type references | 35 |
| Unexported `type modelBuilder struct` declarations | 31 |
| Files containing both `modelBuilder` and a bare `builder` identifier | 20 |

Two consequences:

- The 743 cross-package call sites make this codemod-only work, exactly as the
  PRD's §8 "codemod-first" clause requires.
- The 20 files where `builder` already appears as an identifier mean the
  unexported half (`modelBuilder` → `builder`) is **not** safe as a textual
  substitution. It needs a type-aware rename or a per-file compile check.

**Seven `atlas-channel` packages already declare both constructors.** Go has no
overloading, so FR-12's "rename all 59 to `NewBuilder`" cannot be a rename
there. Their signatures:

| Package | `NewModelBuilder` | `NewBuilder` | Relationship |
|---|---|---|---|
| `channel/channel` | `()` | `()` | identical |
| `channel/inventory` | `(characterId uint32)` | same | identical |
| `channel/note` | `()` | `()` | identical |
| `channel/world` | `()` | `()` | identical |
| `channel/compartment` | `(id, characterId, it, capacity)` | same | identical |
| `channel/transport/route` | `(name string)` | same | identical |
| `channel/asset` | `(id uint32, compartmentId, templateId)` | `(compartmentId, templateId)` | **differs** |

Six of the seven are not real collisions. `NewBuilder` is already a one-line
alias delegating to `NewModelBuilder`, carrying the comment "alias for
NewModelBuilder for backward compatibility" — see
`services/atlas-channel/atlas.com/channel/inventory/builder.go:26` and
`services/atlas-channel/atlas.com/channel/transport/route/builder.go:42`. Only
`channel/asset` has genuinely divergent arity
(`builder.go:119` vs `builder.go:127`).

### 2.4 DOM-01 — FR-15's 69 constructors, classified

The PRD's open question 1 asks whether FR-15's triage might be large enough to
split out. It is not. Classifying all 69 against DOM-01's own trigger
(`audit-checklist.md:94` — "package has `model.go`"):

| Class | Count | Disposition |
|---|---|---|
| Sole builder in a package that has `model.go` | 9 | Rename to `NewBuilder` |
| Sibling builders over distinct types in one package | 55 | Keep `New<Type>Builder`; record as exempt |
| Package has no `model.go` — DOM-01's trigger never fires | 5 | Record as N/A, not exempt |

Nine renames is a rounding error next to the 59 of FR-12. FR-15 stays in scope.

### 2.5 FILE-05 — 100 declarations, 72 packages

| Fact | Count |
|---|---|
| Distinct packages | 72 |
| Target package already has a `builder.go` | 6 |
| Target package needs a new `builder.go` | 94 |

By source file:

| Source file | Declarations |
|---|---|
| `model.go` | 83 |
| `reference_data.go` | 7 |
| `entity_builder.go` | 4 |
| `skill_usage_info.go`, `recipients.go`, `producer.go`, `message.go`, `context.go`, `connection.go` | 1 each |

The dominant case is a single builder co-located with `Model` in `model.go`.

The four `entity_builder.go` files declare `type entityBuilder struct` with
`func NewEntityBuilder()` — see
`services/atlas-tenants/atlas.com/tenants/tenant/entity_builder.go:5,15`,
`services/atlas-tenants/atlas.com/tenants/configuration/entity_builder.go:9,17`,
and `services/atlas-quest/atlas.com/quest/quest/entity_builder.go:10,24`. These
build the GORM `entity`, not the domain `Model`.

### 2.6 Toolchain

- `go.work` lists 96 modules; `find . -name go.mod` returns 111. The repo is a
  multi-module workspace, so any type-aware tool must be workspace-aware.
- `gopls` and `gorename` are **not installed** in this environment.
- `tools/verify.sh:182` enumerates modules with
  `find "$ROOT/services" "$ROOT/libs" -name go.mod`. A Go module outside those
  two trees is invisible to the gate.

---

## 3. Decisions

### D1 — `Transform` reads unexported fields directly

**Decision.** Generated `Transform` bodies read the `Model`'s unexported fields
directly. No new accessors are minted.

```go
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:   m.id,
		Name: m.name,
		Gm:   m.gm,
	}, nil
}
```

This is the exact syntactic inverse of the `Extract` it pairs with:

```go
func Extract(rm RestModel) (Model, error) {
	return Model{id: rm.Id, name: rm.Name, gm: rm.Gm}, nil
}
```

**Alternatives rejected.**

*Accessor-based, minting the missing ones* — matches the visible house style
(`services/atlas-buddies/atlas.com/buddies/buddy/rest.go` and
`services/atlas-ban/atlas.com/ban/ban/rest.go` both use accessors). Rejected
because 33 of the 104 tier-A packages lack an accessor for at least one field,
so a conformance sweep would silently widen dozens of packages' exported API.
The PRD's §5 states this task introduces no API surface; minting accessors
would contradict it.

*Hybrid — accessor where present, field otherwise* — rejected because it
produces mixed-style bodies within one function and makes the codemod's output
depend on an accessor-resolution step that the direct form does not need.

**Consequence.** `Transform` is symmetric with `Extract`: both construct across
the package's own encapsulation boundary, which is where they both already
live. The house style in `buddy`/`ban` is not violated — those packages have
complete accessor sets and their hand-written `Transform`s are untouched. Only
newly generated bodies use the direct form.

### D2 — 14 packages get named variants and an exemption entry

**Decision.** The 14 packages of §2.2 get `Transform`-prefixed converters
following FR-2/FR-3 (`TransformCard`, `TransformCollection`,
`TransformEquipment`, `TransformConsumable`, …) **and** an entry in
`exemptions.md` recording that the literal `func Transform(` detector cannot
fire because no `type RestModel` exists.

**Alternatives rejected.**

*Amend the detector* — change `file-responsibilities.md:193` and the sweep from
`func Transform(` to a `Transform`-prefixed converter per `RestModel`. This is
the technically cleaner fix and changes the *verification procedure*, not the
*rule* at `audit-checklist.md:97`. Rejected because the PRD's §2 non-goals name
"changing the checklist itself" and the distinction between a rule and its
detector is too fine to unilaterally claim as inside scope. It remains the
right follow-up if a future audit finds the exemption list is being ignored.

*Force a `Transform` alias per package* — designate a primary wire type and
name its converter `Transform`. Rejected because `data/tradeability`'s five
compartment RestModels are peers; any choice of primary is arbitrary and would
read as noise to the next person.

**Consequence.** `backend-guidelines-reviewer` will still record DOM-04 as FAIL
on these 14 unless it reads `exemptions.md`. That is an accepted, documented
residue — see §9, risk R3.

### D3 — the seven duplicate constructors merge

**Decision.** For the six packages of §2.3 whose signatures are identical:
inline `NewModelBuilder`'s body into `NewBuilder`, delete `NewModelBuilder`,
repoint call sites to `NewBuilder`, and drop the now-false "alias for
NewModelBuilder for backward compatibility" comment.

For `channel/asset`, the two constructors differ in arity and are not
interchangeable. `NewBuilder(compartmentId, templateId)` is retained unchanged;
`NewModelBuilder(id, compartmentId, templateId)` is renamed
`NewBuilderWithId`. This satisfies FR-16 (zero `NewModelBuilder` in the sweep)
and DOM-01 (a `NewBuilder()` exists), and it is the only option that does not
change a call site's semantics.

**Alternatives rejected.**

*Disambiguate all seven by parameter* — mechanical and uniform, but leaves six
packages carrying two constructors that do literally the same thing, which is
the duplication the checklist row exists to prevent.

*Exempt the seven* — fails the PRD's first acceptance criterion outright.

**Consequence.** The six merges are behavior-preserving by inspection: the
deleted function's only distinguishing property was its name. `channel/asset`
is the one hand-reviewed case in this workstream.

### D4 — FR-15 resolves to 9 renames, 55 exemptions, 5 N/A

**Decision.** Apply the §2.4 classification. The 55 sibling-builder packages are
recorded in `exemptions.md` with their sibling constructors listed as the
evidence FR-15 demands. The 5 packages without `model.go` are recorded as N/A
against DOM-01's stated trigger, not as exemptions — a trigger that never
fires is a different disposition from a rule that cannot be satisfied
(`audit-checklist.md:25-44`).

**Resolves PRD open question 1.** FR-15 stays in this task.

### D5 — `entity_builder.go` is correct placement

**Decision.** The four `entity_builder.go` files are recorded in
`exemptions.md` as out of FILE-05's scope, with the evidence that
`entityBuilder` constructs the GORM `entity` and not the domain `Model`.
FILE-05's file table assigns `builder.go` to the *domain* builder and
`entity.go` to entity concerns; a single-purpose `entity_builder.go` is exactly
the "genuine single-purpose utility" FILE-06 permits. They are not moved.

**Satisfies FR-10**, which requires the question be resolved explicitly rather
than left unexamined.

### D6 — the codemod is a throwaway Go module outside `services/` and `libs/`

**Decision.** The AST codemod lives at
`docs/tasks/task-263-backend-guideline-conformance/codemod/` with its own
`go.mod`, deliberately **not** added to `go.work`. Because `tools/verify.sh:182`
enumerates modules only under `services/` and `libs/`, the gate never builds or
lints it. It is deleted in the final commit alongside `sweep.sh` (D7).

**Alternatives rejected.**

*Promote it into `tools/`* — `tools/` holds durable guards
(`atlasguards`, `goroutineguard`, `envguard`) that CI runs forever. A
one-shot migration tool would rot there.

*Run it from `/tmp`* — rejected because the plan phase and the reviewer need to
read the transformation that produced 104 files; an unreviewable generator is
worse than hand edits.

### D7 — `sweep.sh` and the inventories are scaffolding, deleted at task end

**Decision.** `sweep.sh`, `split-by-model.sh`, `codemod/`, and every
`inventory-*.txt` survive through execution — the plan consumes them as its
work inventory — and are deleted in the branch's final commit. No guard is
promoted into `tools/` and `verify.sh` is not touched.

**Amends the PRD.** This drops goal 3 ("leave a re-runnable inventory") and
rewrites acceptance criteria 1–3, which are phrased as "`sweep.sh` reports N
entries". Those become **execution-time gates**: the final implementation
commit must be preceded by a recorded `sweep.sh` run showing the required
counts, and that run's output is pasted into the task's `progress.md` as the
evidence. The check happens; the script does not survive it.

`exemptions.md` (FR-6) is unaffected and remains the branch's durable artifact.

### D8 — three workstreams, never interleaved in one commit

**Decision.** The work partitions into three independent workstreams that touch
disjoint symbol sets, so they can be ordered freely and reviewed separately:

| # | Workstream | Rows | Nature |
|---|---|---|---|
| W1 | Builder relocation | FILE-05 | Intra-package file moves, 72 packages |
| W2 | Builder rename | DOM-01 | Type-aware rename, 59 + 9 constructors |
| W3 | `Transform` + round-trip test | DOM-04 | 104 generated + 81 hand, 185 packages |

Ordering is **W1 → W2 → W3**. W1 first because moving a builder into
`builder.go` gives W2 a single file per package to rename in. W3 last because
it is the only workstream that adds tests, and it should run against a tree
where the builder churn has already settled — its round-trip tests construct
`Model` values through those builders.

**Satisfies the PRD's §8 reviewability constraint**: no commit mixes two
checklist rows.

---

## 4. Architecture

### 4.1 The codemod

One Go program, three subcommands, one shared front-end. It uses
`golang.org/x/tools/go/packages` in `NeedSyntax|NeedTypes|NeedTypesInfo` mode so
that rename operations are type-checked rather than textual, loading each
module from `go.work`.

```
codemod/
  go.mod
  main.go            // subcommand dispatch
  load.go            // packages.Load over a module list, shared by all three
  relocate.go        // W1: move builder decls model.go -> builder.go
  rename.go          // W2: type-aware NewModelBuilder/ModelBuilder rename
  transform.go       // W3: invert Extract -> Transform, tier A only
  report.go          // per-package PASS / SKIPPED(reason) ledger
```

**The ledger is the contract.** Every subcommand emits one line per candidate
package: `APPLIED`, or `SKIPPED` with a machine-readable reason. A package the
codemod skips is not silently dropped — it lands on the hand-work list for that
workstream. The union of `APPLIED` and `SKIPPED` must equal the input
inventory, and the run fails if it does not. This is what prevents the
"prevalence is not compliance" failure the PRD's §1 describes from recurring in
a new form.

### 4.2 W1 — builder relocation (`relocate.go`)

For each of the 72 packages, from the AST of the source file:

1. Collect the builder's declaration set: `type <X>Builder struct`, its
   constructor(s), every method with a `*<X>Builder` receiver, and any
   `Clone*` free function returning `*<X>Builder`.
2. Move those declarations verbatim into `builder.go` in the same package,
   creating the file with the package clause if absent, appending if present
   (6 packages).
3. Recompute the import sets of both files from the moved declarations' actual
   references — not by copying the source file's import block — and run
   `gofumpt` on both.

The move is byte-identical at the declaration level; only import blocks and
file boundaries change. `git diff -M` should show the hunks as moves.

**Skip conditions** (→ hand work): a builder method whose receiver is also used
by a non-builder declaration in the same block; a builder declared inside a
file whose remaining content would become empty; `reference_data.go` (FR-11,
7 builders in 1038 lines, hand-split).

### 4.3 W2 — builder rename (`rename.go`)

Type-aware, driven by `types.Object` identity rather than by name:

1. Resolve the `*types.TypeName` for `ModelBuilder` / `modelBuilder` and the
   `*types.Func` for `NewModelBuilder` / `CloneModelBuilder` in each target
   package.
2. Rewrite every `*ast.Ident` whose `TypesInfo.Uses`/`Defs` entry is that
   object, across every module in `go.work` — this reaches the 743 cross-package
   call sites and the 35 qualified type references.
3. Before rewriting an unexported `modelBuilder` → `builder`, check the target
   scope for an existing object named `builder`. If one exists, skip the
   package to hand work. This is the guard for the 20 files of §2.3.

The 6 merges and `channel/asset` (D3) are **not** codemod work — they are
hand edits applied before the codemod runs, so that by the time `rename.go`
executes, every target package has exactly one constructor to rename.

The 9 FR-15 renames (D4) reuse the same machinery with a different symbol list.

### 4.4 W3 — `Transform` generation (`transform.go`)

Tier A only (104 packages). For each:

1. Locate `func Extract(rm RestModel) (Model, error)` and require its body to
   be a single `return &?Model{…}, nil` with a flat `*ast.CompositeLit`.
2. Invert each `KeyValueExpr`: key `id`, value `rm.Id` becomes key `Id`, value
   `m.id`. Where the value is a conversion `T(rm.X)`, emit
   `U(m.x)` where `U` is the `RestModel` field's declared type, taken from
   `TypesInfo` — never guessed.
3. Emit `func Transform(m Model) (RestModel, error)` appended to `rest.go`,
   and the round-trip test (§6) into `rest_test.go`.
4. Verify the emitted function type-checks in the package before writing. A
   package whose generated `Transform` does not type-check is reverted and
   skipped to hand work — the codemod never writes code it has not checked.

Step 4 is what makes the 8 conversion cases safe: if the inverse conversion is
wrong, the type check catches it rather than the reviewer.

**Tiers B1/B2/C (81 packages) are hand work.** They are the part the PRD's §8
identifies as genuinely requiring per-package judgment, and they are the
majority of the review burden despite being a minority of the file count.

---

## 5. Hand-work inventory

The codemod is expected to leave roughly this much for people:

| Workstream | Codemod | Hand | Why hand |
|---|---|---|---|
| W1 | ~65 packages | ~7 | `reference_data.go` split (FR-11), skip conditions |
| W2 | 50 + 9 renames | 7 | The 6 merges + `channel/asset` (D3) |
| W3 | 104 packages | 81 | Tiers B1 (37), B2 (37), C (7) |
| — | — | 74 entries | `exemptions.md` (14 + 55 + 4 + 1 `libs/atlas-packet`) |

The 81 tier-B/C packages are the schedule's long pole. Per
`docs/codemod-vs-agents.md`, they are the *only* part of this task where
fanning out implementer agents is the right instrument — the 100 moves and the
59 renames must not be agent work.

---

## 6. Testing strategy

**FR-4's round-trip test is the whole point.** Without it, 185 `Transform`
functions are unreachable code that no compiler and no test protects, and the
first `Model` field added after this task silently desynchronises them.

Generated form, per package:

```go
func TestTransformRoundTrip(t *testing.T) {
	m, err := NewBuilder(/* every field set to a distinct non-zero value */).Build()
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

Three properties this design depends on:

- **Every field distinct and non-zero.** A test that leaves a field at its zero
  value passes when `Transform` forgets that field. The generator derives the
  value from the field's type; a package where it cannot is skipped to hand
  work, not given a zero-valued test.
- **`reflect.DeepEqual` on the whole `Model`**, not field-by-field assertions.
  Field-by-field assertions grow stale the same way `Transform` does.
- **Built through the package's builder** (per CLAUDE.md's builder-pattern rule,
  and no `*_testhelpers.go`). This is the dependency that forces W3 to run
  after W1 and W2.

Where a package has no builder, the test constructs `Model` directly with a
composite literal — legal in-package, and consistent with D1.

Beyond the round-trip tests, the gate is unchanged: flagless `tools/verify.sh`
exits 0 before the branch is called done.

---

## 7. Commit partition and sequencing

One commit per workstream per service, never mixing rows (PRD §8):

```
W1  refactor(<service>): move builders into builder.go        × ~40
W2  refactor(<service>): rename ModelBuilder to Builder       × ~25
W3  feat(<service>): add Transform and round-trip tests       × ~40
    chore(task-263): record guideline exemptions              × 1   (exemptions.md)
    chore(task-263): remove conformance scaffolding           × 1   (D7)
```

Each commit is independently buildable — the codemod runs per module and the
per-service commit is cut only after that module's `go build ./...` and
`go test ./...` pass.

**Merge-conflict sequencing (resolves PRD open question 4).** Nine worktrees are
in flight (task-240, 241, 246, 250, 251, 254, 256, 259, 262). The conflict
surface is asymmetric:

- **W2 is nearly conflict-free.** It renames a symbol. A concurrent branch that
  calls `NewModelBuilder` conflicts only if it touches the same line, and the
  rename is trivially re-appliable to a rebased branch by re-running the
  codemod.
- **W1 is the conflict generator.** Moving 83 builders out of `model.go`
  rewrites the file another branch is most likely editing.
- **W3 appends to `rest.go` and adds a new `rest_test.go`.** Appends conflict
  rarely; new files never do.

Therefore: **land W2 and W3 first, W1 last**, inverting the *development* order
of D8. W1's relocations are re-derivable from a one-command codemod re-run
after a rebase, so if W1 does conflict, the resolution is "re-run the tool",
not "merge by hand". The PRD's per-row commit isolation is what makes this
selective landing possible.

If a rebase is needed, per CLAUDE.md the branch stays single and the clean PR
branch is produced by rebase at PR time.

---

## 8. Artifacts

### 8.1 `exemptions.md` structure

One entry per package, grouped by checklist row. Every entry carries
`file:line` evidence — the acceptance criterion forbids "out of scope" without
it.

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

Sections: DOM-04 no-`Model` (176 packages, FR-5), DOM-04 no-`RestModel`
(14, D2), DOM-01 sibling builders (55, D4), DOM-01 trigger not fired (5, D4),
FILE-05 entity builders (4, D5), FILE-05 excluded tree (1 —
`libs/atlas-packet/model/skill_usage_info.go:143`, PRD §7/FR-18).

The 176 no-`Model` packages are listed by path with a single shared evidence
paragraph per service cluster rather than 176 individual write-ups; the PRD's
FR-5 evidence for `atlas-data` is the model.

### 8.2 The 14 no-`RestModel` packages (D2)

```
services/atlas-cashshop/atlas.com/cashshop/rewardpool/rest.go
services/atlas-channel/atlas.com/channel/data/tradeability/rest.go
services/atlas-channel/atlas.com/channel/monsterbook/rest.go
services/atlas-dragons/atlas.com/dragons/dragon/rest.go
services/atlas-drops/atlas.com/drops/data/foothold/rest.go
services/atlas-inventory/atlas.com/inventory/data/tradeability/rest.go
services/atlas-messengers/atlas.com/messengers/character/rest.go
services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go
services/atlas-parties/atlas.com/parties/character/rest.go
services/atlas-pets/atlas.com/pets/data/position/rest.go
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/rates/rest.go
services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/reactor/drop/rest.go
services/atlas-summons/atlas.com/summons/summon/rest.go
services/atlas-tenants/atlas.com/tenants/configuration/rest.go
```

The plan phase regenerates this list from the classifier before starting W3 —
it is derived, not authored.

**Resolves PRD open question 2** for `npc/conversation`: FR-1 applies to the
types the package's `rest.go` actually declares a `RestModel` for and that
`Extract` already round-trips. A builder-backed domain type with no wire
representation gets no `Transform`.

**Resolves PRD open question 3**: `exemptions.md` stays in the task folder. It
is the branch's durable artifact (D7), and moving exemptions into
`.claude/skills/backend-dev-guidelines/resources/audit-checklist.md` would be
the checklist change the PRD's §2 forbids.

---

## 9. Risks

| # | Risk | Mitigation |
|---|---|---|
| R1 | The codemod writes 104 `Transform` bodies that compile but are semantically wrong (a field mapped to the wrong sibling of the same type). | The round-trip test (§6) with every field set to a **distinct** value catches exactly this. A same-type field swap is invisible to the compiler and visible to `reflect.DeepEqual`. |
| R2 | `modelBuilder` → `builder` collides with a local identifier and the codemod silently produces a shadowed reference that still compiles. | `rename.go` step 3 checks the target scope and skips rather than rewrites; the 20 candidate files of §2.3 are the known population. |
| R3 | `backend-guidelines-reviewer` re-raises DOM-04 on the 14 D2 packages, since it follows the literal grep. | Accepted residue. `exemptions.md` is the answer; if it proves insufficient in practice, the detector amendment rejected in D2 becomes a follow-up task. |
| R4 | 743 call-site rewrites land a change to a `Build()` validation rule or a JSON tag by accident (FR-17). | The rename touches identifiers only, never literals or struct tags. The PRD's acceptance criterion requires an explicit reviewer pass over the diff for exactly this; `git diff -G'json:"' main...HEAD` is the mechanical form. |
| R5 | W1 lands and immediately conflicts with all nine in-flight worktrees. | §7 lands W1 last, and W1 is re-derivable by re-running the codemod after a rebase. |
| R6 | Deleting `sweep.sh` (D7) removes the ability to confirm the rows stayed closed. | Accepted, per the explicit decision. The pre-deletion run's output is recorded in `progress.md` as the evidence; regression detection reverts to the next `backend-guidelines-reviewer` audit. |

---

## 10. Acceptance criteria — deltas from the PRD

The PRD's §10 stands except where D7 changes the mechanism:

- Criteria 1–3 (`sweep.sh` reports N entries) become **execution-time gates**.
  The final `sweep.sh` run's verbatim output is recorded in the task's
  `progress.md` before the scaffolding-removal commit. The counts required are
  unchanged: 0 `NewModelBuilder`, 0 FILE-05 entries outside `exemptions.md`,
  and DOM-04 inventories containing only exempted packages.
- Criterion 4 (`exemptions.md` with `file:line` evidence) is unchanged and is
  now the branch's only durable conformance artifact.
- All remaining criteria — round-trip tests, flagless `tools/verify.sh` exit 0,
  no FAIL on DOM-01/DOM-04/FILE-05 outside the exemption list, FR-17/FR-18
  diff checks, and code review before the PR — are unchanged.

One addition: the codemod's ledger (§4.1) must show
`APPLIED + SKIPPED == input inventory` for all three subcommands, and every
`SKIPPED` package must appear either in a hand-work commit or in
`exemptions.md`. A package that is skipped and then forgotten is the specific
failure this task exists to end.
