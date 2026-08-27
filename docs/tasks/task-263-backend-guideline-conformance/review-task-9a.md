# Review: Task 9a — DOM-01 type rename (FR-13 scope gap)

Commit range: `59062ab04..01e2cc2c3` (16 commits: 15 per-service `refactor(<service>): rename
ModelBuilder type to Builder` commits + `01e2cc2c3` ledger/artifact commit).

Verdict: **APPROVED**

## 1. `targetsFromTSV` change (additive default-pairs directive)

`docs/tasks/task-263-backend-guideline-conformance/codemod/rename.go` (+38/-2) adds a branch at
the top of `targetsFromTSV`'s per-line loop: for a 2-field or 3-field row whose non-pkgDir fields
are all blank, the pkgDir is registered in `order`/`seen` and the loop `continue`s — it never
enters `pairsByDir`. The later target-construction loop then produces `Target{PkgDir: pkgDir,
Pairs: pairs}` with `pairs == nil` for that pkgDir (`rename.go` around line 472-478), and
`renameImpl`'s existing `if len(t.Pairs) > 0 { pairs = t.Pairs }` (line 106) naturally falls back
to the fixed `renamePairs` set — exactly the mechanism the brief and controller note describe.

- Triples path unchanged: the 3-field-with-content branch is reached only when the new
  "all-blank" check is false, and is otherwise byte-identical to the prior code (just wrapped in
  a `seen[pkgDir]` bool map instead of `pairsByDir[pkgDir]` presence check — same semantics).
  Confirmed by re-reading `rename.go:425-478` and by the pre-existing
  `TestTargetsFromTSV/well-formed_triples_merged_per_pkgDir` and
  `TestTargetsFromTSV/malformed_row_errors` subtests, both untouched and still passing (`go test
  ./...` in the codemod module: `ok atlas-task-263-codemod`).
- `renamePairs` (`rename.go:52`) is untouched and remains the fallback.
- The R2 collision guard (`rename.go:93-101`, `"builder identifier already in scope"`) is
  byte-identical — confirmed by reading `renameImpl` in full.
- Ledger emits exactly one row per input pkgDir: `NewLedger("rename", input)` where `input` is
  built 1:1 from `targets` (`rename.go:78-82`), and every target (whether a triples row or a
  default-pairs row) reaches exactly one of `ledger.Skipped`/`ledger.Applied`. Verified against
  the actual `ledger-rename-type.tsv` (39 rows: 37 APPLIED + 2 SKIPPED, matching the 39-line
  inventory 1:1, no duplicates).
- Covering test: `TestTargetsFromTSV/empty-stem_row_falls_back_to_renamePairs`
  (`rename_test.go:409-427`) exercises both the 2-field (`"services/a/drop\t"`) and 3-field
  (`"services/b/drop\t\t"`) forms and asserts `len(target.Pairs) == 0` for both. This genuinely
  covers the new code path (it fails against the pre-change code, which would error on either
  row shape per the old `len(fields) != 3` / `empty field` checks the report quotes from
  `git show 2b07d2776:...`).

## 2. The two SKIPs

**`services/atlas-quest/atlas.com/quest/quest`** — `git diff 59062ab04..01e2cc2c3 --
services/atlas-quest` is empty; the service is genuinely untouched. `dom01-type-targets.tsv:36`
still lists `services/atlas-quest/atlas.com/quest/quest` (not hand-excluded — left to the guard),
and the collision is real: `quest/builder.go:11` declares `type modelBuilder struct`, while
`quest/processor.go:851` and `:921` each declare a local `builder := sagaproducer.NewBuilder(...)`
— exactly the `modelBuilder` + `builder`-in-scope collision the guard checks for
(`pkg.Types.Scope().Lookup("modelBuilder") != nil && hasObjectNamed(pkg, "builder")`). Ledger row
(`ledger-rename-type.tsv`) reads `SKIPPED  builder identifier already in scope`, verbatim from
the codemod's own message at `rename.go:100`.

**`services/atlas-character/atlas.com/character/character`** — excluded from
`dom01-type-targets.tsv` by hand (not present in the 38-row file, though present in the 39-line
raw inventory). The report's claim that a dry run against the full 39-row list would have
*applied* here (because the guard only checks `modelBuilder`/`builder`, not the pre-existing
`Builder`/`NewBuilder`) is consistent with reading the guard code: it only calls
`scope.Lookup("modelBuilder")` and `hasObjectNamed(pkg, "builder")` — a distinctly-cased
`Builder`/`NewBuilder` pair does not trip either check, so nothing would have stopped the rename
from firing. The Task 5 ruling is intact in the commit range: `git diff 59062ab04..01e2cc2c3 --
services/atlas-character` shows zero changes under `character/character/`, and
`character/character/model.go:211,244,248` still declare `type modelBuilder`, `func
NewEmptyBuilder() *modelBuilder`, and `func CloneModel(m Model) *modelBuilder` unchanged. The
ledger row for this pkgDir is `SKIPPED  Task 5 exemption: type modelBuilder and CloneModel
deliberately left unrenamed alongside the pre-existing type Builder/NewBuilder (see task-5
report; controller resolution R3)` — text that is honest about being a hand-written / pre-emptive
exclusion rather than a codemod-detected collision, and cites its authority.

**`atlas-character`'s other three declarations** (`teleport_rock`, `saved_location`,
`pending_change`) carry no exemption and are renamed in commit `a0e318ce9`: `git diff
59062ab04..01e2cc2c3 -- .../teleport_rock/builder.go .../saved_location/builder.go
.../pending_change/builder.go` shows `type modelBuilder`→`type builder`,
`NewBuilder()`→returns `*builder`, and every method receiver/return type updated consistently.
`go build ./...` for `services/atlas-character/atlas.com/character` passes clean from the module
root (workspace mode).

## 3. Rename completeness per service

Full residue re-check (identical grep to Step 1/Step 6 of the brief):

```
grep -rn --include='*.go' -E 'type (M|m)odelBuilder' services libs | grep -v 'libs/atlas-packet/'
services/atlas-quest/atlas.com/quest/quest/builder.go:11:type modelBuilder struct {
services/atlas-character/atlas.com/character/character/model.go:211:type modelBuilder struct {
```

Exactly the two recorded SKIPs — no forgotten declaration. `dom01-type-targets.tsv` (38 rows)
and `ledger-rename-type.tsv` (39 rows: 37 APPLIED + 2 SKIPPED) both check out against the 39-line
`inventory-dom01-modelbuilder-type.txt`, and per-service declaration counts in the inventory
(cashshop 6, npc-shops 4, character 4, query-aggregator 3, pets 3, merchant 3, login 3, inventory
3, consumables 3, one each for channel/doors/drops/monsters/quest/storage/summons) match the
brief's numbers exactly.

Each per-service commit diff was inspected (`git show --stat`) and cross-package call sites
(e.g. `inventory/asset/processor.go`, `inventory/asset/processor_test.go` inside the
`atlas-inventory` commit; `consumables/equipable/processor.go`,
`consumables/equipable/producer.go` inside the `atlas-consumables` commit) are consistent
`*ModelBuilder`→`*Builder` identifier updates from the same codemod run, not separate hand edits.
Hand-fixed doc comments (`cashshop/inventory/model.go`, `cashshop/inventory/compartment/model.go`,
`npc-shops/commodities/builder.go`, `npc-shops/shops/builder.go`,
`monsters/monster/builder.go`) were diffed line-by-line: every changed line is a `//`-prefixed
comment, no identifier text was touched by the `sed` fixups.

**FR-15 substring-exception check:** `grep -rln "NewSharedVesselModelBuilder\|
NewTripScheduleModelBuilder\|RewardModelBuilder" services libs` still finds all three untouched
occurrences (`atlas-channel/transport/route/builder.go`,
`atlas-consumables/data/consumable/model.go`, `atlas-consumables/consumable/reward_test.go`) —
confirming the rename stayed `types.Object`-identity based and did not textually match these
`ModelBuilder`-containing-but-different identifiers. The commit-range diff for those three files
was also confirmed empty in the range's diffstat.

Build/vet/test sampled across several touched modules (workspace mode, from each service's
module root): `atlas-consumables`, `atlas-character`, `atlas-quest` (untouched, sanity build),
`atlas-monsters` (`go build && go vet` clean), `atlas-inventory/asset` and
`atlas-monsters/monster` test packages (`go test` — both `ok`). No failures.

## 4. FR-17 — no JSON tag / REST route / Kafka topic literal changed

`git diff 59062ab04..01e2cc2c3 -- 'services/*' | grep -iE 'json:"|route|topic|kafka'` returns
exactly one line, which is an unchanged context line (`model.Provider[[]kafka.Message]` in the
consumables diff, not a `+`/`-` line). No struct tag, route string, or topic literal appears
anywhere in the range's diff.

## 5. FR-18 — `libs/atlas-packet/` untouched

`git diff --stat 59062ab04..01e2cc2c3 | grep atlas-packet` returns nothing. Confirmed untouched.

## 6. Commit hygiene

All 15 per-service commits were inspected with `git show --stat --format=""`; each stages files
under exactly one `services/<S>/` tree (cross-package files within the same service are expected
and present, e.g. `asset/processor.go` + `asset/processor_test.go` alongside `asset/builder.go`
in the `atlas-inventory` commit). No commit reaches outside its named service. `01e2cc2c3` stages
only the codemod fix, its test, the two new TSVs, and the `progress.md` append — matching the
brief's Step 5. `git status --porcelain` at HEAD is clean; no stray untracked artifact from this
task's work.

## Not evaluable

- The report's "Concerns for the controller" section flags a `progress.md`/`agent-ledger.tsv`
  discrepancy about a prior "Task 9 review" section and a claim about `rename.go`'s pre-change
  state that the implementer says does not match `git show 2b07d2776:...`. That is a
  process/reconciliation question about a different task's artifact, not something this review
  can adjudicate from the 9a diff alone — flagged here for the controller, not scored as a
  finding against this unit.
- `review-task-9.md` appearing as untracked in the report's working tree is outside this task's
  scope (Task 9, not 9a) and is not part of the reviewed range.

## Summary

- No blocking findings.
- No non-blocking findings — every check in scope (targets-format additivity, both SKIP
  dispositions, per-service completeness, FR-15/17/18, commit hygiene) is verified with
  file:line/command evidence above.
