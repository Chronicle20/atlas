# Task 7 report — W2: apply the rename to `atlas-channel`

## What I implemented

Ran the Task 6 codemod's `rename` subcommand scoped to `services/atlas-channel`,
then hand-fixed the doc comments and test names left stale by the identifier-only
rename, following the precedent set by Task 4's hand-rename of `channel/asset`.

### Step 1 — Apply

The literal command in the brief (`go run ./docs/.../codemod rename ...` from the
worktree root) fails: the worktree root has no `go.mod`, only a `go.work`, so
`GOWORK=off` from the root can't resolve a main module (same issue task-6's report
documented). Ran it the way Task 6 did — from inside the codemod module directory,
with an absolute `-repo` path:

```
$ cd docs/tasks/task-263-backend-guideline-conformance/codemod && GOWORK=off go run . rename \
    -repo <worktree root> -only services/atlas-channel \
    -ledger <worktree root>/docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv
(no output, no error)
```

### Step 2 — Ledger check

```
$ cut -f2 docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv | sort | uniq -c
     19 APPLIED
```
19 of 19 APPLIED, 0 SKIPPED — matches the brief's count for atlas-channel exactly.
No hand resolution needed, nothing to record in progress.md.

### Step 3 — Confirm no `NewModelBuilder`/`ModelBuilder` survives

The codemod itself only renames identifiers the Go type system resolves (FR-12/13
scope); it does not touch comment text or `_test.go` function names. A first grep
after Step 1 alone still showed ~30 stale hits: doc comments ("// NewModelBuilder
creates a new builder instance") left over the now-renamed `NewBuilder`/`Builder`
declarations, `TestNewModelBuilder`/`TestModelBuilder_SetMaxMp` test names, and a
couple of `t.Fatalf`/comment strings in `monstermagnet_test.go` and `chakra_test.go`
naming the now-renamed `monster.NewBuilder`/`character.NewBuilder`. Task 4's
`492ad9bf4` set the precedent for updating exactly this kind of accompanying text
(it renamed `TestNewModelBuilder` -> `TestNewBuilderWithId` and the doc comment in
the same commit), so I did the same here — updated 15 doc comments/test names and
2 stray `t.Fatalf`/comment strings across the touched files by hand (`sed`, one
file per invocation).

Two categories of `ModelBuilder` hits remain by design, both correctly excluded
from FR-12/13 scope:
- `skill/handler/mistcast/mistcast.go:42` — a comment describing **atlas-monsters'**
  `ModelBuilder.AddStatusEffect` (a different service, out of this task's `-only
  services/atlas-channel` scope; not yet renamed, Task 8 territory).
- `transport/route/builder.go` — `NewSharedVesselModelBuilder`/
  `NewTripScheduleModelBuilder`, two distinct-type builders in one package. Per
  PRD FR-15, packages with several builders over distinct types keep
  `New<Type>Builder` names to disambiguate; these are FR-15, not FR-12/13, so the
  codemod correctly left them (they weren't in the 19-entry ledger). The
  co-resident `modelBuilder`/`ModelBuilder` type in that same file (for the
  package's own `Model`) *was* renamed to `builder`/`Builder` — confirmed in the
  diff.

Final check:
```
$ grep -rn --include='*.go' 'NewModelBuilder\|ModelBuilder' services/atlas-channel
services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast.go:42:// atlas-monsters' ModelBuilder.AddStatusEffect REPLACES a same-type POISON on
services/atlas-channel/atlas.com/channel/transport/route/builder.go:165:// NewSharedVesselModelBuilder creates a new builder for SharedVesselModel
services/atlas-channel/atlas.com/channel/transport/route/builder.go:166:func NewSharedVesselModelBuilder() *sharedVesselBuilder {
services/atlas-channel/atlas.com/channel/transport/route/builder.go:174:	return NewSharedVesselModelBuilder()
services/atlas-channel/atlas.com/channel/transport/route/builder.go:224:// NewTripScheduleModelBuilder creates a new builder for TripScheduleModel
services/atlas-channel/atlas.com/channel/transport/route/builder.go:225:func NewTripScheduleModelBuilder() *tripScheduleBuilder {
services/atlas-channel/atlas.com/channel/transport/route/builder.go:245:	return NewTripScheduleModelBuilder()
```
Both categories are the documented, in-scope exceptions above.

### Step 4 — Build, vet, test

From `services/atlas-channel/atlas.com/channel`:
```
$ go build ./...
(no output — clean)
$ go vet ./...
(no output — clean)
$ go test ./...
(all `ok` or `[no test files]`; grep for anything other than `^ok`/`no test files`
returned nothing — pristine, no FAIL)
```
Per the brief, ran the workspace build (not `GOWORK=off go build ./...`); the
known pre-existing `go.sum` gap in this module was not encountered/relevant here.

### Step 5 — Confirm no literal changed (FR-17)

The literal command in the brief has a typo (final `grep -v` lacks `-E`, so the
`|` is treated literally, not as alternation, and the filter does nothing). Ran
it with `-E` added to get the intended filter:
```
$ git diff -U0 -- services/atlas-channel | grep '^[+-]' | grep -E '"|`' | grep -vE 'ModelBuilder|modelBuilder'
(no output)
```
No string/backtick literal content changed — every hit before the fix was an
identifier rename inside an unchanged literal (e.g. `pet.NewModelBuilder(1, ..., "Pet")` -> `pet.NewBuilder(1, ..., "Pet")`).

### Step 6 — Commit

```
git add services/atlas-channel docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv
git commit -m "refactor(atlas-channel): rename ModelBuilder to Builder" (+body)
```
Landed at `fdcac2dbd`.

## Files changed

- `services/atlas-channel/atlas.com/channel/**` — 81 files touched by the codemod
  plus the hand comment/test-name fixups (full list in the commit diff; largest
  clusters: `asset/`, `cashshop/**`, `character/`, `monster/`, `pet/`, `reactor/`,
  `npc/shops/**`, `drop/`, `channel/`, `compartment/`, `inventory/`, `note/`,
  `transport/route/`, `world/`, plus call sites across `kafka/consumer/**`,
  `socket/**`, `skill/handler/**`, `movement/`, `party/hpsync/`, `respawn/`,
  `pointreset/`).
- `docs/tasks/task-263-backend-guideline-conformance/ledger-rename-channel.tsv` — new,
  19 APPLIED entries.

Not staged (per instructions): `docs/tasks/task-263-backend-guideline-conformance/agent-ledger.tsv`
and `progress.md` (both already showed pre-existing local modifications from
prior tasks, not touched by me).

## Self-review

- Diff is 81 files of pure identifier rename + comment/test-name text fixups; no
  struct field, JSON tag, route, or Kafka topic changed (confirmed by Step 5).
- Verified by hand that the two remaining `ModelBuilder` categories are legitimate
  exclusions (cross-service comment reference; FR-15 distinct-type builders in
  `transport/route`), not codemod misses — cross-checked against the PRD's FR-13/
  FR-15 definitions and Task 6's report on the asset-package precedent.
- No `*_testhelpers.go` files added; no new domain types; no invented values.
- Build, vet, and full test suite all clean for the module.

## Concerns

None blocking. Two things worth the controller's awareness for Task 8/25:
1. The brief's Step 5 command has a `grep -v` missing `-E`; I used `-E` to get
   the intended alternation filter and note the typo here rather than silently
   "fixing" the brief.
2. The `atlas-monsters' ModelBuilder.AddStatusEffect` comment in
   `skill/handler/mistcast/mistcast.go` will become stale once Task 8 renames
   atlas-monsters' `ModelBuilder` — worth a follow-up grep in that task.
