# task-263 — Plan Context

Companion to [plan.md](plan.md). Records what the plan phase derived, where it
departs from [design.md](design.md), and what an executor needs that the task
sections do not repeat.

Branch point: `eaa5ce6f7`. Plan written at `47939df2f`.

---

## 1. Key facts, re-derived at plan time

Every count below was confirmed against the tree in this worktree, not copied
from the design's prose.

| Fact | Value | How confirmed |
|---|---|---|
| `inventory-dom01-newmodelbuilder.txt` | 59 rows | `wc -l` |
| `inventory-dom01-other.txt` | 69 rows | `wc -l` |
| `inventory-dom04-has-model.txt` | 185 rows | `wc -l` |
| `inventory-dom04-no-model.txt` | 176 rows | `wc -l` |
| `inventory-file05-builders.txt` | 100 rows, 72 distinct packages | `wc -l`; `cut -d: -f1 \| xargs -n1 dirname \| sort -u` |
| FILE-05 packages that already have `builder.go` | 6 | `atlas-character/character`, `atlas-families/family`, `atlas-quest/quest`, `atlas-quest/quest/progress`, `atlas-tenants/configuration`, `atlas-tenants/tenant` |
| `entity_builder.go` declarations | 4 | `quest/quest:10`, `quest/quest/progress:5`, `tenants/configuration:9`, `tenants/tenant:5` |
| `reference_data.go` builders | 7 | lines 73, 397, 684, 738, 785, 842, 930 |
| `libs/atlas-packet` FILE-05 entry | 1 | `model/skill_usage_info.go:143` |
| Alias-comment constructors | 6 | `grep -rn 'alias for NewModelBuilder'` — channel/{world:24, note:25, transport/route:41, compartment:33, inventory:26, channel:32} |
| `has-model` `rest.go` with exactly one `^func Extract` | 158 | `xargs grep -c` over the inventory |
| `has-model` `rest.go` with zero / two+ | 13 / 14 | same |
| `golang.org/x/tools` in the module cache | v0.49.0 (and 14 other versions) | `ls ~/go/pkg/mod/golang.org/x/` |
| `gofumpt`, `goimports`, `gopls`, `gorename` | not installed | `which` |
| Go toolchain | go1.27.0 | `go version` |

The design's tier split (A=104, B1=37, B2=37, C=7) is consistent with the 158/13/14
split above once body flatness is applied, but the classifier that produced it was
not persisted. **Task 2 rebuilds it as a codemod subcommand**, and every later task
reads the derived TSV rather than the design's numbers. Task 2 Step 5 requires the
derived counts be pasted into `progress.md` and flags a divergence of more than ±3
as a finding.

---

## 2. Departures from the design

### 2.1 An eighth constructor collision the design missed — `atlas-character/character`

Design §2.3 lists **seven** packages declaring both `NewModelBuilder` and
`NewBuilder`, all in `atlas-channel`. The tree has **eight**:
`services/atlas-character/atlas.com/character/character` declares
`func NewBuilder(c BuilderConfiguration, accountId uint32, worldId world.Id, name string, skinColor byte, gender byte, hair uint32, face uint32) *Builder`
at `builder.go:91` and `func NewModelBuilder() *modelBuilder` at `model.go:242`,
with **88 call sites** across `services/atlas-character`.

They are genuinely distinct — one is the character-creation builder applying
`BuilderConfiguration` defaults, the other the zero-value model-reconstruction
builder used by the Kafka consumer (`kafka/consumer/character/consumer.go:371`)
and the REST layer (`provider.go:60`), with a companion `CloneModel` at
`model.go:246`.

**Plan Task 5 handles it** under D3's `channel/asset` precedent, with two
sub-decisions the design does not make:

1. `NewModelBuilder()` → `NewEmptyBuilder()`, satisfying FR-16 without colliding.
2. `modelBuilder` is **exempt from FR-13**. Renaming it to `builder` would leave
   `Builder` (builder.go:91) and `builder` in one package differing only in case.

Both go into `exemptions.md` under a new section, `## DOM-01 — NewBuilder name
already taken`, alongside `channel/asset`.

**This is the one substantive judgment call the plan makes that the design did
not authorise.** If `NewEmptyBuilder` is the wrong name, it is a one-line change
in Task 5 Step 2 plus its 88 call sites; nothing downstream depends on it.

### 2.2 Workstream order is W2 → W3 → W1, not D8's W1 → W2 → W3

Design §D8 orders development W1 → W2 → W3; design §7 orders *landing* W2 → W3 → W1
and calls this "inverting the development order of D8". The plan resolves the
conflict by using **W2 → W3 → W1 for both**, because:

- D8's stated reason for W1-first — "moving a builder into `builder.go` gives W2 a
  single file per package to rename in" — does not hold. W2's rename is type-aware
  (driven by `types.Object` identity across every module in `go.work`), so it is
  entirely indifferent to which file a declaration lives in.
- W3's round-trip tests construct `Model` through the package's builder, so W3 does
  depend on W2's rename having landed. It does **not** depend on W1, which changes
  no signature.
- §7's ordering is the operative constraint: it resolves PRD open question 4 and it
  is what keeps W1's `model.go` rewrites — the conflict generator against the nine
  in-flight worktrees (task-240, 241, 246, 250, 251, 254, 256, 259, 262) — last.

### 2.3 The generated round-trip test prefers the builder but falls back conditionally

Design §6 requires construction through the package's builder, falling back to a
composite literal "where a package has no builder". The plan's Task 10 adds a third
condition: the builder is used only when it exists **and** every field `Extract`
maps has a matching `Set<Field>` setter resolvable from `TypesInfo`. A builder that
cannot set a mapped field would produce a test that silently leaves that field at
its zero value — the exact failure §6's "every field distinct and non-zero"
property exists to prevent.

`services/atlas-monster-book/atlas.com/monster-book/data/consumable` is the
composite-literal case: it declares no builder at all.

### 2.4 The generator skips rather than guesses

Design §4.4 step 4 type-checks before writing. The plan adds three skip conditions
the design does not enumerate, all of which route a package to hand work rather
than to a weak test (Task 10):

- two or more `bool` fields — no way to give them distinct values
- any field whose type is not integer / string / bool / float / `uuid.UUID` /
  `time.Time` — no derivable distinct non-zero value
- generated code that does not type-check

---

## 3. Task sizing

Twenty-seven tasks. The F4 warning (>6 files, or >1 service) fires on many of them
**by design**, because they are the same mechanical change repeated — the case
Step 5a explicitly permits batching. Specifically:

| Task | Why deliberately large |
|---|---|
| 5 | 88 call sites, but one service and one identifier; a scripted sweep |
| 8 | 23 services, but one commit per service and the identical codemod invocation |
| 12 | ~40 services, same — one commit per service, one command per service |
| 13 | 3 services, but it is deliberately the *first* hand-work task: the four #1498 packages between them exercise every hand-work shape (FR-2 named variants, FR-3 five-compartment, plain tier A), so the pattern the later tasks copy is established once |
| 14 | 8 services / 11 packages, but the identical treatment repeated — enumerate `Extract*`, write the mirrored `Transform*`, write the round-trip test, record the exemption evidence |
| 15, 16 | ~37 packages each, but explicitly partitioned into one-service batches for `atlas-implementer` fan-out at dispatch time |
| 20, 21 | The 72 FILE-05 packages split in half; one commit per service |
| 23 | 20 builder sets in one 2430-line file, but a single file pair in a single package |
| 27 | 7 files, all in the task folder; it is a single closeout sequence (final sweep evidence → gate → review → delete scaffolding) whose steps are strictly ordered and cannot be split without losing the "record the evidence before deleting the script that produced it" property D7 depends on |

Tasks 15 and 16 are the **only** places in this plan where fanning out implementer
agents is correct (design §5, `docs/codemod-vs-agents.md`). The 100 FILE-05 moves
(Tasks 19–23) and the 59 DOM-01 renames (Tasks 6–9) are codemod work and must not
be handed to agents at the same transformation — that is PRD §8's codemod-first
clause.

---

## 4. Files an executor should read before starting

| Purpose | Path |
|---|---|
| The tier-A `Extract` shape the codemod inverts | `services/atlas-monster-book/atlas.com/monster-book/data/consumable/rest.go:40-42` |
| The house `Transform` shape | `services/atlas-ban/atlas.com/ban/ban/rest.go:36-47` |
| The house named-variant shape (`TransformCheck`) | `services/atlas-ban/atlas.com/ban/ban/rest.go:91` |
| The house `TestTransform` shape | `services/atlas-ban/atlas.com/ban/ban/rest_test.go:10-45` |
| The alias-constructor shape Task 3 merges | `services/atlas-channel/atlas.com/channel/inventory/builder.go:18-29` |
| The divergent-arity case (D3) | `services/atlas-channel/atlas.com/channel/asset/builder.go:119,126` |
| The eighth collision (§2.1 above) | `services/atlas-character/atlas.com/character/character/builder.go:91`, `model.go:242` |
| The five-compartment `Transform` case (FR-3) | `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go:17-100` |
| The named-`Extract` case (FR-2) | `services/atlas-channel/atlas.com/channel/monsterbook/rest.go:102,107` |
| The module enumeration the gate uses | `tools/verify.sh:181-186` |

Module roots for `go build` / `go test`, from `plan-context.sh`: `services/<S>/atlas.com/<short-name>`
(e.g. `services/atlas-channel/atlas.com/channel`,
`services/atlas-npc-conversations/atlas.com/npc`,
`services/atlas-monster-book/atlas.com/monster-book`,
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`), plus
`libs/atlas-constants`, `libs/atlas-database`, `libs/atlas-script-core`,
`libs/atlas-packet`.

---

## 5. Dependencies and gotchas

- **`GOWORK=off` is mandatory for every codemod command.** The repo's root
  `go.work` lists 96 modules and would otherwise capture the codemod module,
  which D6 requires stay invisible to `tools/verify.sh:182`'s
  `find services libs -name go.mod` enumeration.
- **`golang.org/x/tools` must be pinned to v0.49.0.** That version is already in
  the module cache, so the codemod builds without a network fetch. Any other
  version may not be.
- **No `gofumpt` in this environment.** The codemod formats with `format.Source`
  from the standard library. `tools/verify.sh` runs no gofmt check, so this is
  sufficient; do not add a formatter dependency.
- **Preserve line endings.** CLAUDE.md forbids a CRLF→LF normalisation as a side
  effect. Both `rename.go` and `relocate.go` must read the original bytes, detect
  `\r\n`, and re-apply it on write. `relocate_test.go` has a case for this.
- **`tools/verify.sh` runs per *changed* module.** A branch touching ~40 services
  makes the flagless run long. Dispatch `atlas-verifier` for it (Task 27 Step 3)
  rather than running it in a large context.
- **`CloneModel` is not `CloneModelBuilder`.** FR-14 renames only the latter.
  `CloneModel(m Model) *modelBuilder` appears in `channel/inventory`,
  `channel/transport/route`, and `atlas-character/character` and keeps its name;
  its return type changes with FR-13, its name does not. Task 6's test asserts this.
- **`sweep.sh` and `split-by-model.sh` must be re-run from the worktree root**, not
  from the task folder — they hard-code `OUT="docs/tasks/task-263-..."` relative to
  the repo root.

---

## 6. Accepted residue

- **R3 will fire at review.** `backend-guidelines-reviewer` follows the literal
  `func Transform(` grep at `file-responsibilities.md:193` and will re-raise DOM-04
  on the 14 `NO-RESTMODEL` packages, which cannot host that signature because no
  `type RestModel` exists. D2 accepts this and answers with `exemptions.md`. Any
  *other* DOM-01/DOM-04/FILE-05 FAIL is real.
- **R6: `sweep.sh` does not survive the branch.** D7 deletes it; the evidence is the
  pre-deletion run recorded verbatim in `progress.md` (Task 27 Steps 1–2). This
  drops PRD goal 3 and converts acceptance criteria 1–3 into execution-time gates.
  Regression detection reverts to the next `backend-guidelines-reviewer` audit.
- **Design §2.3's collision count is 7; the tree's is 8.** Recorded in §2.1 above
  and handled by Task 5. The design is not amended — the plan carries the
  correction.
