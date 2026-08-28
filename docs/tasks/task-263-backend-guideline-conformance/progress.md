# task-263 backend-guideline-conformance — progress

## Task 2 — derived classification

Produced by `classify` (`docs/tasks/task-263-backend-guideline-conformance/codemod`), run
from the worktree root:

```
GOWORK=off go run ./docs/tasks/task-263-backend-guideline-conformance/codemod classify \
  -repo . -out docs/tasks/task-263-backend-guideline-conformance
```

Row counts (sum checks, all exact): `classify-dom04.tsv` 185, `classify-dom01-fr15.tsv` 69,
`classify-file05.tsv` 100.

### DOM-04 tier (`cut -f2 classify-dom04.tsv | sort | uniq -c`)

```
     94 A
     11 B1
     61 B2
      7 C
     12 NO-RESTMODEL
```

Design §2.1 expectation: A≈104 B1≈37 B2≈37 C≈7, NO-RESTMODEL=14.

### DOM-01 FR-15 class (`cut -f2 classify-dom01-fr15.tsv | sort | uniq -c`)

```
      5 NO-MODEL-GO
      3 RENAME
     61 SIBLING-EXEMPT
```

Design §2.4 expectation: RENAME=9, SIBLING-EXEMPT=55, NO-MODEL-GO=5.

### FILE-05 disposition (`cut -f4 classify-file05.tsv | sort | uniq -c`)

```
      4 ENTITY-BUILDER
      1 EXCLUDED-TREE
      2 HAS-BUILDER-GO
     93 RELOCATE
```

Design §2.5 expectation: RELOCATE≈89, HAS-BUILDER-GO=6, ENTITY-BUILDER=4, EXCLUDED-TREE=1.

### Divergence finding

Every bucket except DOM-04 `C` (7=7), FILE-05 `ENTITY-BUILDER` (4=4) and FILE-05
`EXCLUDED-TREE` (1=1) diverges from the design's §2 prose numbers by more than ±3:

- DOM-04: A −10, B1 −26, B2 +24, NO-RESTMODEL −2 (within tolerance).
- FR-15: RENAME −6, SIBLING-EXEMPT +6.
- FILE-05: RELOCATE +4, HAS-BUILDER-GO −4.

Spot checks against source (`services/atlas-channel/atlas.com/channel/account/rest.go`,
`.../merchant`, `.../messenger`, `.../party`, `services/atlas-trades/.../escrow`,
`services/atlas-maps/.../map/timer`) confirm the derived tiers are correct per the rules in
task-2's brief — e.g. `account`'s `Extract` is genuinely non-flat (calls `strconv.ParseUint`
and a builder chain before returning), and several single-builder DOM-01 packages
(`escrow.NewItemBuilder`, `map/timer.NewEntryBuilder`) build a domain type (`Item`, `Entry`)
rather than a type literally named `Model`, so they are correctly `SIBLING-EXEMPT` rather than
`RENAME` under the brief's literal rule ("its return type's `Build()` returns the package's
`Model`"). The design's §2 counts were never persisted as a script (this task's premise) and
read as a rough manual estimate; the derived TSVs above are the authoritative counts going
forward, and downstream tasks should size against them, not the design prose.

## Tasks 2-3 review (backfilled) — APPROVED_WITH_FINDINGS

Artifact: `review-task-2-3.md`. 0 blocking, 2 non-blocking. Controller ruling: both are
carried into **Task 10** (`transform` subcommand), which is the first task whose correctness
depends on tier `A` actually meaning "safe to auto-collapse":

1. `codemod/classify.go:412-429` — `isFlatFieldValue` checks a composite-literal field
   value's base is *some* identifier, not that it is `Extract`'s actual parameter. Not
   exploited by any of the 94 real tier-`A` packages today. Tighten before `transform`
   consumes tier `A`.
2. `codemod/classify_test.go:346-416` — no `EXCLUDED-TREE` case for the
   `libs/atlas-packet/` prefix branch. Add the case.

Reviewer also noted a pre-existing, unrelated `go.sum` gap that makes `GOWORK=off go build ./...`
fail in `services/atlas-channel` on both sides of Task 3 — not caused by this branch; the
workspace `go.work` build passes. Flagged for Task 27's final gate.

## Task 5 — complete

Commit `c861e29b4` — `atlas-character/character.NewModelBuilder` → `NewEmptyBuilder`,
88 call sites across 19 files. `type modelBuilder`, `CloneModel`, `type Builder` and
`NewBuilder` left unchanged per the plan's exemption ruling (to be recorded in
`exemptions.md` in Task 25). Review APPROVED (`review-task-5.md`).

Gate `tools/verify.sh --quick --base 83adfa9ea` → PASS. **Last gated commit: `c861e29b4`.**
Earlier range `eb7100154..83adfa9ea` (Tasks 1-4) also PASS.

## Task 6 — complete

Commit `27c5fe431` — codemod `rename` subcommand (type-aware, driven by `types.Object`
identity). Tree-wide dry run: 59 ledger lines, **58 APPLIED, 1 SKIPPED**. The skip is
`atlas-quest/quest`, a genuine R2 collision that needs hand work in a later task —
carry this forward. True remaining `NewModelBuilder` declaration count is **51**, not
the plan's estimated 52.

Implementer concern recorded in `task-6-report.md`: a rename-local loader with
`Tests: true` was added in `rename.go` rather than changing shared `load.go`, to reach
`_test.go` call sites without regressing `classify.go`. Referred to the Task 6 review.

Gate `tools/verify.sh --quick --base c861e29b4` → PASS. **Last gated commit: `27c5fe431`.**

## Task 6 review — CHANGES_REQUIRED, blocking finding DISMISSED by the controller

Artifact: `review-task-6.md`. The single blocking finding — "~83 files of uncommitted,
undisclosed modifications to services/atlas-channel plus an untracked
`ledger-rename-channel.tsv`" — is a **stale-snapshot artifact**, not a defect. The
controller ran the Task 6 review concurrently with Task 7's implementer, which was at that
moment applying the rename to `atlas-channel` and writing that ledger. Evidence: those exact
82 files and the ledger are commit `fdcac2dbd` (Task 7), which landed while the review was
running. No action required.

The reviewer's sub-claim that `TestNewModelBuilder`→`TestNewBuilder` "cannot be produced by
the committed object-identity rename.go" is correct but not a defect either: `fdcac2dbd`'s
commit message discloses that test names and stale doc-comment text were brought up to date
by hand, following the Task 4 precedent. Verified in the diff.

**Process lesson: do not schedule a per-task review concurrently with an implementer that
touches the same paths.** Serialize the review against the tree it is reviewing, or gate it
on the next commit.

Non-blocking findings carried forward:

1. `task-6-report.md`'s stated evidence for needing `Tests: true` is wrong — the colliding
   `builder` local is in production code (`services/atlas-quest/atlas.com/quest/quest/processor.go:851,921`),
   reachable without it. The guard itself is correct; only the cited justification is.
2. `rename_test.go` never exercises `loadModuleForRename`'s `Tests: true` path — no
   regression coverage. Fold into a later codemod task.
3. The R2 collision guard covers only the `modelBuilder`/`builder` pair; `Builder` /
   `NewBuilder` / `CloneBuilder` collisions are unchecked. Residual risk for Tasks 7-9 —
   accepted, because such a collision surfaces as a compile failure in the per-service
   build rather than silently.

## Task 7 — complete

Commit `fdcac2dbd` — codemod `rename` applied to `atlas-channel`: 19 declarations renamed
across 82 files, all APPLIED, none SKIPPED. Ledger: `ledger-rename-channel.tsv`. Test names
and stale doc-comment text updated by hand in the same commit, per the Task 4 precedent.
Module `go build`/`go vet`/`go test` clean.

Gate `tools/verify.sh --quick --base 27c5fe431` → PASS. **Last gated commit: `fdcac2dbd`.**

Task 7 review not yet run — schedule it in the next session, serialized against the tree
(see the Task 6 review process lesson above).

## Task 8 — PARTIAL (segment 1 of 2)

Commits `83ce8faf1..e8e28fc1b` (10, one per service): `atlas-query-aggregator`, `atlas-pets`,
`atlas-monsters`, `atlas-world`, `atlas-tenants`, `atlas-skills`, `atlas-quest` (quest/progress
only — quest/quest SKIP correctly left intact), `atlas-monster-book`, `atlas-consumables`,
`atlas-storage`. Each service's module-local build/vet/test clean before staging.

Cap reached with 12 services remaining plus the combined ledger and repo-wide check.
Continuation brief: `.superpowers/sdd/plan/task-8-brief-cont.md`, dispatched to a fresh
implementer against the same report file.

Operational note from segment 1: the plan's `-append` flag **does not exist** on the codemod;
per-service temp ledgers merged afterwards is the working approach.

### Controller-verified counts at `e8e28fc1b`

- `func NewModelBuilder` outside `libs/atlas-packet/`: **13** — the 12 remaining services plus
  the `atlas-quest/quest` R2 collision. Expected end state for Task 8 is therefore **1**, not 0.

### FR-13 scope finding — NEEDS A USER DECISION, do not action silently

The task inventory (`inventory-dom01-newmodelbuilder.txt`) was derived from `NewModelBuilder`
**constructors** only. FR-13's `ModelBuilder`/`modelBuilder` → `Builder`/`builder` **type**
rename has a materially wider footprint:

```
grep -rn --include='*.go' -E 'type (M|m)odelBuilder' services libs | grep -v 'libs/atlas-packet/'
```
→ ~50 declarations across 24 services, including four never in the inventory at all
(`atlas-login`, `atlas-doors`, `atlas-merchant`, `atlas-summons`) and several services segment 1
already "finished" (`atlas-query-aggregator` 3, `atlas-pets` 3, `atlas-consumables` 3,
`atlas-storage` 1, `atlas-monsters` 1), largely the `asset`/`compartment`/`inventory`
Clone-based builders.

Consequence: the plan's Step 2 repo-wide "no `ModelBuilder` outside `libs/atlas-packet`" check
**cannot pass** at the end of Task 8 as the plan assumes, and Task 18's DOM-01 close-out is
sized against the same incomplete inventory. Raised with the user; not expanded into by any
implementer.

Gate `tools/verify.sh --quick --base fdcac2dbd` (Task 8 segment 1, all 10 service modules) → PASS.
**Last gated commit: `e8e28fc1b`.**

## Task 8 — complete (segment 2)

Commits `8d79ab916..49106bbcd` (13): the remaining 12 services one per commit
(`atlas-rps`, `atlas-reactors`, `atlas-npc-shops`, `atlas-mounts`, `atlas-messages`,
`atlas-maps`, `atlas-keys`, `atlas-inventory`, `atlas-expressions`, `atlas-drops`,
`atlas-data`, `atlas-cashshop`) plus `49106bbcd` recording the combined
`ledger-rename-rest.tsv`. Each service module-local build/vet/test clean before staging.

**FR-16 repo-wide check meets the stated bar:** `func NewModelBuilder` outside
`libs/atlas-packet/` is now exactly **1** — the `atlas-quest/quest` R2 collision reserved
for hand work. W2's bulk phase is done.

Bare-`ModelBuilder` type grep still shows 56 files (the out-of-inventory Clone-based builder
pattern). See the FR-13 scope finding above — open, awaiting a plan decision.

Gate `tools/verify.sh --quick --base e8e28fc1b` (Task 8 segment 2, 12 modules) → PASS.
**Last gated commit: `49106bbcd`.**

## Tasks 7-8 review — APPROVED_WITH_FINDINGS

Artifact: `review-task-7-8.md`. 0 blocking, 1 non-blocking, verified across all 24 commits.
Independently confirmed: exactly 1 residual `func NewModelBuilder`
(`services/atlas-quest/atlas.com/quest/quest/builder.go:22`) with the R2 collision real
(`quest/processor.go:851,921`); `NewBuilderWithId`/`NewEmptyBuilder` unregressed; no commit
staged outside its own service; combined ledger 58 lines / 57 APPLIED / 1 SKIPPED matches the
per-commit ledgers; FR-17 holds (no JSON tag, route, or Kafka topic literal changed); and the
object-identity guarantee held — the FR-15 exceptions `NewSharedVesselModelBuilder`,
`NewTripScheduleModelBuilder`, `RewardModelBuilder(Type)` are untouched despite containing
`ModelBuilder` as a substring.

**Open non-blocking fix, do this early next session (small):**
`services/atlas-pets/atlas.com/pets/pet/builder_test.go` lines 7,14,24,34,44,54,64,74,84,94,
153,164 — stale `TestModelBuilder_*` / `TestModelBuilderSetName*` test-function names left
unrenamed, inconsistent with the `TestModelBuilder_*`→`TestBuilder_*` fixup Task 8 applied to
every other service in the range, and not disclosed in `task-8-report.md`'s atlas-pets row.

## USER DECISION — FR-13 scope gap: add a task to this branch

Asked and answered 2026-08-26. The user chose **"Add a task to this branch."**

Concretely, for the next controller session:

1. Insert a new plan task (suggest **Task 9a**, after Task 9's FR-15 triage and before Task 10)
   that runs the codemod `rename` over the out-of-inventory `type ModelBuilder` /
   `type modelBuilder` declarations — ~50 across 24 services, largely the `asset` /
   `compartment` / `inventory` Clone-based builders, plus the four services never in the
   inventory at all: `atlas-login`, `atlas-doors`, `atlas-merchant`, `atlas-summons`.
   Derive the authoritative list with:
   `grep -rn --include='*.go' -E 'type (M|m)odelBuilder' services libs | grep -v 'libs/atlas-packet/'`
   Expect the R2 collision guard to SKIP some; those become hand work, same as
   `atlas-quest/quest`.
2. Re-size **Task 18**'s DOM-01 close-out against the true count rather than the constructor
   inventory — as written it asserts a bar the constructor inventory cannot reach.
3. Task 25's `exemptions.md` then records only genuine exemptions (`atlas-quest/quest`'s
   collision, `atlas-character`'s `modelBuilder`, `channel/asset`'s `NewBuilderWithId`), not
   this gap.

## Next session starts at: Task 9

Task 9 = "W2 — FR-15 triage and the nine sole-builder renames". Note Task 2's derived
`classify-dom01-fr15.tsv` says **RENAME=3, SIBLING-EXEMPT=61, NO-MODEL-GO=5**, not the design's
RENAME=9 — size Task 9 against the TSV, not the plan prose.

## Plan amendment — Task 9a inserted (controller, session starting at Task 9)

Per the user decision above, **Task 9a** is now in `plan.md`, between Task 9 and Task 10.
It runs the codemod `rename` over the out-of-inventory `type ModelBuilder` / `type modelBuilder`
declarations.

Authoritative inventory re-derived at `1f2dbeee9` and persisted to
`inventory-dom01-modelbuilder-type.txt`: **39 declarations** (32 exported `ModelBuilder`,
7 unexported `modelBuilder`) across **16 services** — not the ~50/24 estimated at `e8e28fc1b`,
because Task 8 segment 2's `atlas-data` and `atlas-cashshop` commits took some of them along.
Breakdown: `atlas-cashshop` 6, `atlas-npc-shops` 4, `atlas-character` 4,
`atlas-query-aggregator` 3, `atlas-pets` 3, `atlas-merchant` 3, `atlas-login` 3,
`atlas-inventory` 3, `atlas-consumables` 3, and one each in `atlas-summons`, `atlas-storage`,
`atlas-quest`, `atlas-monsters`, `atlas-drops`, `atlas-doors`, `atlas-channel`.

### Controller finding — `renameImpl` ignores `Target.From`/`Target.To`

Inspected while sizing Task 9. `renameImpl` (`codemod/rename.go:69-131`) loops the fixed
package-level `renamePairs` (`rename.go:44-49`) against each target package's scope and never
reads `t.From`/`t.To`; `targetsFromInventory` sets them as dead decoration. Two consequences,
both now written into the briefs/plan:

1. Task 9's `-targets` flag (which the plan assumes already exists — it does not) must carry a
   **stem** per package and derive the four pairs from it, plus a per-target pair override in
   `Target` that falls back to `renamePairs` when empty. Issued to the Task 9 implementer as a
   correction appended to `task-9-brief.md`.
2. Task 9a therefore needs **no new rename logic at all** — `renamePairs` already has
   `ModelBuilder`→`Builder` and `modelBuilder`→`builder`. It only needs a way to name a package
   that has no `NewModelBuilder`, which is precisely the empty-stem fallback. This is why 9a is
   sequenced after 9.

### Task 18 needs no re-sizing

The user decision's item 2 said to re-size "Task 18's DOM-01 close-out". On inspection Task 18
is **DOM-04 only** (`Transform`/`sweep.sh`); it asserts nothing about `ModelBuilder`. The DOM-01
close-out is Task 9a Step 6's repo-wide grep, sized against the real 39. No edit to Task 18.

Task 25's `exemptions.md` scope is unchanged and correct: it records the genuine exemptions
(`atlas-quest/quest`'s R2 collision, `atlas-character/character`'s Task 5 `modelBuilder`
ruling, `channel/asset`'s `NewBuilderWithId`) plus whatever Task 9a Step 3's dry run adds — not
this scope gap, which is now closed by a task rather than excused.

### Task 9 dispatch — in flight

Dispatched `atlas-implementer` (sonnet) with `task-9-brief.md`. Controller resolutions carried
in the brief: R1 the RENAME class is **3 rows, not 9** (all three in `atlas-drop-information`:
`continent/drop`, `monster/drop`, `reactor/drop`); R2 + its correction, the `-targets` design
above; R3 fold in the open Tasks 7-8 review fix (stale `TestModelBuilder_*` names in
`services/atlas-pets/atlas.com/pets/pet/builder_test.go`); R4 do not touch the Task 9a
declarations.

**Note:** `SendMessage` is disabled this session, so the R2 correction could only be appended to
the brief file after dispatch. If the Task 9 report shows a from/to-triple `-targets` design
that renames nothing, that is why — feed the correction back as a fix round.

## Task 9 — complete

Commits `2b07d2776`, `13de0adaf`, `d8240e381`, `f07717eae`.

- `2b07d2776` — codemod `rename` gains a `-targets <tsv>` flag. Per the controller finding
  above, `renameImpl` ignored `Target.From`/`Target.To`, so the implementer added an additive
  `Target.Pairs [][2]string` used only when non-empty, falling back to `renamePairs` when empty.
  The `-inventory` path is unchanged.
- `13de0adaf` — the three FR-15 `RENAME` packages, all in `atlas-drop-information`
  (`continent/drop`, `monster/drop`, `reactor/drop`). **3 rows, not the design's 9** — Task 2's
  TSV was authoritative, as ruled.
- `d8240e381` — the carried-forward Tasks 7-8 review fix: stale `TestModelBuilder_*` names in
  `services/atlas-pets/atlas.com/pets/pet/builder_test.go` renamed to `TestBuilder_*`. That
  non-blocking finding is now **closed**.
- `f07717eae` — `ledger-rename-fr15.tsv` and `fr15-targets.tsv`.

Module-local build/vet/test clean for the codemod module, `atlas-drop-information`, and
`atlas-pets/pet`.

Gate `tools/verify.sh --quick --base 49106bbcd` → **PASS** (exit 0; go build/vet on
`atlas-drop-information` and `atlas-pets`, plus analyzer/skill-id/scope/producer-seam/env-domain
and lint guards; docker bake skipped as expected for `--quick`).
**Last gated commit: `59062ab04`** (the range included the Task 9a plan-amendment docs commit).

Task 9 review dispatched (`atlas-reviewer`, sonnet) over `49106bbcd..f07717eae`, explicitly
scoped to the commit range rather than the working tree so it cannot repeat the Task 6
stale-snapshot false finding while Task 9a runs concurrently.

## Task 9a — dispatched

`atlas-implementer` (sonnet) with `.superpowers/sdd/plan/task-9a-brief.md`. 39 declarations,
16 services. `PARTIAL` is an expected outcome at this width.

## Task 9 review — APPROVED_WITH_FINDINGS

Artifact: `review-task-9.md`. 0 blocking, 1 non-blocking. Verified directly against file
content, not the implementer's report: `renameImpl` genuinely reads `Target.Pairs` (the dead-field
bug is fixed); `renamePairs` and the `-inventory` path are byte-identical, so Tasks 6-8 remain
reproducible; the R2 collision guard is untouched; the ledger emits one row per `pkgDir`; all
three `atlas-drop-information` renames reached every call site including cross-package `_test.go`
(repo-wide grep for the old names is empty); no `Clone*Builder` existed to rename; no JSON/route/
Kafka literal changed; the `atlas-pets` commit touches only `Test*` function names; and the Task
9a scope fence held. Codemod, `atlas-drop-information` and `atlas-pets` modules independently
rebuilt and retested — pass.

### Non-blocking finding — controller ruling: ACCEPT, no fix round

The finding: `targetsFromTSV` implements the pre-correction **3-column `pkgDir\tfrom\tto`**
format rather than the **2-column `pkgDir\tstem`** derivation the brief's R2 correction mandated,
and the deviation is undisclosed in the implementer's self-review.

Ruling: accept as implemented. I read `rename.go:405-440` directly. The triples format is
strictly more expressive than stem-derivation, and — decisively — the parser already implements
the **default-pairs directive** Task 9a depends on: a row of `pkgDir\t` or `pkgDir\t\t` registers
the directory with no Pairs, so `renameImpl` falls back to `renamePairs`. That is the exact
mechanism Task 9a Step 2 needs, documented in the function's own doc comment. Rewriting a working,
reviewed, tested parser to match the letter of a mid-flight correction would buy nothing.

The reviewer's real point — that a future `-targets` caller would be surprised — is addressed by
appending the verified format to `task-9a-brief.md`, which supersedes that brief's R1.

## Task 9a — DOM-01 type rename

`atlas-implementer` (sonnet) closed the FR-13 type-rename scope gap: 39 `type ModelBuilder`/
`type modelBuilder` declarations across 16 services, none reachable from the `NewModelBuilder`
constructor inventory Tasks 6-8 drove off.

**R1 finding (contradicts the ruling above):** at the point this task started, `targetsFromTSV`
(base commit `49106bbcd`, unchanged through `2b07d2776`) did **not** implement an empty-stem
fallback — a 2- or 3-field row with an empty `from`/`to` hit the `pkgDir == "" || from == "" ||
to == ""` guard and errored `"empty field"`. Verified directly: `git show
2b07d2776:.../codemod/rename.go` lines 410-440 show the unconditional 3-field, all-non-empty
parse. This is the small additive fix R1 anticipated, made rather than worked around: `rename.go`
now accepts a `pkgDir\t` (2-field) or `pkgDir\t\t` (3-field, both blank) row as a default-pairs
directive — it registers the directory with `Target.Pairs` left nil, so `renameImpl` falls back to
the fixed `renamePairs` set exactly as Step 2 needs. Covered by
`TestTargetsFromTSV/empty-stem_row_falls_back_to_renamePairs`. Codemod module: `go build ./... &&
go vet ./... && go test ./...` → pass.

Inventory re-derivation (Step 1) matched `inventory-dom01-modelbuilder-type.txt` exactly: 39
lines, no diff.

**`atlas-character/atlas.com/character/character` excluded from the targets file, not left to a
natural SKIP.** The R3 exemption (Task 5: `type modelBuilder`, `CloneModel`, the pre-existing
`type Builder`/`NewBuilder` left alone) does **not** trip the codemod's collision guard, because
the guard only checks the lowercase `modelBuilder`/`builder` pair, and the exempted package's
`Builder`/`NewBuilder` are a different, differently-cased identifier — a dry run against the full
39-row targets file confirmed this package would actually APPLY, not skip. Per R2 ("never force a
skip... revert and record by hand"), the package directory was removed from
`dom01-type-targets.tsv` before any apply run, so the codemod never touched it, and the ledger
carries a hand-written `SKIPPED` row with the Task 5/R3 reason. `git status` after every
`atlas-character` codemod run confirmed `character/character/model.go` stayed untouched.

38 packages went through the codemod's apply path across 16 per-service commits (largest-first:
`atlas-cashshop` 6, `atlas-npc-shops` 4, `atlas-character` 3 (of 4 — `character/character`
excluded per above), `atlas-query-aggregator`/`atlas-pets`/`atlas-merchant`/`atlas-login`/
`atlas-inventory`/`atlas-consumables` 3 each, then `atlas-channel`/`atlas-doors`/`atlas-drops`/
`atlas-monsters`/`atlas-storage`/`atlas-summons` 1 each). `atlas-quest/quest` SKIPPED on the known
R2 `builder` collision (its `modelBuilder`/`builder` renamer target). Each service's module-local
`go build ./... && go vet ./... && go test ./...` passed before its commit; stale `// ModelBuilder`
doc comments were fixed by hand in `atlas-cashshop/cashshop/inventory{,/compartment}` and
`atlas-npc-shops/{commodities,shops}/builder.go`, and `atlas-monsters/monster/builder.go`. No
`TestModelBuilder_*` test-function names were found in the 39 declarations' packages.

Step 6 repo-wide residue re-check (`grep -rn --include='*.go' -E 'type (M|m)odelBuilder' services
libs | grep -v 'libs/atlas-packet/'`):

```
services/atlas-character/atlas.com/character/character/model.go:211:type modelBuilder struct {
services/atlas-quest/atlas.com/quest/quest/builder.go:11:type modelBuilder struct {
```

Both are already-recorded SKIPs (Task 5/R3 exemption, R2 `builder`-collision). No forgotten
package.

**Concern for the controller:** `progress.md` and `agent-ledger.tsv` carried uncommitted working-
tree edits from a concurrent process at the start of this task, referencing commit hashes
(`13de0adaf`, `d8240e381`, `f07717eae`, `59062ab04`) that do not exist in this branch's `git log`
and a "Task 9 review — APPROVED_WITH_FINDINGS" claim that the empty-stem fallback "already
implements" in `rename.go` — which the R1 finding above shows was false at `2b07d2776`. This
task's brief described the concurrent agent as a read-only reviewer; the working tree shows
otherwise. This section was appended without altering that content; the controller should
reconcile it.

## CORRECTION to the Task 9 review ruling above

The ruling (accept the 3-column `-targets` format, no fix round) stands, but **its stated
evidence was wrong and must not be relied on.**

I wrote that I had "read `rename.go:405-440` directly" and found the default-pairs directive
already implemented at Task 9. I had in fact read the **working tree**, which the Task 9a
implementer was concurrently modifying — it had already applied its own additive fix. The
committed Task 9 code did not contain that directive. Verified after the fact:

```
$ git show 2b07d2776:.../codemod/rename.go   # targetsFromTSV
		if len(fields) != 3 {
			return nil, fmt.Errorf("targets file: line %d: expected 3 tab-separated fields ...")
		}
		...
		if pkgDir == "" || from == "" || to == "" {
			return nil, fmt.Errorf("targets file: line %d: empty field: %q", i+1, line)
		}
```

Three non-empty fields required; no empty-stem fallback. The Task 9a implementer's R1 finding is
correct and mine was not. The `CONTROLLER NOTE` appended to `task-9a-brief.md` was therefore also
wrong at the time it was written, and the implementer rightly ignored it in favor of direct
inspection and made the fix R1 directed.

**Process lesson, a sharper form of the Task 6 one:** the earlier lesson was "do not schedule a
per-task review concurrently with an implementer that touches the same paths." The real rule is
broader — **while any implementer is running in this worktree, the working tree is not evidence.**
Read `git show <commit>:<path>`, never the file, when the claim is about what a commit contains.

## Task 9a — complete (DONE_WITH_CONCERNS)

Commits `a034a8242..a58dcdc97` (15 per-service refactors) plus `01e2cc2c3` (ledger/artifacts).

- **15 of 16 services renamed.** `atlas-quest` produced no diff: its sole target hit the known
  R2 `builder` collision and SKIPped, as designed.
- **`atlas-character/character` was excluded before any apply run.** Dry-run testing showed it
  would have *applied* rather than naturally skipping, because the collision guard is
  case-sensitive and does not see the package's pre-existing `Builder`/`NewBuilder`. The
  implementer excluded it from the targets file and hand-wrote the SKIP into the ledger,
  preserving the Task 5 ruling exactly. Good catch — the guard's case-sensitivity is a real
  latent hazard for any future `-targets` run and belongs in `exemptions.md`'s notes (Task 25).
- Additive fix to `targetsFromTSV` adding the default-pairs directive, with a covering test.
- All 15 touched service modules plus the codemod module: `go build`/`vet`/`test` clean.

### FR-13 / DOM-01 repo-wide residue — controller-verified at `01e2cc2c3`

```
$ grep -rn --include='*.go' -E 'type (M|m)odelBuilder' services libs | grep -v 'libs/atlas-packet/'
services/atlas-character/atlas.com/character/character/model.go:211:type modelBuilder struct {
services/atlas-quest/atlas.com/quest/quest/builder.go:11:type modelBuilder struct {

$ grep -rn --include='*.go' -E 'func NewModelBuilder' services libs | grep -v 'libs/atlas-packet/'
services/atlas-quest/atlas.com/quest/quest/builder.go:22:func NewModelBuilder() *modelBuilder {
```

Exactly the two recorded exemptions and nothing else. **The FR-13 scope gap is closed.** Both
residual entries are genuine `exemptions.md` rows for Task 25, not unfinished work.

## Task 9a review — APPROVED

Artifact: `review-task-9a.md`. **0 blocking, 0 non-blocking**, full range `59062ab04..01e2cc2c3`
reviewed: the `targetsFromTSV` default-pairs directive and its test, both SKIP dispositions,
per-service rename completeness including cross-package call sites and hand-fixed doc comments,
FR-15 substring-exception non-interference, FR-17/FR-18 literal safety, and commit hygiene across
all 15 per-service commits.

Two `not_evaluable` items, both already reconciled by the controller and needing no action:

1. The `progress.md` / `agent-ledger.tsv` conflict the Task 9a implementer raised — answered by
   the `CORRECTION` section above. The uncommitted content it saw was the controller's own
   in-progress ledger writing; the substantive part (my claim about `rename.go`) was wrong and is
   now corrected in place with `git show` evidence.
2. The untracked `review-task-9.md` — committed in `858edb209`.

Gate `tools/verify.sh --quick --base 59062ab04` → **PASS** (exit 0; 15 modules through
build/vet, analyzer/skill-id/scope/producer-seam/env-domain and lint guards; docker bake skipped
as expected for `--quick`). **Last gated commit: `858edb209`.**

## Next session starts at: Task 10

Task 10 = "W3 — `transform` subcommand". W2 (the builder-rename work stream, Tasks 3-9a) is
**closed**: DOM-01/FR-12/13/14/15/16 residue is the two recorded exemptions and nothing else.

Carry into Task 10 — the two Tasks 2-3 review findings the controller deferred to exactly this
task, because Task 10 is the first whose correctness depends on tier `A` meaning "safe to
auto-collapse":

1. `codemod/classify.go:412-429` — `isFlatFieldValue` checks that a composite-literal field
   value's base is *some* identifier, not that it is `Extract`'s actual parameter. Not exploited
   by any of the 94 real tier-`A` packages today. **Tighten before `transform` consumes tier `A`.**
2. `codemod/classify_test.go:346-416` — no `EXCLUDED-TREE` case for the `libs/atlas-packet/`
   prefix branch. Add it.

Also size Task 10 against Task 2's derived TSVs (A=94, B1=11, B2=61, C=7, NO-RESTMODEL=12), not
the design's §2 prose.

Open items for later tasks (no action now):

- Task 25 `exemptions.md` must record: `atlas-quest/quest`'s R2 collision; `atlas-character/character`'s
  Task 5 ruling; `channel/asset`'s `NewBuilderWithId`; and **the collision guard's case-sensitivity
  hazard** found in Task 9a — the guard checks only the `modelBuilder`/`builder` pair and misses a
  package's pre-existing `Builder`/`NewBuilder`, so any future `-targets` run must dry-run first.
- Codemod test debt from the Task 6 review: `rename_test.go` never exercises
  `loadModuleForRename`'s `Tests: true` path.
- Pre-existing `go.sum` gap making `GOWORK=off go build ./...` fail in `services/atlas-channel`
  (not caused by this branch; the `go.work` build passes). Flagged for Task 27's final gate.

## Task 10 — complete (DONE_WITH_CONCERNS)

Commit `b1d090265` — "chore(task-263): add Transform generation to codemod". W3 opens.

- `transform.go` + `transform_test.go` added, `transform` subcommand wired into `main.go`.
  New suites pass: `TestGenerateTransform` 7/7, `TestGenerateRoundTripTest` 3/3.
- **Both carried-forward Task 2-3 review findings landed in this same commit** — the
  `isFlatFieldValue` tightening in `classify.go` and the `EXCLUDED-TREE` case in
  `classify_test.go` — each with a covering regression case.
- Module suite (`go build`/`go vet`/`go test`) clean.

### Step 5 dry run over all 94 tier-`A` rows — 82 APPLIED / 12 SKIPPED

No real file was touched (`git status` clean after the run). Skip histogram:

| reason | n | detail |
|---|---|---|
| `unsupported field type` | 6 | map, `[]string`, `*time.Time` ×2, `[]uint32` |
| `two or more bool fields` | 3 | values would be indistinguishable, so the round-trip test could not catch R1 |
| `Extract maps no fields` | 1 | |
| `Transform already declared` | 1 | `atlas-storage/asset` — hand-written earlier |
| `does not type-check` | 1 | `atlas-npc-conversations/petdata` — a real int→`[]EvolutionRestModel` field-count mismatch the overlay gate correctly caught |

All 12 join the tier-B/C hand-work list of Tasks 13-18. The one `does not type-check` row is the
type-check-before-write gate doing exactly its job on real code, not a generator bug.

**Implementer concern for the record:** the TDD RED step was not captured as a literal failing
run — `transform.go` and `transform_test.go` were written together against the interface locked
by the brief. Handed to the task reviewer to rule on whether the tests are genuinely
discriminating, which is what RED exists to establish.

### Gate — PASS

`tools/verify.sh --quick --base 858edb209` → exit 0. Note the selection: because the codemod
module lives under `docs/`, the gate reported "no Go module changed" and skipped the Go lint &
format layer; what ran was the analyzer, skill/job-id, and scope guards. The codemod's own
build/vet/test was covered by the implementer's module-local run, not by this gate.
**Last gated commit: `b1d090265`.**

## Task 10 review — CHANGES_REQUIRED → fixed in round 1

Artifact: `review-task-10.md`. **1 blocking, 3 non-blocking.** The blocking finding justified
holding Task 11's dispatch.

**Blocking:** `transform.go`'s `typeChecksWithOverlay` omitted `Tests: true`, so `packages.Load`
never loaded the generated `rest_test.go` at all — overlay or not. The generated
`TestTransformRoundTrip`, the only defense against R1 (a field mapped to the wrong same-typed
sibling), was shipping to every applied package with zero type-check verification. The reviewer
proved it empirically: the gate returned `nil` for a scratch package whose `_test.go` returned
`"not an int"` from a func declared `int`. `rename.go:279-285` already set `Tests: true` for
exactly this reason.

Non-blocking, all fixed in the same round: no fixture with two same-typed fields (so nothing
proved the derived values were actually distinct — the round-trip test's whole purpose); no
CRLF preservation in `mergeIntoRestGo`/`mergeIntoRestTestGo`. The missing-RED process note was
ruled non-blocking — the assertions were genuinely discriminating; the gap was fixture coverage.

### Fix round 1 — commit `c5b76c6b9` (DONE)

"fix(task-263): type-check the generated round-trip test, prove distinct sibling values,
preserve CRLF". RED captured literally this time. Module suite and `go vet` clean.

### Dry run re-run: 76 APPLIED / 18 SKIPPED (was 82 / 12)

The strengthened gate newly skips **6** packages whose generated tests would not have compiled —
byte-range overflow and builder `Build()` return-arity mismatches, previously merged unverified:

`atlas-cashshop/character`, `atlas-channel/character`, `atlas-maps/data/map/info`,
`atlas-messages/character`, `atlas-monsters/monster/consumable`, `atlas-npc-shops/character`.

**The count drop is the fix working, not a regression.** No gate was loosened and no package
special-cased. `atlas-channel/character` is inside Task 11's scope and will now correctly SKIP
to hand work rather than receive a broken generated test.

These 6 join the tier-B/C hand-work list of Tasks 13-18, bringing that list to 18.

**Not yet gated: `b1d090265..c5b76c6b9`.** Per the execute-task fix loop the fix commit is not
gated separately; it joins the next gate's range, which will run from `858edb209`.

## Task 10 fix round 1 review — APPROVED

Artifact: `review-task-10-fix1.md`. **0 blocking, 0 non-blocking**, range `b1d090265..c5b76c6b9`.

The reviewer did not take the fix on report. It independently reproduced RED (fix reverted) and
GREEN (fix present) for the type-check-gate finding without touching git history, re-ran the
module suite / build / vet, re-ran the Step 5 dry run and matched **76 APPLIED / 18 SKIPPED**
byte-for-byte, and traced two of the six newly-skipped packages to actual service source:

- `atlas-cashshop/character` — `services/atlas-cashshop/atlas.com/cashshop/character/rest.go:40,88-120`:
  29 fields, so the value sequence reaches 319 for a `stance byte` → genuine overflow.
- `atlas-maps/data/map/info` — `services/atlas-maps/atlas.com/maps/data/map/info/model.go:29-36`:
  `Build() Model` returns a **single** value, against `transform.go:401-406`'s always-two-value
  assumption.

Both are legitimate skips, not an over-broad gate. `Tests: true` at `transform.go:543`; CRLF
restore at `transform.go:449-458` and `513-520`.

Two `not_evaluable`, neither needing action: four of the six newly-skipped packages were not
hand-traced (the brief asked for at least two), and whether any of the 76 still-APPLIED packages
harbor an undiscovered defect class is unchanged surface from the first review.

**Task 10 is closed.** Commits `b1d090265` + `c5b76c6b9`.

### Open item carried forward — single-return `Build()`

`transform.go:401-406` assumes `Build()` returns `(Model, error)`. A package whose builder
returns a bare `Model` (like `atlas-maps/data/map/info`) is therefore pushed to hand work rather
than generated. That is *safe* — the type-check gate catches it — but it is a **generator
limitation, not a property of those packages**. If the tier-B/C hand-work list of Tasks 13-18
turns out to hold several of these, teaching the generator both arities is likely cheaper than
hand-writing each. Count them before deciding; do not widen the generator speculatively.

## Task 10 gate — PASS

`tools/verify.sh --quick --base 858edb209` over `858edb209..2146a4ab4` → exit 0 (analyzer,
skill/job-id, scope guards; docker bake skipped as expected for `--quick`). Covers both Task 10
commits and the fix round. **Last gated commit: `2146a4ab4`.**

## Task 11 — complete (DONE)

Commit `ed54e564e` — "feat(atlas-channel): add Transform and round-trip tests". First task to put
generated code into the real service tree.

### 23 tier-A rows in `atlas-channel` → 15 APPLIED / 8 SKIPPED

All 15 generated `TestTransformRoundTrip` pass; module `go build`/`go vet`/`go test ./...` clean.
**No generator change was needed** — Task 10's dry-run tally stands and no re-gate of the codemod
is required.

Skipped, all joining the tier-B/C hand-work list of Tasks 13-18:

| package | reason |
|---|---|
| `buddylist/buddy`, `data/quest`, `minigame`, + 1 | two or more bool fields (4) |
| `data/consumable`, `data/equipment`, `mts/listing`, `mts/wish` | unsupported field type — map/slice/pointer, `*time.Time` (3, one package overlaps the bool set) |
| `character` | generated code does not type-check — the byte-range overflow class, 4 byte fields (1) |

*(Corrected: an earlier draft of this table listed `character` as a ninth skip separate from the
does-not-type-check row. There are 8 skips, 4+3+1, and the ledger TSV's 23 rows — 15 APPLIED +
8 SKIPPED — match the histogram exactly. Verified by the Task 11 reviewer against source.)*

Note the plan's §Task 11 preamble says `atlas-channel` holds 50 of the 185 has-model packages;
only **23** of those are tier `A` and in scope here. The other 27 are tier B1/B2/C/NO-RESTMODEL
and belong to later tasks. Not a shortfall.

### FR-17 no-deletion check — controller-verified

`git diff -U0 2146a4ab4..ed54e564e -- services/atlas-channel | grep '^-'` returns two
`-import "testing"` lines. The implementer flagged them; I checked the diff at `-U3` myself:

```
-import "testing"
+import (
+	"reflect"
+	"testing"
+)
```

in `data/monster/rest_test.go` and `mts/holding/rest_test.go`. Both are `astutil.AddImport`
folding a single-line import into a grouped block to add `reflect`. **No content was deleted;
FR-17 holds.** Expect this same benign pattern in every later W3 apply task — a bare
`grep '^-'` will be non-empty whenever a target `rest_test.go` had a single-line import, so the
check needs the `-U3` look, not just the count.

## Task 11 review — APPROVED

Artifact: `review-task-11.md`. **0 blocking, 0 non-blocking**, commit `ed54e564e` reviewed in full.

The reviewer verified rather than trusted: `git diff --stat` on `codemod/` for the range is empty
(generator untouched, so Task 10's 76/18 tally stands); all 15 `Transform` functions are true
inverses of their `Extract`, including the genuinely risky cases — `drop` (X/Y vs DropperX/DropperY,
both `int16`), `guild/member` (Level/Title/AllianceTitle, three same-typed `byte` fields), and the
typed conversions `mts/holding` (`byte(m.worldId)`, correctly drawn from the RestModel field's
declared `byte` rather than `world.Id`) and `character/skill`. Round-trip fixtures use distinct
sequential non-zero values throughout, so a sibling swap would fail `reflect.DeepEqual`. All **8**
skips spot-checked against source, not the 2 asked for — every one legitimate. Ledger: 23 rows,
matching the histogram with no discrepancy.

## Task 11 gate — FAIL (gofmt), generator defect

`tools/verify.sh --quick --base 2146a4ab4` → exit 1. One failing check: **lint & format guard**,
`FMT FAIL — services/atlas-channel/atlas.com/channel`. Build, vet, and all tests passed; this is
purely `gofmt`.

Quoted failing block (same shape in all three files):

```
diff data/monster/rest_test.go.orig data/monster/rest_test.go
@@ -24,6 +24,7 @@
 		t.Errorf("FixedDamage=%d, want 5", m.FixedDamage())
 	}
 }
+
 func TestTransformRoundTrip(t *testing.T) {
```

Affected: `data/monster/rest_test.go`, `mts/holding/rest_test.go`, `mts/transaction/rest_test.go`.

### Diagnosis

`mergeIntoRestTestGo` appends the generated `TestTransformRoundTrip` **without a blank line
separating it from the preceding top-level declaration**. `gofmt` requires one. This did not
surface in Task 10 because the codemod's own tests compare generated *fragments*, and it did not
surface in the implementer's module gate because `go build`/`go vet`/`go test` are all
indifferent to the blank line — only `gofmt` sees it.

**It is in the generator, so it will recur in every remaining W3 apply task.** Note it appears
only where the generated func follows an existing declaration in a pre-existing `rest_test.go`
(3 of 15 here); a newly created test file is unaffected, which is why the defect is easy to miss.

Fix belongs in `codemod/transform.go` — never in the generated files — then re-run the
`atlas-channel` apply. `mergeIntoRestGo` needs the same audit; the `Transform` additions to
`rest.go` happened not to trip it here, but the merge path is the same shape.

Per the execute-task fix loop this does not get its own gate; the fix commit joins the next
gate's range, which continues to run from `2146a4ab4`.

## Task 11 fix round 1 — commit `20765f8d3` (DONE)

"fix(task-263): generate gofumpt-clean rest_test.go merges".

### CORRECTION to the diagnosis above

My fix brief said "`gofmt` requires the blank line". **That is wrong, and the implementer
disproved it empirically.** Plain `gofmt` does *not* require it. The repo gate is
`golangci-lint fmt` (gofumpt + goimports), and **only gofumpt** enforces a blank line between
top-level declarations. Verifying with `gofmt -l` alone would have shown a clean tree while the
gate still failed.

The fix therefore pipes `mergeIntoRestTestGo`'s `format.Node` output through
`mvdan.cc/gofumpt/format.Source`. An initial attempt using `gofumpt/format.File` (AST mutation)
did **not** fix it — recorded because the distinction is easy to get wrong twice.

**This adds `mvdan.cc/gofumpt` as a real dependency of the codemod module** (`go mod tidy`'d).
Acceptable — the codemod is a task-scoped throwaway tool under `docs/`, not a shipped service —
but it is a new third-party dependency and should be named as such in the PR description.

`mergeIntoRestGo` was audited per the brief and does **not** have the defect: its textual merge
already inserts an explicit blank line. Left unchanged.

### Verification — controller-checked

Regenerated `atlas-channel` from base `2146a4ab4` with the fixed generator: **15 APPLIED / 8
SKIPPED**, ledger byte-identical. I confirmed the diff scope myself:

```
$ git diff --stat ed54e564e -- services/atlas-channel
 .../data/monster/rest_test.go    | 1 +
 .../mts/holding/rest_test.go     | 1 +
 .../mts/transaction/rest_test.go | 1 +
 3 files changed, 3 insertions(+)

$ gofmt -l services/atlas-channel/atlas.com/channel
(no output)
```

Exactly the three expected blank lines and nothing else — the approved generated logic is
unchanged, so Task 11's content review still stands.

Regression test added asserting the **merged file** is format-clean (RED before, GREEN after),
for both the pre-existing-test-file and new-file cases. The prior fragment-comparison tests
structurally could not catch this class.

### Process note

The fix implementer used **124 tool calls**, over the 120 cap, and reported `DONE` rather than
`PARTIAL`. No work was lost, but the cap did not bind as designed — worth watching on the next
long fix round.

### Task 11 gate (re-run after fix) — PASS

`tools/verify.sh --quick --base 2146a4ab4` over `2146a4ab4..20765f8d3` -> exit 0, 8 checks green
including the previously failing **lint & format guard**. **Last gated commit: `20765f8d3`.**

## Next session starts at: Task 12

Tasks 10 and 11 are closed: both reviewed APPROVED, both gated PASS. W3 is proven end to end —
the generator is hardened (type-checks the generated test, gofumpt-clean output) and applied to
its first service.

Carry into Task 12 and every later W3 apply task:

1. **The FR-17 removed-line check is expected to be non-empty** whenever a target `rest_test.go`
   had a single-line import; `astutil.AddImport` folds it into a group. Look at `-U3`, do not
   count lines.
2. **Verify formatting with the repo gate, not `gofmt -l`.** Only gofumpt enforces the
   blank-line rule that failed Task 11; `gofmt -l` prints a clean tree while the gate fails.
3. **Tier-A is not the package count.** Size each service against `classify-dom04.tsv` rows, not
   the plan prose (atlas-channel: plan said 50 packages, only 23 were tier A).
4. The 18 repo-wide SKIPPED packages plus per-service skips are the input to Tasks 13-18.
5. Open item from Task 10: `transform.go:401-406` assumes `Build()` returns `(Model, error)`.
   Single-return builders are pushed to hand work. Count them across the remaining skips before
   deciding whether teaching the generator both arities beats hand-writing each.

## Task 12 — controller ruling: split into 3 sequential batches

Task 12 as planned is 29 services / 106 tier-A packages. At ~6 tool calls per service that is
~174 calls against a 120-call cap, so it is split into three disjoint, **sequential** batches
(sequential, not parallel: three agents committing in one worktree would contend on `index.lock`).

Authoritative per-service tier-A counts, from
`awk -F'\t' '$2=="A"' classify-dom04.tsv | cut -f1 | sed 's|^services/\([^/]*\)/.*|\1|' | sort | uniq -c`:

```
     23 atlas-channel        (Task 11, done)
      8 atlas-monster-death        8 atlas-messages           7 atlas-consumables
      4 atlas-npc-shops            4 atlas-inventory          3 atlas-query-aggregator
      3 atlas-pets                 3 atlas-monsters           3 atlas-maps
      2 atlas-trades               2 atlas-reactors           2 atlas-mini-games
      2 atlas-login                2 atlas-character          2 atlas-chairs
      2 atlas-cashshop             2 atlas-ban                1 atlas-summons
      1 atlas-storage              1 atlas-saga-orchestrator   1 atlas-rankings
      1 atlas-party-quests         1 atlas-npc-conversations   1 atlas-monster-book
      1 atlas-marriages            1 atlas-guilds              1 atlas-fame
      1 atlas-doors                1 atlas-buddies
```

Note the plan's Task 12 prose over-counted several services (e.g. it lists `atlas-character` at 7
and `atlas-monsters` at 5); the tsv is authoritative, as Task 11 already established for
`atlas-channel` (plan said 50, actual 23).

- Batch 1 (43 pkgs, 9 services): monster-death, messages, consumables, npc-shops, inventory,
  query-aggregator, pets, monsters, maps -> `ledger-transform-rest-1.tsv`
- Batch 2 (19 pkgs, 11 services): trades, reactors, mini-games, login, character, chairs,
  cashshop, ban, summons, storage, saga-orchestrator -> `ledger-transform-rest-2.tsv`
- Batch 3 (9 pkgs, 9 services): rankings, party-quests, npc-conversations, monster-book,
  marriages, guilds, fame, doors, buddies -> `ledger-transform-rest-3.tsv`

Each batch gets its own ledger file so no two agents append to the same file. Plan Steps 2-4
(cross-batch ledger reconciliation, `handwork-dom04.tsv`) are the controller's, run once after
batch 3. Step 2's reconciliation must glob `ledger-transform-*.tsv`, not the two filenames the
plan names.

Briefs: `.superpowers/sdd/plan/task-12-batch-{1,2,3}-brief.md`.

### Task 12 batch 1 — DONE (9 services, 36 APPLIED / 7 SKIPPED)

Commits `81922b270`(monster-death) `3c21ef059`(messages) `7a7e489b0`(consumables)
`60c3ec452`(npc-shops) `e66e25ceb`(inventory) `9185d8bbb`(query-aggregator) `8ed6edaf3`(pets)
`0dddbda0d`(monsters) `f949577ac`(maps), plus `d18c4e883` ledger and `a7fd717e1` report.
All 9 modules `go build && go vet && go test` clean on the first generator pass. 4 of the 7
skips are the known single-return-builder limitation.

**Two invocation corrections — the plan's Task 12 command does not run as written:**

1. The codemod is its own Go module nested in a worktree whose root is not a Go module, so
   `go run ./docs/.../codemod transform` fails with `go: cannot find main module`. The working
   form is `GOWORK=off go run -C docs/tasks/task-263-backend-guideline-conformance/codemod .
   transform ...` with **absolute** `-repo` / `-classification` / `-ledger` paths.
2. **There is no `-append` flag** (the plan invented it; it errors "flag provided but not
   defined"). Each run overwrites `-ledger` wholesale, so a per-service loop must write to a
   scratch path and concatenate onto the batch ledger by hand.

Batch 1 hit both, worked around them, and the corrections are now baked into the batch 2 and 3
briefs. Anyone re-running W3 from the plan text alone will hit them again.

Gate for `20765f8d3..a7fd717e1` launched (log: `.superpowers/sdd/gates/gate-12b1.log`); batch 1
review and batch 2 implementer dispatched concurrently.

### Task 12 batch 1 gate — PASS

`tools/verify.sh --quick --base 20765f8d3` over `20765f8d3..a7fd717e1` -> exit 0.
Selection was exactly the 9 batch-1 modules (`verify.sh: change base 20765f8d3 — 76 changed
path(s)`, `9 changed Go module(s)`) — batch 2 was committing concurrently and did **not** leak
into this range, so the concurrency did not contaminate the verdict. The **lint & format guard
(9 modules)** passed, which is the gofumpt check that failed in Task 11; the Task 11 generator
fix holds across all 9 services.

**Last gated commit: `a7fd717e1`.**

### Task 12 batch 1 review — APPROVED (0 blocking, 0 non-blocking)

Artifact: `task-12-batch-1-review.md`. The reviewer re-derived every number rather than taking
the report's, and the manual `/tmp` ledger concatenation — the workaround most likely to drop or
duplicate a row — came back clean: 43 ledger rows == 43 tier-A rows in `classify-dom04.tsv` for
these 9 services, and the 36 APPLIED rows are set-identical to the 36 package dirs actually
touched by the diff. No SKIPPED package has any diff.

All 7 SKIPPED reasons confirmed against source: 4 single-return `Build() Model` builders,
1 `map[SpecType]int32` field, 1 `[]uint32` field, and 1 Extract that discards its RestModel
entirely (`atlas-messages/.../data/map/rest.go`). The only 2 deletions in the whole batch are
single-line import folds — the documented FR-17 known-good behavior.

Running count of the single-return-builder limitation (`transform.go:401-406`): **4 so far**
(batch 1). Keep accumulating across batches 2-3; the total decides whether teaching the generator
both `Build()` arities beats hand-writing each in Tasks 13-18.

### Task 12 batch 2 — DONE (11 services, 10 commits)

Commits `0efabf17f`(trades) `fa083d6ea`(reactors) `4b152c1fd`(mini-games) `2283a9ef8`(login)
`a9baaece7`(character) `e33a308f2`(chairs) `e68bd9a05`(cashshop) `885e987ed`(ban)
`eb8422431`(summons) `6707f13e3`(saga-orchestrator), plus `2a84f41f9` ledger+report.
10/11 services built/vetted/tested clean. `atlas-storage` produced 0 APPLIED and correctly got
no commit — its `asset` package already has a hand-written `Transform`.

**Two new skip categories, neither in the Task 11 carry-forward list:**

1. `atlas-cashshop/character` — type-check failure: the generated round-trip fixture used `319`
   for a `byte`-typed field (overflow). The generator's own type-check caught its bad output and
   skipped rather than committing a broken file, which is the fail-safe behaving correctly — but
   it is a generator defect, not an inherent property of the package, and it will recur wherever
   a narrow integer field exists. Sent to the batch-2 reviewer for independent confirmation that
   nothing partial was left behind.
2. `atlas-storage/asset` — "Transform already declared". This is **already conformant**, a
   categorically different thing from a generator limitation. It must not be counted into
   `handwork-dom04.tsv` as hand work; there is nothing to do for it.

**Single-return-builder tally: 4 after batch 2** (0 new this batch; all 4 from batch 1).

Gate for `a7fd717e1..2a84f41f9` launched (`.superpowers/sdd/gates/gate-12b2.log`); batch 2 review
and batch 3 implementer dispatched concurrently.

### Task 12 batch 2 review — APPROVED (0 blocking, 0 non-blocking, 2 not-evaluable)

Artifact: `task-12-batch-2-review.md`. All 19 tier-A rows for the 11 services present exactly
once — the hand-concatenated ledger is clean again. All 10 commits strictly scoped to their own
service tree and insertion-only.

The three flagged judgment calls all resolved in favor of the implementer:

- `atlas-storage/atlas.com/storage/asset/processor.go:100` has a pre-existing, correctly-signed
  `Transform(m Model) (RestModel, error)`. The "already declared" skip is a true no-op, not a
  generator blind spot — confirming it stays **out** of `handwork-dom04.tsv`.
- `atlas-cashshop/atlas.com/cashshop/character` has no `rest_test.go` and **zero diff in range**:
  the byte-overflow type-check failure rolled back cleanly, nothing partial committed.
- The ledger's only two non-APPLIED reasons are the byte overflow and "already declared";
  neither is a single-return builder, so the **tally stays at 4**, independently confirmed.

Two not-evaluable, both benign: the reviewer did not re-run build/vet/test for the 10 services
(that is precisely what the `a7fd717e1..2a84f41f9` gate covers — do not close batch 2 until that
gate reports), and gofumpt conformance is the gate's job by instruction, not the reviewer's.

### Task 12 batch 2 gate — PASS

`tools/verify.sh --quick --base a7fd717e1` over `a7fd717e1..2a84f41f9` -> exit 0.
Selection was exactly the 10 committed batch-2 modules (`change base a7fd717e1 — 39 changed
path(s)`); batch 3 was committing concurrently and did not leak in. **lint & format guard
(10 modules)** green, so gofumpt holds across batch 2. This closes the batch-2 review's first
not-evaluable (build/vet/test not re-run by the reviewer) and its second (gofumpt).

**Batch 2 fully closed: implemented, reviewed APPROVED, gated PASS. Last gated commit
`2a84f41f9`.**

### Task 12 batch 3 — DONE (9 services, 8 commits)

Commits `449d2c0`(rankings) `fcb2902`(party-quests) `7a5ed8621`(monster-book) `6c2a44a43`(marriages)
`5d28ddf1b`(guilds) `5a92f7c01`(fame) `d651a8181`(doors) `0f4dc6e17`(buddies), plus `d0ba529db`
ledger+report. 8/8 applied services build/vet/test clean. One skip:
`atlas-npc-conversations/.../npc/petdata` — the generator assigned a scalar `evolutions int` into
a `[]EvolutionRestModel` field; its own type-check caught it, clean skip, no commit.

Gate for `2a84f41f9..d0ba529db` launched (`.superpowers/sdd/gates/gate-12b3.log`); batch 3 review
dispatched concurrently.

## Task 12 Steps 2-4 (controller) — reconciliation complete

### Step 2 — ledger reconciliation: EXACT

```
$ cat ledger-transform-channel.tsv ledger-transform-rest-*.tsv | cut -f2 | sort | uniq -c
     76 APPLIED
     18 SKIPPED
$ cat ledger-transform-channel.tsv ledger-transform-rest-*.tsv | wc -l
94
$ awk -F'\t' '$2=="A"' classify-dom04.tsv | wc -l
94
```

Stronger than the plan's count check, which only compares totals: I diffed the two **path sets**
and they are `SET-IDENTICAL` with zero duplicate rows. Every tier-A package is accounted for
exactly once across Task 11 + the three batches, so the hand-concatenated ledgers neither dropped
nor double-counted anything.

Note the plan's Step 2 names only `ledger-transform-channel.tsv` and `ledger-transform-rest.tsv`;
the batch split means the correct reconciliation globs `ledger-transform-*.tsv`.

### The 18 skips, categorized

| Category | Count | Nature |
|---|---|---|
| unsupported field type (`map[SpecType]int32`×2, `[]string`, `*time.Time`×2, `[]uint32`) | 6 | generator gap |
| single-return `Build() Model` | 4 | generator gap (`transform.go:401-406`) |
| two or more bool fields | 3 | generator gap |
| byte overflow in generated fixture | 2 | generator defect |
| `Extract` maps no fields | 1 | package-specific |
| scalar-into-slice mismatch | 1 | generator defect |
| **`Transform` already declared** | **1** | **already conformant — not hand work** |

**Correction to the batch-2 note above:** the byte-overflow category was *not* new in batch 2.
`atlas-channel/character` hit it in Task 11 (`330` into a `byte`) and it never made the
carry-forward list; `atlas-cashshop/character` (`319`) is the second instance. The Task 11
carry-forward list was incomplete — treat the ledger, not that list, as the authoritative
inventory of generator gaps.

**Final single-return-builder tally: 4** (atlas-messages/character, atlas-npc-shops/character,
atlas-monsters/monster/consumable, atlas-maps/data/map/info) — all from batch 1, confirmed by two
reviewers and by the ledger. Only 4 packages, so teaching the generator both `Build()` arities is
almost certainly not worth it versus hand-writing 4; that closes Task 10's open question.

### Step 3 — `handwork-dom04.tsv`: 108 rows

91 non-tier-A rows + 17 tier-A skips. **The plan predicted "≈81 rows plus any tier-A skips"; the
actual non-tier-A count is 91, not 81** — another instance of the plan's prose counts being off,
as with the per-service tier-A counts. Size Tasks 13-18 from this file, not from the plan.

`atlas-storage/atlas.com/storage/asset` ("Transform already declared") is deliberately **excluded**
— it is already conformant, so counting it as hand work would invent work that does not exist.
The batch-2 reviewer independently confirmed a correctly-signed `Transform` at
`services/atlas-storage/atlas.com/storage/asset/processor.go:100`.

### Enforced-rule violation found and fixed in the committed ledgers

6 ledger rows embedded the literal absolute worktree path
(`<HOME>/source/atlas-ms/atlas/.worktrees/...`) inside type-check error text — 1 in
`ledger-transform-channel.tsv` (landed back in Task 11), 4 in `-rest-1.tsv`, 1 in `-rest-3.tsv`.
These are committed files under `docs/`, where CLAUDE.md's "never a literal home/absolute path"
rule is *enforced*. Sanitized to repo-relative with sed; row count unchanged at 94.

Two things worth carrying:
- The absolute path originates from passing an **absolute `-repo`** to the codemod, which then
  appears in `go/types` error strings. Batch 2 passed a relative repo path and produced clean
  rows, which is why its reviewer correctly reported "no absolute paths" for that file alone.
- **The verify.sh path guard did not catch this** across three PASSing gates, so it presumably
  scopes to `docs/**/*.md` and not `.tsv`. Worth a look before relying on it for non-markdown
  artifacts under `docs/`.

### Task 12 batch 3 gate — PASS

`tools/verify.sh --quick --base 2a84f41f9` over `2a84f41f9..d0ba529db` -> exit 0. Exactly the
8 committed batch-3 modules (`change base 2a84f41f9 — 22 changed path(s)`), **lint & format
guard (8 modules)** green. **Last gated commit: `d0ba529db`.**

All three batch gates PASS. The Steps 2-4 reconciliation commit that follows is docs-only and
joins the next gate's range.

### Task 12 batch 3 review — APPROVED (0 blocking, 0 non-blocking, 2 not-evaluable)

Artifact: `task-12-batch-3-review.md`. Ledger rows counted and de-duplicated, per-service commit
scoping confirmed, and the `npc-conversations/petdata` skip independently confirmed as a clean
zero-diff skip with an accurately characterized failure mode. The 0-single-return-builder claim
was checked against **all three** batches' ledgers, not just batch 3.

Both not-evaluable items are closed by the batch-3 gate, which ran after the review was
dispatched: it built/vetted all 8 modules and ran the gofumpt lint & format guard green. The
reviewer re-ran only 2 of 8 and had no gofumpt available; the gate covers both.

## Task 12: COMPLETE

All three batches implemented, reviewed APPROVED, and gated PASS. Reconciliation (Steps 2-4)
committed as `9de2c0b6b`.

- 29 services, 94 tier-A packages, 76 APPLIED / 18 SKIPPED, ledger set-identical to
  `classify-dom04.tsv` with no dupes.
- `handwork-dom04.tsv` written: **108 rows** (91 non-tier-A + 17 tier-A skips). This is the input
  to Tasks 13-18 — size them from this file, not from the plan's prose.
- Last gated commit: `d0ba529db`. The `9de2c0b6b` reconciliation commit is docs-only and joins
  the next gate's range.

## Next session starts at: Task 13

Carry into Task 13 and the remaining W3 hand-work tasks:

1. **Size from `handwork-dom04.tsv` (108 rows), not the plan.** The plan predicted ~81; it also
   over-counted per-service tier-A packages in Task 12 and named a `-append` flag that does not
   exist. Treat plan prose counts as estimates and re-derive from the tsv every time.
2. **The generator's ledger is the authoritative gap inventory, not the Task 11 carry-forward
   list** — that list omitted the byte-overflow category, which had already occurred in Task 11.
3. **Only 4 single-return-builder packages exist.** Hand-write those 4; do not teach the
   generator both `Build()` arities. This closes Task 10's open question.
4. `atlas-storage/.../asset` is already conformant and is deliberately excluded from the
   hand-work list. Do not "fix" it.
5. Pass a **relative** `-repo` to the codemod. An absolute one leaks the worktree path into
   `go/types` error text and thence into committed ledgers under `docs/`, violating the enforced
   no-absolute-path rule. 6 such rows were sanitized in `9de2c0b6b`.
6. The verify.sh path guard did not flag those rows across three PASSing gates — it likely scopes
   to `docs/**/*.md`, not `.tsv`. Do not rely on it for non-markdown artifacts under `docs/`.

## Task 13 — implemented, reviewed APPROVED

Commits `22bb8c1`, `39c6b08`, `19e500d` (code) + `0ae9f94` (report). Four packages:
`atlas-channel/data/tradeability`, `atlas-channel/monsterbook`, `atlas-inventory/data/tradeability`,
`atlas-monster-book/data/consumable`.

- `monster-book/data/consumable`'s `Transform` was already generated in Task 12; verified, not
  regenerated. Only the test was added.
- The plan's prose for the monsterbook `collection` subtest described a `[]Card` field that
  `Collection` does not have. The implementer built against `Collection`'s real fields; the
  reviewer independently confirmed that call and that the round trip is still meaningful.
- Review: `task-13-review.md`, 0 blocking / 0 non-blocking / 0 not-evaluable.
- **Process incident, worth carrying:** the Task 13 reviewer ran a bare `git stash pop` in this
  worktree and surfaced an unrelated pre-existing stash entry as a conflict on `agent-ledger.tsv`.
  It resolved cleanly (`git checkout HEAD -- <path>`, stash list unchanged), but this is exactly
  the shared-stash hazard CLAUDE.md forbids. Reviewer dispatches should say so explicitly.

## Task 14 — scope correction (controller ruling)

The plan's Task 14 named **11** remaining `NO-RESTMODEL` packages including
`services/atlas-dragons/atlas.com/dragons/dragon` and
`services/atlas-summons/atlas.com/summons/summon`. Re-derived from `classify-dom04.tsv`: neither
is a `NO-RESTMODEL` row, and both `rest.go` files are handler-only (`InitResource` +
`handleGet*`, zero `Extract`, no `RestModel`). **They are out of scope — do not "fix" them.**

The real set is 12 `NO-RESTMODEL` rows, 3 of them closed by Task 13, leaving **9**:

| package | `Extract` count |
|---|---|
| `atlas-cashshop/.../rewardpool` | 0 (wire type only — needs a ruling) |
| `atlas-drops/.../data/foothold` | 1 |
| `atlas-messengers/.../character` | 1 |
| `atlas-parties/.../character` | 1 |
| `atlas-pets/.../data/position` | 1 |
| `atlas-saga-orchestrator/.../rates` | 1 |
| `atlas-saga-orchestrator/.../reactor/drop` | 1 |
| `atlas-npc-conversations/.../npc/conversation` | 18 |
| `atlas-tenants/.../tenants/configuration` | 9 |

Split into three batches: **A** = the first four rows above; **B** = pets/position, saga/rates,
saga/reactor/drop; **C** = npc/conversation; **D** = tenants/configuration. Batches are
serialized, not parallel — concurrent implementers would race on the index in a single worktree.

### Task 13 gate — FAIL (gofumpt), fix queued

`tools/verify.sh --quick --base d0ba529db` -> exit 1, **lint & format guard (3 modules)**:
`lint.sh: FMT FAIL — services/atlas-channel/atlas.com/channel`, gofumpt wants a blank line between
`fields()` and `setFields()` on each of the five compartment wire types in
`data/tradeability/rest.go`. Cosmetic only; no build/vet/test failure.

`atlas-inventory/.../data/tradeability` is the same generated shape and almost certainly has the
same defect — the guard reported 3 modules. Fix both, plus whatever else `gofumpt -l` names in the
Task 13 range.

**Last gated commit is still `d0ba529db`** — the Task 13 range is not yet clean. The fix commit
joins the next gate's range; do not gate it separately.

Carry-forward: hand-written `Transform`/helper additions need `gofumpt -w` before commit. The
module-local `go build/vet/test` an implementer runs does **not** catch this — only the gate's lint
& format guard does. Say so in hand-work briefs.

## Task 14 batch A — implemented, reviewed APPROVED_WITH_FINDINGS

Commits `d936016` (cashshop/rewardpool), `d633c9a` (drops/data/foothold), `3ef2660`
(messengers/character), `f74f2b0` (parties/character), `1769bdb` (handwork-notes + report).
Review: `task-14a-review.md` — 0 blocking, 2 non-blocking, 0 not-evaluable.

Two non-blocking findings, both carried forward rather than fixed:

1. **`cashshop/rewardpool` — `Transform` without an `Extract`.** The package has no `Extract` at
   all; the implementer wrote `TransformReward` against the inline wire->domain mapping at
   `processor.go:37-43`. The reviewer called it defensible and grounded in real code, but noted a
   recorded exemption would have been an equally reasonable, more conservative reading. **Ruling:
   keep the `Transform`.** It is a real inverse of a real mapping, and an exemption for a package
   that *can* round-trip would be the weaker artifact. Revisit only if Task 25 finds the exemption
   set reads better without it.
2. **No RED run was staged** — tests and implementation were written together rather than
   fail-then-pass. The reviewer independently confirmed all four round-trip tests use distinct
   non-zero values and are non-tautological (would fail on a mis-mapped field), and re-ran them
   GREEN. Accepted. **Future hand-work briefs should state that the RED transcript is required
   evidence, not a formality** — this is the second task where it was skipped.

`handwork-notes.md` now carries one design-§8.1 entry per batch-A package, absolute-path clean.

### Gate over Task 13 + 14a + fmt fix — PASS

`tools/verify.sh --quick --base d0ba529db` -> exit 0. `29 changed path(s), 7 changed Go module(s)`;
go build/vet (7 modules), go analyzer guards, skill/job id guard, scope guard, producer seam guard,
env domain guard, and **lint & format guard (7 modules)** all green.

**Last gated commit: `ac45438a8`.**

The gofumpt fix is `9eb0a2f12`. Only 2 of the 15 changed `.go` files needed it — both
`data/tradeability/rest.go` (atlas-channel and atlas-inventory), the generic `fields()`/`setFields()`
helper pairs.

Two mechanics worth reusing:
- The guard's own entry point is `tools/lint.sh --check --fmt --go <module-root>` (check) and
  `tools/lint.sh --fmt --go <module-root>` (rewrite). Use the guard's own tool rather than a bare
  `gofumpt` binary — it is what the gate actually runs.
- After a rewrite inserts a blank line, **alignment may need a second `--fmt` pass to converge.**
  One pass is not always a fixed point.

## Task 14 batch B — implemented, reviewed APPROVED_WITH_FINDINGS

Commits `0642cd0` (atlas-pets/data/position), `1e6c650` (atlas-saga-orchestrator rates +
reactor/drop), `3c489d8` (handwork-notes + report). Review: `task-14b-review.md` — 0 blocking,
1 non-blocking, 0 not-evaluable.

This batch did stage the RED run, and `tools/lint.sh --check --fmt --go` was clean on both module
roots before commit. The reviewer went further than the batch-A one and proved the tests
non-tautological **by live mutation** — swapped a source field in `position` and in `rates`, saw
each test fail with a field-level diff, reverted. That is the bar; ask for it explicitly.

### Pre-existing gofumpt defect the NEXT gate will surface

`services/atlas-pets/atlas.com/pets/data/position/rest.go` fails `gofumpt -l` on **import
grouping**. The reviewer confirmed it is pre-existing — present at `0642cd0^`, not introduced by
this batch — but the file is now in the changed set, so the lint & format guard will flag it the
first time a gate covers `atlas-pets`.

**Expect that gate FAIL and fix it directly; it is not a regression of batch B.** Use
`tools/lint.sh --fmt --go services/atlas-pets/atlas.com/pets`, then re-check with `--check`.
Remember a rewrite can need a second `--fmt` pass to converge.

## Task 14 status: batches A and B done, C and D remain

Remaining `NO-RESTMODEL` packages (the 9-row set ruled on above, 7 now closed):

| package | `Extract` count | batch |
|---|---|---|
| `atlas-npc-conversations/.../npc/conversation` | 18 | **C — not started** |
| `atlas-tenants/.../tenants/configuration` | 9 | **D — not started** |

Both are large (`rest.go` is 980 and 1093 lines). Give each its own implementer dispatch; do not
combine them. `npc/conversation` carries the design §8.2 scope note: FR-1 applies only to types
`rest.go` already declares a `RestModel` for and that `Extract` already round-trips — the 20+
builder-backed types in `model.go` (2430 lines) get **no** `Transform`.

Batch briefs live at `.superpowers/sdd/plan/task-14-brief-a.md` and `-b.md`; write `-c.md` and
`-d.md` in the same shape. The `-b.md` brief is the better template — it carries the RED-run
requirement and the gofumpt step that `-a.md` lacked.

## Next session starts at: Task 14 batch C

Carry forward, in addition to the Task 12/13 items above:

1. **Last gated commit: `ac45438a8`.** Everything after it (batch B + tracking commits) is
   ungated and joins the next gate's range.
2. `tools/lint.sh --check --fmt --go <module-root>` is the guard's own entry point — use it, not a
   bare `gofumpt`. A rewrite may need two `--fmt` passes to converge.
3. Require the **RED transcript** and **mutation-proof of non-tautology** in hand-work briefs.
   Batches A and B differed on exactly this and A was the weaker unit.
4. `cashshop/rewardpool` keeps its `TransformReward` despite having no `Extract` — ruled on above.
   Do not re-open it outside Task 25.
5. Serialize implementer dispatches. Two implementers committing in one worktree race on the
   git index. Reviewers and verifiers are read-only and can overlap an implementer.
6. `SendMessage` was disabled this session, so a running agent could not be corrected mid-flight.
   Put everything an implementer needs in the brief up front.

## Gate: `ac45438a8..13dabad91` — PASS

`tools/verify.sh --quick --base ac45438a8`, exit 0, log `.superpowers/sdd/gates/gate-14b.log`.
Selected 2 modules (`atlas-pets`, `atlas-saga-orchestrator`); `lint & format guard (2 module(s))`
passed. **The predicted `atlas-pets/.../data/position/rest.go` gofumpt failure did not
materialize** — the guard did not flag it. Treat the batch-B warning about it as closed; no fix
needed.

**Last gated commit is now `13dabad91`.**

## Task 14 batch C — dispatched

Brief `.superpowers/sdd/plan/task-14-brief-c.md`. Controller pre-derived the inventory so the
implementer does not: `conversation/rest.go` has **18** `Extract*` and **14** `Transform*`; the
four missing pairs are `Choice`, `Operation`, `Outcome`, `Option`, and all four are currently
**inlined at their call sites**. Scope is: extract those four into named `Transform*`, repoint the
inline callers, cover all 18 pairs with a round-trip test. Wire-type naming here is `Rest<X>Model`,
not `<X>RestModel`. `RestConditionModel` has neither `Extract` nor `Transform` — out of scope,
recorded in the note.

## Task 14 batch D — brief written, NOT yet dispatched (serialized behind C)

Brief `.superpowers/sdd/plan/task-14-brief-d.md`. **Controller finding that changes batch D's
shape:** `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` has 9 `Extract*` and
**9 matching `Transform*` already present** — coverage is complete pre-task. There is no
`Transform` to write. Batch D is therefore round-trip *test* coverage plus the exemption note,
not implementation.

`KiteConfig` (`rest_test.go:252`) and `RpsReward` (`rest_test.go:14`) already have real round-trip
tests; `InstanceRoute` has partial coverage (`rest_test.go:189`, effect-attributes only). The
brief's Step 1 makes the implementer confirm whether `TradeConfig` and `Rankings` are already
covered in `trade_config_test.go` / `rankings_test.go` before writing duplicates.

Because nothing is implemented, the per-test **mutation proof** is the only non-tautology
evidence for this batch — the brief makes it the explicit gate.

Dispatch D only after C's implementer has committed (one implementer per worktree at a time).

## Task 14 batch C — complete, gate PASS

Commits `d714923` (four `Transform*` + de-inlined callers + `rest_transform_test.go`, 18/18
round-trip subtests) and `f1ef34e` (handwork notes). Implementer reported DONE, no concerns;
module-local `go build`/`go vet`/`go test` clean across 23 packages and
`tools/lint.sh --check --fmt --go services/atlas-npc-conversations/atlas.com/npc` → OK.

Gate `13dabad91..f1ef34e`: `tools/verify.sh --quick --base 13dabad91`, exit 0, log
`.superpowers/sdd/gates/gate-14c.log`. **Last gated commit is now `f1ef34e`.**

`task-14c-review.md` was dispatched separately; its verdict is recorded below when it lands.

### Task 14c review: APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable

Artifact `docs/tasks/task-263-backend-guideline-conformance/task-14c-review.md`. Strongest review
unit in this plan so far. The reviewer independently: confirmed all four inline mapping blocks are
gone and replaced by calls to the named `Transform*`; read all four `Extract`/`Transform` pairs
side by side and confirmed true inverses over the mapped fields; and proved non-tautology by
**two live mutations of its own** (zeroing `Meso` in `TransformOption`, blanking `NextState` in
`TransformOutcome`), each producing field-level-diff failures in the expected subtests, both
reverted and confirmed clean. It also re-ran `tools/lint.sh --check` and the module build/vet
itself rather than trusting the report, and cross-checked all 18 `rest.go:<line>` references in the
handwork note against the file.

Scope confirmed clean: `model.go` untouched, no `TransformCondition`, no files outside the package.

**Task 14 batch C is closed.** Batch D is the last unit of Task 14.

## Task 14 batch D — implemented, gate + review in flight

Commits `c4ff028` (5 new round-trip tests in `configuration/rest_test.go`) and `4f6252b`
(handwork notes). Implementer DONE, no concerns. 12/12 round-trip tests pass (5 new, 7
pre-existing); full-module `go build`/`go vet`/`go test` clean;
`tools/lint.sh --check --fmt --go services/atlas-tenants/atlas.com/tenants` → OK. Mutation proof
captured and reverted-clean for all 5 new tests, with `rest.go` verified byte-identical to backup
afterward. No asymmetry bug found; `rest.go` left read-only as the brief required.

**Observation carried forward (not acted on, out of batch D's scope):** pre-existing
`services/atlas-tenants/atlas.com/tenants/configuration/processor_test.go:848`
`TestMtsConfigRoundTrip` never calls `ExtractMtsConfig`, so it was never real pair evidence
despite its name. The new `TestMtsConfigTransformExtractRoundTrip` fills that gap. The batch-D
reviewer was asked to confirm this claim independently.

### Gate `f1ef34e..4f6252b` — PASS

`tools/verify.sh --quick --base f1ef34e`, exit 0, log `.superpowers/sdd/gates/gate-14d.log`.
**Last gated commit is now `4f6252b`.** Batch D's review verdict is recorded below when it lands.

### Task 14d review: APPROVED — 0 blocking, 1 non-blocking, 0 not-evaluable

Artifact `docs/tasks/task-263-backend-guideline-conformance/task-14d-review.md`. The reviewer
re-derived the mutation proof independently rather than trusting the report: mutated 3 of the 5
`Transform*` bodies itself (`Route`/`startMapId`, `MtsConfig`/`commissionRate`,
`InstanceRoute`/`capacity`), confirmed each new test fails with a specific field-level diff, and
confirmed `rest.go` byte-identical after revert. It also independently read
`trade_config_test.go:48`, `rankings_test.go:37`, and `processor_test.go:848` to check the
no-duplicate and flagged-observation claims — all confirmed as stated, including that
`TestMtsConfigRoundTrip` never calls `ExtractMtsConfig`. Exemption note at
`handwork-notes.md:26` is repo-relative and matches the batch A–C form.

The single non-blocking item is not a defect in the batch: the working tree carried unrelated
pre-existing local modifications (`agent-ledger.tsv`, `progress.md`, untracked
`task-14c-review.md`) outside the reviewed range. Those are this controller's own tracking
artifacts.

## TASK 14 IS COMPLETE — all four batches (A, B, C, D) landed, gated, and reviewed

Every `NO-RESTMODEL` package from the 9-row set is closed. Last gated commit: `4f6252b`.

## Task 15 (tier B1) — Step 1 partition, and a correction to the plan's sizing

The plan estimates "≈37 packages" for tier B1. **The actual B1 row count is 11 packages across 9
services.** Verified against `docs/tasks/task-263-backend-guideline-conformance/classify-dom04.tsv`:

| service | B1 packages |
|---|---|
| `atlas-channel` | 3 — `channel/merchant`, `channel/messenger`, `channel/party` |
| `atlas-consumables` | 1 — `consumables/data/consumable` |
| `atlas-doors` | 1 — `doors/data/map` |
| `atlas-drops` | 1 — `drops/party` |
| `atlas-guilds` | 1 — `guilds/party` |
| `atlas-inventory` | 1 — `inventory/data/consumable` |
| `atlas-npc-shops` | 1 — `npc/data/consumable` |
| `atlas-party-quests` | 1 — `party-quests/party` |
| `atlas-trades` | 1 — `trades/data/inventory` |

Nine batches, one service each, per the plan's own batching rule. Confirmed by grep that **every
one of the 11 packages currently has zero `Transform*` functions** — 23 `Transform*` to write in
total. Unlike Task 14, these packages all have a `type RestModel`, so Task 15 requires **no**
exemption note and touches no `docs/` file.

Two shape clusters worth exploiting (noted in the briefs so implementers copy rather than
re-derive): five near-identical `party` packages (`Extract` + `ExtractMember`) and three
near-identical `data/consumable` packages (`Extract` + `ExtractReward`).

Briefs written: `.superpowers/sdd/plan/task-15-common.md` (shared steps, naming, RED + mutation
evidence requirements) plus nine `task-15-brief-<service>.md` carrying the per-package `### Files`
inventory and the exact pair table. Generator kept at `.superpowers/sdd/plan/gen-task-15-briefs.sh`.

Dispatches are **serialized** — one implementer at a time, per the standing ruling that two
implementers committing in one worktree race on the git index.

### Task 15 batch `atlas-channel` — implemented, gate + review in flight

Commit `e6ed5f642` — 7 `Transform*` across `channel/merchant` (3), `channel/messenger` (2),
`channel/party` (2). Implementer DONE. RED confirmed (`undefined: Transform*` in all three
packages) → GREEN 7/7 subtests → mutation proof passed per package. Full-module
`go build`/`go vet`/`go test` clean; `tools/lint.sh --check --fmt --go` clean on first pass.

**Process near-miss worth propagating:** during its own mutation proof the implementer reverted
with a blind `sed` and briefly corrupted an unrelated line in `party` that carried identical text.
`go build` caught it and it repaired the line by hand. The reviewer was asked to independently
confirm nothing survived — every changed line in the diff must be explained by a new `Transform*`.

**Ruling carried into every subsequent Task 15 brief:** do the mutation proof with a precise,
uniquely-anchored edit, never a blind `sed`, and confirm `git diff --exit-code` on the file after
reverting.

Gate launched for `4f6252b..e6ed5f642` → `.superpowers/sdd/gates/gate-15-channel.log`.
Next batch dispatched: `atlas-consumables`.

#### Task 15 `atlas-channel` review: APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable

Artifact `docs/tasks/task-263-backend-guideline-conformance/task-15-channel-review.md`.

- All 7 pairs verified field-for-field as true inverses by reading both bodies.
- `TransformBlacklistName` correctly emits only `Name`, leaving `BlacklistRestModel.Id` unmapped —
  matching `Extract`, which also ignores `Id`. Same for `TransformVisitEntry` / `VisitRestModel.Id`.
- Reviewer ran 3 independent live mutations (one per package), each producing the expected
  field-level subtest failure, then restored and confirmed `git diff --exit-code` clean.
- **The reported `sed` near-miss left no trace.** The reviewer read the full diff of all three
  `rest.go` against `4f6252b`: every hunk is pure insertion (67/27/31 lines, **zero `-` lines**),
  and the `party` region around the corrupted literal shows no diff at all.
  `git status --porcelain -- services/atlas-channel` is empty.
- No accessors minted, no `docs/` file touched, exactly 6 files in the commit.

**Scope note discovered in review, not in my brief's pair table:** the commit also added
`transformListing` in `services/atlas-channel/atlas.com/channel/merchant/listing.go:53`, the
inverse of the `ExtractListing` that `merchant`'s `Extract` calls. That is a displaced `Extract`
the controller's `rest.go`-only grep did not see. The reviewer confirmed it is a correct, required
part of the merchant round trip and approved it.

**Carry forward:** the per-service pair tables were built from `grep '^func Extract' <pkg>/rest.go`.
A package whose `Extract` delegates to a helper in a sibling file (`listing.go` here) has pairs the
table misses. Remaining batches should grep the whole package directory, not just `rest.go`.

### Gate `4f6252b..e6ed5f642` (atlas-channel) — PASS

`tools/verify.sh --quick --base 4f6252b`, exit 0, log `.superpowers/sdd/gates/gate-15-channel.log`.
**Last gated commit is now `e6ed5f642`.**

### Task 15 batch `atlas-consumables` — implemented, gate + review in flight

Commit `4f82f105d` — `Transform` + `TransformReward` for `consumables/data/consumable`.
Implementer DONE, no concerns; 2/2 subtests, module-local build/vet/test clean,
`tools/lint.sh --check --fmt --go` OK. Gate launched for `e6ed5f642..4f82f105d` →
`.superpowers/sdd/gates/gate-15-consumables.log`.

Its reviewer was additionally asked to run `grep -rn '^func Extract' <pkg>/` over the whole package
directory and treat any delegated `Extract*` with no inverse as **blocking** — the completeness gap
the `atlas-channel` batch exposed.

Next batch dispatched: `atlas-inventory` (near-identical to `atlas-consumables`; brief points it at
`4f82f105d` as template while warning that field lists must still be confirmed package by package).

**Task 15 progress: 2 of 9 batches landed** (`atlas-channel`, `atlas-consumables`).
Remaining: `atlas-inventory` (in flight), `atlas-npc-shops`, `atlas-doors`, `atlas-drops`,
`atlas-guilds`, `atlas-party-quests`, `atlas-trades`.

### Gate `e6ed5f642..4f82f105d` (atlas-consumables) — PASS

`tools/verify.sh --quick --base e6ed5f642`, exit 0, log
`.superpowers/sdd/gates/gate-15-consumables.log`. **Last gated commit is now `4f82f105d`.**

#### Task 15 `atlas-consumables` review: APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable

Artifact `docs/tasks/task-263-backend-guideline-conformance/task-15-consumables-review.md`.
Both pairs verified as true field-for-field inverses. Reviewer ran its own uniquely-anchored
mutations on both (`incFatigue` in `Transform`, `worldMsg` in `TransformReward`), got field-level
failures, reverted to a clean `git diff --exit-code`. Independent `grep` for `Extract*` outside
`rest.go` found **none** — no delegated-pair gap in this package. Commit is fully additive
(`rest.go +91`, `rest_test.go +112/-1`); the two pre-existing tests are byte-identical; no
accessors minted (D1); no `docs/` file and no sibling consumable package touched.

**Task 15 progress: 2 of 9 batches landed, gated PASS, and reviewed APPROVED**
(`atlas-channel` `e6ed5f642`, `atlas-consumables` `4f82f105d`).

### Task 15 batch `atlas-inventory` — implemented, gate + review in flight

Commit `056cd3b` — `Transform` + `TransformReward` for `inventory/data/consumable`. Implementer
DONE, no concerns; 2/2 subtests, full-module build/vet/test green, `tools/lint.sh --check` OK.
Gate launched for `4f82f105d..056cd3b` → `.superpowers/sdd/gates/gate-15-inventory.log`.

**Important finding: the three `data/consumable` packages are NOT identical.** The implementer
confirmed field lists by direct read of `model.go`/`rest.go` rather than copying the
`atlas-consumables` template, and reports this package's `RewardModel` has **fewer** fields. Its
reviewer was asked to verify that claim field by field, treating any carried-over sibling-only
field as blocking.

**Ruling carried into the `atlas-npc-shops` brief:** near-identical siblings are a template for
*shape*, never for *field lists*. Each implementer confirms its own package's fields and states in
its report which fields differ from the siblings.

Next batch dispatched: `atlas-npc-shops` (last of the three `data/consumable` packages).

### Gate `4f82f105d..056cd3b` (atlas-inventory) — PASS

`tools/verify.sh --quick --base 4f82f105d`, exit 0, log `.superpowers/sdd/gates/gate-15-inventory.log`.
**Last gated commit is now `056cd3b`.**

#### Task 15 `atlas-inventory` review: APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable

Artifact `docs/tasks/task-263-backend-guideline-conformance/task-15-inventory-review.md`.

**The copied-template risk was checked and is clean.** `atlas-consumables`' `RewardModel`
(`services/atlas-consumables/atlas.com/consumables/data/consumable/model.go:257-264`) carries
`effect`, `worldMsg`, `period` on top of `itemId`/`count`/`prob`; `atlas-inventory`'s
(`services/atlas-inventory/atlas.com/inventory/data/consumable/model.go:202-206`) has only the
three shared fields, and `TransformReward` (`rest.go:254-260`) emits exactly those three — **no
field carried over from the template.**

Reviewer ran its own line-anchored mutations on both pairs (`rest.go:113` `Price`, `rest.go:257`
`Count`), each producing a genuine field-level failure, then reverted to a clean
`git diff --exit-code`. `grep -rn '^func Extract'` over the whole package directory returns only
`Extract` and `ExtractReward` — no delegated-pair gap. Commit touches only `rest.go` and
`rest_test.go`; no accessors minted, no `docs/` file, siblings untouched.

**Task 15 progress: 3 of 9 batches landed, gated PASS, and reviewed APPROVED**
(`atlas-channel` `e6ed5f642`, `atlas-consumables` `4f82f105d`, `atlas-inventory` `056cd3b`).
Remaining: `atlas-npc-shops` (in flight), `atlas-doors`, `atlas-drops`, `atlas-guilds`,
`atlas-party-quests`, `atlas-trades`.

### Task 15 batch `atlas-npc-shops` — implemented, gate + review IN FLIGHT AT HANDOFF

Commit `38810baa9` — `Transform` + `TransformReward` for `npc/data/consumable`. Implementer DONE:
RED confirmed, GREEN 3/3, mutation proof caught a field-level swap and reverted cleanly,
module-local lint/build/vet/test all clean, no `docs/` file touched.

The implementer claims its `Model`/`RewardModel` field lists are **byte-identical** to
`atlas-inventory`'s. The reviewer was asked to verify that claim by direct diff rather than accept
it — the equivalent assumption was already disproved once for `atlas-consumables`, which carries
three extra `RewardModel` fields (`effect`, `worldMsg`, `period`). It was also asked to establish
what the third test covers, since the pair count is 2.

- Gate launched: `056cd3b..38810baa9` → `.superpowers/sdd/gates/gate-15-npc-shops.log`
- Review dispatched → `docs/tasks/task-263-backend-guideline-conformance/task-15-npc-shops-review.md`

**Both were still running when this controller session handed off. Reconcile them from those two
files before dispatching the next batch** — the gate log's last lines carry the verdict, and the
review artifact opens with `verdict:`.

## NEXT SESSION STARTS AT: Task 15, batch `atlas-doors`

State at handoff:

1. **Last gated commit: `056cd3b`.** `38810baa9` is covered by the in-flight gate above; if that
   log shows exit 0, the last gated commit becomes `38810baa9`.
2. **Task 15 is 4 of 9 batches landed.** Done: `atlas-channel` `e6ed5f642`, `atlas-consumables`
   `4f82f105d`, `atlas-inventory` `056cd3b`, `atlas-npc-shops` `38810baa9`. Remaining, one
   dispatch each: **`atlas-doors`, `atlas-drops`, `atlas-guilds`, `atlas-party-quests`,
   `atlas-trades`.**
3. **Briefs are already written** — `.superpowers/sdd/plan/task-15-common.md` (shared steps,
   naming, RED + mutation requirements) plus `task-15-brief-<service>.md` for each remaining
   service. They carry the `### Files` inventory and the exact pair table. Do not regenerate them;
   the generator is at `.superpowers/sdd/plan/gen-task-15-briefs.sh` if a regeneration is ever
   needed.
4. **Dispatch prompts should reuse the shape used for batches 1-4** (see the dispatch record in
   this file): worktree cwd discipline, the four carried-forward rulings below, and an explicit
   "ambiguity I have already resolved" section.
5. **Standing rulings for every remaining Task 15 batch:**
   - Confirm the pair table by grepping the **whole package directory**, not just `rest.go`. Extra
     `Extract*` found are **in scope**; a missing entry or a pre-existing `Transform*` is a
     `NEEDS_CONTEXT` disagreement.
   - Mutation proof uses a precise uniquely-anchored edit, **never a blind `sed`**, with
     `git diff --exit-code` confirmed after revert.
   - Near-identical sibling packages are a template for **shape only, never field lists**.
   - Task 15 needs **no** exemption note and touches **no** `docs/` file.
6. **Serialize implementers** — one at a time. Reviewers and gates are read-only and may overlap.
7. **After Task 15:** Tasks 16-27 remain (16 tier B2, 17 tier C, 18 DOM-04 close-out, 19-21 W1
   `relocate`, 22-23 W1 hand work, 24 FILE-05, 25 `exemptions.md`, 26 behavior preservation,
   27 final gate + review).

### Gate `056cd3b..38810baa9` (atlas-npc-shops) — PASS

`tools/verify.sh --quick --base 056cd3b`, exit 0, log `.superpowers/sdd/gates/gate-15-npc-shops.log`.
**Last gated commit is now `38810baa9`.** This resolves item 1 of the handoff state above: no gate
is outstanding. Only the `atlas-npc-shops` review artifact remains to be reconciled.

#### Task 15 `atlas-npc-shops` review: APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable

Artifact `docs/tasks/task-263-backend-guideline-conformance/task-15-npc-shops-review.md`. Landed
just after the handoff note above was written — **nothing is left outstanding; the next session
does not need to reconcile anything.**

The byte-identity claim was verified, not accepted: the reviewer diffed sorted field lists and full
struct bodies against `atlas-inventory` and confirmed identical (57 `Model` fields, 3
`RewardModel` fields), refuting the `atlas-consumables`-style divergence risk. It then
programmatically enumerated all 57 `Model` fields and confirmed each appears in `Transform`
(`rest.go:92-171`), with `Model.HandsIncrease()` — a hardcoded-0 accessor, not a field — correctly
absent from both `Transform` and `Extract`. Own live mutation on `TransformReward`
(`Prob: m.prob` → `m.count`) failed both subtests with field-level diffs, reverted clean.
`grep -rn '^func Extract'` over the package found no orphaned delegate. Fixture uses distinct
non-default values throughout.

**Task 15 progress: 4 of 9 batches landed, gated PASS, and reviewed APPROVED**
(`atlas-channel` `e6ed5f642`, `atlas-consumables` `4f82f105d`, `atlas-inventory` `056cd3b`,
`atlas-npc-shops` `38810baa9`). Last gated commit `38810baa9`. Next batch: `atlas-doors`.

### Task 15 batch `atlas-doors` — commit `75c61ac`, gate PASS, review APPROVED

`Transform` + `TransformPortal` for `data/map` (2 pairs). Implementer DONE, no concerns:
inventory matched the brief exactly (2 `Extract*`, no pre-existing `Transform*`), RED confirmed,
mutation proof a precise field-drop edit with clean revert, module-local build/vet/test/lint clean.

- Gate `38810baa9..75c61ac`: `tools/verify.sh --quick --base 38810baa9` exit 0,
  log `.superpowers/sdd/gates/gate-15-doors.log`. **Last gated commit is now `75c61ac`.**
- Review APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
  `docs/tasks/task-263-backend-guideline-conformance/task-15-doors-review.md`.
  The reviewer enumerated every `Portal`/`Model`/`PortalRestModel`/`RestModel` field from
  `model.go` and confirmed full coverage in the new `Transform*`; ran its own live mutation
  (`Name: p.name` → `"MUTATED"`), saw field-level failures in both subtests, reverted clean.
  No hardcoded-value accessors exist on these structs. Fixtures use distinct non-zero values
  with sign-differentiated X/Y. Commit touches only `rest.go`/`rest_test.go` — no `docs/` file.

**Task 15 progress: 6 of 9 batches — 5 landed+gated+reviewed** (`atlas-channel` `e6ed5f642`,
`atlas-consumables` `4f82f105d`, `atlas-inventory` `056cd3b`, `atlas-npc-shops` `38810baa9`,
`atlas-doors` `75c61ac`); **`atlas-drops` implementer in flight.** Remaining after it:
`atlas-guilds`, `atlas-party-quests`, `atlas-trades`.

### Task 15 batch `atlas-drops` — commit `a614c62c9`, gate PASS, review APPROVED

`Transform` + `TransformMember` for `party` (2 pairs). Implementer DONE, no concerns: package
sweep found no extras and no pre-existing `Transform*`, no sibling `party` package touched.

- Gate `75c61ac..d12f656da`: `tools/verify.sh --quick --base 75c61ac` exit 0, log
  `.superpowers/sdd/gates/gate-15-drops.log`. **Last gated commit is now `d12f656da`**
  (covers `a614c62c9` plus the docs-only record commit).
- Review APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
  `docs/tasks/task-263-backend-guideline-conformance/task-15-drops-review.md`. The reviewer
  enumerated every field of `Model`/`MemberModel`/`RestModel`/`MemberRestModel` **and** the
  embedded `field.Model` from `libs/atlas-constants/field/model.go`, confirming coverage on both
  sides. Own live mutation (`WorldId: m.field.WorldId()` -> `world.Id(0)`) failed both subtests on
  `reflect.DeepEqual`, reverted clean. No hardcoded-value accessors on these types. Fixtures use
  distinct non-zero values including distinct non-nil UUIDs.

**Task 15 progress: 7 of 9 batches — 6 landed+gated+reviewed** (adds `atlas-drops` `a614c62c9`);
**`atlas-guilds` implementer in flight.** Remaining after it: `atlas-party-quests`, `atlas-trades`.

### Task 15 batch `atlas-guilds` — commit `69d96c7ce`, gate PASS, review APPROVED

`Transform` + `TransformMember` for `party` (2 pairs).

- Gate `d12f656da..9c425c940`: `tools/verify.sh --quick --base d12f656da` exit 0, log
  `.superpowers/sdd/gates/gate-15-guilds.log`. **Last gated commit is now `9c425c940`**
  (covers `69d96c7ce` plus the docs-only record commit).
- Review APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
  `docs/tasks/task-263-backend-guideline-conformance/task-15-guilds-review.md`.
  - The "3 passing subtests for 2 pairs" discrepancy I flagged is benign: it is Go's own count of
    the parent test plus 2 subtests. `grep -rn '^func Extract\|^func Transform' party/` shows
    exactly the 2 documented pairs, no hidden third.
  - Every field of `Model`/`MemberModel`/`RestModel`/`MemberRestModel`, including the embedded
    `field.Model` reached via `Field()`/`WorldId()`/`ChannelId()`/`MapId()`/`Instance()`, is
    mapped on both sides. Those accessors are pure delegations, not hardcoded-value accessors.
  - The reviewer deliberately mutated a *different* field than the implementer's proof did
    (`Level: 0` in `TransformMember` vs. the implementer's `Instance`), saw field-level failure,
    reverted clean.

**Task 15 progress: 8 of 9 batches — 7 landed+gated+reviewed** (adds `atlas-guilds` `69d96c7ce`);
**`atlas-party-quests` implementer in flight.** Remaining after it: `atlas-trades`.

### Task 15 batch `atlas-party-quests` — commit `4d3ab3b`, gate PASS, review APPROVED

`Transform` + `TransformMember` for `party` (2 pairs).

- Gate `9c425c940..6c840dba1`: `tools/verify.sh --quick --base 9c425c940` exit 0, log
  `.superpowers/sdd/gates/gate-15-party-quests.log`. **Last gated commit is now `6c840dba1`**
  (covers `4d3ab3b` plus the docs-only record commit).
- Review APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
  `docs/tasks/task-263-backend-guideline-conformance/task-15-party-quests-review.md`.
  - **Worth noting for the remaining conformance work:** this package's `MemberRestModel` carries
    five JSON fields (`Name`, `Level`, `JobId`, `MapId`, `Online`) that `TransformMember` does
    *not* map. That is correct here — the pre-existing `ExtractMember` (`rest.go:134-140`) never
    mapped them either, so the inverse is faithful to the existing contract. The reviewer
    confirmed this against `ExtractMember`'s own body rather than by sibling-package analogy.
    This package's `Model`/`MemberModel` are genuinely narrower than `atlas-guilds`'
    (`id, leaderId, members` / `id, worldId, channelId`; no embedded `field.Model`) — another
    data point that the five `party` packages are NOT interchangeable.
  - Independent mutation (dropped `WorldId: m.worldId` from `TransformMember`) produced
    field-level failures, reverted clean.
  - The reviewer flagged a modified `services/atlas-trades/.../rest_test.go` in the working tree.
    **Benign and expected** — that is the concurrently running `atlas-trades` implementer, the
    ninth and final Task 15 batch. Not a stray edit.

**Task 15 progress: 9 of 9 batches — 8 landed+gated+reviewed** (adds `atlas-party-quests`
`4d3ab3b`); **`atlas-trades` implementer in flight — the final batch.**

### Task 15 batch `atlas-trades` — commits `5e2b905ac` + `21c22e538`, gate PASS, review APPROVED

`Transform` + `TransformAsset` for `data/inventory` (2 pairs). Implementer returned
**DONE_WITH_CONCERNS**: its first commit overwrote a pre-existing `rest_test.go` (it used `Write`
on a file it believed was new, destroying five existing tests); it caught this in self-review via
`git show --stat`, recovered the originals from `git show HEAD~1:...`, and landed `21c22e538` as a
corrective commit.

- Gate `6c840dba1..c3ad6b3c4`: `tools/verify.sh --quick --base 6c840dba1` exit 0, log
  `.superpowers/sdd/gates/gate-15-trades.log`. **Last gated commit is now `c3ad6b3c4`.**
- Review APPROVED — 0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
  `docs/tasks/task-263-backend-guideline-conformance/task-15-trades-review.md`. The overwrite was
  made the review's primary charge and the recovery was verified, not accepted:
  `git diff 6c840dba1..21c22e538 --stat -- .../data/inventory/` → `2 files changed, 78
  insertions(+)`, **zero deletions**; all five original test functions intact byte-for-byte, with
  the new `TestTransformRoundTrip` appended after them (`rest_test.go:142-184`). No other file in
  the package was similarly overwritten.
- Field coverage enumerated from this package's own `model.go`/`rest.go`, explicitly not borrowed
  from the near-identical `atlas-inventory`. Independent mutation on `InventoryType` (the
  implementer had proved with `Flag`) failed at field level and reverted clean.

**Process note carried forward:** the `Write`-on-an-assumed-new-file hazard is real and cost a
recovery commit. Future batches of this shape should `test -f` the target test file before writing
it. Recovery worked here only because the implementer self-reviewed with `git show --stat`.

## TASK 15 COMPLETE — 9 of 9 batches landed, gated PASS, and reviewed APPROVED

`atlas-channel` `e6ed5f642`, `atlas-consumables` `4f82f105d`, `atlas-inventory` `056cd3b`,
`atlas-npc-shops` `38810baa9`, `atlas-doors` `75c61ac`, `atlas-drops` `a614c62c9`,
`atlas-guilds` `69d96c7ce`, `atlas-party-quests` `4d3ab3b`, `atlas-trades`
`5e2b905ac`+`21c22e538`. Zero blocking findings across all nine reviews. Last gated commit
`c3ad6b3c4`; no gate or review outstanding.

## NEXT SESSION STARTS AT: Task 16 (tier B2)

Tasks 16-27 remain: 16 tier B2, 17 tier C, 18 DOM-04 close-out, 19-21 W1 `relocate`, 22-23 W1
hand work, 24 FILE-05, 25 `exemptions.md`, 26 behavior preservation, 27 final gate + review.
Read `plan.md` for Task 16's definition; briefs for Task 16 are not yet generated
(`tools/task-brief.sh docs/tasks/task-263-backend-guideline-conformance/plan.md 16`).

---

# Task 16 — W3 hand work, tier B2

## Step 1: batch partition (recorded per brief Step 1)

`awk -F'\t' '$2=="B2"'` over `classify-dom04.tsv` yields **61 packages across 24 services**, not
the ≈37 the plan estimated. The plan's figure is a stale estimate; the TSV is authoritative and 61
is the number executed. atlas-channel alone carries 21.

Ten batches (no two concurrent batches share a Go module, so per-service commits cannot collide):

| Batch | Packages | Services |
|---|---|---|
| `channel-a` | 5 | atlas-channel (account, buddylist, character/buff, character/teleportrock, data/item) |
| `channel-b` | 5 | atlas-channel (data/map, data/npc/template, data/portal, data/skill, data/skill/effect) |
| `channel-c` | 5 | atlas-channel (door, guild, guild/thread, monster, monster/information) |
| `channel-d` | 6 | atlas-channel (mts/configuration, parcel, pet, quest, reactor, trade) |
| `character` | 5 | atlas-character ×4, atlas-consumables ×1 |
| `messages-storage` | 7 | atlas-messages ×4, atlas-storage ×3 |
| `doors-summons-reactors` | 7 | atlas-doors ×2, atlas-summons ×2, atlas-reactors ×3 |
| `login-monsters-npcconv-qagg` | 8 | atlas-login ×2, atlas-monsters ×2, atlas-npc-conversations ×2, atlas-query-aggregator ×2 |
| `misc-a` | 6 | atlas-cashshop, atlas-dragons, atlas-inventory, atlas-kites, atlas-maps, atlas-merchant |
| `misc-b` | 7 | atlas-mts, atlas-npc-shops, atlas-portals, atlas-saga-orchestrator ×2, atlas-trades, atlas-transports |

Briefs: `.superpowers/sdd/plan/task-16-brief-<batch>.md`, each carrying the fact block, the plan
section verbatim, and a per-package `### Files` inventory that states for every package whether
`rest_test.go` already exists — the direct countermeasure to the Task-15 `atlas-trades`
`Write`-on-an-assumed-new-file overwrite.

**Ruling — no codemod for tier B2.** `docs/codemod-vs-agents.md` asks this before a second
implementer at the same transformation. B2 is by construction the residue that is *not* templated:
every row's evidence string names a distinct disqualifying reason (control flow, intermediate
assignment, builder chain, displaced declaration, non-composite first result). The plan already
partitions the templated work into Tasks 19-21 (`relocate` codemod); Task 16 is titled "hand work"
for this reason. Hand fan-out stands.

**Last gated commit entering Task 16: `c3ad6b3c4`.** (`596dbf1ad` is docs-only and joins the next
gate's range.)

### Task 16 batch `channel-a` — commit `aea948a4a`, implementer DONE

5 atlas-channel packages (account, buddylist, character/buff, character/teleportrock, data/item).
Implementer reported 5/5 `TestTransformRoundTrip` PASS and a clean module `go build && go vet &&
go test`. It confirmed resolution #3 held in practice: `buff.RestModel.Id` and
`teleportrock.RestModel.Id` are fields the existing `Extract` never reads, so `Transform` does not
populate them — faithful to `Extract`, not to the JSON surface. No `handwork-notes.md` entry (no
genuinely lossy round trip).

**Gate deferred deliberately.** Six implementers are still mid-edit in this worktree, so
`tools/verify.sh` would be reading a working tree that is never clean and could report a failure
belonging to another batch's half-finished state. Per the "at most one gate in flight / the gate
covers a range, not a task" rule, Task 16's commits accumulate and one gate runs over
`c3ad6b3c4..HEAD` once the concurrent wave settles. Last gated commit remains `c3ad6b3c4`.

### Task 16 batch `messages-storage` — commits `8167f428d` + `1d5205164`, implementer DONE

7 packages: atlas-messages (data/skill, data/skill/effect, pet, rate) and atlas-storage
(data/consumable, data/etc, data/setup). TDD RED confirmed (`undefined: Transform` ×7) then GREEN;
full `go build && go vet && go test` clean on both modules.

**Concurrency hazard confirmed real, and caught.** During `git add services/atlas-storage` this
implementer's index picked up another concurrent agent's staged `services/atlas-summons/...` files.
It unstaged them with `git restore --staged services/atlas-summons` and verified via
`git show --stat HEAD` that the commit touched only its 6 intended atlas-storage files. This is
exactly the failure the `git commit -- <pathspec>` instruction was meant to prevent — the agent
used `git add` + `git commit` rather than the pathspec form, and the shared index bit it. The
recovery was correct and atlas-summons' working tree was untouched (`git restore --staged` does not
write the working tree), so no damage. Reviewers for the affected batches must confirm commit
contents against the batch's declared file list rather than trusting the message.

### Task 16 batches `character` and `doors-summons-reactors` — implementers DONE

- `character` — `1cf400a29` (atlas-character ×4) + `f2a89d447` (atlas-consumables ×1). All 5
  round-trips pass; both module gates clean.
- `doors-summons-reactors` — `5a292ffba` (atlas-doors ×2), `2bac98f26` (atlas-summons ×2),
  `e4d483d9b` (atlas-reactors ×3). All 7 round-trips pass; three module gates clean. No lossy
  `Extract` found, so no `handwork-notes.md` entry.

**Correction to the `character` implementer's own report.** It reported that its
`handwork-notes.md` addition "was independently folded into another concurrent agent's commit
`4796e262a`". That is false: `git show --stat 4796e262a` contains exactly two files, both under
`services/atlas-portals/atlas.com/portals/portal/` (`rest.go`, `rest_test.go`), 42 insertions, no
docs file. The agent also described the `docs/tasks/.../` directory as "likely gitignored" — it is
tracked. What is actually true: `handwork-notes.md` was last committed in `de782187a` (the
atlas-login commit, which therefore swept a docs file outside its declared `services/atlas-login`
pathspec), and the file is presently `M` in the working tree carrying the `character` and `misc-a`
entries. The controller will land it as an explicit docs commit once the wave settles — no content
is lost, but no implementer's account of where it went can be trusted.

**Concurrency cost tally so far.** Three of the five landed batches reported index interference
(`messages-storage` cross-staged atlas-summons; `character` cross-staged unrelated files and hit
`index.lock` contention; `de782187a` swept a docs file). Every instance was caught by the agent's
own `git diff --cached --stat` check and recovered without data loss, and every commit's contents
are being independently re-verified in review rather than taken on the implementer's word. The
lesson for the remaining batches: the `git commit -- <pathspec>` form is load-bearing under
concurrency, and `git add` + `git commit` is not safe in a shared worktree.

### Task 16 batch `channel-a` — review APPROVED

0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
`docs/tasks/task-263-backend-guideline-conformance/task-16-channel-a-review.md`. Field coverage was
re-derived per package from each `model.go` (account 14, buddylist 5, character/buff 7,
teleportrock 2, data/item 2) rather than by sibling analogy. The implementer's
"`Extract` never reads `RestModel.Id`" claim was checked against the actual `Extract` bodies
(`character/buff/rest.go:51-65`, `character/teleportrock/rest.go:41-43`) and holds — both fields
are `json:"-"` and neither `Model` carries an id. `git show --stat`: 267 insertions, 1 deletion,
and the reviewer chased that single deletion down to a mechanical `import "testing"` →
import-block expansion in `data/item/rest_test.go`, not lost test content. Independent mutation
(dropped `Gender` from `account`'s `Transform`) failed at field level and reverted clean.

### Task 16 batch `misc-b` — commits `d4b753fb6`, `8562ea7c4`, `4796e262a`, `95b3d37c7`, `9eba1cb3e`, `59eab9fe1`, implementer DONE

7 packages across 6 services (atlas-mts, atlas-npc-shops, atlas-portals, atlas-saga-orchestrator ×2,
atlas-trades/configuration, atlas-transports). All six module gates clean; 7/7 round-trips pass and
all pre-existing tests in every touched package still pass.

**The Task-15 `Write`-on-an-existing-test-file hazard recurred — in atlas-trades again, and despite
an explicit named warning in this batch's dispatch prompt.** The implementer used `Write` on
`atlas-trades/atlas.com/trades/configuration/rest_test.go`, caught it before any build or commit,
restored the original content verbatim from its prior `Read`, and verified additivity twice
(`git diff --stat` pre-stage, `git diff --cached --stat` pre-commit).

Controller-verified independently, not accepted on report: `git show --stat 9eba1cb3e` →
`2 files changed, 41 insertions(+)`, **zero deletions**. The recovery is real.

**Process conclusion: a prose warning in the brief is not a sufficient control for this failure.**
It has now fired in two consecutive tasks, both times in atlas-trades, and both times was caught
only by the implementer's own post-hoc self-check. The durable fix is a preflight `test -f` (or
`Read`-before-`Write`) discipline that the agent cannot skip, not stronger wording. Recorded here
for Task 27's close-out and for whatever revises the implementer contract.

### Task 16 batch `messages-storage` — review APPROVED

0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
`docs/tasks/task-263-backend-guideline-conformance/task-16-messages-storage-review.md`. The
cross-staging incident was resolved to the repository, not to the implementer's word: both commits'
paths stay wholly inside their own service, and
`git log --oneline c3ad6b3c4..HEAD -- services/atlas-summons` shows `2bac98f26` intact — the
unstage worked and nothing leaked or was lost. All 7 `Transform`s were checked field-by-field
against their own `model.go`; no cross-package borrowing despite the look-alike siblings.
`data/skill/effect/rest.go:64`'s `RestModel.CardStats` is confirmed never read by `Extract` and
correctly left unset. Both commits purely additive (287 / 104 insertions, 0 deletions). Mutation on
`consumable`'s `Rechargeable` failed at field level, reverted clean.

### Task 16 batch `login-monsters-npcconv-qagg` — commits `de782187a`, `127f007`, `04af499`, `8bdcf00`, implementer DONE

8 packages across 4 services — the widest batch. All four module gates green.

**One genuinely lossy round trip found, and handled per resolution #4 rather than papered over.**
`atlas-login/atlas.com/login/character`'s `Extract` (`rest.go:97`) is a builder chain that never
calls `SetSpawnPoint`, `SetPets`, `SetEquipment`, `SetInventory`, `SetRank`, `SetRankMove`,
`SetJobRank`, or `SetJobRankMove`, so eight `Model` fields (`model.go:44`, `:47`-`:53`) cannot
survive `Extract(Transform(m))` — and `RestModel` has no field at all for seven of them. The
implementer asserted only the 26 fields that do round-trip, named them explicitly in the test, and
recorded the gap with `file:line` in `handwork-notes.md`. It did **not** add a `RestModel` field to
close it (PRD §5) and did not relax a builder rule (FR-17). This is the correct outcome for the
case the plan anticipated.

### Task 16 batch `character` — review APPROVED

0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
`docs/tasks/task-263-backend-guideline-conformance/task-16-character-review.md`. Both commits
confirmed purely additive (0 deletions) and scoped to their own service — the implementer's
unreliable git narrative was set aside and the state read from the repository, as instructed.
Field counts re-derived per package from its own `model.go` (configuration 1, atlas-character
data/portal 8, data/skill 5, data/skill/effect 51 + nested, atlas-consumables portal 8). The
`CardStats` claim was verified against `Extract`'s actual body (`rest.go:73-126`) — never read, no
`Model` counterpart, correctly unpopulated. `configuration.Extract` confirmed to have no error
return (`rest.go:29`) and `Transform` matches that shape (`rest.go:38`) — no signature was invented.
Mutation on `data/skill/effect`'s `MonsterStatus` failed at field level and reverted clean.

**Note on the two same-named `portal` packages in this one batch:** the reviewer derived
`atlas-character/data/portal` (8 fields) and `atlas-consumables/portal` (8 fields) separately
rather than asserting equivalence from the matching count. Same field count is not the same field
set, and treating it as such is the exact error the `atlas-party-quests`/`atlas-guilds` finding in
Task 15 was about.

### Task 16 batch `doors-summons-reactors` — review APPROVED

0 blocking, 0 non-blocking, 0 not-evaluable. Artifact
`docs/tasks/task-263-backend-guideline-conformance/task-16-doors-summons-reactors-review.md`. All
three commits `+N/-0`, each scoped to its own service. Both atlas-summons packages confirmed fully
present in `2bac98f26` — the `messages-storage` stage/unstage incident cost this batch nothing.
All seven `Transform`s checked field-by-field against their own `model.go`; no borrowing among the
look-alikes (`atlas-doors/data/skill` vs `atlas-summons/data/skill`, the two `data/skill/effect`s,
the three `reactor/data*` siblings).

Two substantive confirmations rather than assertions:
- The "no lossy fields" claim was tested at its weakest point. `atlas-doors/party` was flagged as
  the likely-lossy case (largest `Extract`, nested members); its `MemberRestModel` carries only
  `Id`, so there is no dropped member attribute. Claim holds for a reason, not by assumption.
- `atlas-summons/.../data/skill/effect/rest.go`'s `Transform` reads the unexported `m.hp` directly
  rather than the narrowing `Hp() int16` getter (`effect/model.go:39`), so no truncation is
  introduced on the way out. This is D1 (read unexported fields directly) doing real work — the
  getter would have silently narrowed the round trip.

Mutation on `reactor/data/state`'s `NextState` failed at field level and reverted clean.

### Task 16 batch `misc-a` — commits `749366aaa`, `a07cefc18`, `c30eb446a`, `159dd280a`, `87dfe8a`, `1d4d9ed`, `8f1e02e`, implementer DONE

6 packages across 6 services. All six module gates pass; every `TestTransformRoundTrip` confirmed
RED (`undefined: Transform`) before implementation and GREEN after.

Two field gaps documented rather than closed (PRD §5 forbids adding a `RestModel` field):
- `atlas-maps/atlas.com/maps/reactor` — `Model.updateTime` (`model.go:24`) has no `RestModel`
  field, so the round trip cannot restore it; the test asserts field-by-field over the 13 fields
  `RestModel` does carry and deliberately omits `UpdateTime`.
- `atlas-merchant/atlas.com/merchant/data/portal` — `RestModel.Target` and `RestModel.ScriptName`
  have no `Model` counterpart and are never read by the existing `Extract`, so `Transform` does not
  emit them (resolution #3).

This batch also landed `8f1e02e`, a docs commit carrying `handwork-notes.md`. That clears the
working-tree modification flagged earlier — the `character` and `misc-a` entries are now committed.
The `login-monsters-npcconv-qagg` entry went in earlier via `de782187a`. No handwork content was
lost anywhere, contrary to the `character` implementer's mistaken account of where its entry went.

**All seven non-channel batches are now landed.** Remaining: `channel-b` (in flight), then
`channel-c` and `channel-d`, which serialize behind it on the shared atlas-channel module.

### Task 16 batch `channel-b` — commits `d2227f582` + `f0a756762`, implementer DONE

5 atlas-channel packages (data/map, data/npc/template, data/portal, data/skill, data/skill/effect).
5/5 round-trips pass; module gate clean.

**Correction to the controller's own commit guidance — this was my error, not the implementer's.**
I instructed every Task 16 batch to use `git commit -m "..." -- <pathspec>` to survive the shared
index under concurrency. That form is correct for *tracked* files but silently skips *untracked*
ones, so the five brand-new `rest_test.go` files could not be committed that way. The implementer
correctly diagnosed this and split the batch into two commits — `d2227f582` for the five `rest.go`
changes, `f0a756762` for the five new test files via `git add` — and verified both are scoped
exclusively to its own five packages. Two commits here is the right outcome, not a deviation.

The corrected recipe for the remaining channel batches: `git add <explicit new file paths>` first,
then `git commit -m "..." -- services/atlas-channel`, then `git show --stat HEAD` to confirm
scope. Never `git add -A`/`git add .`.

**Pre-existing ambiguity noted, correctly not "fixed":** in `data/skill/effect`, `Extract` collapses
a nil `*PointRestModel` and an explicit `&PointRestModel{0,0}` to the same zero `point.Model` for
the `LT`/`RB` fields. The implementer identified this as pre-existing, left it alone, and recorded
it in its report. It is not a `handwork-notes.md` case because no `Model` field is dropped — the
round trip over `Model` still holds. Changing `Extract` to disambiguate would be a behavior change
and out of scope.

### Task 16 batch `login-monsters-npcconv-qagg` — review APPROVED

0 blocking / 0 non-blocking / 0 not-evaluable. Artifact `task-16-login-monsters-npcconv-qagg-review.md`.

The lossy carve-out was verified at its premise, which was the point of making it the primary
charge. `atlas-login/character`'s `Extract` (`rest.go:130-157`) genuinely never calls the eight
setters, **and the test's 26-field assertion list matches exactly the 26 setters `Extract` does
call** — nothing round-trippable was quietly excluded to make a narrowed test pass, and no
non-round-tripping field was wrongly included. That is the failure mode a narrowed assertion
invites, and it is confirmed absent.

Both near-twin pairs were diffed rather than assumed: `atlas-login/guild` `Model` has 3 fields
vs `atlas-query-aggregator/guild`'s 13; `atlas-npc-conversations/pet` is
`{id,templateId,name,level,slot}` vs `atlas-query-aggregator/pet`'s `{id,slot,templateId,closeness}`.
Genuinely different, independently derived. The two pre-existing `rest_test.go` files show only an
import-line reformat as their sole deletion; all pre-existing tests intact. Mutation on
`atlas-query-aggregator/pet`'s `Closeness` failed at field level, reverted clean.

### Task 16 batch `misc-b` — review APPROVED

0 blocking / 0 non-blocking / 0 not-evaluable. Artifact `task-16-misc-b-review.md`.

The `Write` incident is fully closed. The reviewer diffed
`9eba1cb3e^:.../trades/configuration/rest_test.go` against the committed blob directly: purely
additive, and both pre-existing tests (`TestRestModelDecodesTheTenantsWireDocument`,
`TestWireDocumentFoldsIntoTheDomainModel`) are **byte-identical** and passing, alongside the new
`TestTransformRoundTrip`. The rest of the package suite is green too. All six commits show zero
deletions.

Displaced-`Extract` handling confirmed correct: `Transform` at
`services/atlas-npc-shops/atlas.com/npc/data/setup/rest.go:22`, with `Extract` left untouched at
`processor.go:40` — placed per DOM-04 without the out-of-scope relocation.

**Strongest available evidence against cross-package borrowing so far:** atlas-npc-shops populates
`Id` in `Transform` because its `Extract` reads it, while atlas-mts and atlas-trades correctly
leave `Id` unset because theirs never do — three same-shaped packages in one batch diverging
exactly where their own sources diverge. Scope confirmed: `9eba1cb3e` touches only
`trades/configuration`; Task 15's `data/inventory` has zero diff. Mutation on
`atlas-saga-orchestrator/data/npc`'s `TrunkGet` failed at field level, reverted clean.

### Task 16 batch `misc-a` — review APPROVED

0 blocking / 0 non-blocking / 0 not-evaluable. Artifact `task-16-misc-a-review.md`.

Both displaced-`Extract` packages handled correctly: `atlas-cashshop/.../commodity` has `Extract`
untouched at `model.go:37` with `Transform` placed at `rest.go:35-46` (8/8 fields);
`atlas-inventory/.../data/setup` has `Extract` untouched at `processor.go:40` with `Transform` in
`rest.go` (11/11 fields). Neither `Extract` was relocated — the out-of-scope "helpful move" did not
happen.

Both documented field gaps verified at the premise, not the conclusion:
- `atlas-maps/reactor` — `RestModel` genuinely carries no field able to hold `updateTime`
  (`model.go:24`), and `TestTransformRoundTrip` (`rest_test.go:43-81`) asserts exactly the 13
  fields `Transform` emits with nothing restorable silently omitted. Mutation (dropped `Direction`)
  failed with `rest_test.go:74: Direction mismatch. Expected 11, got 0`; reverted clean.
- `atlas-merchant/data/portal` — `Extract`'s body confirmed never to read `rm.Target` or
  `rm.ScriptName`, matching the handwork-notes claim exactly.

No narrowing getters: the accessor-based `Transform`s (dragons, maps) use same-type,
non-narrowing accessors consistent with the `atlas-ban` reference. **The `test -f`-style
inventory in the brief did its job** — `atlas-kites`' pre-existing `rest_test.go` was appended to,
not overwritten, and its five original tests are present and passing. That is the control the
Task-15/`misc-b` `Write` hazard needed, working as intended.

**Task 16 status: 7 of 10 batches landed AND reviewed APPROVED, 0 blocking findings across all
seven.** Remaining: `channel-c` (implementer in flight), `channel-b` (review in flight),
`channel-d` (6 packages, not yet dispatched — must follow `channel-c` on the shared atlas-channel
module).

### Task 16 batch `channel-c` — commit `623378a`, implementer DONE

5 atlas-channel packages (door, guild, guild/thread, monster, monster/information). 5/5 round-trips
pass; module gate clean. Two resolution-#3/#4 cases recorded in `handwork-notes.md` under
"Batch `channel-c`": `guild/thread`'s `tenantId`/`guildId` (never populated by `Extract`) and
`monster`'s `DamageEntries` (never read by `Extract`).

Landed as a single commit — the corrected `git add` + explicit-pathspec `git commit` recipe worked.

### Task 16 batch `channel-b` — review APPROVED

0 blocking / 0 non-blocking / 0 not-evaluable. Artifact `task-16-channel-b-review.md`.

The `LT`/`RB` ambiguity claim was verified in all three parts rather than accepted: the collapse is
real (`point.Model` zero-value equality), it is genuinely pre-existing (`git show
d2227f582^:...rest.go` shows identical `Extract` nil-check logic before the commit), and — the part
that mattered — `data/skill/effect/rest_test.go:69-70` uses **non-zero** points, so the passing
test is not masking the ambiguity. Field coverage independently derived per package; both commits
purely additive, 10 expected files, no deletions, no overlap with `channel-a`'s ten packages.
Mutation on `data/portal`'s `ScriptName` failed as expected and reverted clean.

## TASK 16 STATE — 8 of 10 batches landed; 8 reviewed APPROVED, 0 blocking findings overall

Landed + reviewed APPROVED: `channel-a` `aea948a4a`; `channel-b` `d2227f582`+`f0a756762`;
`character` `1cf400a29`+`f2a89d447`; `messages-storage` `8167f428d`+`1d5205164`;
`doors-summons-reactors` `5a292ffba`+`2bac98f26`+`e4d483d9b`;
`login-monsters-npcconv-qagg` `de782187a`+`127f007`+`04af499`+`8bdcf00`;
`misc-a` `749366aaa`+`a07cefc18`+`c30eb446a`+`159dd280a`+`87dfe8a`+`1d4d9ed`+`8f1e02e`;
`misc-b` `d4b753fb6`+`8562ea7c4`+`4796e262a`+`95b3d37c7`+`9eba1cb3e`+`59eab9fe1`.

Landed, review NOT yet dispatched: `channel-c` `623378a` (door, guild, guild/thread, monster,
monster/information).

NOT yet dispatched at all: `channel-d` — 6 atlas-channel packages (mts/configuration, parcel, pet,
quest, reactor, trade). Brief already generated at
`.superpowers/sdd/plan/task-16-brief-channel-d.md`.

## NEXT SESSION — three things remain before Task 16 is complete

1. **Dispatch `channel-d`** (`task-implementer`, `model: sonnet`) with
   `.superpowers/sdd/plan/task-16-brief-channel-d.md`. Use the corrected git recipe: `git add` the
   explicit new `rest_test.go` paths first (a plain `git commit -- <pathspec>` will NOT pick up
   untracked files), then `git commit -m "..." -- services/atlas-channel`, then `git show --stat
   HEAD`. Carry forward resolutions #1-#8 as written into the `channel-c` dispatch — that prompt is
   the best template; reuse it verbatim with the package list swapped.
2. **Dispatch the `channel-c` review** (`task-reviewer`, `model: sonnet`) over `623378a`, and the
   `channel-d` review once it lands. Primary charge for `channel-c`: verify at the premise that
   `guild/thread`'s `tenantId`/`guildId` and `monster`'s `DamageEntries` are genuinely never
   touched by `Extract`, and that no restorable field was excluded from the assertion to make a
   narrowed test pass.
3. **Run the deferred gate** once no implementer is editing:
   `tools/verify.sh --quick --base c3ad6b3c4 > "$CLAUDE_JOB_DIR/tmp/gate-16.log" 2>&1`
   backgrounded. It was deliberately held for the whole task because 7 concurrent implementers
   meant the working tree was never clean and a failure could not be attributed to the right batch.
   **Last gated commit is still `c3ad6b3c4`** — one gate covers the entire Task 16 range.
   Also check `git status` for an uncommitted `handwork-notes.md` (batch `channel-c` appended to
   it) and land it as a docs commit.

Then Tasks 17-27 remain: 17 tier C, 18 DOM-04 close-out, 19-21 W1 `relocate` codemod, 22-23 W1 hand
work, 24 FILE-05, 25 `exemptions.md`, 26 behavior preservation, 27 final gate + review.

### Task 16 batch `channel-c` — review APPROVED_WITH_FINDINGS (0 blocking, 2 non-blocking)

Artifact `task-16-channel-c-review.md`, over `623378a` (10 non-doc files, purely additive, no
overlap with `channel-a`/`channel-b`).

Both carve-outs verified at the premise rather than accepted: `guild/thread`'s `tenantId`/`guildId`
(`model.go:11-12`) are never set anywhere in the package (full-package grep sweep), and `monster`'s
`RestModel.DamageEntries` (`rest.go:33`) is never read by `Extract`. The reviewer built its own
field-by-field inventory for all five packages and found **no restorable field excluded from an
assertion to narrow a passing test**; fixtures use distinct non-zero values throughout. Its own live
mutation (dropping `TownY` in `door/rest.go`'s `Transform`) failed with an isolated diff and reverted
clean.

Non-blocking, both accepted as-is — no fix round:
1. `door/rest.go:52`, `monster/rest.go:62`, `monster/information/rest.go:29` build the `RestModel`
   literal through exported getters instead of direct unexported field access (D1's stated shape),
   while `guild` and `guild/thread` use direct access. Functionally identical — no getter touched
   has side effects — but stylistically inconsistent inside the batch. **Ruling: not worth a fix
   round.** The round trip is what DOM-04 asserts and it holds; a getter-vs-field style sweep, if
   wanted, belongs to Task 18's DOM-04 close-out where it can be done uniformly across all 61 B2
   packages rather than in one batch.
2. `monster/information/rest.go:44-49` — `Extract` silently defaults `id` to 0 on a parse failure.
   Pre-existing, confirmed unmodified by this commit (`git show 623378a^`), observation only; PRD §2
   puts pre-existing deviations out of scope.

**Task 16: 9 of 10 batches landed and reviewed, 0 blocking findings overall. `channel-d`
implementer in flight — the final batch.** Last gated commit still `c3ad6b3c4`; the single Task 16
gate runs once `channel-d` lands.

### Task 16 batch `channel-d` — commit `d4731b0`, implementer DONE (final batch)

6 atlas-channel packages (mts/configuration, parcel, pet, quest, reactor, trade). 6/6 round-trips
GREEN; module-local build/vet/test clean; no concerns. Two carve-outs appended to
`handwork-notes.md`: `pet.RestModel.Lead` (never read by `Extract` — resolution #3) and
`reactor.Model.updateTime` (genuinely lossy — resolution #4, narrowed assertion), with
mutation-tested proof of non-tautology on `reactor`.

**All 10 Task 16 batches have now landed.** Gate launched over `c3ad6b3c4..d4731b0` — the single
gate covering the entire Task 16 range, deliberately deferred until no implementer was editing the
shared worktree.

# Task 17 — W3 hand work, tier C

## Step 1 re-sizing: the plan's tier-C premise does not hold for 5 of the 7 rows

The plan describes tier C as "≈7 packages that declare both a `Model` and a `RestModel` but no
converter in either direction." The 7 `C` rows of `classify-dom04.tsv` are real, but the classifier's
evidence string for every one of them is only `no Extract in package` — it never checked that a
`Model` exists or that a `Transform` is absent. Read directly, the 7 rows split four ways:

**Genuine tier C — 1 package.** `Model` + `RestModel`, no converter either direction:
- `services/atlas-channel/atlas.com/channel/maps/location` — `Model` `model.go:15`, `RestModel`
  `rest.go:20`, no `rest_test.go`. This is the only row matching the plan's description.

**Misnamed converter — 1 package.**
- `services/atlas-character/atlas.com/character/session/history` — `Model` `model.go:10`, `RestModel`
  `rest.go:11`, and a converter that already exists under a non-conforming name:
  `TransformToRest(m Model) RestModel` (`rest.go:33`) and `TransformSliceToRest` (`rest.go:44`). The
  sole external caller is `resource.go:66` (`TransformSliceToRest`); `TransformToRest` has no caller
  outside `rest.go`. **Ruling:** rename to `Transform` with FR-1's signature
  `func Transform(m Model) (RestModel, error)` returning a nil error, and adapt
  `TransformSliceToRest` to call it. This is a rename plus an error-return widening, not a behavior
  change — the function is total today and stays total.

**Already conformant, misplaced file — 2 packages.** `Transform` exists with the right name and
FR-1 signature; it simply lives in `resource.go` next to the `RestModel`, while `rest.go` in these
packages holds the HTTP handlers (an inverted file convention):
- `services/atlas-dragons/atlas.com/dragons/dragon` — `Transform` `resource.go:41`
- `services/atlas-summons/atlas.com/summons/summon` — `Transform` `resource.go:44`
Task 18's `sweep.sh` greps `^func Transform(` package-wide, so both already satisfy DOM-04. **Ruling:
no DOM-04 code work and no file move here** — file placement is FILE-* business and belongs to Tasks
19-21 (`relocate`) / Task 24, per the same "a displaced declaration is a separate pre-existing
deviation" rule applied throughout Task 16. Task 17 adds only the missing `TestTransform` coverage.

**Structurally exempt — 3 packages.** A `RestModel` with **no `Model` anywhere in the package**;
these are inbound-only DTOs deserialized from another service's JSON:API response, so there is no
domain type to convert from and `Transform` cannot be written by construction:
- `services/atlas-consumables/atlas.com/consumables/monster` — `RestModel` `rest.go:11`, no `Model`
- `services/atlas-data/atlas.com/data/skill` — `RestModel` `rest.go:8`, no `Model` (the `Model` lives
  in the `effect/` subpackage, which has its own `Transform` at `effect/rest.go:132`)
- `services/atlas-maps/atlas.com/maps/character` — `RestModel` `rest.go:11`, no `Model`; the file's
  own comment states it is "the minimal projection of the atlas-character JSON:API resource needed by
  atlas-maps"
Verified by grepping `type Model` / `Model struct` / `Model interface` across every `.go` file in
each package — zero hits. **Ruling:** these are genuine exemptions, not deferred work. Record each in
`handwork-notes.md` with the `file:line` of the `RestModel` and the absence finding, and carry them
into Task 25's `exemptions.md` as a new `NO-MODEL` class alongside D2's `NO-RESTMODEL`.

Net Task 17 code work: **2 packages** (`maps/location` new `Transform`; `session/history` rename),
plus `TestTransform` coverage for those 2 and for `dragons/dragon` + `summons/summon`, plus 3
documented exemptions. One batch.

### Task 16 batch `channel-d` — review APPROVED (0 blocking, 1 non-blocking)

Artifact `task-16-channel-d-review.md`, over `d4731b0` (12 files, purely additive, confined to
`services/atlas-channel`, no overlap with the channel-a/b/c package sets).

Both carve-outs verified against source rather than against the implementer's notes:
- `pet.RestModel.Lead` (`rest.go:49-74`, `model.go:77-79`) — `Extract` never reads it and `Model` has
  no backing field, so the full `reflect.DeepEqual` test is correctly *un*narrowed.
- `reactor.Model.updateTime` (`model.go:16`, `rest.go:14-28`) — `RestModel`'s 13 fields carry no
  timestamp, and the test narrows exactly one field (13 explicit comparisons plus a zero-value
  assertion on `UpdateTime`). Nothing else was silently dropped.

Independently derived field inventories for all 6 packages found no restorable field excluded from an
assertion. The reviewer's own mutation on `parcel` (`FeePaid: m.FeePaid() + 1` — a package the
implementer did not mutate) failed the round trip and reverted clean. Fixtures use distinct non-zero
values including non-empty slices in `pet.Excludes` and `quest.Progress`; `Extract` error paths assert
`err == nil` throughout.

Non-blocking: `pet` and `reactor` build the `RestModel` through exported getters where same-package
unexported access was available — the same getter-vs-field inconsistency flagged in `channel-c`.
Accepted; tracked as input to Task 18's DOM-04 close-out, where it can be swept uniformly across all
61 B2 packages.

## TASK 16 COMPLETE — 10 of 10 batches landed and reviewed, 0 blocking findings across the whole task

`channel-a` `aea948a4a`; `channel-b` `d2227f582`+`f0a756762`; `channel-c` `623378a`;
`channel-d` `d4731b0`; `character` `1cf400a29`+`f2a89d447`; `messages-storage` `8167f428d`+`1d5205164`;
`doors-summons-reactors` `5a292ffba`+`2bac98f26`+`e4d483d9b`;
`login-monsters-npcconv-qagg` `de782187a`+`127f007`+`04af499`+`8bdcf00`;
`misc-a` `749366aaa`+`a07cefc18`+`c30eb446a`+`159dd280a`+`87dfe8a`+`1d4d9ed`+`8f1e02e`;
`misc-b` `d4b753fb6`+`8562ea7c4`+`4796e262a`+`95b3d37c7`+`9eba1cb3e`+`59eab9fe1`.

61 tier-B2 packages, one `Transform` and one round-trip test each. The single deferred gate over
`c3ad6b3c4..d4731b0` is the only outstanding item.

### Task 16 gate `c3ad6b3c4..d4731b0` — **FAIL** (lint & format guard, 24 modules)

Log `.superpowers/sdd/gates/gate-16.log`. Everything else passed: `go build/vet` across all selected
modules, go analyzer guards, skill/job id guard, scope guard, producer seam guard, env domain guard.
The single failing check is `lint & format guard`, 9 failing targets across 5 services.

**Caution for future sessions — the background-task notification reported "exit code 0" and the gate
had in fact FAILED.** The launch command was `tools/verify.sh ... > log 2>&1; echo "GATE EXIT: $?"`,
so the reported status was the trailing `echo`'s exit, not `verify.sh`'s. Launch a gate as the last
command in the pipeline with nothing after it, and always confirm the verdict by reading the log's
final lines rather than trusting the notification's exit code.

Two failure classes, both inside Task 16's own new code:

1. **gofumpt formatting — 8 new `rest_test.go` files** (mechanical):
   `atlas-channel` `data/skill/effect/rest_test.go:3`, `data/skill/rest_test.go:3`,
   `guild/rest_test.go:4`; `atlas-character` `data/skill/effect/rest_test.go:3`,
   `data/skill/rest_test.go:3`; `atlas-messages` `data/skill/effect/rest_test.go:3`,
   `data/skill/rest_test.go:3`; `atlas-monsters` `monster/information/rest_test.go:78`.
2. **staticcheck S1016 — 4 sites**, "should convert X to Y instead of using struct literal", i.e. a
   field-identical struct conversion written as a literal:
   `atlas-channel/.../monster/information/rest.go:32:29` (`AttackInfo` -> `AttackInfoRestModel`);
   `atlas-monsters/.../monster/information/rest.go:92:27` (`Skill` -> `skill`) and `:96:29`
   (`AttackInfo` -> `AttackInfoRestModel`); `atlas-summons/.../data/skill/effect/rest.go:31:29`
   (`StatChange` -> `StatupRestModel`).

Per `/execute-task` Step 4c a gate FAIL becomes a review finding fed to an implementer, not a
controller-side fix. The fix commit joins the next gate's range. **Last gated commit remains
`c3ad6b3c4`** — nothing has passed a gate since.

### Task 16 fix round — commit `6a575d0`, implementer DONE

Clears both gate FAIL classes. `tools/lint.sh --check --fmt --go` now reports `lint.sh: OK` on all 5
affected modules; `go build/vet/test` clean on each.

12 files, +27/-39. I inspected the diff rather than accepting the report:
- The 9 `rest_test.go` files are **whitespace and import-order only**. The largest
  (`atlas-monsters/.../monster/information/rest_test.go`, 14 +/-) is purely struct-literal key
  alignment — every value, and every assertion, is byte-identical. No reviewed test assertion moved.
- The 4 S1016 sites became genuine Go struct conversions — `AttackInfoRestModel(a)`, `skill(s)`,
  `StatupRestModel(su)`. Go only compiles a struct conversion between field-identical types, so the
  build passing *is* the proof the premise held; no struct was reshaped to make one legal.

One file beyond the brief's enumeration: `atlas-channel/.../guild/thread/rest_test.go` needed the
same import-order fix and was included. Correct call — it is the same defect class from the same
task, and leaving it would have failed the next gate.

### Task 16 gate re-run `c3ad6b3c4..37a8799f1` — **PASS**

`tools/verify.sh --quick --base c3ad6b3c4`, exit 0, log `.superpowers/sdd/gates/gate-16-rerun.log`.
All checks green including `lint & format guard (24 modules)`. Docker bake skipped as expected for
`--quick` — per CLAUDE.md this is the inner loop, not a pre-PR pass; the flagless `tools/verify.sh`
still runs once at branch end.

**Last gated commit is now `37a8799f1`.** This range covers the entire Task 16 body: all 10 batches,
the lint fix round `6a575d0`, and the docs commits.

**TASK 16 IS COMPLETE AND GATED.** 61 tier-B2 packages, each with a `Transform` and a round-trip
test; 10/10 batches reviewed APPROVED; 0 blocking findings; gate PASS.

### Task 17 — commits `7d81d43`, `9680019`, `a71607d`, `dc1eab8`, implementer DONE

Executed against the four-group re-sizing above, not the plan's stale tier-C description.

- Group 1 (`atlas-channel/.../maps/location`) — new `Transform` + `TestTransform` (`7d81d43`).
  A field-pairing mismatch was found and recorded rather than papered over: `RestModel.Id` /
  `Model.characterId`.
- Group 2 (`atlas-character/.../session/history`) — `TransformToRest` renamed to `Transform` and
  widened to FR-1's `(RestModel, error)` (`9680019`).
- Group 3 (`atlas-dragons/dragon`, `atlas-summons/summon`) — `TestTransform` only (`a71607d`,
  `dc1eab8`). No `Transform` written and no file moved, as ruled. The implementer correctly reported
  **no RED phase** for these two rather than fabricating one, since `Transform` pre-existed.
- Group 4 — three NO-MODEL exemptions grep-verified and written to `handwork-notes.md`.

All four module-local gates clean, and `tools/lint.sh --check --fmt --go` OK on all four — the
gofumpt class that failed the Task 16 gate was caught in-batch this time.

### Task 17 gate `37a8799f1..8203e0d03` — **PASS**

`tools/verify.sh --quick --base 37a8799f1`, exit 0, log `.superpowers/sdd/gates/gate-17.log`.
All checks green including `lint & format guard (4 modules)`. **Last gated commit is now
`8203e0d03`.** Task 17 review still in flight at the time this was written.

### Task 17 review — **CHANGES_REQUIRED** (1 blocking, 1 non-blocking)

Artifact `task-17-review.md`. Groups 2, 3, and 4 all clean; the blocking finding is in Group 1.

**Blocking — `atlas-channel/.../maps/location/rest.go:44`: `Transform` leaves `RestModel.Id` at zero
instead of `m.CharacterId()`.** I verified the reviewer's claim at both ends rather than accepting it:

- `services/atlas-maps/atlas.com/maps/character/location/rest.go:56-63` is the **producer** of this
  exact wire shape and maps `Id: m.CharacterId()` alongside the identical
  `WorldId`/`ChannelId`/`MapId`/`Instance`/`State` set. `atlas-channel`'s `maps/location` is the
  consumer of that same resource.
- Group 3's own `dragon` and `summon` `Transform`s, in this same task, derive `Id` from the entity's
  natural key.

So the `handwork-notes.md` justification — "no case-insensitive name match for `Id`, `Model` exposes
it as `CharacterId()`" — is wrong. This is an omitted legitimate mapping, not an unpairable field.
The plan's Step 1 field-pairing rule ("pair each exported `RestModel` field with the `Model` field
whose name matches case-insensitively") was applied too literally: the JSON:API resource id is
conventionally the entity's natural key and never name-matches. **Serializing a resource id of `0`
is a real defect, not a cosmetic one.** Rule for the rest of the conformance work: a `RestModel.Id`
paired with an obvious natural-key accessor is a mapping, not an exemption.

Non-blocking, and the reason the defect survived: `rest_test.go` never asserts `rm.Id`, which masked
the missing mapping. The assertion goes in with the fix.

### Task 17 fix round 1 — commit `e3faf58`, implementer DONE; blocking finding CLEARED

Diff inspected, not accepted on report: exactly 2 files, +5/-3. `rest.go` adds `Id: m.CharacterId()`
and deletes the 3-line comment that asserted the field was unpairable; `rest_test.go` adds the single
`rm.Id` assertion. No other field mapping touched, no struct field added/removed/reordered.

RED was shown before GREEN — `Id mismatch. Expected 100, got 0` — which is the proof the test was
previously blind to the gap rather than merely proof the fix works. `atlas-channel` module
build/vet/test pass; `tools/lint.sh --check --fmt --go` OK.

`handwork-notes.md` corrected: the `maps/location` entry now records `Id` -> `CharacterId()` matching
the `atlas-maps` producer, instead of claiming an exemption.

## HANDOFF — controller context past threshold after Tasks 16 and 17

**State: Tasks 16 and 17 are both complete. Task 16 is gated PASS. Task 17 is gated PASS at
`8203e0d03`, but its fix commit `e3faf58` is NOT yet gated** — per `/execute-task` Step 4c a fix
commit joins the next gate's range rather than getting its own, so the next session's first gate is
`tools/verify.sh --quick --base 8203e0d03`, covering `e3faf58` plus whatever Task 18 lands.

**Last gated commit: `8203e0d03`.**

### What was done this session

- Task 16 finished: `channel-d` (`d4731b0`) landed and reviewed APPROVED, `channel-c` (`623378a`)
  reviewed APPROVED. All 10 batches / 61 tier-B2 packages done, 0 blocking findings.
- Task 16's single deferred gate FAILED on `lint & format guard` (gofumpt in 9 new `rest_test.go`
  files, staticcheck S1016 at 4 sites). Fix round `6a575d0` cleared it; re-run PASS at `37a8799f1`.
- Task 17 re-sized (see the four-group ruling above — the plan's tier-C premise held for only 1 of 7
  rows), implemented in `7d81d43`/`9680019`/`a71607d`/`dc1eab8`, gated PASS at `8203e0d03`, reviewed
  CHANGES_REQUIRED, fixed in `e3faf58`.

### Two rules established this session, carry them forward

1. **A `RestModel.Id` paired with an obvious natural-key accessor is a mapping, not an exemption.**
   The Task 17 blocking finding was a `Transform` leaving the JSON:API resource id at zero because
   no `Model` field name-matched "Id". The plan's case-insensitive field-pairing rule must not be
   applied to the resource id. Check the *producer* of the wire shape when in doubt.
2. **Launch a gate as the only command in its Bash call.** A trailing `; echo "GATE EXIT: $?"` made
   the background notification report exit 0 for a gate that had FAILED. Always confirm a gate
   verdict by reading the log's final lines.

Also standing from Task 16, unresolved by design: several `Transform`s build the `RestModel` through
exported getters rather than direct unexported field access (D1's stated shape). Flagged non-blocking
in both `channel-c` and `channel-d` reviews and deliberately deferred to **Task 18's DOM-04
close-out**, where it can be swept uniformly across all 61 B2 packages instead of patched per batch.

### Next step

Task 18 (DOM-04 close-out and ledger confirmation). Its plan section is `plan.md` "## Task 18";
generate the brief with `tools/task-brief.sh docs/tasks/task-263-backend-guideline-conformance/plan.md 18`.
Then Tasks 19-27 remain: 19-21 W1 `relocate` codemod, 22-23 W1 hand work, 24 FILE-05,
25 `exemptions.md` (must now include the new **NO-MODEL** class from Task 17 Group 4 alongside D2's
`NO-RESTMODEL`), 26 behavior preservation, 27 final gate + review.

## Task 18 — DOM-04 closeout

Step 1 (`sweep.sh` then `split-by-model.sh`, regenerated in place):

```
   30 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-has-model.txt
  176 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-no-model.txt
```

Step 2 cross-check against `handwork-notes.md` and design §8.2's 14-package `NO-RESTMODEL` list
resolved the 30-line residue as follows:

- **8 already documented** in `handwork-notes.md` (Task 14 batches / Task 17 NO-MODEL section):
  `cashshop/rewardpool`, `consumables/monster`, `data/skill`, `maps/character`,
  `messengers/character`, `npc-conversations/conversation`, `parties/character`,
  `tenants/configuration`.
- **3 false residues** — `sweep.sh` only greps the literal `^func Transform(` inside `rest.go`
  itself; these three packages have a conforming `Transform(Model) (RestModel, error)`, it just
  lives in `resource.go`/`processor.go`, not `rest.go`: `dragons/dragon` (`resource.go:41`),
  `summons/summon` (`resource.go:44`), `storage/asset` (`processor.go:100`). Not defects; documented
  here rather than in `handwork-notes.md` since no hand-work was needed.
- **2 accepted NO-RESTMODEL residue, part of design §8.2's 14 but never added to
  `handwork-notes.md`**: `channel/data/tradeability` and `inventory/data/tradeability` — no
  `type RestModel`; each already has `TransformCash`/`TransformConsumable`/`TransformEquipment`/
  `TransformEtc`/`TransformSetup` (five wire types, five `Model`-typed `Transform*` variants),
  mirroring the `rewardpool` pattern documented in Task 14. No hand-work needed; recorded here as
  the batch-D14 gap-fill.
- **17 genuine forgotten packages — a real DOM-04 gap, not accepted residue.** Every one of these
  is a `Model`+`RestModel` package the W3 codemod's ledger marked `SKIPPED` (design §4.1: "two or
  more bool fields", "unsupported field type ...", "generated code does not type-check ...",
  "Extract maps no fields" — `handwork-dom04.tsv` rows for these paths), and design §10 requires
  "every `SKIPPED` package must appear either in a hand-work commit or in `exemptions.md`" — neither
  happened. Confirmed by direct inspection: none of the 17 has a `func Transform` anywhere in the
  package (not just missing from `rest.go`).

  ```
  services/atlas-cashshop/atlas.com/cashshop/character
  services/atlas-channel/atlas.com/channel/buddylist/buddy
  services/atlas-channel/atlas.com/channel/character
  services/atlas-channel/atlas.com/channel/data/consumable
  services/atlas-channel/atlas.com/channel/data/equipment
  services/atlas-channel/atlas.com/channel/data/quest
  services/atlas-channel/atlas.com/channel/minigame
  services/atlas-channel/atlas.com/channel/mts/listing
  services/atlas-channel/atlas.com/channel/mts/wish
  services/atlas-consumables/atlas.com/consumables/cash
  services/atlas-maps/atlas.com/maps/data/map/info
  services/atlas-messages/atlas.com/messages/character
  services/atlas-messages/atlas.com/messages/data/map
  services/atlas-monsters/atlas.com/monsters/monster/consumable
  services/atlas-monsters/atlas.com/monsters/monster/mobskill
  services/atlas-npc-conversations/atlas.com/npc/petdata
  services/atlas-npc-shops/atlas.com/npc/character
  ```

  **This is a plan-sizing problem, not a single forgotten package** (brief's escalation threshold is
  2; this is 17, spanning ~9 services, each needing an individually-reasoned `Transform` given the
  codemod's own SKIP reasons — ambiguous bool-field mapping, byte-range mismatches, unsupported
  slice/map field types). Per the brief's own instruction ("If it is more than two packages, stop
  and report it as a finding instead"), this was **not** closed in this task. It needs at least one
  follow-up task (a new Task, sized similarly to Tasks 14/16/17) before DOM-04 can be called closed.
  `handwork-dom04.tsv`'s `SKIPPED` rows are the per-package reason and are still read-only input for
  that follow-up.

### Task 18 — D1 getter close-out

Ruling implemented verbatim from the controller addendum: a `Transform` this branch created or
rewrote that reads `m.Field()`/`m.GetField()` instead of `m.field` is a D1 violation and gets fixed
to direct-field form; a pre-existing `Transform` (even one just renamed by this branch) is out of
scope and untouched.

```
git diff --diff-filter=AM eaa5ce6f7..HEAD -- '*rest.go'
```

produced 185 branch-added `func Transform`/`func Transform*` bodies. Filtering the added bodies for
`m.[A-Z]...()` calls found 22 such calls spread over 18 files. One,
`services/atlas-character/atlas.com/character/session/history/rest.go`, is a rename of pre-existing
code (`968001916 feat(atlas-character): rename TransformToRest to Transform in session/history` —
the getter-based body predates this branch) and was left untouched per the ruling. The remaining 21
were genuinely authored by this branch's W3 hand-work (confirmed per-file via
`git log --oneline eaa5ce6f7..HEAD -- <file>`, each landing in a "feat: add Transform..." commit) and
were rewritten to direct unexported-field access — **21 `Transform` functions across the remaining
17 files** (`guilds/party` contributes 2 and `npc/conversation` 4, which is why the function count
exceeds the file count):

- `services/atlas-channel/atlas.com/channel/door/rest.go` — `Transform`
- `services/atlas-channel/atlas.com/channel/maps/location/rest.go` — `Transform` (also updated the
  matching `handwork-notes.md` prose from `m.CharacterId()` to `m.characterId`; the natural-key
  mapping documented in Task 17 is unchanged, only the accessor spelling)
- `services/atlas-channel/atlas.com/channel/monster/information/rest.go` — `Transform`
- `services/atlas-channel/atlas.com/channel/monster/rest.go` — `Transform`
- `services/atlas-channel/atlas.com/channel/mts/configuration/rest.go` — `Transform`
- `services/atlas-channel/atlas.com/channel/parcel/rest.go` — `Transform`
- `services/atlas-channel/atlas.com/channel/pet/rest.go` — `Transform`
- `services/atlas-channel/atlas.com/channel/quest/rest.go` — `Transform`
- `services/atlas-channel/atlas.com/channel/reactor/rest.go` — `Transform`
- `services/atlas-channel/atlas.com/channel/trade/rest.go` — `Transform`
- `services/atlas-character/atlas.com/character/configuration/rest.go` — `Transform`
- `services/atlas-dragons/atlas.com/dragons/character/rest.go` — `Transform`
- `services/atlas-guilds/atlas.com/guilds/party/rest.go` — `Transform` and `TransformMember`
- `services/atlas-inventory/atlas.com/inventory/data/setup/rest.go` — `Transform`
- `services/atlas-maps/atlas.com/maps/reactor/rest.go` — `Transform`
- `services/atlas-npc-conversations/atlas.com/npc/conversation/rest.go` — `TransformChoice`,
  `TransformOperation`, `TransformOutcome`, `TransformOption`
- `services/atlas-trades/atlas.com/trades/data/inventory/rest.go` — `Transform`

Nested `field.Model` access (e.g. `m.field.WorldId()`) was kept as-is — that reads a private field
holding another package's value type and calls *its* own public API, which is the established
pattern already used by non-violating `Transform`s (`atlas-channel/party/rest.go:164-167`,
`atlas-drops/party/rest.go:147-150`). Only `m.<ExportedGetter>()` calls on the `Model`/`MemberModel`
parameter itself were rewritten.

Module-local `go build ./... && go test ./...` (or the package-scoped equivalent) was run and passed
for every module touched: atlas-channel, atlas-character, atlas-dragons, atlas-guilds,
atlas-inventory, atlas-maps, atlas-npc-conversations, atlas-trades. No behavior change — every
rewritten field read returns the identical value the getter it replaces returned.

## CONTROLLER RULING — Task 18b inserted to close the 17 forgotten packages

Task 18's escalation is accepted as correct: 17 `Model`+`RestModel` packages were marked `SKIPPED`
by the W3 codemod and then never given hand-work nor an `exemptions.md` entry, which design §10
forbids. Task 18 was right to stop rather than absorb them.

**They are not deferred and not exempted.** Reviewing every `SKIPPED` reason in
`handwork-dom04.tsv` for these 17 paths, not one says a `Transform` is impossible — they fall into
three closable classes:

1. **Ambiguous bool pairing** (3: `channel/buddylist/buddy`, `channel/data/quest`,
   `channel/minigame`) — "two or more bool fields" means the codemod refused to guess which
   `RestModel` bool pairs with which `Model` bool. A human reads the names and decides.
2. **Unsupported field type** (6: `channel/data/consumable`, `channel/data/equipment`,
   `channel/mts/listing`, `channel/mts/wish`, `consumables/cash`, `monsters/monster/mobskill`) —
   `map[SpecType]int32`, `[]string`, `*time.Time`, `[]uint32`. The codemod only emitted scalar
   copies; a hand-written mapping is ordinary Go.
3. **Generated *test* did not type-check** (7: `cashshop/character`, `channel/character`,
   `maps/data/map/info`, `messages/character`, `messages/data/map`,
   `monsters/monster/consumable`, `npc-conversations/petdata`, `npc-shops/character`) — byte
   overflow from the fixture generator's `330`/`319` literals, `Build()` returning 1 value not 2,
   and one bad `m.evolutions` conversion. **The blocker was the generator, not the package.** A
   hand-written `Transform` plus a hand-written test closes these.

Per CLAUDE.md ("the bar is: can I produce this myself right now?") every one of these is producible,
so **Task 18b** is inserted before Task 19 and splits the 17 across three serial hand-work batches
sized like Tasks 14/16/17. DOM-04 is NOT closed until 18b lands and Task 18's Steps 1-2 re-run clean.

- **18b-A** (6, atlas-channel): `buddylist/buddy`, `character`, `data/consumable`, `data/equipment`,
  `data/quest`, `minigame`
- **18b-B** (5): `channel/mts/listing`, `channel/mts/wish`, `consumables/cash`,
  `cashshop/character`, `maps/data/map/info`
- **18b-C** (6): `messages/character`, `messages/data/map`, `monsters/monster/consumable`,
  `monsters/monster/mobskill`, `npc-conversations/petdata`, `npc-shops/character`

Serial, not parallel — 18b-A and 18b-B both touch `atlas-channel` and all three commit into the one
worktree. Reference pattern for both files:
`services/atlas-channel/atlas.com/channel/door/rest.go` and its `rest_test.go` (D1 direct-field form,
post-getter-fix).

Task 18 itself stays committed at `775e6aacf` and is otherwise complete (Steps 1-3, 3b).

### Gate over `8203e0d03..775e6aacf` — ERROR (raced a running implementer), NOT a FAIL

The gate reported `1 check(s) FAILED — lint & format guard`, on exactly one line:

```
buddylist/buddy/rest_test.go:26:14: undefined: Transform (typecheck)
lint.sh: LINT FAIL — services/atlas-channel/atlas.com/channel
```

That file is **not in any commit**. It is Task 18b-A's in-flight TDD RED test, written by the
implementer that was running concurrently:

```
$ git status --porcelain services/atlas-channel/atlas.com/channel/buddylist/
 M services/atlas-channel/atlas.com/channel/buddylist/buddy/rest.go
?? services/atlas-channel/atlas.com/channel/buddylist/buddy/rest_test.go
```

Everything gated from the actual commit passed — all 8 `go build/vet` modules, the analyzer, scope,
producer-seam and env-domain guards. `775e6aacf` is not implicated, and this is classified ERROR
(the gate did not validly run) rather than FAIL. Re-run after 18b-A commits.

**Third procedural rule for this branch — `tools/verify.sh` reads the WORKING TREE, not the commit
range.** `--base` selects which modules to check, but the content checked is whatever is on disk.
So the `/execute-task` Step 4c "launch the gate, then immediately dispatch the next task" pattern is
only sound when the gate and the next implementer touch different modules. When they share one — as
here, both `atlas-channel` — the gate reads the implementer's half-written tree and reports a
phantom failure. **Do not run a gate concurrently with an implementer working the same module.**
Sequence it after that implementer's commit instead. This also means a TDD implementer's RED phase
is guaranteed to trip a concurrent gate's typecheck.

## SURFACED FOR RULING — `atlas-channel/character` drops `spawnPoint` on the REST->domain path

Found by the 18b-A fix-round implementer (`04a1c22`) while proving its test discriminates. **Not
fixed, deliberately** — needs a decision, and it is a behavior change this branch's PRD forbids.

Two mutually-corroborating symptoms in `services/atlas-channel/atlas.com/channel/character`:

1. `model.go:240-242` — `Model.SpawnPoint()` is a hardcoded `return 0` stub, not a field read.
2. `rest.go` — `Extract` never assigns `spawnPoint` on the returned `Model` at all.

So `spawnPoint` round-trips to zero regardless of what the wire carried. Note this makes the new
`character` round-trip test **structurally blind to this particular omission**: it compares two
values that have both passed through `Extract`, so a field `Extract` never sets is equal on both
sides whatever the fixture says. The test is still correct for its purpose — it was proven to catch
a `Transform`-side regression via a deliberate break/restore of the `Fame` mapping — but it does not
and cannot cover this.

**Why this is not a drive-by fix.** Both symptoms point the same way: `spawnPoint` looks
*deliberately unimplemented* in this service, not accidentally dropped. Assigning it in `Extract`
would change the REST->domain behavior of a live path, which task-263 is explicitly a
behavior-preserving conformance sweep and must not do. The real question — does `atlas-channel`'s
character model carry `spawnPoint` at all, and who owns it — is a design decision outside this
branch, so it is surfaced rather than absorbed.

**Recommendation: separate task.** Do not fold it into task-263.

### Second instance — this is a pattern, not a one-off

18b-B's review found the **same defect in a different service**:
`services/atlas-cashshop/atlas.com/cashshop/character/rest.go`'s pre-existing `Extract` also never
sets `Model.spawnPoint` from `RestModel.SpawnPoint`. Neither commit touched that code; both
reviewers found it independently while checking round-trip coverage.

Two services' `character` packages dropping the *same* field on the REST->domain path is much more
likely a shared-provenance artifact (these `character` models are near-copies of each other) than
two coincidental typos. That raises the follow-up's scope from "fix one `Extract`" to **"sweep every
`character`-shaped package for fields `Extract` silently drops, and decide whether `spawnPoint` is
owned here at all"** — which is well outside a behavior-preserving conformance sweep.

The follow-up task should start from a repo-wide sweep for `RestModel` fields with no corresponding
assignment in the package's `Extract`, not from these two known sites.

### Third instance — confirmed by 18b-C's review

`services/atlas-npc-shops/atlas.com/npc/character/rest.go`'s pre-existing `Extract` never populates
`Model.x`/`y`/`stance` from `RestModel.X`/`Y`/`Stance`. Same shape, third service, found by a third
independent reviewer.

**Known affected sites (all pre-existing, none touched by this branch):**

| package | field(s) `Extract` drops |
|---|---|
| `atlas-channel/.../channel/character` | `spawnPoint` |
| `atlas-cashshop/.../cashshop/character` | `spawnPoint` |
| `atlas-npc-shops/.../npc/character` | `x`, `y`, `stance` |

All three are `character` packages, which confirms the shared-provenance reading. Note the
consequence for this branch's own tests: in each of these packages the round-trip test is **inert
for the dropped fields** — both sides pass through `Extract`, so an unset field compares equal
whatever the fixture holds. The tests are correct for catching `Transform`-side regressions (proven
by deliberate break/restore in 18b-A fix round 1) but provide no signal on these specific fields.
That is a limitation to state in the PR description, not a defect to patch here.

**Still recommended: a separate task.** Fixing `Extract` changes REST->domain behavior, which
task-263 must not do.


## Task 18b — complete; DOM-04 CLOSED

All three batches landed and reviewed:

| batch | commit | reviewer verdict |
|---|---|---|
| 18b-A (6, atlas-channel) | `56298756d` + fix `04a1c2262` | CHANGES_REQUIRED → fixed |
| 18b-B (5) | `89c396334` | APPROVED |
| 18b-C (6) | `21c1f2367` | APPROVED_WITH_FINDINGS |

### Gate over `8203e0d03..21c1f2367` — PASS

`/tmp/gate-18b.log`, `verify.sh: change base 8203e0d03 — 66 changed path(s)`, 13 changed Go modules.
All 13 `go build/vet` green plus go analyzer guards, skill/job id guard, scope guard, producer seam
guard, env domain guard, and `lint & format guard (13 module(s))`. Closing line: `All checks passed,
but docker bake was skipped — not a pre-PR pass.` This supersedes the earlier ERROR gate over
`8203e0d03..775e6aacf`, which had raced 18b-A's in-flight RED test.

**Last gated commit: `21c1f2367`.** (The two docs-only commits on top — `9303349bf`, `9af1df5f9`,
and this session's review-artifact commit — carry no Go change.)

### Task 18 Steps 1-2 re-run — DOM-04 residue drops 30 → 13, exactly as predicted

`sweep.sh` then `split-by-model.sh` regenerated in place:

```
   13 inventory-dom04-has-model.txt
  176 inventory-dom04-no-model.txt
```

30 − 17 = 13. The remaining 13 are precisely the three accepted classes Task 18 enumerated, with
zero unexplained lines:

- **8 documented** in `handwork-notes.md`: `cashshop/rewardpool`, `consumables/monster`,
  `data/skill`, `maps/character`, `messengers/character`, `npc-conversations/conversation`,
  `parties/character`, `tenants/configuration`.
- **3 false residues** (conforming `Transform` lives outside `rest.go`): `dragons/dragon`,
  `summons/summon`, `storage/asset`.
- **2 accepted NO-RESTMODEL**: `channel/data/tradeability`, `inventory/data/tradeability`.

All 17 forgotten `SKIPPED` packages now carry a hand-written `Transform`. **DOM-04 is closed.**

### Task 19 — W1: `relocate` subcommand — DONE

Added `codemod/relocate.go` + `relocate_test.go`, wired into `main.go`. `Relocate(pkg, srcFile,
builderType)` moves a builder's whole declaration set (struct, constructors, methods, Clone
functions) from srcFile into `builder.go` in the same package, recomputing each output file's
import block from `TypesInfo`-verified actual references (never copying the original import
block), formatting both with `format.Source`, and preserving srcFile's line endings on both
outputs. `TestRelocate` covers all seven table cases plus a "HAS-BUILDER-GO: methods already in
builder.go" case added after the dry-run surfaced it (see finding below). `go build/vet/test`
clean.

Two corrections found only by running the dry-run against the real tree, both now baked into
`relocate.go`:

1. **classify-file05.tsv's `builderType` column is stale for most rows.** It was generated by
   Task 2 before W2 (Tasks 3-9a) renamed most of the repo's builders `ModelBuilder`/`modelBuilder`
   → `Builder`/`builder`. `Relocate` now tries the TSV name first and falls back to its renamed
   counterpart (`resolveBuilderType`, reusing `rename.go`'s own `renamePairs` mapping) before
   reporting "no declarations found" — otherwise nearly every already-renamed package would
   wrongly land in the ledger as a data-integrity failure rather than being relocated.
2. **The shared-receiver guard's first draft was too strict.** It skipped
   `services/atlas-families/.../family` (a HAS-BUILDER-GO package) because every method on
   `Builder` already lives in `builder.go`, not `model.go` — which the first draft's "any method
   in another file" check read as a disallowed split. Fixed to exempt `builder.go` itself (the
   move's own destination) alongside srcFile from the check; only a method in some *third* file
   is a genuine cross-file split. Locked in with a new `TestRelocate` case.

Dry run against the real tree (`-repo .` , `-classification classify-file05.tsv`, `-dry-run`,
`-ledger /dev/stdout`, no file written — verified `git status` clean afterward):

```
64 APPLIED
 9 SKIPPED
```

Skip reasons, all verbatim from the brief's table: `excluded tree` (1, `libs/atlas-packet/model`),
`reference_data.go excluded` (1, cashshop `asset`), `multiple builders in file` (3:
`conversation/model.go`'s 20, `conversation/quest/model.go`'s 2, `pets/data/pet/model.go`'s 2),
`entity builder` (4, the quest/tenants `entity_builder.go` files). 64/9 vs. design §5's
estimated ≈65/≈7 — the small gap is the file-grouping collapsing multi-builder-row files (e.g.
reference_data.go's 7 rows) into one ledger entry each, changing the unit being counted from
"row" to "(pkgDir, file)".

Files: `codemod/relocate.go`, `codemod/relocate_test.go`, `codemod/main.go` (wired). No
`services/`/`libs/` file was modified — Tasks 20-21 apply the move.

## Task 19 — complete (`8c568d6b7`), gate PASS, review APPROVED_WITH_FINDINGS

`relocate` subcommand added to the task-local codemod module. `TestRelocate` covers all 7 brief
table cases plus 1; `GOWORK=off go build/vet/test ./...` clean. Real-tree dry run: **64 APPLIED /
9 SKIPPED**, working tree clean afterward, no `services/`/`libs/` file touched.

**Gate over `21c1f2367..8c568d6b7` — PASS** (`.superpowers/sdd/gates/gate-19.log`, `change base
21c1f2367 — 11 changed path(s)`). Note this gate is *thin*: `verify.sh` reported `lint & format
guard, Go layer (no Go module changed)` — the task-local codemod module is **not** in `verify.sh`'s
module set, so only the analyzer/skill-id/scope guards ran. The codemod's own `go test` is the real
coverage for W1 tooling; do not read a codemod-only gate PASS as evidence the codemod works.

**Last gated commit: `8c568d6b7`.**

### Review: 0 blocking, 2 non-blocking — one is a live thread, carry it into Tasks 21/25

1. *(minor)* `relocate.go:243-257` (`resolveBuilderType`) and the new "no declarations found"
   `skipError` at `:149,160` have no dedicated fixture. Of the two dry-run bugs the implementer
   fixed, only the shared-receiver one (bug 2) has a regression test; the post-W2-rename
   builder-type fallback (bug 1) is exercised only by the real-tree run.

2. **CARRY FORWARD — the 64/9 split does not reconcile with design §5.** 64 + 9 = **73**, but
   design.md:141-147 enumerates **72** distinct FILE-05 packages (§5's ≈65 + ≈7). The report
   explains the gap as a grouping-unit effect `(pkgDir,file)` vs row "collapsing multi-builder files
   into one entry" — the reviewer showed this is **arithmetically backwards**: collapsing yields
   *fewer* groups than 72, not 73. The reviewer's grounded alternative is that
   `conversation/quest/model.go` and `pets/data/pet/model.go` are additional multi-builder files
   absent from design's evidence base.

   This is not blocking for Tasks 20-21 (the codemod is correct either way; it is the *count* that
   is unexplained), but it **must** be reconciled before Task 25 writes `exemptions.md`, because
   design §10 requires every `SKIPPED` row resolve to hand-work or an exemption — and an
   off-by-one in the denominator means one package is either double-counted or unaccounted for.
   Task 21 should reconcile `ledger-relocate-a.tsv` + `ledger-relocate-b.tsv` against design's
   72-package list and record which reading is right.

Reviewer could not independently reproduce the dry-run split — the worktree had concurrent Task 20
activity under `services/atlas-query-aggregator/` at review time. Recorded as not_evaluable, not
as a defect.

## CONTROLLER RULING — Task 21 split into three serial batches

Task 21's plan section lists ~24 modules. Task 20 covered **7** and hit the tool-call cap at 128
calls (DONE_WITH_CONCERNS, one flagged item — the CRLF spot-check — left undone because of it).
Sizing Task 21 as one dispatch would guarantee at least two `PARTIAL`s, which per `/execute-task`
Step 4d is itself the signal that the plan under-decomposed. Splitting up front instead:

- **21-A** (7): `atlas-quest` (3, excl. `entity_builder.go`), `atlas-tenants` (2, excl.
  `entity_builder.go`), `atlas-channel` (2), `libs/atlas-constants` (2), `libs/atlas-database` (1),
  `atlas-storage` (1), `atlas-saga-orchestrator` (1). Creates `ledger-relocate-b.tsv`.
- **21-B** (9, 1 declaration each — wide but shallow): `atlas-notes`, `atlas-messages`,
  `atlas-marriages`, `atlas-inventory`, `atlas-families`, `atlas-drops`, `atlas-doors`,
  `atlas-character-factory`, `atlas-character`.
- **21-C** (2 services + close-out): `atlas-npc-conversations` (6 files, excl.
  `conversation/model.go` → Task 23) and `atlas-cashshop` (9 of 16, excl. `reference_data.go` →
  Task 22). Owns plan Steps 5-6 (ledger completeness, ledger commit).

Serial, not parallel — all three append to the one `ledger-relocate-b.tsv` and commit into the one
worktree. All 18 paths confirmed to exist by `ls` before briefs were written.

**The 72/73 reconciliation from Task 19's review is assigned to 21-C as a required step**, not left
for Task 25. `conversation/quest/model.go` — one of the two files the Task 19 reviewer named as the
likely cause — is in 21-C's own batch, so that batch is the one positioned to settle it from the
ledgers rather than from either prior narrative.

### Scheduling note — gate-20 forced 21-A to wait

`tools/verify.sh --quick --base 8c568d6b7` fanned out to **every** module (`73 changed path(s)`;
`libs/atlas-script-core` changed, which triggers repo-wide fan-out). It was therefore building
`atlas-inventory`, `atlas-families` and others that Tasks 21-A/B touch. Per this branch's third
procedural rule — `verify.sh` reads the **working tree**, not the commit range — 21-A was held until
gate-20 finished rather than dispatched concurrently. This is the second time a `libs/` change has
turned a "cheap" per-task gate into a full fan-out; expect the same after 21-A, which touches
`libs/atlas-constants` and `libs/atlas-database`.

## Task 20 — complete (`8c568d6b7..c9582f0ec`), review APPROVED_WITH_FINDINGS

7 services relocated, one commit each, preceded by a codemod fix commit:

| commit | module |
|---|---|
| `a433d6513` | codemod fix (see below) |
| `2301a7f71` | atlas-query-aggregator |
| `e25c5b5f3` | atlas-login |
| `af27afccb` | atlas-pets |
| `7f58dc448` | atlas-maps |
| `0a573f178` | atlas-consumables |
| `767c65d1a` | atlas-npc-shops |
| `c9582f0ec` | libs/atlas-script-core |

Ledger `ledger-relocate-a.tsv`: **33 APPLIED / 1 SKIPPED** (34 rows, 35 declarations).

### Two more real codemod defects, found only by applying to the real tree

`a433d6513` fixes both, and both matter for Tasks 21-A/B/C since they share the codemod:

1. **Silent comment loss** — moved declaration bodies could drop comments.
2. **Silent multi-builder-package overwrite** — a second builder written into an existing
   `builder.go` could clobber the first.

A regression test was added for both. Note the pattern across Tasks 19 and 20: **four defects in
this codemod have now been found by real-tree application and zero by the unit tests written ahead
of it.** The Step 2 asymmetry check, not the test suite, is what is actually holding the line on
behavior preservation — batches 21-A/B/C must run it per module and not skip it.

### Review — 0 blocking, and it independently re-derived the evidence

The reviewer did not take the report's word for anything: it re-ran the Step 2 asymmetry check
against all 7 commits itself and confirmed every non-import content line pairs 1:1 `-`/`+` with
identical text. It further confirmed `a433d6513` is genuinely first in the range and that neither
defect reached committed output (checked `NewValidationContextBuilderWithLogger` and both builders'
full method sets are present in `query-aggregator/validation/builder.go`; the comments the report
describes live on getters that never moved).

**CRLF question closed, negatively:** the implementer flagged that it never spot-checked CRLF
round-tripping before hitting the cap. The reviewer checked all 65 touched files and found **none
was CRLF at the base commit** — so the risk does not materialize in this range. It is *not*
evidence the codemod preserves CRLF; that remains untested against a live file, and batches
21-A/B/C should check if any of their files are CRLF.

Non-blocking finding: `design.md:400-403`'s skip-condition prose lists 3 conditions, but
`staticSkipReason` (`codemod/relocate.go:90-109`) implements a 4th — "multiple builders in file"
(distinct-builder-type count > 1). Pre-existing and correctly used; the *doc* is what is stale.
Fold into a later docs task.

Not evaluable: `go test ./...` was re-run by the reviewer for only 3 of 7 modules, deliberately, to
avoid duplicating the concurrent gate. The implementer had run all 7 green.

Nothing in this range bears on the 72/73 thread; it stays assigned to 21-C.

### Gate over `8c568d6b7..c9582f0ec` — FAIL (`lint & format guard`), and it is a codemod defect

`.superpowers/sdd/gates/gate-20.log`. One check failed — `lint & format guard (91 module(s))` — with
**12 failing targets**, an `fmt` and a `lint` failure in each of six modules:

```
lint.sh: FAIL — 12 failing target(s):
lint.sh:   fmt:services/atlas-consumables/atlas.com/consumables
lint.sh:   lint:services/atlas-consumables/atlas.com/consumables
lint.sh:   fmt:services/atlas-login/atlas.com/login
lint.sh:   lint:services/atlas-login/atlas.com/login
lint.sh:   fmt:services/atlas-maps/atlas.com/maps
lint.sh:   lint:services/atlas-maps/atlas.com/maps
lint.sh:   fmt:services/atlas-npc-shops/atlas.com/npc
lint.sh:   lint:services/atlas-npc-shops/atlas.com/npc
lint.sh:   fmt:services/atlas-pets/atlas.com/pets
lint.sh:   lint:services/atlas-pets/atlas.com/pets
lint.sh:   fmt:services/atlas-query-aggregator/atlas.com/query-aggregator
lint.sh:   lint:services/atlas-query-aggregator/atlas.com/query-aggregator
```

Representative detail:

```
services/atlas-consumables/atlas.com/consumables/character/model.go:12:1: File is not properly formatted (gofumpt)
	"strconv"
services/atlas-consumables/atlas.com/consumables/compartment/builder.go:4:1: File is not properly formatted (gofumpt)
	"atlas-consumables/asset"
```

**Root cause: `relocate` formats with `format.Source`, which is `gofmt`-clean but not
`gofumpt`-clean** — it does not enforce gofumpt's import grouping/blank-line conventions. Every
affected file is codemod output. `libs/atlas-script-core` escaped only because its import shape
happened to already agree.

This is the **fifth** defect in this codemod found by real-tree application (see the Task 20 note:
still zero found by its unit tests). It is fixed **in the codemod, not by hand-patching the six
services**, because Tasks 21-A/B/C run the same codemod over ~24 more modules and a hand-patch would
reproduce this identical gate failure three more times. Fix brief additionally requires the codemod
be made to agree with `tools/lint.sh`'s gofumpt invocation *by construction* rather than by
imitation, plus a regression test asserting emitted bytes are gofumpt-clean.

Per `/execute-task` Step 4c the fix commit is not gated on its own; it joins the next gate's range.
**Last gated commit remains `8c568d6b7`.**

Note this also retro-explains a Task 16 event: that task's single deferred gate failed on the same
guard (gofumpt in 9 new `rest_test.go` files). Generated Go in this repo has now failed the gofumpt
guard twice; any future generator on this branch should be checked against `tools/lint.sh` before
its output is committed, not after.

### Task 20 fix — PARTIAL (`876d4cb63`), continuation dispatched

The codemod fix landed; the re-apply did not. The implementer hit the tool-call cap at 148 calls and
**deliberately reverted its regenerated service files** rather than commit output it had not
re-verified against its own second fix. So `876d4cb63` contains `relocate.go` only, and the six
affected modules on disk still carry the original gate-failing formatting. Tree confirmed clean of
source changes afterward.

`renderFile`'s pipeline is now: build text → `format.Source` →
`goimports.Process(FormatOnly: true, LocalPrefix: atlasLocalPrefix)` → `gofumpt.Source`.
`FormatOnly: true` is deliberate — `neededImports` already computes the exact import set from
`pkg.TypesInfo`, so goimports must not add or remove imports, only group and sort them.
`atlasLocalPrefix` must stay in sync with `.golangci.yml`'s
`formatters.settings.goimports.local-prefixes`.

**A second, worse defect was found in the same pass — and it would have broken Task 21-A.**
`neededImports` omitted the alias for packages whose declared package name differs from their import
path's last component (the `libs/atlas-constants/map`, `atlas-tenant` shape). **`libs/atlas-constants`
is in 21-A's batch**, so had this not surfaced here, 21-A would have emitted code that does not
compile. That makes six defects found in this codemod by real-tree application, still zero by its
unit tests — the pattern is now strong enough to treat as a property of this tool: **its unit tests
do not constrain its output; only application does.**

Continuation brief: `.superpowers/sdd/plan/task-20-fix1-brief-cont.md`. It carries the two missing
regression tests (gofumpt-clean output; alias correctness), the verified
`git checkout 8c568d6b7 -- ...` restore block, the ledger-unchanged assertion, and the
behavior-preservation diff against the pre-Task-20 base.

### Task 20 fix — continuation DONE (`852c472b4`), gate FAIL cleared

Two regression tests added (gofumpt-clean output; alias correctness), then the codemod re-applied to
all six modules. Verification, all green:

- codemod module `GOWORK=off go test ./...` — pass, including the 2 new tests
- all six service modules — `go build/vet/test` clean
- **`tools/lint.sh --check --go` over all six: `0 issues.` × 6, `lint.sh: OK`** — this is the exact
  guard that failed gate-20, now clean
- behavior preservation confirmed per module against pre-Task-20 base `8c568d6b7`: formatting-only

`ledger-relocate-a.tsv` unchanged as predicted (33 APPLIED / 1 SKIPPED stands) — not restaged.

One refinement worth carrying into 21-A/B/C: the brief's exclusion regex for the
behavior-preservation diff does not skip **aliased** import lines, which surfaced in `atlas-maps`
once the alias fix started emitting them. Batches running that check will need the same refinement
or they will see phantom asymmetry on any file with an aliased import. Documented in
`.superpowers/sdd/plan/task-20-fix1-report.md`.

## STATE AT HANDOFF

**Last gated commit: `8c568d6b7`.** Per `/execute-task` Step 4c, `876d4cb63` and `852c472b4` are fix
commits and are NOT gated separately — they join the next gate's range. **The next session's first
action is `tools/verify.sh --quick --base 8c568d6b7`**, which will cover Task 20's seven service
commits plus both fixes. That gate previously FAILED on `lint & format guard` and the fix is
believed complete, but *believed* is not *gated* — do not treat `lint.sh --check` passing as the
gate having passed.

Expect a full fan-out again (`libs/atlas-script-core` is in the range), so per this branch's third
procedural rule, do not dispatch 21-A concurrently with it — the gate reads the working tree.

### Completed: Tasks 1-20 (18b included). Remaining: 21-A/B/C, 22-27.

Briefs already written and ready to dispatch, fact blocks not yet prepended to B and C:
- `.superpowers/sdd/plan/task-21a-brief.md` (7 modules, incl. `libs/atlas-constants` — the alias fix
  matters here)
- `.superpowers/sdd/plan/task-21b-brief.md` (9 modules, 1 declaration each)
- `.superpowers/sdd/plan/task-21c-brief.md` (2 services + ledger close-out + the **required** 72/73
  reconciliation)

Then: 22 (`reference_data.go` hand split, FR-11), 23 (`conversation/model.go`, 20 builders),
24 (FILE-05), 25 (`exemptions.md` — needs the NO-MODEL class from Task 17 Group 4 alongside D2's
NO-RESTMODEL, and consumes both relocate ledgers), 26 (behavior preservation), 27 (final gate +
review).

### Open threads the next session must not lose

1. **72/73 reconciliation** — assigned to 21-C as a required step. Design enumerates 72 FILE-05
   packages; the dry run produced 73 groups. Must be settled before Task 25 writes `exemptions.md`.
2. **`design.md:400-403` prose is stale** — lists 3 skip conditions, `staticSkipReason`
   (`codemod/relocate.go:90-109`) implements 4. Fold into a docs task.
3. **CRLF preservation is still untested against a live file.** Task 20's range happened to contain
   no CRLF file. Batches 21-A/B/C should check whether any of theirs are.
4. **`spawnPoint` / `Extract`-drops-fields** — three `character` packages, recommended as a separate
   task, explicitly NOT folded into task-263.
5. **The codemod's unit tests do not constrain its output.** Six defects, all found by real-tree
   application, none by tests. The per-module asymmetry check and `tools/lint.sh --check --go` are
   the real gates for W1 batches.

## Session resume (2026-08-27)

### Gate 20 re-run: **PASS**

`tools/verify.sh --quick --base 8c568d6b7` → exit 0, covering Task 20's seven service commits plus
both fix commits (`876d4cb63`, `852c472b4`). The previously-failing **`lint & format guard`
(91 modules)** is green, along with `go build/vet` × 91, the analyzer guards, and the scope /
producer-seam / service-registration / env-domain guards. Docker bake skipped (`--quick`), so this
is an iteration gate, not a pre-PR pass.

**Last gated commit is now `61032d0fa`.** Task 21-A dispatches on top of it.

### Task 21-A: complete (`61032d0fa..59afb53af`, 8 commits)

Six modules relocated — `atlas-quest`, `atlas-channel`, `libs/atlas-constants`, `libs/atlas-database`,
`atlas-storage`, `atlas-saga-orchestrator`. `atlas-tenants` produced no source change (both classify
rows are entity-builders, correctly skipped). Per module: `go build/vet/test` PASS and
`tools/lint.sh --check --go` → `0 issues.`

One non-mechanical commit in the range: `af4f138fe` adds a `-append` flag to `relocate` so batches
can share one ledger across contexts.

**Review verdict: APPROVED_WITH_FINDINGS** (`review-task-21a.md`). All six module diffs confirmed
pure relocations; the `libs/atlas-constants/field` import-alias fix emits `_map ".../atlas-constants/map"`
correctly and is not duplicated. A bare-`gofumpt` false positive on
`atlas-storage/.../projection/builder.go` was chased down and disproved via `golangci-lint fmt --diff`
— recorded in the artifact so it is not rechased.

**Blocking finding — resolved by the controller.** The report's narrative claimed
"9 APPLIED, 4 SKIPPED, 13 rows" for `ledger-relocate-b.tsv`, contradicting both the committed file
and the report's own table. Verified directly: `cut -f2 … | sort | uniq -c` → **8 APPLIED, 4 SKIPPED**,
`wc -l` → **12**. The committed ledger was always correct; only the prose was wrong. Report corrected
in place. **Task 21-C's 72/73 reconciliation and Task 25 must use 8/4/12.**

**Non-blocking, carry to 21-B/C:** the `-append` regression tests cover `mergeLedgerLines` /
`splitLedgerLines` in memory but never `runRelocate`'s on-disk read branch (missing file,
trailing-newline vs. not). Correct by hand-trace and by the 7-invocation real apply — but this
codemod's six defects were all found on the real tree and none by its unit tests, so the gap is real.

Also from 21-A: the Step-2 asymmetry-check regex has a blank `-`/`+` line filter imbalance, fixed in
the implementer's local procedure and documented in `task-21a-report.md` for B/C to reuse. No
committed file encodes the buggy regex.

**Gate 21-A: PASS.** `tools/verify.sh --quick --base 61032d0fa` → exit 0 over all eight commits;
`go build/vet` × 91, analyzer guards, and `lint & format guard` (91 modules) all green. Docker bake
skipped (`--quick`). **Last gated commit is now `59afb53af`.**

### Task 21-B: PARTIAL (`59afb53af..b2a35ab8c`, 10 commits, cap reached)

Seven of nine modules applied — `atlas-notes`, `atlas-messages`, `atlas-marriages`, `atlas-inventory`,
`atlas-drops`, `atlas-doors`, `atlas-character-factory`. All seven: `go build/vet/test` PASS and
`tools/lint.sh --check --go` → `0 issues.`

**Seventh codemod defect — `services/atlas-families`, reverted not patched.** `Relocate`'s
`ast.NewCommentMap` attachment silently drops **free-floating section-header comments** (a comment
separated from the next decl by a blank line, so not its doc comment). Two such comments vanished from
both output files; diff was asymmetric 14 `+` / 17 `-`. Module fully reverted, ledger row `SKIPPED`.
Consistent with the standing pattern: every defect in this codemod has been found by real-tree
application, none by its unit tests.

**Controller ruling: fix `relocate.go`, do not hand-carve the module.** Comment loss is silent source
loss, and batch C plus later tasks still have to run this codemod; a per-module carve-out leaves the
defect live for all of them. Assigned to the continuation, regression-test-first, ahead of
`atlas-character` so that module is generated by the fixed codemod.

**New pattern worth carrying: `--new-from-rev` unmasking.** Moving code into a brand-new `builder.go`
makes pre-existing dead fields / unchecked errors newly visible to `tools/lint.sh`. Hit twice
(`atlas-messages` dead `stance` field, `atlas-marriages` unchecked `Put`); each fixed in a separate
labeled commit *after* its relocation commit, never folded in. Expect it again in batch C.

**Ledger, verified by the controller against the file** (`cut -f2 | sort | uniq -c`, `wc -l`):
**15 APPLIED / 5 SKIPPED / 20 rows.** Batch A 8/4/12 + batch B's 7 APPLIED + 1 SKIPPED = 20. ✓
The report's prose said 14/5/19 and was corrected in place. **This is the second consecutive batch to
miscount its own ledger by one in prose** — batch A's was the review's blocking finding. Standing rule
now in the continuation brief: the ledger *file* is authoritative, never a report narrative, and counts
must be pasted from command output.

Continuation brief: `.superpowers/sdd/plan/task-21b-brief-cont.md` (codemod comment fix +
`atlas-families` re-apply, then `atlas-character`). Per Step 4d, task review runs once over the whole
21-B range after the continuation lands, not per segment.

**Gate 21-B: PASS.** `tools/verify.sh --quick --base 59afb53af` → exit 0 over the ten partial-batch
commits; `go build/vet` × 91, analyzer guards, and `lint & format guard` (91 modules) green. Docker
bake skipped (`--quick`). **Last gated commit is now `b2a35ab8c`.** The continuation's commits join
the next gate's range.

### Task 21-B: complete (`59afb53af..b7f3e8d90`, 13 commits)

Ran across three implementer segments — the first hit the tool-call cap at 7/9 modules; the second
completed the codemod fix and `atlas-families`, then **ended its turn waiting on a backgrounded
`go test` it never read**. `SendMessage` is disabled this session, so the controller inspected the tree
and dispatched a fresh cont2 implementer — whereupon the stalled agent woke on its own and finished.
cont2 was stopped (`TaskStop`) before it made any edit; tree verified clean afterward. **Procedural note
for the remaining batches: a stalled agent may still resume, so before dispatching a replacement,
prefer stopping the original first — the two overlapped on the same files here and only luck kept it
from being a conflict.**

Final commits: 9 relocations (`atlas-notes`, `atlas-messages`, `atlas-marriages`, `atlas-inventory`,
`atlas-drops`, `atlas-doors`, `atlas-character-factory`, `atlas-families` re-apply, `atlas-character`),
2 lint-gate fixes (`f1faf17aa`, `da3ece073`), the codemod fix `ac3fa65a3`, and the ledger commit.

**Review verdict: APPROVED_WITH_FINDINGS, 0 blocking** (`review-task-21b.md`). The reviewer verified
RED→GREEN on the comment-fix regression test by checking out pre/post-fix `relocate.go`, and probed four
further comment shapes (EOF-trailing, two consecutive floating, floating-before-a-moving-decl,
interleaved line/block) against the fixed codemod with a throwaway test — **all handled correctly**.
Both lint-gate fixes confirmed genuinely pre-existing against `git show 59afb53af:<path>`.

**Non-blocking, worth acting on before Task 25:** `relocate_test.go:557-651`
(`TestRelocatePreservesFloatingSectionComments`) only covers "floating comment before a decl that stays",
twice. The fix handles the other four shapes — but by probe, not by committed test. Given this codemod's
record (seven defects, all found on the real tree, none by its tests), the committed test under-specifies
the contract it is supposed to protect. Candidate for a small test-hardening commit.

**Ledger final, verified against the file** (`cut -f2 | sort | uniq -c` → 17 APPLIED / 4 SKIPPED;
`wc -l` → 21): **17 APPLIED / 4 SKIPPED / 21 rows.** `atlas-families` correctly flipped SKIPPED→APPLIED.
This is batch B's closed-out contribution; batch C appends to it.

**Gate 21-B continuation: PASS.** `tools/verify.sh --quick --base b2a35ab8c` → exit 0 over
`ac3fa65a3`, `d21d54298`, `b7f3e8d90`; `go build/vet` × 91, analyzer guards, and `lint & format guard`
(91 modules) green. Docker bake skipped (`--quick`). **Last gated commit is now `b7f3e8d90`.**

## STATE AT HANDOFF (session 2)

**Last gated commit: `b7f3e8d90`.** Tasks 1-21-B are complete, reviewed, and gated. Working tree holds
only controller-owned edits (`progress.md`, `agent-ledger.tsv`), the long-standing dirty `go.work.sum`,
an untracked local `.gatelogs/` (gate logs, disposable — not in `.gitignore` because this worktree's
`.git` is a file, so `.git/info/exclude` was not writable), and the two untracked review artifacts
`review-task-21a.md` / `review-task-21b.md`.

### Completed: Tasks 1-20, 21-A, 21-B. Remaining: 21-C, 22-27.

**Next action: Task 21-C.** Brief at `.superpowers/sdd/plan/task-21c-brief.md` — it has a `### Files`
section but **no fact block yet**; prepend one with
`tools/task-facts.sh task-263 --base b7f3e8d90` before dispatching. Scope: 2 services + relocate
ledger close-out + the **required** 72/73 reconciliation.

Then: 22 (`reference_data.go` hand split, FR-11), 23 (`conversation/model.go`, 20 builders),
24 (FILE-05), 25 (`exemptions.md`), 26 (behavior preservation), 27 (final gate + review).

### What 21-C must carry forward

- **Relocate ledger is at 17 APPLIED / 4 SKIPPED / 21 rows** (`ledger-relocate-b.tsv`, verified against
  the file). Batch C appends via the codemod's `-append`; never hand-edit the TSV.
- **State ledger counts from pasted command output, never prose arithmetic.** Batches A and B *both*
  miscounted their own ledger by one; A's was a blocking review finding.
- **The `--new-from-rev` unmasking pattern**: moving code into a new `builder.go` exposes pre-existing
  dead fields / unchecked errors to `tools/lint.sh`. Hit twice in batch B. Verify pre-existence with
  `git show <base>:<path>`, then fix in a separate commit *after* the relocation commit.
- **Behavior-preservation diff filters** must skip aliased import lines and balance blank `-`/`+` lines,
  or they report phantom asymmetry. Corrected filter set is in `.superpowers/sdd/plan/task-21b-report.md`.
- **`tools/lint.sh` / `golangci-lint fmt --diff` is the sole formatting authority** — a bare `gofumpt`
  finding is not actionable; batch A chased one such false positive to ground.
- **CRLF preservation remains unexercised.** No file in Task 20's, batch A's, or batch B's range had CRLF.
  Batch C should check and record the result either way.

### Open threads

1. **72/73 reconciliation** — required step of 21-C. Design enumerates 72 FILE-05 packages; the dry run
   produced 73 groups. Must be settled before Task 25 writes `exemptions.md`.
2. **`relocate_test.go:557-651` under-specifies the comment contract** — covers 1 of 5 shapes the fix
   handles (21-B review, non-blocking). Candidate test-hardening commit before Task 25.
3. **`design.md:400-403` prose is stale** — lists 3 skip conditions; `codemod/relocate.go:90-109`
   implements 4. Fold into a docs task.
4. **`spawnPoint` / `Extract`-drops-fields** — three `character` packages, explicitly NOT in task-263.
5. **The codemod's unit tests do not constrain its output.** Seven defects now, every one found by
   real-tree application, none by its tests. The per-module asymmetry check plus
   `tools/lint.sh --check --go` are the real gates.
6. **`SendMessage` is disabled this session** — a stalled subagent cannot be resumed, but it may still
   wake and finish on its own. Stop the original before dispatching a replacement.

## Task 21-C — batch C + ledger close-out — DONE

Applied `relocate` to the two remaining services, per brief (excluding the two files reserved for
Tasks 22/23):

- `services/atlas-npc-conversations` — 5 files applied (`conversation/item`, `conversation/npc`,
  `conversation/recipe`, `saved_location`, `validation`), `conversation/quest/model.go` SKIPPED
  (codemod's own `multiple builders in file` check — it has 2 builder rows, `Builder` +
  `StateMachineBuilder`), `conversation/model.go` never touched (Task 23's 20-builder file).
  Commit `180176796` "refactor(atlas-npc-conversations): move builders into builder.go".
- `services/atlas-cashshop` — all 9 non-`reference_data.go` rows applied
  (`cashshop/inventory`, `cashshop/inventory/asset`, `cashshop/inventory/compartment`, `character`,
  `character/compartment`, `character/inventory`, `coupon`, `coupon/batch`, `coupon/redemption`);
  `reference_data.go` never touched (Task 22's hand split). Commit `c3335b9b6`
  "refactor(atlas-cashshop): move builders into builder.go".

`git status` confirmed after every codemod invocation that neither `conversation/model.go` nor
`reference_data.go` was modified.

### Step 2 — asymmetry check, corrected filter

Reused batch A/B's filter but it needed one more fix: `grep -E` in this shell does **not** expand
`\t` to a tab (confirmed: `printf '+\t"errors"\n' | grep -E '^\+\t"'` → exit 1, no match), so the
batch A/B filter silently passed every import line through unfiltered here. Switching to `grep -P`
fixes that (`grep -P '^\+\t"'` matches). Separately, `saved_location/model.go`'s single aliased
import (`_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"`) is a form the batch A/B
filter's `^\+\t"` pattern doesn't match (`\t_map "..."` — identifier before the quote, not a bare
quote) — this is exactly the "aliased import" case flagged as carried-forward note 4. Added a second
pattern for it. **Corrected filter for Batch C (both `-P` and the alias addition are load-bearing):**

```
grep -vP '^\+package |^[+-]$|^\+import|^\+\t"|^\+\)|^-import|^-\t"|^-\)|^\+\t[A-Za-z_][A-Za-z0-9_]* "|^-\t[A-Za-z_][A-Za-z0-9_]* "'
```

Every one of the 14 touched files (5 npc-conversations + 9 cashshop) came back symmetric (each
moved block appears exactly once as `-` and once as `+`, byte-identical) after applying this filter.
Verified per-file; the `saved_location` case above was the only one that needed the alias addition.

### Step 3 — build/vet/test

- `services/atlas-npc-conversations/atlas.com/npc`: `go build ./... && go vet ./... && go test ./...`
  → `Go test: 295 passed in 35 packages`.
- `services/atlas-cashshop/atlas.com/cashshop`: `go build ./... && go vet ./... && go test ./...`
  → `Go test: 279 passed in 56 packages`.

No `--new-from-rev`-style pre-existing dead-code/unchecked-error exposure was found in either
module (neither module needed a separate lint-gate fix commit).

### CRLF check

`od -c <file> | grep -c '\\r'` → `0` for all 14 touched files (both services). No CRLF anywhere in
this batch, same as Task 20 and batches A/B.

### Step 5 — ledger completeness across both halves

```
$ cat ledger-relocate-a.tsv ledger-relocate-b.tsv | cut -f2 | sort | uniq -c
     64 APPLIED
      6 SKIPPED
$ wc -l ledger-relocate-a.tsv ledger-relocate-b.tsv
 34 ledger-relocate-a.tsv
 36 ledger-relocate-b.tsv
 70 total
```

**64 APPLIED / 6 SKIPPED / 70 rows total**, final and closed.

### Step 6 — the 72/73 reconciliation (required)

Derived from `classify-file05.tsv` and the two ledgers directly, not from either prior narrative.

```
$ awk -F'\t' '{print $4}' classify-file05.tsv | sort | uniq -c
      4 ENTITY-BUILDER
      1 EXCLUDED-TREE
      2 HAS-BUILDER-GO
     93 RELOCATE
$ awk -F'\t' '($4=="RELOCATE"||$4=="HAS-BUILDER-GO"){print $1"\t"$2}' classify-file05.tsv | sort -u | wc -l
68
```

100 raw declaration rows (93 `RELOCATE` + 2 `HAS-BUILDER-GO` + 4 `ENTITY-BUILDER` + 1
`EXCLUDED-TREE`) collapse to **68 distinct `(pkgDir,file)` groups** for `RELOCATE`/`HAS-BUILDER-GO`
(the collapse comes from four files carrying more than one builder row: `reference_data.go` 7,
`conversation/model.go` 20, `conversation/quest/model.go` 2, `pets/data/pet/model.go` 2). Add the 4
`ENTITY-BUILDER` groups (each is its own file, 1 row = 1 group): **68 + 4 = 72 — exactly
design.md:141-147's stated FILE-05 package count.** The 1 `EXCLUDED-TREE` group
(`libs/atlas-packet/model/skill_usage_info.go`, confirmed: `grep -P '\tEXCLUDED-TREE$'
classify-file05.tsv` → exactly this one row) is *not* part of design's 72 — design.md §8.1
(`FILE-05 excluded tree (1 — libs/atlas-packet/model/skill_usage_info.go:143, PRD §7/FR-18)`)
and D5/D6 both treat "out of FILE-05's transform scope entirely" as a different disposition from
"one of the 72 packages FILE-05 covers." Add it back: **72 + 1 = 73 — exactly Task 19's real-tree
dry-run total** (`progress.md:2753-2754`, 64 APPLIED + 9 SKIPPED).

**Both prior narratives are wrong, as the brief warned:**
- Task 19's report ("grouping-unit effect... collapsing multi-builder-row files into one ledger
  entry") is arithmetically backwards, exactly as the Task 19 reviewer showed: collapsing 100 raw
  rows into `(pkgDir,file)` groups produces *68*, which is *below* design's 72, not above it. It
  cannot explain a total that is 1 *more* than 72.
- The reviewer's alternative — that `conversation/quest/model.go` and `pets/data/pet/model.go` are
  "additional multi-builder files absent from design's evidence base" — is also not the source of
  the 72/73 gap. They are not "extra": they are two of the four multi-builder files already folded
  into the 68-group total that sums exactly to design's 72 (with `ENTITY-BUILDER`). Removing them
  from the count would make it 70, not 73; adding them back is required to *reach* 72, not to
  explain an *excess* over it.
- **The actual source of the +1 is the `EXCLUDED-TREE` row.** It is a third, distinct disposition
  category — "not part of FILE-05's transform surface at all" (PRD §7/FR-18), as opposed to
  "part of FILE-05 but requires hand-work" (`ENTITY-BUILDER`, D5, or the two reserved
  `HAS-BUILDER-GO`/`RELOCATE` files). Design's own 72-count correctly omits it; Task 19's dry run,
  which iterated every classify row including `EXCLUDED-TREE` ones and tallied every non-`APPLIED`
  outcome into "`SKIPPED`", did not. That is the entire gap. No package is double-counted and none
  is unaccounted for.

This also explains design §5's `~65 applied / ~7 hand` estimate (`design.md:456`) cleanly:
`~7 hand` = 4 `entity_builder.go` (D5) + 1 `excluded tree` (D6/FR-18) + `reference_data.go`
(Task 22) + `conversation/model.go` (Task 23) = 7, and `~65 applied` ≈ 68 minus those same two
Task-22/23 reservations minus the two genuinely-multi-builder files the codemod's own
`multiple builders in file` check discovered at implementation time (`conversation/quest/model.go`,
`pets/data/pet/model.go`) = 64, which is this batch's final, exact `APPLIED` count.

**Every `SKIPPED` row across both ledgers, checked for attribution:**

```
$ cat ledger-relocate-a.tsv ledger-relocate-b.tsv | awk -F'\t' '$2=="SKIPPED"'
services/atlas-pets/atlas.com/pets/data/pet/model.go	SKIPPED	multiple builders in file
services/atlas-npc-conversations/atlas.com/npc/conversation/quest/model.go	SKIPPED	multiple builders in file
services/atlas-quest/atlas.com/quest/quest/entity_builder.go	SKIPPED	entity builder
services/atlas-quest/atlas.com/quest/quest/progress/entity_builder.go	SKIPPED	entity builder
services/atlas-tenants/atlas.com/tenants/configuration/entity_builder.go	SKIPPED	entity builder
services/atlas-tenants/atlas.com/tenants/tenant/entity_builder.go	SKIPPED	entity builder
```

- The 4 `entity builder` rows → attributable to D5 / `exemptions.md`'s planned "FILE-05 entity
  builders (4)" section (`design.md:576-578`).
- `services/atlas-pets/atlas.com/pets/data/pet/model.go` (`multiple builders in file`) →
  **NOT attributable to any known destination.** It is not Task 22 (`reference_data.go`), not
  Task 23 (`conversation/model.go`), not a D5 `ENTITY-BUILDER`, not the `EXCLUDED-TREE` row, and
  design.md's §8.1 exemptions list (`design.md:576-579`) has no entry for it. **Finding: this
  package needs either a hand-split commit (same shape as Tasks 22/23, since it genuinely has 2
  builders in one `model.go`) or a new `exemptions.md` entry with its own justification — Task 25
  cannot silently drop it.**
- `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/model.go`
  (`multiple builders in file`, `Builder` + `StateMachineBuilder`) → **same finding, NOT
  attributable to any known destination.** Not covered by Task 23 (which is scoped to
  `conversation/model.go` only), not D5, not `EXCLUDED-TREE`, not in `design.md`'s §8.1 list.
  **Same recommendation: hand-split or a new exemption, decided before Task 25.**

Neither of these two findings blocks 21-C itself (the codemod's `SKIPPED` disposition for both is
correct — routing a 2-builder file to hand review rather than silently merging both builders'
fields is the safe behavior) — but both must be resolved (hand-work commit or `exemptions.md`
entry) before Task 25 writes `exemptions.md`, per design §10's rule that every `SKIPPED` package
resolves to one or the other.

### Step 7 — ledger commit

`docs/tasks/task-263-backend-guideline-conformance/ledger-relocate-a.tsv` (read-only, unchanged),
`ledger-relocate-b.tsv`, and this `progress.md` entry committed together.

## Task 21-C review — APPROVED_WITH_FINDINGS (0 blocking, 2 non-blocking)

Artifact: `review-task-21c.md`. The reviewer independently re-derived all 14 relocation purity
checks (not trusting the report's filter), recomputed the ledger counts (64/6/70), confirmed the
append was additive-only and `ledger-relocate-a.tsv` untouched, confirmed both reserved files absent
from the diff, and re-ran `go build`/`vet`/`test` clean for both modules.

**Non-blocking 1 — the 70-row total needs its chain written down.** The brief asked for
`APPLIED + SKIPPED` = the `RELOCATE`+`HAS-BUILDER-GO` row count; the report answered the 72/73
question instead. The reviewer verified the chain independently and calls it airtight:
**70 = 66 (68 groups − 2 Task-22/23 reserved, never ledgered) + 4 (`ENTITY-BUILDER` rows also
recorded `SKIPPED`)**, against 95 raw rows / 68 groups. Task 25 should carry that sentence forward
so the next reader does not re-derive it.

**Non-blocking 2 — CONTROLLER RULING: hand-split, do not exempt.** The two unattributed `SKIPPED`
rows —

- `services/atlas-pets/atlas.com/pets/data/pet/model.go`
- `services/atlas-npc-conversations/atlas.com/npc/conversation/quest/model.go`

— are **multi-builder files**, which is exactly the class the plan already routes to hand-work:
Task 22 (`reference_data.go`, 7 builders) and Task 23 (`conversation/model.go`, 20 builders) are
both hand-splits of precisely this shape, and both were carved out of the codemod for precisely
this reason. These two are the same class; they were simply missed when the plan enumerated its
hand-work. Exempting them would exempt a package on the grounds that the *codemod* could not do it,
which is not what design §8.1's exemption list is for.

**Therefore: a new Task 23-B hand-splits both files**, following Task 22's and Task 23's procedure
verbatim (verbatim moves, recomputed imports, no renames, no `Transform` additions, module-local
build/vet/test, one commit per service). Neither file goes into `exemptions.md`. Task 25 must be
briefed that these two rows resolve to Task 23-B, not to an exemption — with Task 23-B landed,
every `SKIPPED` row across both ledgers is then attributable and design §10 is satisfied.

**Sequencing correction.** An earlier draft of this ruling said 23-B was a prerequisite of Task 25
only and could run in parallel with Task 24. That is wrong. Task 24 Step 2 requires the regenerated
`inventory-file05-builders.txt` residue to be *exactly* five documented entries (one `EXCLUDED-TREE`
plus four `ENTITY-BUILDER`). Both of these files still hold builders in a non-`builder.go` file, so
the sweep will list them as residue and Task 24 will fail its own acceptance check until they move.

**Task 23-B is therefore a prerequisite of Task 24 as well as Task 25, and must land before either.**

## Gate 21-C — ERROR (raced with an in-flight implementer), NOT a FAIL

`tools/verify.sh --quick --base b7f3e8d90` was launched at `HEAD=9a3a85e29` and reported
`4 check(s) FAILED` (scope guard, producer seam guard, and two others). **This verdict is void.**
Every failure traces to one cause, visible in `.gatelogs/gate-21c.log:212-272`:

```
asset/reference_data.go:73:6:  EquipableReferenceDataBuilder redeclared in this block
asset/reference_data.go:100:6: NewEquipableReferenceDataBuilder redeclared in this block
... (through NewEtcReferenceDataBuilder) ...
asset/reference_data.go:791:6: too many errors
```

Those are **pre-Task-22 line numbers** in a file Task 22 truncated to 245 lines, reported as
`redeclared` against the `asset/builder.go` Task 22 was creating at that moment. The gate observed
the tree after `builder.go` was written but before `reference_data.go` was truncated — a mid-edit
snapshot that never existed as a commit. Task 22's own module-local `go build`/`vet`/`test` passed
(279/279), and its reviewer independently re-ran all three clean on the committed state.

**Structural finding — this changes how the gate may be scheduled.** `tools/verify.sh` reads the
**working tree**, not the commit range implied by `--base`. The `/execute-task` guidance to "launch
the gate and immediately dispatch the next implementer" is therefore only safe when the next
implementer touches files the gate is not compiling — which, on a repo-wide fan-out
(`modules_selected=91`), is never. On this plan the gate and an implementer **must not overlap**.

**Consequence:** the 21-C gate must be re-run on a quiescent tree, and the re-run's range widens to
cover Tasks 22, 23, and 23-B as well — still `--base b7f3e8d90`, since no gate has validly passed
since. **The last validly gated commit remains `b7f3e8d90`.** Do not re-run it while the Task 23-B
implementer is in flight; wait for it to report, then run the gate with nothing else dispatched.

## Task 23 review — APPROVED_WITH_FINDINGS (0 blocking, 1 non-blocking)

Artifact: `review-task-23.md`. Commit `26e1ec1e4`. The reviewer re-derived everything independently
rather than re-reading the report, and the evidence is unusually strong for a hand-split of this
size:

- Pre-image (`git show b7f3e8d90:.../conversation/model.go`) has exactly **20** `type <X>Builder
  struct`; `builder.go` has the same 20 **in the same order**.
- Ordered full-diff line comparison (not a multiset/Counter check — this also rules out
  same-content reordering, which a multiset cannot): `minus == plus` exactly, **1362 lines each**.
- Function/method/constructor/`Clone` count: **149 in the pre-image, 149 in `builder.go`, 0 left in
  `model.go`**.
- No `Transform` added anywhere; all 20 constructors keep `New<Type>Builder`.
- Imports correctly recomputed: `strconv` retained in `model.go` (1 use), correctly dropped from
  `builder.go` (0 uses); `errors`/`uuid`/`field` used in both. `model_json.go` references only
  `CraftActionModel`, a domain type — no ripple.
- Commit touches only the two named files.

**Non-blocking — Task 24 must not reuse this check verbatim.** The brief's Step 4 said
`grep -c 'Builder' model.go` should return `0`; it actually returns **1**, at
`services/atlas-npc-conversations/atlas.com/npc/conversation/model.go:444` — a pre-existing
*doc-comment prose* mention of `CraftActionBuilder` inside `NewCraftActionModelDirect`'s comment,
not a declaration. A declaration-specific grep confirms 0 builder declarations remain, so the code
is correct and the *check* is what was wrong: a bare substring grep counts prose.

Task 24 regenerates `inventory-file05-builders.txt` via `sweep.sh` and asserts the residue is
exactly five entries. **If `sweep.sh` matches on the substring `Builder` rather than on a
declaration pattern, this comment will surface as a sixth residue entry and must be recognised as a
false positive, not a forgotten package.** Check `sweep.sh`'s pattern before interpreting its output.

`tools/lint.sh` / `golangci-lint` was unavailable in the reviewer's environment as well
(`which golangci-lint` → not found), so the `--new-from-rev` unmasking hazard on this new
20-builder file — the largest such surface in the plan — remains genuinely unchecked and falls to
the repo-wide gate.

## STATE AT HANDOFF (session 3)

**Last VALIDLY gated commit: `b7f3e8d90`** — unchanged from session 2. See "Gate 21-C — ERROR"
above: no gate has validly passed since, because the one run raced an implementer.

### Completed this session: Tasks 21-C, 22, 23, 23-B. Remaining: 24, 25, 26, 27.

| Task | Commits | Review |
|---|---|---|
| 21-C | `180176796`, `c3335b9b6`, `9a3a85e29` | APPROVED_WITH_FINDINGS (0 blocking) |
| 22 | `006158302` | APPROVED (0 findings) |
| 23 | `26e1ec1e4` | APPROVED_WITH_FINDINGS (0 blocking) |
| 23-B | `27c5629fc`, `46cc37833`, `b88bd1ed8` | APPROVED (0 findings) |

**Task 23-B review landed after the handoff was written: APPROVED, 0 blocking, 0 non-blocking**
(`review-task-23b.md`). Verified by independent ordered full-diff against the `b7f3e8d90` pre-image.
Pet: `Builder` + `SkillModelBuilder`. Quest: `Builder` + `StateMachineBuilder` — 2 each, confirming
`classify-file05.tsv`'s recorded count rather than assuming it. Both `model.go` files now show
`grep -n 'Builder'` → **no matches at all** (a true 0, not a masked prose hit like
`conversation/model.go:444`). Each commit touches only its intended files, and `b88bd1ed8` records
both SHAs, the correct type names, and the explicit "neither becomes an `exemptions.md` entry".

**Only item 1 (the gate, `.gatelogs/gate-23b.log`) remains outstanding at handoff.**

**Task 23-B is a controller-created task, not in `plan.md`.** Its brief is at
`.superpowers/sdd/plan/task-23b-brief.md`; the ruling that created it is under "Task 21-C review"
above.

### Two things were in flight when this session handed off

1. **The re-run gate** — `tools/verify.sh --quick --base b7f3e8d90 > .gatelogs/gate-23b.log`,
   launched on a quiescent tree, covering Tasks 22, 23 and 23-B as well as 21-C. **Read
   `.gatelogs/gate-23b.log` first thing.** On PASS, the last gated commit becomes `b88bd1ed8`.
   On FAIL, quote the failing block and route it as a fix commit — do not re-gate the fix
   separately.
2. **The Task 23-B review** — artifact will be at `review-task-23b.md`. Read it only if the verdict
   is not APPROVED; reconcile and ledger it either way.

### Next action: Task 24

Brief already generated at `.superpowers/sdd/plan/task-24-brief.md` (has a `### Files` section, no
fact block — prepend one with `tools/task-facts.sh task-263 --base <last gated commit>`).
Briefs for 25, 26 and 27 are also pre-generated and all carry `### Files`.

**Task 24 cannot start until 23-B's gate and review reconcile**, since it asserts on the residue
23-B just removed.

### What the next controller must carry forward

- **Never run the gate concurrently with an implementer.** `tools/verify.sh` reads the *working
  tree*, not the `--base` commit range. This voided one full gate run this session. Reviewers are
  read-only and *may* overlap the gate; implementers may not.
- **`golangci-lint` is unavailable in every agent environment tried so far.** The `--new-from-rev`
  unmasking hazard on four new `builder.go` files (cashshop `asset`, npc `conversation`, pets
  `data/pet`, npc `conversation/quest`) is therefore unchecked and rides entirely on the repo-wide
  gate's `lint & format guard`. If that guard fails, this is the first place to look, and the fix
  belongs in a *separate* commit after the relocation per the established pattern.
- **A bare `grep -c 'Builder'` counts prose, not declarations** — see the Task 23 review finding
  above. Task 24 must check `sweep.sh`'s pattern before treating a sixth residue entry as real.
- **State ledger counts from pasted command output, never prose arithmetic.** Batches A and B each
  miscounted by one; A's was blocking.
- **The relocate ledgers are closed at 64 APPLIED / 6 SKIPPED / 70 rows** and must not be
  hand-edited. `70 = 66 (68 groups − 2 reserved) + 4 ENTITY-BUILDER`. Task 25 needs that sentence.
- **All four `SKIPPED` multi-builder rows now resolve to hand-work** (Tasks 22, 23, 23-B). Task 25
  should find **no** package needing an exemption on "codemod declined it" grounds; the only
  `exemptions.md` entries are the `SIBLING-EXEMPT` constructor sets (D4) and the documented
  `EXCLUDED-TREE` / `ENTITY-BUILDER` dispositions.

## Task 23-B — remaining multi-builder hand splits

Both `SKIPPED` rows left without a plan destination after Task 21-C's review are now hand-split,
following Task 22's and Task 23's procedure verbatim.

**`services/atlas-pets/atlas.com/pets/data/pet/model.go` → `builder.go`** — commit `27c5629`.
Two builder declaration sets moved (from Step 1's enumeration):

```
type Builder struct { ... }              (NewBuilder, SetId, SetHunger, SetCash, SetLife,
                                           SetSkills, AddSkill, SetReqPetLevel, SetReqItemId,
                                           SetEvolutions, Build)
type SkillModelBuilder struct { ... }    (NewSkillModelBuilder, SetId, SetIncrease,
                                           SetProbability, Build)
```

`grep -c 'Builder' model.go` after the move: `0`. No package imports were required by either file
(the original file had none); `builder.go` and `model.go` both carry no import block.

**`services/atlas-npc-conversations/atlas.com/npc/conversation/quest/model.go` → `builder.go`** —
commit `46cc378`. Two builder declaration sets moved, matching `classify-file05.tsv`'s count for
this file:

```
type Builder struct { ... }              (NewBuilder, SetId, SetQuestId, SetNpcId, SetQuestName,
                                           SetStartStateMachine, SetEndStateMachine, SetCreatedAt,
                                           SetUpdatedAt, Build)
type StateMachineBuilder struct { ... }  (NewStateMachineBuilder, SetStartState, SetStates,
                                           AddState, Build)
```

`grep -c 'Builder' model.go` after the move: `0`. Both `model.go` and `builder.go` keep the
identical four-import block (`atlas-npc-conversations/conversation`, `errors`, `time`,
`github.com/google/uuid`) — every import is used on both sides of the split (domain accessors use
`uuid.UUID`/`time.Time`/`conversation.StateModel`/`errors.New`; the builders use the same set).

Both moves were verified byte-identical per Step 3 (`git diff --cached -M`, balanced `-`/`+` pairs
after accounting for the blank-line filter asymmetry the brief warned about — `^\+$` was excluded
by the filter while `^-$` was not, producing a phantom imbalance on blank lines only; re-running
with both blank-line forms excluded showed a clean balance). Module-local `go build ./... && go vet
./... && go test ./...` passed for both `services/atlas-pets/atlas.com/pets` and
`services/atlas-npc-conversations/atlas.com/npc`.

**Neither row is an `exemptions.md` entry.** Both `ledger-relocate-a.tsv`/`ledger-relocate-b.tsv`
`SKIPPED` rows for these two files now resolve to hand-work landed in this task, exactly like Task
22's and Task 23's rows. Task 25 should attribute both to Task 23-B, not to the exemption list.

## Gate 23-B — FAIL (`lint & format guard`), one target

`.gatelogs/gate-23b.log`, `tools/verify.sh --quick --base b7f3e8d90`, `53 changed path(s)` →
`shared-lib change` → repo-wide fan-out, `91 changed Go module(s)`.

Everything before the lint guard passed: `go-analyzer-guards: PASS`, `skill-job-id-guard: clean`,
`scopeguard` 91 modules, `producerseamguard` 68 modules, `service-registration-guard: clean`,
`envguard` 68 modules.

**One real failure, and it is exactly the hazard the previous handoff predicted** — the
`--new-from-rev` unmasking on a newly-created `builder.go`:

```
services/atlas-cashshop/atlas.com/cashshop/asset/builder.go:42:7: S1016: should convert model (type EquipableReferenceData) to EquipableReferenceDataBuilder instead of using struct literal (staticcheck)
	*b = EquipableReferenceDataBuilder{
	     ^
services/atlas-cashshop/atlas.com/cashshop/asset/builder.go:305:7: S1016: should convert model (type CashEquipableReferenceData) to CashEquipableReferenceDataBuilder instead of using struct literal (staticcheck)
	*b = CashEquipableReferenceDataBuilder{
	     ^
2 issues:
* staticcheck: 2
lint.sh: LINT FAIL — services/atlas-cashshop/atlas.com/cashshop
```

Both sites are `Clone` methods that Task 22 moved **verbatim** out of `reference_data.go`. The code
is unchanged and was previously invisible to `golangci-lint --new-from-rev`; creating `builder.go`
made those lines "new" and unmasked the pre-existing diagnostic. Per the standing pattern the fix
is a **separate commit after the relocation**, never folded back into the move commit.

**The run did complete** (the log was still being written when this session first read it; the
background job then reported `exit 1`). Final tally, quoted:

```
lint.sh: FAIL — 1 failing target(s):
...
1 check(s) FAILED — the branch is not ready.
```

`grep -E 'LINT FAIL|FMT FAIL|failing target'` over the whole 450-line log returns exactly those two
lines — **this cashshop staticcheck pair is the only failure in the entire range**, and no `fmt`
target failed. Every other check passed.

**Last validly gated commit remains `b7f3e8d90`.** The fix commit joins the next gate's range, which
still covers Tasks 21-C, 22, 23 and 23-B.

## Gate 23-B fix — `fdd466783`, review APPROVED (0 blocking, 0 non-blocking)

Both `Clone` methods now use a type conversion; 2 insertions, 51 deletions, one file:

```go
*b = EquipableReferenceDataBuilder(model)
*b = CashEquipableReferenceDataBuilder(model)
```

Artifact: `review-gate23b-fix.md`. **The review was dispatched for one specific reason and it earned
its keep:** a type conversion copies *every* field unconditionally, while the struct literal it
replaced copied an explicitly enumerated list. Those are equivalent only if the literal was a
complete, untransformed copy — and the conversion compiling proves only that the underlying types
match, *not* that the literal omitted nothing. A field deliberately left at its zero value would
compile clean and still be a silent behavior change. The reviewer checked this directly against the
pre-image rather than trusting the report:

- `EquipableReferenceDataBuilder` (`builder.go:9-32`) vs `EquipableReferenceData`
  (`reference_data.go:9-33`) — identical **23** fields, same order/names/types; pre-image literal
  enumerated all 23 1:1, no omission, no transform.
- `CashEquipableReferenceDataBuilder` vs `CashEquipableReferenceData` (`reference_data.go:73-98`) —
  identical **24** fields (`cashId` + the same 23); pre-image literal enumerated all 24.

Commit touches only `asset/builder.go`, and only the two `Clone` bodies; the other five builder
types in the file are untouched.

**Not evaluable (1):** `golangci-lint` is unavailable in the reviewer's environment too — another
agent environment on this branch without it. Whether S1016 is actually silenced is owned by the
repo-wide gate, not by any agent.

## Task 24 — FILE-05 closeout

Re-ran `sweep.sh` as-is (no hand edits to the script or any generated inventory file). It
regenerates five inventory files; only two of the five changed against the pre-Task-24 tree,
because they were last regenerated before Tasks 19-23-B landed (`inventory-dom04-outbound.txt`,
`inventory-dom04-inbound.txt`, and `inventory-dom01-modelbuilder-type.txt`/
`inventory-dom01-newmodelbuilder.txt` were already current and printed no diff):

- `inventory-file05-builders.txt` — 99 lines → **5 lines**. The 94-line drop is exactly the 64
  APPLIED relocations (Tasks 19-21) plus the four hand-split multi-builder files (Tasks 22, 23,
  23-B) leaving `builder.go`, now correctly excluded by the script's own `grep -v '/builder.go:'`
  filter, plus the `_test.go` filter removing nothing new.
- `inventory-dom01-other.txt` — 67 lines, same count before and after, but 39 of the 67 rows'
  `func New*Builder` path moved from the old `model.go`/`reference_data.go`/etc. location to
  `builder.go` at its new post-move line number (verified via `git diff`; 39 removed / 39 added,
  no row lost or gained).

### FILE-05 residue — verbatim contents of `inventory-file05-builders.txt`

```
libs/atlas-packet/model/skill_usage_info.go:143:type SkillUsageInfoBuilder struct {
services/atlas-quest/atlas.com/quest/quest/entity_builder.go:10:type entityBuilder struct {
services/atlas-quest/atlas.com/quest/quest/progress/entity_builder.go:5:type entityBuilder struct {
services/atlas-tenants/atlas.com/tenants/configuration/entity_builder.go:9:type entityBuilder struct {
services/atlas-tenants/atlas.com/tenants/tenant/entity_builder.go:5:type entityBuilder struct {
```

**Exactly the five documented dispositions from the brief, in the same order, no sixth entry.**
Cross-checked against the relocate ledgers (64 APPLIED / 6 SKIPPED / 70 rows, closed) and the
Task 22/23/23-B hand-split commits: `reference_data.go`, `conversation/model.go`,
`data/pet/model.go`, and `conversation/quest/model.go` do not appear — all four multi-builder
SKIPs are resolved by hand-work, not exemption, matching the controller's note. The doc-comment
prose mention of `CraftActionBuilder` still exists, unchanged, at
`services/atlas-npc-conversations/atlas.com/npc/conversation/model.go:444`
(`// cannot be produced through the validated CraftActionBuilder.`) — confirmed directly with
`grep -n`. It does not surface as a sixth entry because the declaration-anchored `^type ...
Builder struct` pattern cannot match prose, exactly as the controller's note said; the real
`type CraftActionBuilder struct` declaration lives at `conversation/builder.go:722`, inside the
`grep -v '/builder.go:'` exclusion. **FILE-05 is closed: no code change required, no forgotten
package found.**

## Task 24 review — APPROVED_WITH_FINDINGS (0 blocking, 1 non-blocking)

Artifact: `review-task-24.md`. Commit `9c66d7c`. The reviewer re-derived rather than re-read:

- **Re-ran `sweep.sh` itself** and confirmed `git status` shows zero diff against the committed
  inventories — so they are genuinely re-derivable output, not hand-edited. Tree left clean.
- **`inventory-dom01-other.txt` churn is path-only.** 39 removed / 39 added, row count unchanged
  (67 → 67). The reviewer built `(package-dir, declaration-text)` multisets for both versions,
  ignoring path basename and line number, and diffed them — **identical**. Every changed row moved
  only its file path, all landing on `builder.go`; no declaration was altered, added, or dropped.
  That is the check that would have caught a relocation quietly renaming a constructor.
- The three "already current" inventories genuinely re-generated to identical content (not skipped).
- `git log -- sweep.sh`: only the original PRD commit ever touched the script. Unmodified since.

**Non-blocking → carried into Tasks 25/26.** The reviewer found `progress.md`'s "already current"
clause folds in `inventory-dom01-modelbuilder-type.txt`, a tracked file `sweep.sh` never writes.
**The controller checked the full set and it is broader than the finding states — three tracked
inventory files are outside the script's write set entirely:**

```
inventory-dom01-modelbuilder-type.txt
inventory-dom04-has-model.txt
inventory-dom04-no-model.txt
```

`sweep.sh` writes exactly five: `inventory-dom04-outbound.txt`, `inventory-dom04-inbound.txt`,
`inventory-file05-builders.txt`, `inventory-dom01-newmodelbuilder.txt`, `inventory-dom01-other.txt`.
The other three are hand-derived analysis artifacts from earlier tasks and are **frozen at whatever
tree they were produced against** — re-running the sweep does not refresh them.

**Tasks 25-27 must not cite those three as current evidence** without re-deriving them, and any
final write-up that says "the inventories are re-derivable via `sweep.sh`" is true only of the five.
This does not affect the FILE-05 conclusion, which rests on `inventory-file05-builders.txt`.

## Gate over `b7f3e8d90..fdd466783` — PASS

`.gatelogs/gate-fix23b.log`. `tools/verify.sh --quick --base b7f3e8d90`, exit 0,
`55 changed path(s)` → shared-lib fan-out → 91 modules.

```
✓ service registration guard
✓ env domain guard
✓ lint & format guard (91 module(s))

All checks passed, but docker bake was skipped — not a pre-PR pass.
```

Zero occurrences of `LINT FAIL` / `FMT FAIL` / `failing target` / `FAILED` in the whole log. The
`lint & format guard` — the single check that failed on gate-23b — is now clean, which **confirms
the staticcheck S1016 fix**. That confirmation could only come from here: `golangci-lint` is absent
from every agent environment on this branch, so no implementer or reviewer could establish it.

**Last validly gated commit: `fdd466783`.** This range covers Tasks 21-C, 22, 23, 23-B and the
gate-23B fix. Note the trailing caveat — `--quick` skips the docker bake, so per CLAUDE.md this is
the iteration gate, not a pre-PR pass. The flagless `tools/verify.sh` still has to run once at
branch end.

Commits landed after this gate (`9c66d7c` Task 24, plus the docs commits) join the next range.

## Task 25 — controller ruling: split into 25-A and 25-B, and its sources are stale

`plan.md`'s Task 25 writes one `exemptions.md` across eight sections, ~250 entries, each needing
`file:line` confirmed at HEAD. As a single dispatch that is a near-certain `PARTIAL`. Split:

- **25-A** — the three DOM-04 sections; creates the file. Brief: `.superpowers/sdd/plan/task-25a-brief.md`.
- **25-B** — the DOM-01 and FILE-05 sections, plus plan Steps 3-5 (evidence check, counts, commit).

### Re-derived counts — `plan.md`'s section table is wrong in two places

| section | `plan.md` says | actually in source |
|---|---|---|
| DOM-04 — no `type RestModel` | 14 | **12** `NO-RESTMODEL` rows in `classify-dom04.tsv` |
| DOM-01 — sibling builders | 55 | **61 rows**, but only **25 distinct packages** |

`classify-dom04.tsv` dispositions in full: `94 A / 61 B2 / 12 NO-RESTMODEL / 11 B1 / 7 C`.
`classify-dom01-fr15.tsv`: `61 SIBLING-EXEMPT / 5 NO-MODEL-GO / 3 RENAME`.

### STALE-CITATION HAZARD — the real risk in Task 25

The classify TSVs and inventories were generated by **Task 2**. Tasks 19-23B have since relocated
builder declarations across ~900 files. **Every `file:line` in those sources is suspect and many are
provably wrong.** Proof, from `classify-dom01-fr15.tsv`'s `atlas-cashshop/asset` row:

```
siblings: NewBuilder (model.go:116), NewEquipableReferenceDataBuilder (reference_data.go:100),
NewCashEquipableReferenceDataBuilder (reference_data.go:425), ... (reference_data.go:947)
```

Task 22 moved all seven of those into `asset/builder.go` and truncated `reference_data.go` to 245
lines — so four of those offsets **do not exist in the file at all**. Plan Step 2 already says "an
entry citing a stale line is worse than no entry"; both briefs make re-derivation at HEAD mandatory
and forbid transcribing a `file:line` from a TSV.

Corollary for both implementers: **a package that has since gained the thing it was exempted for
must not be listed as exempt.** `progress.md` records that 3 of the 12 `NO-RESTMODEL` rows were
closed by Task 13, leaving 9 genuinely exempt.

## Task 25-A — complete (`4ed1cef25`), DONE_WITH_CONCERNS

Three DOM-04 sections written into a new `exemptions.md`. Every `file:line` re-derived at HEAD with
`grep -n`/`sed -n` rather than transcribed — the stale-citation discipline held.

| section | plan.md said | emitted |
|---|---|---|
| no domain `Model` | 176 | **171** in clusters + **5** called out separately (see ruling) |
| no `type RestModel` | 14 | **12** |
| lossy `Extract` | "as found" | **19** (12 dropped-field + 4 reference-type-copy + 3 both) |

**The `NO-RESTMODEL` count resolved differently than the brief predicted, and the implementer was
right to override it.** The brief said 3 of 12 were closed by Task 13, leaving 9 exempt. Re-confirmed
at HEAD: **all 12 now have named `Transform*` coverage** — Task 14's batches A-D closed the rest
after that `progress.md` note was written. All 12 therefore carry the same disposition ("exempt from
the literal `func Transform(` detector; satisfied by the named variants"), and the 9/3 split in the
note above is superseded.

### CONTROLLER RULING — the 5 "false positive" rows are legitimately one-directional

25-A found that 5 of the 176 rows are misclassified by the inventory: they DO have a domain type,
under a name other than literally `Model`, and each has only one direction of `Extract`/`Transform`.
It correctly declined to write "the package has no domain Model at all" for them and escalated
rather than inventing a disposition.

**The criterion was already mechanical and did not need a judgment call** — `sweep.sh` itself splits
DOM-04 by whether a package serves JSON:API (`resource.go` present) or is an inbound-only read
client (absent). Checked directly:

| package | `resource.go` | direction present | verdict |
|---|---|---|---|
| `services/atlas-channel/atlas.com/channel/party_quest` | absent | `ExtractTimer` | correct |
| `services/atlas-maps/atlas.com/maps/data/map/monster` | absent | `Extract` | correct |
| `services/atlas-npc-conversations/atlas.com/npc/cosmetic` | absent | `ExtractAppearance` | correct |
| `services/atlas-transports/atlas.com/transports/instance` | **present** | `TransformRoute`, `TransformInstanceStatus` | correct |
| `services/atlas-marriages/atlas.com/marriages/marriage` | **present** | `TransformMarriage`, `TransformProposal` | correct |

An inbound-only read client consumes a REST response, so `Extract` (RestModel → domain) is the only
direction it needs; a package serving JSON:API produces one, so `Transform` (domain → RestModel) is
the only direction it needs. **In all five the direction present is exactly the direction the
package's role requires. None is a gap.** The split is 3 absent / 2 present with zero
counter-examples — this is derived, not asserted.

**Disposition for all five: N/A by trigger, not "exempt for lack of a Model."** They have a domain
type; DOM-04's literal `type Model` detector simply does not fire on it, the same structural reason
as the `NO-MODEL-GO` class. **Task 25-B must rewrite 25-A's "Excluded from this section"
subsection** to carry this disposition and this table, so the artifact does not read as five
unresolved findings. The `atlas-marriages` note stands: `marriage` was that service's only row, so
its cluster is legitimately empty in section 1.

## Task 25-A review — APPROVED_WITH_FINDINGS (6 blocking, 2 non-blocking)

Artifact: `review-task-25a.md`. Commit `4ed1cef25`.

**The stale-citation risk did not materialize, and this was checked properly.** The reviewer sampled
**30+ citations across all three DOM-04 sections**, deliberately targeting files this branch
rewrote (`builder.go`, `model.go`, `rest.go`, and the heavily-relocated
`atlas-npc-conversations/conversation`, `atlas-tenants/configuration`, `atlas-transports/instance`).
**Every sampled citation resolved exactly at HEAD — 100%.** That substantiates 25-A's claim to have
re-derived rather than transcribed, which was the main thing this review existed to test.

Also independently confirmed: all 12 `NO-RESTMODEL` packages genuinely have the named `Transform*`
the document cites (re-derived from `classify-dom04.tsv`, not read off the report) — so 25-A's
override of the stale 9/3 split is correct; 14 of the 176 "no domain `Model`" rows spot-checked for
absence of `type Model`, zero hits; all 19 lossy-`Extract` findings traced to `handwork-notes.md` at
the cited lines with matching content.

**6 blocking — all one mechanical defect, all in Section 3.** Six of 62 `### ` headings cite a bare
filename (`rest.go`, `model.go`, no line number) or only a `handwork-notes.md:NN` reference, which
is not a `.go` file. That fails the plan's hard acceptance criterion (every heading must carry a
backtick-quoted `<path>.go:<line>` before the next heading). Lines 87, 318, 338, 342, 350, 358.

**Routed to Task 25-B as Step 0b, not a separate fix round** — 25-B already owns plan Step 3, the
whole-file evidence sweep, so the fix and its verification land in one place. The line numbers are
derivable; the reviewer demonstrated one (`map/processor.go:40` for the `Exists` function that entry
names in prose but does not cite). Section 3's content is sound and must not be restructured — only
the citations are deficient.

## Task 26 — FR-17/FR-18 verification

Behavior-preservation audit over the full branch diff (`main...HEAD`, 923 files changed). Every
step's diff was reduced mechanically before reading — never read raw whole — as instructed. The
audit found **no defects**: every deletion pairs with an identical (or identically-renamed, per
the branch's own `ModelBuilder`→`Builder` constructor-rename work) addition.

### Step 1: FR-18 — excluded trees

```
git diff --name-only main...HEAD | grep -E '\.sql$|docker-compose.*\.ya?ml$|\.tpl$|^libs/atlas-packet/'
```
Output: (none — exit code 1, no match). **PASS.**

### Step 2: FR-17 — no JSON tag changed

Reduction: ran with `-U0` (no context lines) instead of the brief's default-context form (7601
lines with context vs 11 actual changed lines), then split into `+`/`-` sets and sorted.

```
git diff -U0 -G'json:"' main...HEAD | grep -E '^[+-].*json:"' | grep -v '^[+-][+-][+-]'
```
Result: **13 lines total, all 13 are `+` (additions), 0 are `-` (deletions)**. (Correction: the
figure originally pasted here was "11"; re-running the identical command reproduces 13. The 2-line
undercount was immaterial to the conclusion — the two extra lines are
`docs/tasks/task-263-backend-guideline-conformance/design.md`'s rename-safety table row (prose
quoting `` `json:"modelBuilder"` `` and `"NewModelBuilder"` as example literals) and
`docs/tasks/task-263-backend-guideline-conformance/codemod/rename_test.go`'s
`mustContain(t, got, \`json:"modelBuilder"\`)` assertion — a codemod unit test verifying the tag
*survives* the rename, not a tag edit.) All 13 lines are additions inside docs/test-assertion
strings, confirmed via `git diff -U0 -G'json:"' main...HEAD | grep -n '^diff --git\|^+++'`: every
hit is in `rename_test.go`, `design.md`, `plan.md`, `progress.md`, a `review-*.md`, or
`teleportrock/rest_test.go` — no `.go` source file has a changed JSON struct tag (confirmed
separately: `git diff -U0 main...HEAD -- '*.go' | grep -E '^[+-].*json:"'` → empty). **PASS.**

### Step 3: FR-17 — no `GetName()` return value changed

```
git diff -U0 main...HEAD -- '*.go' | grep -E '^[+-].*GetName\(\) string'
```
Output: (none — exit 1). **PASS** — no `GetName() string` declaration touched anywhere on the
branch. (Scoped to `-- '*.go'`: the unscoped form now also matches this section's own prose once
this commit lands, since `main...HEAD` includes the sentence you are reading; scoping to `.go`
files avoids that self-reference.)

### Step 4: FR-17 — no route registration or Kafka topic changed

Reduction: extracted `+`/`-` lines (188 total) from
`git diff -U0 main...HEAD -- '*/resource.go' '*/producer.go' '*/consumer.go'`, then per-file
inspection since the destination files (new `builder.go`s) fall outside this glob and can't
appear as the `+` counterpart within the same diff.

Files touched: `atlas-channel` consumers (asset, drop, message, monster, pet, reactor),
`atlas-character/character` consumer, `atlas-consumables/equipable/producer.go`,
`atlas-drops` consumer, `atlas-pets/pet/resource.go`, `atlas-quest/kafka/producer/saga/producer.go`,
`atlas-reactors` consumer, `atlas-skills/macro` consumer.

Every deletion in this set is one of:
1. A constructor-rename call-site rewrite with **identical arguments**, e.g.
   `asset.NewModelBuilder(e.AssetId, e.CompartmentId, e.TemplateId)` →
   `asset.NewBuilderWithId(e.AssetId, e.CompartmentId, e.TemplateId)`; also
   `drop.NewModelBuilder()`→`drop.NewBuilder()`, `pet.NewModelBuilder(...)`→`pet.NewBuilder(...)`,
   `monster.NewModelBuilder(...)`→`monster.NewBuilder(...)`,
   `reactor.NewModelBuilder(...)`→`reactor.NewBuilder(...)`,
   `character.NewModelBuilder()`→`character.NewEmptyBuilder()`,
   `macro.NewModelBuilder()`→`macro.NewBuilder()`.
2. A type rename: `type Change func(b *asset.ModelBuilder)` → `type Change func(b *asset.Builder)`
   in `atlas-consumables/equipable/producer.go`.
3. The saga `Builder` type (struct + `NewBuilder` + 6 `AddAward*`/`AddConsumeItem`/`AddCreateSkill`
   methods + `Build`/`HasSteps`) relocated verbatim from
   `atlas-quest/.../saga/producer.go` to a new `saga/builder.go` (confirmed by `git show HEAD:.../builder.go`
   byte-diffed against the deleted block from `main`: identical except import grouping and that
   `stepId()` correctly stayed behind in `producer.go`, still present and referenced by both files
   in the same package).
4. One comment-text update (`// petLifespan ... matching NewModelBuilder's` → `... matching NewBuilder's`).

Grep for `topic|route|"/|Path\(` inside the reduced diff: no hits. **No route or Kafka topic
literal changed. PASS.**

### Step 5: FR-17 — no `Build()` validation rule changed

Reduction: `-U0`, scoped to `*.go` files only (346 of 349 raw lines; the other 3 are markdown
prose plus 2 lines from a wholly new file, the codemod tool itself), then paired `-`/`+` sets by
normalized (whitespace-stripped) exact text via `comm`.

```
git diff -U0 main...HEAD -- '*.go' | grep -E '^[+-].*errors\.New\(|^[+-].*fmt\.Errorf\(' | grep -v '^[+-][+-]'
```
136 deletions, 210 additions. `comm -23` (deletions with no identical addition anywhere on the
branch) → **0 unmatched.** Every removed `errors.New(...)`/`fmt.Errorf(...)` call has an
identical-text counterpart added elsewhere (a relocation to `builder.go`); the extra 74 additions
are new validation calls, which FR-17 does not prohibit. **PASS.**

### Step 6: FR-17 — no struct field type changed

**Correction note:** the figure originally pasted here ("17 unmatched, checked individually")
did not reproduce under re-run (see `review-task-26.md`'s independent reconstruction, which got 8,
26, or 1 depending on how the rename was applied), and the write-up itemized only 6 of the 17
items it claimed to check. The evidence below replaces it with a script whose command and output
are pasted verbatim and were run twice to confirm determinism. The substantive PASS conclusion is
unchanged — no `Model`/entity field is dropped — but the residual count is now **3**, not 17 or 1;
see the itemization below for what each residual is and why it is benign.

Reduction: `-U0`, scoped to `*/model.go` and `*/entity.go`:

```
git diff -U0 main...HEAD -- '*/model.go' '*/entity.go' | grep '^-' | grep -v '^---' > /tmp/step6_del.txt
wc -l /tmp/step6_del.txt
```
`6785 /tmp/step6_del.txt` — reproduces the raw deletion count exactly.

Branch-wide addition set (every `+` line in the full diff, not scoped to the two globs, since a
relocated field can land in a new `builder.go` outside them):

```
git diff -U0 main...HEAD | grep '^+' | grep -v '^+++' > /tmp/step6_add.txt
wc -l /tmp/step6_add.txt
```
`46802 /tmp/step6_add.txt`.

First pass — exact multiset match, no rename applied (strip leading `-`/`+` and surrounding
whitespace only, no other transformation): **548 unmatched**, reproducing the report's original
"First pass: 548 unmatched."

Applying a single blind text substitution of `ModelBuilder`→`Builder` (the report's own partial
method) does not converge cleanly: it both under- and over-matches, because the branch's actual
FR-14 rename is **identifier-based, not substring-based** (`design.md` §4.3: rename is driven by
`types.Object` identity, resolved separately for the `ModelBuilder`/`modelBuilder` type and the
`NewModelBuilder`/`CloneModelBuilder` functions — it does not touch a same-spelled substring inside
an unrelated identifier like `SkillModelBuilder`, and the unexported `modelBuilder`→`builder` leg
is itself conditional: `design.md` §4.3 step 3 guards it — "before rewriting an unexported
`modelBuilder` → `builder`, check the target scope for an existing object named `builder`. If one
exists, skip the package to hand work" — which is exactly why `atlas-character/character` kept
`modelBuilder` unrenamed while `atlas-channel/party` renamed it).

The reproducible reconciliation therefore matches in three phases against the *same* fixed
addition pool, each phase consuming what the previous phase already matched, so a line is never
double-counted: (1) exact text, unrenamed — catches packages where the codemod's guard left an
identifier unchanged; (2) the four capital-cased identifiers renamed as whole tokens
(`\bNewModelBuilder\b`→`NewBuilder`, `\bCloneModelBuilder\b`→`CloneBuilder`,
`\bModelBuilder\b`→`Builder`, applied most-specific-first so `NewModelBuilder`/`CloneModelBuilder`
aren't double-substituted via the `ModelBuilder` rule, and `\b` boundaries so `SkillModelBuilder`
is correctly left untouched); (3) the lowercase `\bmodelBuilder\b`→`builder` rename, applied last
since it's the guarded/conditional leg.

```python
import re
from collections import Counter

def normalize(line, prefix):
    assert line.startswith(prefix)
    return line[1:].strip()

del_lines = [normalize(l, '-') for l in open('/tmp/step6_del.txt')]
add_lines = [normalize(l, '+') for l in open('/tmp/step6_add.txt')]

del_c = Counter(del_lines)
add_c = Counter(add_lines)

CAP_REPLACEMENTS = [
    (r'\bNewModelBuilder\b', 'NewBuilder'),
    (r'\bCloneModelBuilder\b', 'CloneBuilder'),
    (r'\bModelBuilder\b', 'Builder'),
]
LOWER_REPLACEMENT = (r'\bmodelBuilder\b', 'builder')

def apply(patterns, s):
    for pattern, repl in patterns:
        s = re.sub(pattern, repl, s)
    return s

# Phase 1: exact (no-rename) match.
common1 = del_c & add_c
residual_del, residual_add = del_c - common1, add_c - common1

# Phase 2: capital-identifier rename.
residual_del_cap = Counter()
for line, cnt in residual_del.items():
    residual_del_cap[apply(CAP_REPLACEMENTS, line)] += cnt
common2 = residual_del_cap & residual_add
residual_del_cap -= common2
residual_add -= common2

# Phase 3: lowercase (guarded/conditional) rename.
residual_del_lower = Counter()
for line, cnt in residual_del_cap.items():
    residual_del_lower[apply([LOWER_REPLACEMENT], line)] += cnt
final_unmatched = residual_del_lower - (residual_del_lower & residual_add)

print("FINAL unmatched:", sum(final_unmatched.values()))
for line, count in final_unmatched.items():
    print(count, repr(line))
```

Run twice; identical output both times:

```
FINAL unmatched: 3
1 'import "github.com/Chronicle20/atlas/libs/atlas-constants/world"'
1 'stance             byte'
1 'import "time"'
```

All 3 itemized and traced:

1. `import "github.com/Chronicle20/atlas/libs/atlas-constants/world"` — from
   `libs/atlas-constants/channel/model.go`. Confirmed via
   `grep -n 'atlas-constants/world"' /tmp/step6_add.txt`: the addition side has the same import
   path (`"github.com/Chronicle20/atlas/libs/atlas-constants/world"`) but without the `import `
   keyword prefix, because it moved from a single-line `import "..."` statement into a grouped
   `import (\n\t"..."\n)` block — the normalization (strip only `-`/`+` and whitespace) does not
   strip the `import ` keyword, so the two forms don't text-match even though it's the same
   import, same package, no import added or removed.
2. `import "time"` — same pattern, from `services/atlas-consumables/atlas.com/consumables/pet/model.go`
   (confirmed via `awk` locating the `diff --git` header immediately preceding the deletion line),
   reformatted into a grouped import block on the addition side (`+\t"time"`, confirmed present at
   multiple lines in `/tmp/step6_add.txt`).
3. `stance             byte` — 6 deletions of this exact normalized text branch-wide, 5 additions
   (confirmed by direct count: `lines.count('stance             byte')` → del 6, add 5). Traced to
   `services/atlas-pets/atlas.com/pets/character/model.go`: this is a field of the
   `ModelBuilder`/`Builder` struct (mirrors `Model`'s own `stance` field but lives in the builder,
   confirmed via wider-context diff — `type Model struct{...}` in that file is untouched, the
   deletion starts immediately after `Model`'s last method), and is confirmed present, unchanged,
   in the new `character/builder.go` at `Build()`. This is the same line the original write-up
   independently traced by hand.

**No `Model`/entity struct field was dropped anywhere on the branch.** Every one of the 3 residuals
is either an import-block reformat (same import, no import added or removed) or the already-traced
`stance` builder field, not a dropped struct field. Every other deletion in this glob is a
`*Builder` declaration relocated to a new `builder.go` (in the same package), in some cases with
the `ModelBuilder`→`Builder`/`modelBuilder`→`builder` rename that the branch's own W1/W2 codemod
applies per the identifier-identity and guard rules in `design.md` §4.3. **PASS.**

### Overall result

FR-17 and FR-18 hold across the full 923-file diff. No behavior-preservation defect found;
Task 26 reports DONE, no findings to route to Task 27.

## Task 25-B — complete (`9ce61b728`), DONE — review APPROVED, 0 findings

The five DOM-01/FILE-05 sections appended to `exemptions.md`, plus the two corrections routed here:
Step 0 (rewrite 25-A's "Excluded from this section" to carry the N/A-by-trigger disposition and the
five-package table) and Step 0b (the 6 blocking citation defects from `review-task-25a.md`).

Counts emitted: **25** sibling-builder packages (from 61 duplicated `SIBLING-EXEMPT` rows), **5**
trigger-not-fired, **2** NewBuilder-name-taken, **4** entity builders, **1** excluded tree. All match
the controller's re-derived table; `plan.md`'s "55" for the first was a row count, not a package count.

**The stale-citation hazard was the whole point of this task and it was handled correctly.** Review
artifact `review-task-25b.md`, verdict **APPROVED — 0 blocking, 0 non-blocking**. 30+ citations
sampled across all five new sections, deliberately targeting the packages Tasks 19-23B relocated
(`atlas-cashshop/asset`, `atlas-npc-conversations/conversation`, `atlas-channel/transport/route`,
`atlas-pets/data/pet`) — every one resolved exactly at HEAD. Acceptance criterion 3 re-run
independently: **62 headings, zero without a `<path>.go:<line>`**. Criterion 4: 4 `out of scope`
hits, all in 25-A content, each carrying a `file:line` on the same line.

### Two content divergences found by re-derivation — both real, both confirmed

These are worth recording because they are the *evidence* that re-derivation actually happened
rather than transcription; a transcribing implementer could not have produced them.

- `services/atlas-channel/atlas.com/channel/transport/route` — lost a duplicate `NewModelBuilder`
  constructor between Task 2 and HEAD. The entry cites **5** siblings, not the TSV's 6.
- `services/atlas-pets/atlas.com/pets/data/pet` — primary constructor renamed
  `NewModelBuilder` → `NewBuilder` by a later task.

Neither changes a package-level count. Both independently re-confirmed by the reviewer with
`grep -n "^func New"`.

## Task 26 — complete (`29b52636b`), DONE, zero defects reported

The FR-17/FR-18 behavior-preservation audit over the full `main...HEAD` diff (**923 files**). All six
steps executed; the implementer reports every deletion mechanically paired with its addition or
individually traced, and **zero behavior-preservation defects**. Evidence pasted verbatim into the
`## Task 26 — FR-17/FR-18 verification` section above.

**This verdict is under adversarial review before it is trusted.** "Zero defects across 923 files" is
the most consequential unverified claim on the branch — Task 27 is all that remains, so a spot-check
presented as a sweep would ship a dropped `Build()` validation rule or a deleted `Model` field. The
reviewer was briefed to re-run all six commands itself, test whether the pairing method can hide a
cross-package move, and probe Step 5 (`errors.New`/`fmt.Errorf`) specifically — a dropped validation
rule is invisible to the compiler and is the defect this branch is most likely to have produced.

### Gate status — no gate was needed for this stretch

`tools/task-facts.sh task-263 --base fdd466783` reports `go_changed=false`: Tasks 24, 25-A, 25-B and
26 are **docs-only**. `1424fc47a` also folded in a stray `go.work.sum` hash line and the Task 25-A
reviewer ledger row that had been left uncommitted.

**Last validly gated commit remains `fdd466783`.** Nothing since it touches Go, so nothing since it
is gate-relevant — but per CLAUDE.md that is still only the `--quick` iteration gate. The flagless
`tools/verify.sh` has not yet run on this branch and is Task 27's job.

## Task 26 review — CHANGES_REQUIRED (1 blocking), fixed in `109d2c474`

Artifact: `review-task-26.md`. **The review found no missed defect — it found an unreproducible
number.** That distinction is the whole finding, and it was worth catching: Task 26's claim is the
last thing standing between this branch and a PR, and "923 files, zero defects" is exactly the shape
of a spot-check wearing a sweep's clothing.

The reviewer re-ran all six commands and rebuilt the Step 5/6 pairing logic from scratch rather than
proofreading the prose. **Steps 1, 3, 4 and 5 reproduce cleanly.** Step 5 — dropped `Build()`
validation rules, the defect this branch was most likely to have produced and the one invisible to
the compiler — it verified to a *stronger* standard than the original, pairing across files and
packages rather than by text multiset, and found **zero false pairs**.

Step 6 was the blocking failure: the pasted "17 unmatched, checked individually" reproduced as 8, 26
or 1 depending on how the rename was applied, and the write-up itemized only 6 of the 17.

### CONTROLLER RULING — the fix converged on 3, not the reviewer's 1, and 3 is the right answer

The fix implementer reported `DONE_WITH_CONCERNS` because its reproducible method yields **3**
residuals where the reviewer's independent reconstruction yielded **1**. It declined to add an
unstated normalization to force agreement. **That was the correct call and the discrepancy is fully
explained, not open.** Verified by reading the corrected section directly:

| residual | disposition |
|---|---|
| `import "github.com/Chronicle20/atlas/libs/atlas-constants/world"` | import-block reformat — single-line `import "..."` became a grouped `import (...)` block, so the `import ` keyword prefix defeats a text match. Same import, none added or removed. |
| `import "time"` | same pattern, `atlas-consumables/consumables/pet/model.go` |
| `stance             byte` | the already-traced `*Builder` field in `atlas-pets/pets/character`; present unchanged in the new `character/builder.go`. `type Model struct` in that file is untouched. |

The reviewer's `1` absorbed the two import reformats via a normalization it did not state; the
implementer's stated method (strip leading `-`/`+` and whitespace, nothing else) does not. **Both
methods agree on the only thing that matters: no `Model` or entity struct field was dropped.**

## Task 27 — final sweep evidence

Ran `bash docs/tasks/task-263-backend-guideline-conformance/sweep.sh`, complete `wc -l` output:

```
   39 docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-modelbuilder-type.txt
    1 docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-newmodelbuilder.txt
   67 docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-other.txt
   13 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-has-model.txt
  155 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-inbound.txt
  176 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-no-model.txt
   34 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-outbound.txt
    5 docs/tasks/task-263-backend-guideline-conformance/inventory-file05-builders.txt
  490 total
```

Ran `bash docs/tasks/task-263-backend-guideline-conformance/split-by-model.sh`, complete `wc -l` output:

```
   13 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-has-model.txt
  176 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-no-model.txt
  189 total
```

**Acceptance criteria check (design §10 / plan Task 27 Step 1):**

1. `inventory-dom01-newmodelbuilder.txt` must be **0** — actual: **1**. FAIL. The single row is
   `services/atlas-quest/atlas.com/quest/quest/builder.go:22:func NewModelBuilder() *modelBuilder {`.
   `exemptions.md:459` documents this package's `NewModelBuilder` as a DOM-01 sibling-builder
   exemption (kept as `NewModelBuilder` rather than renamed to `NewBuilder`, to disambiguate from
   the same package's `NewEntityBuilder`), but that entry exempts the *sibling-builder* finding, not
   the literal `^func NewModelBuilder(` naming pattern this inventory measures. No exemptions.md
   entry addresses the naming pattern itself, and no other package in the repo still uses this name
   (the sibling `atlas-quest/quest/progress` and `atlas-tenants/*` packages in the same DOM-01 cluster
   were all renamed to `NewBuilder`). This count is wrong per the acceptance gate as literally stated.

2. `inventory-file05-builders.txt` must be **5**, all in `exemptions.md` — actual: **5**, all 5
   confirmed present (`exemptions.md` "FILE-05 — entity builders" section lists the 4 entity
   builders; "FILE-05 — excluded tree" section lists the `libs/atlas-packet` builder). PASS.

3. `inventory-dom04-has-model.txt` — every package listed must appear in `exemptions.md`. Actual: 13
   rows; cross-checked each package directory against `exemptions.md` by substring search. 7 found
   (`atlas-cashshop/cashshop/rewardpool`, `atlas-channel/channel/data/tradeability`,
   `atlas-inventory/inventory/data/tradeability`, `atlas-messengers/messengers/character`,
   `atlas-npc-conversations/npc/conversation`, `atlas-parties/parties/character`,
   `atlas-tenants/tenants/configuration`); 6 **MISSING** from `exemptions.md`:
   `services/atlas-consumables/atlas.com/consumables/monster`,
   `services/atlas-data/atlas.com/data/skill`, `services/atlas-dragons/atlas.com/dragons/dragon`,
   `services/atlas-maps/atlas.com/maps/character`, `services/atlas-storage/atlas.com/storage/asset`,
   `services/atlas-summons/atlas.com/summons/summon`. This count/coverage is wrong per the
   acceptance gate as literally stated.

**Verdict: STOP.** Per the brief, a wrong count means a guideline row is not actually closed. Step
2's commit was not performed and the scaffolding was not deleted. This is reported to the controller
as `DONE_WITH_CONCERNS` for a resolution decision (rename `atlas-quest/quest`'s `NewModelBuilder` to
`NewBuilder`, or add an explicit exemptions.md entry for the naming pattern; and either add
`exemptions.md` entries for the 6 missing DOM-04 `has-model` packages or close them with a real
`Transform`/`Extract` implementation).
Reporting the number its own stated method produces, rather than the number that would have looked
like agreement, is the behavior this plan wants.

### Why the naive rename substitution never converged

Recorded because it explains the original 17 and is a trap for any future audit of this branch: the
FR-14 rename is **identifier-based, not substring-based** (`design.md` §4.3). It resolves via
`types.Object` identity, so it leaves a same-spelled substring inside an unrelated identifier
(`SkillModelBuilder`) alone, and the unexported `modelBuilder`→`builder` leg is *conditional* —
§4.3 step 3 skips a package to hand work when `builder` is already taken in the target scope. That
is why `atlas-character/character` kept `modelBuilder` while `atlas-channel/party` renamed it. A
blind `ModelBuilder`→`Builder` substitution therefore both over- and under-matches. The accepted
reconciliation runs three phases against one fixed addition pool, each consuming what the prior
phase matched, with `\b` boundaries and most-specific-first ordering.

Also corrected: Step 2's count (**13**, not 11 — the two extra hits are doc/test-assertion prose,
not Go struct-tag edits) and Step 3's grep, now scoped to `-- '*.go'` so it no longer matches the
audit's own sentence in `progress.md`.

**Blocking finding cleared.** The Step 6 script is pasted verbatim, was run twice with byte-identical
output, and all 3 residuals are itemized and traced by file. No second review round: the requirement
was reproducibility plus itemization, and both are satisfied on inspection.

**FR-17 and FR-18 hold across the full 923-file diff.** Nothing routes to Task 27.

## CONTROLLER RULING — Task 27 Step 1's two failing counts

The Step 1 sweep is acceptance criteria 1-3, and it came back with 2 of 3 failing. The implementer
correctly stopped before Step 2's commit rather than adjusting anything. Both were investigated
directly; **one is a genuine defect and six are detector artifacts.** The distinction matters — the
fix for the first is code, and the fix for the rest is documentation.

### Gap 1 — `inventory-dom01-newmodelbuilder.txt` = 1, expected 0. GENUINE. Fix the code.

```
services/atlas-quest/atlas.com/quest/quest/builder.go:11:type modelBuilder struct {
services/atlas-quest/atlas.com/quest/quest/builder.go:22:func NewModelBuilder() *modelBuilder {
```

**This is a missed FR-14 rename, not a guarded skip.** `design.md` §4.3 step 3 skips a package to
hand work only when an object named `builder` already exists in the target scope. Checked
non-recursively in package `quest/quest` itself: no `type builder`, no `type Builder`, no
`func NewBuilder`. The only hit under that path is
`services/atlas-quest/atlas.com/quest/quest/progress/builder.go:5` — the **`progress` subpackage**,
a different package scope, which cannot collide. The guard had no reason to fire here.

Blast radius is small and fully contained — repo-wide, `NewModelBuilder`/`modelBuilder` appears in
exactly two files, both in this package:

```
15 services/atlas-quest/atlas.com/quest/quest/builder.go
14 services/atlas-quest/atlas.com/quest/quest/builder_test.go
```

No cross-service caller, despite `NewModelBuilder` being exported. Rename `modelBuilder` → `builder`
and `NewModelBuilder` → `NewBuilder`. **This is the branch's only Go change since `fdd466783`, so it
re-arms the gate** — Step 3's flagless `tools/verify.sh` now has something real to check.

### Gap 2 — `inventory-dom04-has-model.txt` has 6 rows not in `exemptions.md`. ALL SIX ARE DETECTOR ARTIFACTS.

Two distinct detector limitations, three packages each. Neither is a conformance gap, and **no code
should change for either** — `sweep.sh`/`split-by-model.sh` are the measuring instrument and must not
be edited to make a count come out right.

**(a) `split-by-model.sh` greps the package directory RECURSIVELY.** Its
`grep -rqs --include='*.go' '^type Model ' "$(dirname "$f")"` walks subdirectories, so a *subpackage's*
`type Model` misfiles the parent into `has-model`. Confirmed non-recursively — none of these three
declares `type Model` in its own package:

| flagged `rest.go` | subpackage that actually holds the `type Model` |
|---|---|
| `services/atlas-consumables/atlas.com/consumables/monster/rest.go` | `monster/drop/position/model.go:3` |
| `services/atlas-data/atlas.com/data/skill/rest.go` | `skill/effect/model.go:16` |
| `services/atlas-maps/atlas.com/maps/character/rest.go` | `character/location/model.go:15` |

`grep -n '^type Model ' <pkg>/*.go` over all three returns nothing. They belong to the already-ruled
"no domain `Model`" class — the same disposition as the other 171.

**(b) `sweep.sh` greps only the `rest.go` file for `^func Transform(`.** These three have both a
domain `Model` and a conforming `func Transform(m Model) (RestModel, error)` — it simply lives in
another file in the same package. DOM-04 is **satisfied**, not exempt:

| package | `type Model` | `Transform` actually at |
|---|---|---|
| `services/atlas-dragons/atlas.com/dragons/dragon` | `model.go:18` | `resource.go:41` |
| `services/atlas-storage/atlas.com/storage/asset` | `model.go:14` | `processor.go:100` (+ `TransformAll` `:138`) |
| `services/atlas-summons/atlas.com/summons/summon` | `model.go:25` | `resource.go:44` |

Note the shape: in all three the `RestModel` is declared in `resource.go` rather than `rest.go`, so
the transform sits beside its `RestModel`. That is a file-placement convention the DOM-04 detector
does not model — the same structural blind spot as the displaced-`Extract` cases Task 16 handled.

**Disposition: document all 6 in `exemptions.md` as a new "DOM-04 — detector artifacts" section**,
split into the two causes above with this evidence. The (b) three must be worded as *satisfied
elsewhere*, never as "exempt" — they conform.

## Task 27 — sweep re-run after the Step 1 fixes

Renamed `modelBuilder` → `builder` and `NewModelBuilder` → `NewBuilder` in
`services/atlas-quest/atlas.com/quest/quest/builder.go` and `builder_test.go` (the only two files
outside this package that reference the identifiers, and both are in this package). Documented the
six DOM-04 detector artifacts in `exemptions.md` under a new `## DOM-04 — detector artifacts`
section, split into (a) recursive-directory false positive (exempt, 3 packages) and (b) displaced
`Transform` (satisfied elsewhere, not exempt, 3 packages).

Re-ran `sweep.sh` then `split-by-model.sh` (unmodified) from the worktree root:

```
$ wc -l docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-newmodelbuilder.txt docs/tasks/task-263-backend-guideline-conformance/inventory-file05-builders.txt docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-has-model.txt docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-no-model.txt
0 docs/tasks/task-263-backend-guideline-conformance/inventory-dom01-newmodelbuilder.txt
5 docs/tasks/task-263-backend-guideline-conformance/inventory-file05-builders.txt
13 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-has-model.txt
176 docs/tasks/task-263-backend-guideline-conformance/inventory-dom04-no-model.txt
194 total
```

`inventory-dom01-newmodelbuilder.txt` = 0 (expected). `inventory-file05-builders.txt` = 5, all
documented in `exemptions.md`'s `## FILE-05 — entity builders` / `## FILE-05 — excluded tree`
sections. `inventory-dom04-has-model.txt` = 13 rows: the 7 pre-existing rows already documented in
`exemptions.md` (`cashshop/rewardpool`, `channel/data/tradeability`, `inventory/data/tradeability`,
`messengers/character`, `npc/conversation`, `parties/character`, `tenants/configuration`) plus the 6
new rows just added under `## DOM-04 — detector artifacts` (`consumables/monster`, `data/skill`,
`maps/character` as (a) exempt; `dragons/dragon`, `storage/asset`, `summons/summon` as (b) satisfied
elsewhere). All 13 accounted for.

`cd services/atlas-quest/atlas.com/quest && go build ./... && go test ./...` — build clean, all
packages `ok` or `[no test files]`.

## Task 27 Step 4 — backend-guidelines-reviewer: CHANGES_REQUIRED (2 blocking)

Artifact: `reviews/audit.md` + `audit.json`. Scope: `main..HEAD`, all 339 changed Go packages,
DOM-01/DOM-04/FILE-05 only — exhaustive re-derivation, not sampling.

**Both blocking findings are `sweep.sh` regex blind spots, which is exactly what this reviewer was
dispatched to find.** The branch's own instrument reported all three rows closed; it was wrong twice,
and neither miss is visible to any grep the branch runs on itself. Both confirmed directly by the
controller before routing.

| finding | why `sweep.sh` missed it |
|---|---|
| `services/atlas-cashshop/atlas.com/cashshop/asset/model.go:106,116` — `type Builder[E any] struct` / `func NewBuilder[E any](...)`, the package's **primary** domain builder, left in `model.go` | the detector is `^type [A-Za-z0-9_]*Builder struct` — a **generic type parameter list** (`Builder[E any] struct`) does not match |
| `services/atlas-channel/atlas.com/channel/account/model.go:52,69` — `type builder struct` / `func NewBuilder() *builder`, and the package has **no `builder.go` at all** | the detector requires `...Builder`; a **lowercase** `builder` does not match |

The `cashshop/asset` miss is the sharper one: that package **already has a `builder.go`** holding all
seven sibling builders Task 22 relocated (`builder.go:9,247,490,530,563,596,643`). The primary
builder was left behind in `model.go` while its siblings moved. `exemptions.md`'s DOM-01
sibling-builder entry even records the fact in passing — "the package's own `NewBuilder` stayed in
`model.go`" — but disposes only of the DOM-01 duplication question and never the FILE-05 *placement*
question. **A fact noted in an exemptions document is not a disposition.** That sentence becomes
false once the builder moves and must be corrected.

**Ruling: fix both, do not exempt either.** FILE-05 is one of the three rows this branch exists to
close, and neither package has any of the structural reasons (`design.md` §4.3 guard, entity builder,
excluded tree) that justify the existing exemptions. **`sweep.sh` must NOT be widened to catch them**
— it is the measuring instrument, and every count on this branch was derived with the current regex;
changing it now would invalidate all of them. The blind spot is recorded here instead.

### Not routed — the 144-package observation is pre-existing debt, and the reviewer was right to say so

The audit also found 144 changed packages with a `model.go` and no builder pattern anywhere — a
literal DOM-01 trigger-fires condition. It checked a 12-package sample **byte-identical against
`main`** and confirmed all 144 are absent from every task-263 target inventory
(`classify-dom01-fr15.tsv`, `inventory-dom01-*.txt`, `classify-file05.tsv`). That is repo debt this
branch neither created nor scoped, per PRD §7. **Non-blocking, not routed.** Recording it because it
is a real finding that a future DOM-01 task should start from.

DOM-04 came back clean: all 13 packages failing the literal `func Transform(` grep are disposed of in
`exemptions.md`, and a 15-package sample of the 172 newly-added `Transform` functions applied the
Task-16 lossy-field policy consistently.

## Gate incident — the flagless run was voided and had to be abandoned

The `task-verifier` dispatched for Step 3 returned `ERROR` after ~16 minutes without an exit code.
Inspection showed **several concurrent `tools/verify.sh` processes** alive at once, all writing the
same log (`.gatelogs/gate-task27-final.log`, 1764 lines, mid-`atlas-query-aggregator`). The agent had
re-launched the gate rather than waiting on the first run. **Concurrent gates writing one log cannot
produce a trustworthy verdict**, so the run was terminated and discarded rather than salvaged.

No verdict is claimed from it. It was superseded regardless: the two FILE-05 fixes above are Go
changes, so the flagless gate has to run again afterward.

**Lesson for the re-run — do not dispatch `task-verifier` for the flagless gate on this branch.** The
full bake plus `-race` across 91 modules runs long enough that an agent polling it will either time
out or restart it. Launch it directly with `run_in_background: true` and reconcile on the completion
notification.

**Last validly gated commit remains `fdd466783`** (`--quick`, `b7f3e8d90..fdd466783`). The flagless
`tools/verify.sh` has still never completed on this branch.

## Task 27 Step 4 fix — complete (`ed4fbb94d`), both blocking findings cleared

`Builder[E any]` + `NewBuilder[E any]` and all their methods relocated from
`cashshop/asset/model.go` into that package's existing `builder.go`; `type builder` /
`func NewBuilder()` and their methods moved out of `channel/account/model.go` into a newly created
`channel/account/builder.go`. Both are pure relocations — generic type parameters preserved, the
lowercase `builder` name preserved, no signature, field, or `Build()` validation rule touched, per
FR-17. `exemptions.md`'s now-false "the package's own `NewBuilder` stayed in `model.go`" sentence was
corrected to cite the new location.

Tests: `atlas-cashshop` 279/279 across 56 packages, `atlas-channel` 2426/2426 across 314 packages,
`gofmt -l` clean on all four touched files.

`inventory-file05-builders.txt` still reads **5**, unchanged, and neither fixed package appears —
which is correct and is the finding restated: the detector never could see them. That is why these
two had to be found by a reviewer reading Go semantics rather than by the branch's own grep.

### `sweep.sh` now aborts early — expected, and a consequence of SUCCESS

The implementer hit and correctly diagnosed this (confirmed with `bash -x`; it did not touch the
script). `sweep.sh` runs under `set -euo pipefail`, and section C is:

```
grep -rn --include='*.go' '^func NewModelBuilder(' services libs | ... > inventory-dom01-newmodelbuilder.txt
```

**`grep` exits 1 when it matches nothing, so the script aborts exactly when acceptance criterion 1 is
met.** Closing the DOM-01 row is the condition that breaks the instrument measuring it. Consequences:

- `inventory-dom01-newmodelbuilder.txt` **is** written (empty, 0 rows) before the abort — the
  criterion-1 evidence is valid.
- `inventory-dom01-other.txt` and the trailing `wc -l` summary **do not run**. Any future "the sweep
  printed its summary" expectation will not be met, and the exit-1 is not a failure signal.

Pre-existing script behavior, not caused by this fix. Not repaired: `sweep.sh` is the measuring
instrument and every count on this branch was derived with it as-is.

## Remaining work — Task 27 Steps 5-8

State at handoff. HEAD `ed4fbb94d`, branch `task-263-backend-guideline-conformance`, tree clean.

1. **The flagless `tools/verify.sh` has still never completed on this branch.** Launch it directly
   with `run_in_background: true`, redirecting to `.gatelogs/`. **Do NOT dispatch `task-verifier` for
   it** — the full bake plus `-race` over 91 modules runs long enough that the agent restarted it and
   left several concurrent runs writing one log, which voided the previous attempt. Reconcile from
   the log on the completion notification. Exit 0 is required before Step 5.
2. **Nothing may edit the working tree while that gate runs.** `tools/verify.sh` reads the *working
   tree*, not a commit range. Reviewers are read-only and may overlap; implementers may not. This has
   already voided one full run on this branch.
3. **Then Steps 5-8** (`.superpowers/sdd/plan/task-27-brief.md`): `git rm` the scaffolding —
   `codemod/`, `sweep.sh`, `split-by-model.sh`, `inventory-*.txt`, `classify-*.tsv`, `ledger-*.tsv`,
   `handwork-dom04.tsv`, `handwork-notes.md`, `fr15-targets.tsv`. Step 6 expects exactly
   `context.md`, `design.md`, `exemptions.md`, `plan.md`, `prd.md`, `progress.md` to remain —
   note `agent-ledger.tsv` and the `review-task-*.md` / `reviews/` artifacts are also present and are
   durable; do not delete them.
4. **Re-run the flagless gate after the deletion** (Step 7). It removes only files outside
   `services/` and `libs/`, so it must still exit 0.
5. Then `superpowers:finishing-a-development-branch`, and `plan-adherence-reviewer` sharded by task
   range (1-10 / 11-20 / 21-27) — never a range shard alongside an unscoped run of the same agent.

The brief names `atlas-verifier` and `atlas-reviewer` for Steps 3-4; **neither agent type exists in
this repo.** The equivalents are `task-verifier` (but see item 1) and `backend-guidelines-reviewer`,
which has already run and returned CHANGES_REQUIRED → both findings now fixed above.

## Task 27 Step 3 — flagless gate PASS at `ea15a4477` (first clean full gate on this branch)

`tools/verify.sh` (no flags) launched directly with `run_in_background: true` — **not** via
`task-verifier`, per the prior handoff ruling — logging to `/tmp/task263-gatelogs/gate-final-flagless.log`.
Exit code **0**, 2463 lines, zero FAIL/ERROR lines.

- `go build/vet/test -race` green across all **91** modules.
- All guards green: go analyzer, skill/job id, scope, producer seam, service registration,
  toolchain pin, operator cancel path, env domain, lint & format (91 modules).
- `docker buildx bake` was **skipped by the gate's own selection** (`− docker buildx bake (no go.mod
  touched)`), not by a flag. This was a flagless run; the CLAUDE.md bar is a flagless exit 0, met.

Before launching, the untracked `.gatelogs/` directory was moved out of the worktree to
`/tmp/task263-gatelogs/` so the tree was fully clean — `tools/verify.sh` reads the working tree, not
a commit range. Nothing edited the tree while it ran.

**Last validly gated commit is now `ea15a4477`.**

## Task 27 Steps 5-6 — scaffolding removed

`git rm`'d from the task folder: `codemod/` (12 tracked files incl. its `go.mod`/`go.sum`),
`sweep.sh`, `split-by-model.sh`, the 8 `inventory-*.txt`, the 6 `classify-*.tsv`,
`dom01-type-targets.tsv`, `fr15-targets.tsv`, the 10 `ledger-*.tsv`, `handwork-dom04.tsv`, and
`handwork-notes.md` — 45 staged deletions. The gitignored 11.9 MB `atlas-task-263-codemod` binary was
deleted from disk too (it would otherwise have surfaced as untracked once `codemod/.gitignore` went).

Confirmed by grep before deleting: **nothing outside the task folder referenced `codemod/`** — it is
absent from `go.work` and `go.work.sum`, so its removal cannot change the 91-module selection.

Durable artifacts deliberately **kept**, correcting the brief's Step 6 "exactly six files" wording:
`context.md`, `design.md`, `exemptions.md`, `plan.md`, `prd.md`, `progress.md` **plus**
`agent-ledger.tsv`, `reviews/audit.{md,json}`, and the 47 `review-task-*.md` / `task-*-report.md` /
`task-*-review.md` evidence files. Those record verdicts, not scaffolding.

## Task 27 Step 7 — post-strip flagless gate PASS at `f8de7249c`

`tools/verify.sh` (no flags) re-run after the scaffolding deletion. Exit **0**, 2460 lines, zero
FAIL/ERROR lines, all 91 modules and all guards green. Log: `/tmp/task263-gatelogs/gate-poststrip.log`.

**Last validly gated commit is now `f8de7249c`.**

## Task 27 Step 8 — plan adherence, all 27 tasks, zero blocking findings

`plan-adherence-reviewer` (sonnet) dispatched as **three non-overlapping range shards** — 1-10,
11-20, 21-27 — with no unscoped run alongside them. They ran concurrently with the Step 7 gate and
were instructed read-only, writing outside the worktree, because `tools/verify.sh` reads the working
tree and a concurrent edit already voided one full run on this branch.

| Shard | Verdict | Blocking | Artifact |
|---|---|---|---|
| Tasks 1-10 (+ inserted 9a) | APPROVED | 0 | `adherence-tasks-1-10.md` |
| Tasks 11-20 | APPROVED | 0 | `adherence-tasks-11-20.md` |
| Tasks 21-27 | APPROVED_WITH_FINDINGS | 0 | `adherence-tasks-21-27.md` |

Every plan task is IMPLEMENTED. **Zero blocking findings across the whole plan.**

The 21-27 shard returned its findings inline rather than writing its artifact; the controller
transcribed them into `adherence-tasks-21-27.md` so that range is not left without a record. That
file is marked as a transcription — the other two are verbatim agent output.

### Non-blocking items to carry into the PR description

1. **Three pre-existing `Extract`-drops-field defects** surfaced by Task 18 and correctly left
   unfixed as out-of-scope behavior changes — they are **not introduced by this branch**:
   `atlas-channel/character` and `atlas-cashshop/character` drop `spawnPoint`;
   `atlas-npc-shops/character` drops `x`/`y`/`stance`. Recording them here so they survive the
   branch rather than being lost with the review artifacts.
2. `model.go:444`'s `CraftActionBuilder` mention — **disposition: no change**, the reference is
   correct prose, not stale. See `adherence-tasks-21-27.md`.
3. Plan Task 27 Step 4's second reviewer dispatch named a nonexistent agent type
   (`atlas-reviewer`); the gap is closed by the plan-adherence audit above.
4. The FILE-05 dry-run group count (73 actual vs the design's ~72 estimate) was deferred from
   Task 19 to Task 21-C; the 21-27 shard audited Task 21 as IMPLEMENTED and raised no count
   discrepancy, so it is treated as settled.

## Branch verified at HEAD `87e11d9fe` — final flagless gate PASS

Exit **0**, 2460 lines, zero FAIL/ERROR, all 91 modules with `-race`, all guards green.
Log: `/tmp/task263-gatelogs/gate-head.log`.

### Why a third full gate was run

A `task-verifier` dispatched in the **previous** session — the one whose ERROR is recorded above —
was still alive and only stopped after 71 minutes, during this session. It had three of its own
`tools/verify.sh` runs going (all died without an exit code) while the Step 3 and Step 7 gates ran.
It then reported that "the worktree changed underneath me" and that `87e11d9fe`'s gate claim needed
re-verification.

Its own self-correction is accepted (it had fabricated a SIGSYS diagnosis for exit 144 and retracted
it). Its inference about the Step 3 / Step 7 gates is not:

- The two commits that landed mid-flight were this session's own, committed **before** each gate was
  launched, not during. The concurrent reviewers were read-only by construction.
- Build-cache contention between concurrent `verify.sh` runs produces failures, not false passes.
  `All checks passed` plus exit 0 means every step returned 0 on the tree that run read.

What *was* a real gap: three `verify.sh` processes were alive on this worktree unknown to the
controller, which is the same condition that voided the original Step 3 gate. So rather than defend
the earlier verdicts, the gate was re-run at HEAD as the **sole** `verify.sh` on a clean, settled
tree. That run is the authoritative verdict for the branch; the Step 3 and Step 7 PASSes are
corroborating, not load-bearing.

**Branch state: all 27 plan tasks implemented, plan adherence APPROVED with zero blocking findings
across three shards, backend guidelines audited with both findings fixed, flagless gate green at
HEAD, tree clean. Ready for code review and PR.**

## Post-merge gate — FAIL, fixed, then PASS at `0f98f9b0c`

Merged `origin/main` twice (the second time picking up `92cb7e4dd`, deploy-metadata only, clean) so
the gate would cover the final tree. The first flagless gate on the merged tree `cb9d04644`
**FAILED**, exit 1 — one check out of 91 modules plus all guards
(`/tmp/task263-gatelogs/gate-merge.log:1515`):

```
--- FAIL: TestTransformRoundTrip (0.00s)
    rest_test.go:285: round trip lost data:
         got {id:11 leaderId:0 members:[]}
        want {id:11 leaderId:0 members:[]}
FAIL	atlas-monster-death/party	0.036s
```

`got` and `want` print identically because the difference was nil-vs-empty slice: main's `Extract`
normalizes to `make([]MemberModel, 0)` while the test's `Model{id: 11}` left `members == nil`.

### The provenance call

This looked like a fourth instance of the pre-existing `Extract`-drops-field class the branch
deliberately left unfixed, which would have argued for weakening the test. It is not.
`git log --follow` and `git show ce26cb43a:` establish:

- `Extract` / `ExtractMember` / `MemberRestModel` are **main's pre-existing code, unchanged here**.
- `Transform` and `TestTransformRoundTrip` are **this branch's own additions**, commit `81922b270`.

The add/add merge resolution recorded in `HANDOFF-pr.md` kept both sides, pairing our new `Transform`
with main's `Extract` for the first time. Our `Transform` returned only `RestModel{Id: m.id}`,
silently dropping `leaderId` and `members` — our defect, and producible work, so it was fixed rather
than deferred.

### Resolution

`21bdd9fec` completes `Transform` into a true inverse of `Extract` and adds `TransformMember`
(decomposing `field.Model` via its real accessors), and strengthens `TestTransformRoundTrip` to carry
a populated `leaderId` and two members. `Extract`/`ExtractMember` untouched; the three deferred
pre-existing defects untouched.

`task-reviewer` over `21bdd9fec`: **APPROVED**, 0 blocking, 0 non-blocking. It confirmed the inverse
is field-for-field, the `field.Model` accessors are real (checked against
`libs/atlas-constants/field/model.go`), the test is non-vacuous, and `Transform`/`TransformMember`
have no other callers whose tests could pin the old behavior. Artifact was written to
`/tmp/task263-pr/review-gate-merge-fix.md` — outside the worktree, because a gate was reading the
working tree at the time.

### Authoritative verdict

Flagless `tools/verify.sh` at **`0f98f9b0c`** — exit **0**, `All checks passed`, zero FAIL/ERROR, all
91 modules with `-race`, all guards green, tree clean at launch and the sole `verify.sh` running.
Log: `/tmp/task263-gatelogs/gate-merge2.log`.

**This supersedes the `87e11d9fe` verdict as the branch's load-bearing gate** — it is the first and
only flagless run on the merged tree.
