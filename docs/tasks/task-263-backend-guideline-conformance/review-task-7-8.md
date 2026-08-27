# Review — Task 7 & Task 8 (commit range `27c5fe431..49106bbcd`)

Reviewer: atlas-reviewer (Sonnet 5)
Scope: `git diff --stat 27c5fe431..49106bbcd`, all 24 commits' full diffs, plus the
two persistent reports (`task-7-report.md`, `task-8-report.md`) as design intent.

## Range under review

24 commits: `fdcac2dbd` (Task 7, atlas-channel), 22 per-service `refactor(atlas-*):
rename ModelBuilder to Builder` commits (Task 8, segments 1+2), and `49106bbcd`
(combined ledger). Verified via `git log --oneline 27c5fe431..49106bbcd`.

## Ruling verification

1. **Exactly 1 remaining `func NewModelBuilder` outside `libs/atlas-packet/`.**
   Confirmed:
   ```
   $ grep -rn --include='*.go' 'func NewModelBuilder' services libs | grep -v 'libs/atlas-packet/'
   services/atlas-quest/atlas.com/quest/quest/builder.go:22:func NewModelBuilder() *modelBuilder {
   ```
   PASS. Verified the stated collision reason is real:
   `services/atlas-quest/atlas.com/quest/quest/processor.go:851` and `:921` bind a
   local `builder := sagaproducer.NewBuilder(...)`, which would collide with a
   package-level `Builder`/`builder` rename in the same package — genuine R2 SKIP,
   not a miss.

2. **`channel/asset` and `character/character` hand-chosen names not regressed.**
   Confirmed:
   ```
   services/atlas-channel/atlas.com/channel/asset/builder.go:127:func NewBuilderWithId(id uint32, ...) *Builder {
   services/atlas-character/atlas.com/character/character/model.go:244:func NewEmptyBuilder() *modelBuilder {
   ```
   PASS — neither was touched in this range (atlas-character isn't even in the
   commit list; `atlas-channel/asset` was in Task 7's ledger as `APPLIED` but the
   diff for that package only renamed the *type* `ModelBuilder`→`Builder`, not
   the already-corrected constructor name).

3. **Bare-`ModelBuilder` type-name population out of scope.** Confirmed present
   and untouched, e.g. `atlas-monsters/monster/builder.go:44`,
   `atlas-npc-shops/npc/shops/builder.go:15`,
   `atlas-npc-shops/npc/commodities/builder.go:14`,
   `atlas-cashshop/cashshop/inventory/{model.go,compartment/model.go}` — all
   `Clone`-based builders with constructor names that were never
   `NewModelBuilder`, hence never in the DOM-01 inventory. Not a defect in this
   range (matches the controller's ruling and both reports' disclosure).

## Structural checks

- **No commit touched paths outside its named service.** Verified via
  `git log --name-only` across the full range (307 lines): every per-service
  commit's files are all under `services/<that-service>/` (plus, for
  `fdcac2dbd`, that commit's own `ledger-rename-channel.tsv`; for `49106bbcd`,
  only `ledger-rename-rest.tsv`). No cross-service staging.
- **Combined ledger matches the per-commit ledgers.** `ledger-rename-rest.tsv`
  has 58 lines, `cut -f2 | sort | uniq -c` → `57 APPLIED / 1 SKIPPED`, exactly
  matching the report and the repo-wide `NewModelBuilder` grep count above.
  atlas-channel contributes 19 entries (asset, cashshop/inventory + 2
  sub-packages, cashshop/item, channel, character, compartment, drop,
  inventory, monster + information, note, npc/shops + commodities, pet,
  reactor, transport/route, world) — matches Task 7's 19/19 APPLIED claim.
- **FR-17 (no literal content change).** `git diff -U0 -- services libs | grep
  '^[+-]' | grep -E '"|`' | grep -vE 'ModelBuilder|modelBuilder'` on the full
  range returns only method-chain call-site lines (`pet.NewBuilder(1, ...,
  "Pet")`, `reactor.NewBuilder(f, 100, "testReactor")`, etc.) and hand-edited
  backtick doc-comment prose — never a JSON tag, route path, or Kafka topic. No
  `json:"`, route, or `Topic` hits in the changed-literal set at all. PASS.
- **Object-identity guarantee held.** The FR-15 distinct-type builders in
  `atlas-channel/transport/route/builder.go`
  (`NewSharedVesselModelBuilder`/`NewTripScheduleModelBuilder`) and
  `atlas-consumables/data/consumable/model.go`'s `RewardModelBuilder`/
  `RewardModelBuilderType` are untouched by the rename despite containing
  `ModelBuilder` as a substring — confirms the codemod matched on resolved
  identifier, not string substring. No `CloneModelBuilder` remains anywhere
  outside `libs/atlas-packet`.
- **Build/vet clean.** Spot-checked module builds for
  atlas-mounts, atlas-quest, atlas-channel, atlas-cashshop, atlas-npc-shops
  (module `npc`), atlas-inventory, atlas-world, atlas-monster-book,
  atlas-monsters, atlas-consumables — all `go build ./...` clean, no output.
  `go test ./...` clean for atlas-quest and atlas-mounts (all `ok`/`[no test
  files]`).
- **Tree hygiene.** `git status --short` shows only
  `docs/tasks/task-263-backend-guideline-conformance/agent-ledger.tsv` and
  `progress.md` modified (expected, controller's bookkeeping, per the task
  brief), plus untracked report/review markdown files from this and prior
  phases — none of those are part of the diff under review.

## Finding — non-blocking

**Stale `TestModelBuilder_*` test-function names left in
`services/atlas-pets/atlas.com/pets/pet/builder_test.go`** (lines 7, 14, 24, 34,
44, 54, 64, 74, 84, 94, 153, 164 — e.g. `TestModelBuilder_Build_Success`,
`TestModelBuilderSetName`). The codemod correctly renamed the identifiers
inside these test bodies (`NewModelBuilder(...)` → `NewBuilder(...)`,
confirmed via `git show dbe0d89c8 -- .../pet/builder_test.go`), but the test
*function names* were left referencing the old `ModelBuilder` name. This is
inconsistent with the hand-fixup convention Task 8 itself applied to every
other service's `builder_test.go` in this same range (atlas-reactors,
atlas-keys, atlas-expressions, atlas-drops, atlas-world, atlas-tenants,
atlas-skills, atlas-quest, atlas-monsters, atlas-rps, atlas-maps — all got
`TestModelBuilder_*` → `TestBuilder_*` / `TestNewModelBuilder` →
`TestNewBuilder` renames). The Task 8 report's table for atlas-pets lists only
`data/cash/builder.go doc comment, pet/resource.go doc comment` as hand
fixups and does not disclose this miss; the report's self-review also claims
"every stale doc-comment/test-name fixup was verified... checked each hit's
context individually," which this instance contradicts.

Not blocking: the stale names don't affect correctness (Go test function
names aren't referenced elsewhere; `go test ./...` for the package is
unaffected either way) and are purely a documentation/naming-hygiene gap
against the convention this same range set for itself. Repo-wide grep
(`grep -rn --include='*_test.go' 'func TestModelBuilder\|func TestNewModelBuilder\|func TestCloneModelBuilder' services libs`)
confirms this is the only in-scope miss; the other hits from that grep
(`atlas-guilds`, `atlas-npc-conversations`) are services outside this range's
22-service inventory and pre-date this work — not this range's responsibility.

## Not evaluable

None. The full diff (24 commits, 259 files per `--stat`) was within slice-first
budget for hunk-level review of the identifier renames; build/vet were run
directly rather than assumed from the reports.

## Verdict rationale

Every controller ruling verified true by direct evidence. No cross-service
staging, no literal/route/topic/JSON-tag drift, no substring over-rename, no
regression of the two hand-chosen names, and the ledger accounting is exact.
The one finding (atlas-pets stale test names) is real but cosmetic and doesn't
undermine the rename's correctness or the ledger's accuracy — `APPROVED_WITH_FINDINGS`.
