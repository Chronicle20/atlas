# Task 8 report — W2: apply the rename to the remaining services (PARTIAL)

## Fact block

```
task=task-263-backend-guideline-conformance
worktree=<repo-root>/.worktrees/task-263-backend-guideline-conformance
branch=task-263-backend-guideline-conformance
go_version=go1.27.0 (module-local builds via workspace go.work; codemod itself run with GOWORK=off from its own module dir)
```

## Deviations from the brief's literal commands (both previously flagged by Tasks 6/7, reconfirmed here)

1. `GOWORK=off go run ./docs/.../codemod rename ...` from the worktree root fails (`cannot find main module`,
   only a `go.work` at root). Ran `cd docs/tasks/task-263-backend-guideline-conformance/codemod && GOWORK=off go run . rename -repo <worktree-abs> -only services/<S> -ledger <path>` instead — same as Tasks 6/7.
2. **`-append` does not exist** on the `rename` subcommand (only `-repo`, `-inventory`, `-ledger`, `-only`,
   `-dry-run` — confirmed via `-h` usage output). `runRename` always `os.WriteFile`s the ledger (overwrite, not
   append). So I ran each service into its own `/tmp/ledger-<service>.tsv`, and the final combined
   `ledger-rename-rest.tsv` (Step 3/4 of the brief) still needs to be assembled by concatenating every
   per-service temp ledger once all services are processed. **This is unfinished** — see "Remaining work."

## atlas-character — confirmed already done, skipped per brief

`services/atlas-character/atlas.com/character/character/model.go` — Task 5 already renamed the constructor to
`NewEmptyBuilder`; the type is `modelBuilder` (unexported, so it was never a `ModelBuilder`/ FR-13 case in the
first place — nothing case-mismatched to fix). No further action needed; not touched in this task.

## Services completed and committed (9 of 22 remaining, plus atlas-character confirmed done)

| Service | Ledger (APPLIED/SKIPPED) | Hand fixups (stale comments/test names) | Commit |
|---|---|---|---|
| atlas-query-aggregator | 6 APPLIED | marriage/party/party_quest/quest/skill doc comments (`ModelBuilder`→`Builder`, `NewModelBuilder`→`NewBuilder` in prose) | `83ce8faf1` |
| atlas-pets | 5 APPLIED | `data/cash/builder.go` doc comment, `pet/resource.go` doc comment | `dbe0d89c8` |
| atlas-monsters | 3 APPLIED | `information/builder.go`, `mobskill/builder.go`, `consumable/builder.go`, `consumable/model.go` doc comments; `information/rest_test.go:TestModelBuilderSetFirstAttack`→`TestBuilderSetFirstAttack`; `banish_producer_test.go:TestModelBuilderSetBanish`→`TestBuilderSetBanish` | `09a131d3a` |
| atlas-world | 2 APPLIED | `world/builder.go`, `channel/builder.go` doc comments; both `builder_test.go:TestNewModelBuilder`→`TestNewBuilder` | `4722fa921` |
| atlas-tenants | 2 APPLIED | `configuration/builder.go`, `tenant/builder.go` doc comments; both `builder_test.go:TestNewModelBuilder`→`TestNewBuilder` | `79e9070cd` |
| atlas-skills | 2 APPLIED | `macro/builder_test.go`, `skill/builder_test.go`: `TestNewModelBuilder`→`TestNewBuilder` | `a8267e42c` |
| atlas-quest | 1 APPLIED (`quest/progress`), 1 **SKIPPED** (`quest/quest`) | `quest/progress/builder_test.go:TestNewModelBuilder_CreatesEmptyBuilder`→`TestNewBuilder_CreatesEmptyBuilder` | `c568e0449` |
| atlas-monster-book | 2 APPLIED | none needed | `2c6bb3703` |
| atlas-consumables | 2 APPLIED | none needed (remaining `ModelBuilder` hits are the pre-existing `asset`/`compartment`/`inventory` Clone-based pattern, and `RewardModelBuilder`, both out of DOM-01 scope — see below) | `f5bcc1952` |
| atlas-storage | 1 APPLIED | `storage/builder.go` doc comment | `e8e28fc1b` |

Every commit above: `go build ./... && go vet ./... && go test ./...` from the service's own module root, all
clean (no FAIL, no vet output) before staging `services/<S>` and committing.

### The `atlas-quest/quest/quest` SKIP — reconfirmed, left in place

```
services/atlas-quest/atlas.com/quest/quest	SKIPPED	builder identifier already in scope
```
Genuine R2 collision (per Task 6's report: `processor.go`'s local `builder := sagaproducer.NewBuilder(...)`
collides with the rename target `builder`). Confirmed `quest/quest/builder.go:22` still has
`func NewModelBuilder() *modelBuilder`, untouched, reserved for hand work as ruled by the controller. Not
invented a name for it.

### mistcast.go cross-service comment — resolved (no change needed)

Brief asked me to resolve the stale comment in
`services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast.go:42` referring to "atlas-monsters'
ModelBuilder.AddStatusEffect". I traced it: `AddStatusEffect` lives on
`services/atlas-monsters/atlas.com/monsters/monster/builder.go`'s `ModelBuilder` type — the **top-level**
`monster` package, not one of the 3 sub-packages (`consumable`, `information`, `mobskill`) that were in the
DOM-01 `NewModelBuilder` inventory. `monster/builder.go` has **no `NewModelBuilder` constructor** (it uses
`Clone(m Model) *ModelBuilder` instead), so it was never in `inventory-dom01-newmodelbuilder.txt` and the
codemod's `-only services/atlas-monsters` run correctly did not touch it (targets are sourced from the
inventory file, not a live grep). Since `monster.ModelBuilder` is genuinely out of this task's inventory-bound
scope (same category as the `asset`/`compartment`/`inventory` Clone-based packages found in
query-aggregator/pets/consumables — pre-existing constructor-already-`NewBuilder`-but-type-still-`ModelBuilder`
mismatches that predate task-263 entirely and were never captured by the `func NewModelBuilder(` grep), the
mistcast.go comment remains **factually accurate as written** — no edit was made, and none is needed until/unless
a future task decides to also rename these out-of-inventory `ModelBuilder` types.

**This surfaces a genuine finding for the controller**: DOM-01/FR-16's repo-wide "no `ModelBuilder` outside
`libs/atlas-packet`" acceptance bar (Step 2 of this brief) will **not** be met purely by processing the
59-entry inventory, because at least these packages have a live, unrenamed `ModelBuilder` type with a
different constructor name (`Clone`, not `NewModelBuilder`) and so were never in the inventory:
- `atlas-monsters/monster` (top-level, `Clone`-based) — the one mistcast.go references
- `atlas-query-aggregator/asset`, `atlas-query-aggregator/compartment`, `atlas-query-aggregator/inventory`
- `atlas-pets/asset`, `atlas-pets/compartment`, `atlas-pets/inventory`
- `atlas-consumables/asset`, `atlas-consumables/compartment`, `atlas-consumables/inventory`
These are likely to recur in the remaining services too (`asset`/`compartment`/`inventory` packages are a
common per-service shape across the monorepo's inventory-consuming services). I have not swept the whole repo
for every instance — flagging the pattern, not a complete inventory of it, per "don't invent, don't
paraphrase, don't chase repo-wide fixes I wasn't asked for."

## Continuation (segment 2): the remaining 12 services + ledger assembly — DONE

Processed all 12 remaining services following the exact recipe established above. Each: codemod
`rename -only services/<S> -ledger /tmp/ledger-atlas-<S>.tsv`, `go build ./... && go vet ./... &&
go test ./...` from the module root (all clean, no FAIL/vet output, before staging), grep for
stale `NewModelBuilder`/`ModelBuilder` text and hand-fix genuine doc-comment/test-name staleness
(leaving the out-of-inventory `asset`/`compartment`/`inventory`/`commodities`/`shops` Clone-based
pattern alone, confirmed case-by-case per service), then `git add services/<S>` + commit.

| Service | Ledger | Hand fixups | Commit |
|---|---|---|---|
| atlas-rps | 1 APPLIED (`game`) | `game/builder.go` doc comments (`ModelBuilder`/`NewModelBuilder`/`CloneModelBuilder`→`Builder`/`NewBuilder`/`CloneBuilder`); `processor_test.go` comment; `model_test.go`: `TestModelBuilderRoundTripsThroughJSON`→`TestBuilderRoundTripsThroughJSON`, `TestModelBuilderRejectsZeroCharacter`→`TestBuilderRejectsZeroCharacter` | `8d79ab916` |
| atlas-reactors | 1 APPLIED (`reactor`) | `reactor/builder_test.go`: `TestNewModelBuilder`→`TestNewBuilder`, 7× `TestModelBuilder_*`→`TestBuilder_*` | `1ab86262a` |
| atlas-npc-shops | 1 APPLIED (`character`) | none needed on `character/model.go`; left `commodities`, `compartment`, `inventory`, `shops` packages' pre-existing `type ModelBuilder`/constructor-`NewBuilder` mismatch alone (out-of-DOM-01-inventory, confirmed) | `1976b4593` |
| atlas-mounts | 1 APPLIED (`mount`) | `mount/builder.go` doc comment, `mount/model.go` doc comment, `docs/domain.md` (2 hits) | `156c4e87d` |
| atlas-messages | 1 APPLIED (`character`) | none needed | `44cbf42aa` |
| atlas-maps | 1 APPLIED (`reactor`) | `reactor/model_test.go`: `TestNewModelBuilder`→`TestNewBuilder`, 6× `TestModelBuilder_*`→`TestBuilder_*`; left `tasks/mist_tick.go`'s comment referencing `atlas-monsters`' out-of-inventory `monster.ModelBuilder` alone (confirmed accurate per segment-1 finding) | `1f7a9e33f` |
| atlas-keys | 1 APPLIED (`key`) | `key/builder.go` doc comments (`ModelBuilder`/`NewModelBuilder`/`CloneModelBuilder`→`Builder`/`NewBuilder`/`CloneBuilder`); `builder_test.go`: 5× `TestModelBuilder_*`→`TestBuilder_*`, `TestCloneModelBuilder_CopiesAllFields`→`TestCloneBuilder_CopiesAllFields` | `3d0e1bec2` |
| atlas-inventory | 1 APPLIED (`data/cash`) | `data/cash/builder.go` doc comment; left `asset`, `compartment`, `inventory` packages' pre-existing out-of-inventory `ModelBuilder` type alone (confirmed via grep: constructor already `NewBuilder`, type still `ModelBuilder`) | `d119bcec6` |
| atlas-expressions | 1 APPLIED (`expression`) | `expression/builder.go` doc comments (3); `builder_test.go`: `TestNewModelBuilder`→`TestNewBuilder`, 12× `TestModelBuilder_*`→`TestBuilder_*`, 2× `TestCloneModelBuilder*`→`TestCloneBuilder*` | `503af8ce2` |
| atlas-drops | 1 APPLIED (`drop`) | `drop/model_test.go`: `TestNewModelBuilder_DefaultValues`→`TestNewBuilder_DefaultValues`, 6× `TestModelBuilder_*`→`TestBuilder_*`, `TestCloneModelBuilder_CopiesAllFields`→`TestCloneBuilder_CopiesAllFields` | `38c828fb7` |
| atlas-data | 1 APPLIED (`skill/effect`) | `skill/effect/model.go` doc comment (2 references to `ModelBuilder`→`Builder`) | `27ff6f209` |
| atlas-cashshop | 1 APPLIED (`character`) | none needed on `character/model.go`; left `cashshop/inventory/asset`, `cashshop/inventory/compartment`, `cashshop/inventory`, `asset`, `character/compartment`, `character/inventory` packages' pre-existing out-of-inventory `ModelBuilder` type alone (confirmed) | `6fbc7f067` |

Every commit: `go build ./... && go vet ./... && go test ./...` clean from the service's own module
root before staging `services/<S>` and committing. No `*_testhelpers.go` added; no domain types
invented; codemod only touches `*ast.Ident` nodes and all hand edits were doc comments/test
function names, never string literals or logic.

### Step 2 — repo-wide confirmation (run from worktree root)

```
grep -rn --include='*.go' 'func NewModelBuilder' services libs | grep -v 'libs/atlas-packet/'
```
Output:
```
services/atlas-quest/atlas.com/quest/quest/builder.go:22:func NewModelBuilder() *modelBuilder {
```
Count: **1**. This matches the controller's ruling exactly — the sole remaining hit is the
`atlas-quest/quest` R2 collision (local `builder := sagaproducer.NewBuilder(...)` in
`processor.go` collides with the rename target), reserved for hand work in a later task. **PASS.**

Informational second grep (bare `ModelBuilder`, expected non-clean per the segment-1 finding —
out-of-DOM-01-inventory `Clone`-based builder types, out of this task's scope):
```
grep -rln --include='*.go' 'ModelBuilder' services libs | grep -v 'libs/atlas-packet/' | wc -l
```
Output: **56** files. Confirms the real, separately-tracked gap flagged in segment 1 — a
controller decision on whether to fold these into a follow-up task remains open. Not expanded
into here per the brief's explicit instruction.

### Step 3/4 — combined ledger

Assembled `docs/tasks/task-263-backend-guideline-conformance/ledger-rename-rest.tsv` by
concatenating, in order: the 6 previously-recovered `atlas-query-aggregator` lines that were
already sitting in that file (untracked leftover from an earlier partial attempt, verified to
match the segment-1 report's table exactly), the 9 remaining segment-1 services' per-service temp
ledgers (`pets`, `monsters`, `world`, `tenants`, `skills`, `quest`, `monster-book`, `consumables`,
`storage` — all still present in `/tmp` and unchanged since segment 1), this segment's 12 new
per-service temp ledgers, and `atlas-channel`'s own `ledger-rename-channel.tsv` from Task 7 (19
entries). Total: **58 lines**, `cut -f2 | sort | uniq -c` → `57 APPLIED`, `1 SKIPPED`. Committed as
`49106bbcd chore(task-263): record combined builder-rename ledger`.

Per the brief's git instructions, `progress.md` and `agent-ledger.tsv` were **not** touched —
those remain the controller's; both still show as modified in `git status` throughout, as
expected.

## Final state

All 22 services in the DOM-01 `NewModelBuilder` inventory are now processed (10 in segment 1 + 12
in segment 2), plus `atlas-channel` (Task 7) and `atlas-character` (Task 5) from earlier tasks.
Repo-wide `func NewModelBuilder` outside `libs/atlas-packet` is down to exactly 1 (the expected
`atlas-quest/quest` SKIP). Task is complete.

## Remaining work (historical — 13 services + finish-up steps, superseded by the continuation above)

Services still to process, each following the exact same recipe used above (codemod run scoped `-only
services/<S>` into its own ledger file, build+vet+test from the module root, grep for stale
`NewModelBuilder`/`ModelBuilder` text and hand-fix genuine doc-comment/test-name staleness — distinguishing
that from the pre-existing out-of-scope `asset`/`compartment`/`inventory` Clone-based pattern described above,
which must be left alone — then `git add services/<S>` + commit):

- `services/atlas-rps` (1 — `game/builder.go:29`)
- `services/atlas-reactors` (1 — `reactor/builder.go:36`)
- `services/atlas-npc-shops` (1 — `character/model.go:330`; module root is `services/atlas-npc-shops/atlas.com/npc`, module name `npc`, **not** `npc-shops`)
- `services/atlas-mounts` (1 — `mount/builder.go:22`)
- `services/atlas-messages` (1 — `character/model.go:310`)
- `services/atlas-maps` (1 — `reactor/model.go:97`)
- `services/atlas-keys` (1 — `key/builder.go:14`)
- `services/atlas-inventory` (1 — `data/cash/builder.go:13`)
- `services/atlas-expressions` (1 — `expression/builder.go:26`)
- `services/atlas-drops` (1 — `drop/model.go:288`)
- `services/atlas-data` (1 — `skill/effect/builder.go:9`)
- `services/atlas-cashshop` (1 — `character/model.go:379`)

Exact next-step command template (proven working in this session), run from the worktree root:
```
cd docs/tasks/task-263-backend-guideline-conformance/codemod && GOWORK=off go run . rename \
  -repo <worktree-abs-path> \
  -only services/<S> -ledger /tmp/ledger-<S>.tsv && cat /tmp/ledger-<S>.tsv
```
then from `services/<S>/atlas.com/<module>`: `go build ./... && go vet ./... && go test ./...`, then
`grep -rln --include='*.go' 'NewModelBuilder\|ModelBuilder' services/<S>` to find stale doc-comment/test-name
text (fix only the genuine ones — the two categories to leave alone are (a) the pre-existing
`asset`/`compartment`/`inventory` Clone-based `ModelBuilder` pattern described above, and (b) any genuine FR-15
distinct-type builder like `RewardModelBuilder`/`SkillModelBuilder`), then `git add services/<S>` + commit
`refactor(<S>): rename ModelBuilder to Builder`.

### After all 22 services are done

- **Step 2 (FR-16/FR-18 repo-wide confirmation)**: run both greps from the brief. Given the finding above, the
  second grep (bare `ModelBuilder`) is expected to show more than just `libs/atlas-packet` — it will show every
  out-of-inventory `asset`/`compartment`/`inventory`-style package plus `atlas-monsters/monster`. Record the
  actual output; do not force a false "clean" result — this is a real, separate finding for the controller to
  decide whether it's in-scope for a follow-up task or was already anticipated as "hand work."
- **Step 3 (combined ledger)**: concatenate every per-service temp ledger (`/tmp/ledger-*.tsv` — currently:
  `query-aggregator`, `pets`, `monsters`, `world`, `tenants`, `skills`, `quest`, `monster-book`, `consumables`,
  `storage`, plus channel's own `ledger-rename-channel.tsv`) into
  `docs/tasks/task-263-backend-guideline-conformance/ledger-rename-rest.tsv`, run the `cut -f2 | sort | uniq -c`
  and `wc -l` checks from the brief, and record the verbatim output in `progress.md`.
- **Step 4**: `git add docs/tasks/task-263-backend-guideline-conformance/ledger-rename-rest.tsv
  docs/tasks/task-263-backend-guideline-conformance/progress.md` + commit
  `chore(task-263): record builder rename ledger`. (Per this task's instructions I did **not** touch
  `progress.md` or `agent-ledger.tsv` myself — those remain the controller's.)

**Note**: none of the 10 per-service temp ledgers above (`/tmp/ledger-*.tsv`) are inside the worktree/repo, so
they will not survive if the sandbox/tmp is cleared between agent sessions. A continuation should either copy
them into the worktree first, or simply re-run the (idempotent, already-verified) codemod commands for the 10
completed services again into a durable location before assembling the combined ledger — the counts are
already known and quoted in the table above if the temp files are gone.

## Self-review

- Every commit's module build+vet+test was run and confirmed clean before staging; no commit was made on a
  failing build.
- Every stale doc-comment/test-name fixup was verified as a genuine post-rename staleness (the old name no
  longer resolves to anything in the package) before editing, not a blanket sed sweep — checked each hit's
  context individually.
- Correctly distinguished and left alone: the `atlas-quest/quest/quest` R2 collision SKIP; the FR-15
  distinct-type builders (`RewardModelBuilder`, `SkillModelBuilder`); the pre-existing, out-of-DOM-01-inventory
  `asset`/`compartment`/`inventory` Clone-based `ModelBuilder` packages (not renamed, not part of this task's
  target set, confirmed via `git log`/`git show main:...` that they predate this branch).
- No `*_testhelpers.go` added; no new domain types invented; no literal string/tag changed (codemod only
  touches `*ast.Ident` nodes, and my hand edits were all doc comments/test function names, never string
  literals).

## Concerns for the controller/reviewer

1. **The out-of-inventory `ModelBuilder` finding above is real and will make the brief's Step 2 "expect no
   output" check fail** even after all 22 services are processed — this is not something Task 8 can fix
   without expanding scope beyond the DOM-01 inventory (which the brief and Contract 3 both say not to do
   unilaterally). Needs a controller decision: either accept it as a known, separately-tracked gap, or spin up
   a follow-up task to extend the inventory to `Clone`-based builders.
2. **13 services remain unprocessed**: `atlas-rps`, `atlas-reactors`, `atlas-npc-shops`, `atlas-mounts`,
   `atlas-messages`, `atlas-maps`, `atlas-keys`, `atlas-inventory`, `atlas-expressions`, `atlas-drops`,
   `atlas-data`, `atlas-cashshop` (all 1-declaration services, mechanically identical work to what's done above).
3. **Combined ledger (Step 3/4) not yet assembled** — needs all 22 services' temp ledgers concatenated once the
   remaining 12 are done.
