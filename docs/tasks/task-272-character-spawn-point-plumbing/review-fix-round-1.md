# Review: fix-round-1 (e2b1723b8..3803625d4)

Reviewer: task-reviewer (Sonnet 5)
Scope: commit range `e2b1723b8..3803625d4`, nine commits closing the six
CHANGES_REQUIRED findings from `review-builder-validation.md` /
`backend-audit/audit-round2.md`, per the controller rulings in
`.superpowers/sdd/plan/fix-round-1-brief.md`.

Diff read via `git show` / `git diff` against the committed range. No tests
run inside `services/atlas-character` (mid-edit by a concurrent fix-round-2
implementer, per instruction); tests were run in the other eight touched
modules.

## Files touched (13, matches the brief's file inventory exactly)

```
services/atlas-cashshop/.../character/builder_test.go        (new)
services/atlas-cashshop/.../character/processor.go
services/atlas-character/.../character/builder_test.go       (new)
services/atlas-consumables/.../character/builder_test.go     (new)
services/atlas-dragons/.../character/builder_test.go         (new)
services/atlas-login/.../character/builder_test.go
services/atlas-messages/.../character/builder_test.go        (new)
services/atlas-messages/.../character/model.go
services/atlas-npc-shops/.../character/builder_test.go       (new)
services/atlas-npc-shops/.../character/processor.go
services/atlas-pets/.../character/builder_test.go            (new)
services/atlas-query-aggregator/.../character/builder_test.go (new)
services/atlas-query-aggregator/.../character/processor.go
```

No file outside this list changed (`git diff e2b1723b8..3803625d4 --name-only`).
`atlas-channel` untouched (correctly — brief noted it already had coverage).
`modelBuilder`/`NewEmptyBuilder()` in atlas-character untouched. No other
builder (`account`, `world`, `coupon`, `compartment`, `drop`, `saga`,
`marriage`, `condition`, pet `Clone`) touched.

## Finding-by-finding

**Finding 1 — atlas-cashshop `InventoryDecorator`** (`character/processor.go:53-56`,
commit `2926b5900`): PASS. `p.l.WithError(err).Errorf("Unable to set
inventory for character [%d].", m.Id())` added immediately before the
`return m` fallback; `p.l` confirmed in scope on `ProcessorImpl`.

**Finding 2 — atlas-npc-shops `InventoryDecorator`** (`character/processor.go:75-78`,
commit `9b70e7e21`): PASS. Same log line added on the `SetInventory`
fallback (the unrelated remote-fetch fallback above it, correctly, was left
alone — outside this finding's scope).

**Finding 3/4 — atlas-query-aggregator `InventoryDecorator`/`GuildDecorator`**
(`character/processor.go:51-54`, `63-66`, commit `9c53062a3`): PASS. Both
fallbacks now log via `p.l.WithError(err).Errorf(...)`. `p.l` confirmed
present on `ProcessorImpl` (`character/processor.go:21`).

**Finding 5 — atlas-messages `SetSkills`** (`character/model.go:237-243`,
commit `d4f3062c4`): PASS. Comment added: "Unreachable in practice: Clone(m)
carries forward m.id, which the Builder already validated as non-zero when m
was constructed." Verified byte-for-byte identical to the wording already
present at `atlas-consumables/.../character/model.go:294` and `:306`
(`SetPets` and a second site). The implementer's report additionally claims
this phrasing is "shared" with atlas-pets' `SetInventory`; that specific
claim doesn't hold — pets' comment there reads differently ("SetInventory's
signature cannot propagate an error…", `atlas-pets/.../model.go:259`). This
is a report-accuracy nit only, not a code defect: the brief required
matching the consumables/pets form, and the added comment matches the
consumables form exactly. Non-blocking.

**Finding 6 — missing invariant tests, all nine services**: PASS for all
nine.
- atlas-cashshop, atlas-npc-shops, atlas-query-aggregator, atlas-messages,
  atlas-consumables, atlas-pets: identical `TestBuild_MissingId` /
  `TestBuild_WithId` pair, asserting `err != nil` on `NewBuilder().Build()`
  and `err == nil` + correct `Id()` once set. Each genuinely pins the
  invariant: with the `id == 0` check removed from `Build()`, `err` would be
  `nil` and `TestBuild_MissingId` would fail (`err == nil, want error`).
  Confirmed `go build ./character/...` and `go test ./character/...` clean
  in all six modules (ran locally, all PASS, matches report).
- atlas-dragons: `TestBuild_ZeroId`/`TestBuild_WithId`, adapted correctly to
  the `NewBuilder(id uint32)` constructor-arg shape rather than a setter.
  `go test ./character/...` PASS locally.
- atlas-login: `TestBuilder_Build_MissingId` appended to the existing
  `builder_test.go` (existing `TestBuilder_Build` already covered the
  success path), matching the file's existing naming convention. `go test
  ./character/...` PASS locally.
- atlas-character: `TestBuilder_Build_MissingAccountId`,
  `TestBuilder_Build_MissingName`, `TestBuilder_Build_WithIdentity` added to
  a new `builder_test.go` (package `character_test`, external), targeting
  `Builder` (builder.go:101) only. Read the committed `builder.go` at
  `3803625d4` directly (not run, per scope note) and confirmed the
  `accountId == 0` / `name == ""` checks the tests exercise are present at
  builder.go:63-68, and that no call in this commit touches `modelBuilder`
  (`NewEmptyBuilder`/`CloneModel`, lines 174+).
- atlas-channel: correctly left alone — brief's own text says it already had
  `TestBuild_MissingId`/`TestBuild_ZeroId`; `git diff --name-only` confirms
  no atlas-channel file appears in the range.

## Controller rulings honored

- No `degrade.Observe` or `model.ErrDecorator` anywhere in the range
  (`git diff e2b1723b8..3803625d4 | grep -c 'degrade.Observe\|ErrDecorator'`
  = 0).
- atlas-consumables and atlas-pets comment-only fallbacks are byte-for-byte
  unchanged; the only new content in those two modules is the new
  `builder_test.go` files.
- `modelBuilder`/`NewEmptyBuilder()` in atlas-character: untouched, no test
  added against it, consistent with the "Open item" section of
  `builder-validation.md`.

## Build/test verification

Ran `go build ./character/...` and `go test ./character/...` from each
module root for the eight non-atlas-character touched modules
(atlas-cashshop, atlas-npc-shops, atlas-query-aggregator, atlas-messages,
atlas-consumables, atlas-dragons, atlas-login, atlas-pets): all built clean,
all tests passed. Did not build/test atlas-character or run
`tools/verify.sh` — atlas-character is mid-edit by a concurrent fix-round-2
implementer per the task instruction, and a repo-wide gate run is out of
scope for a diff-only fix-round review.

## Not evaluable

- atlas-character `go build`/`go test` — module tree is mid-edit by a
  concurrent implementer; correctness of the new test file's use of
  `Builder` was instead confirmed by reading the committed `builder.go` at
  `3803625d4` (the invariant checks it depends on are present at the
  expected lines).
- Repo-wide `tools/verify.sh` — out of scope for this round per both the
  brief and the task instruction.

## Verdict

All six blocking findings are closed exactly as specified, with no
relitigation of the DOM-28/`modelBuilder` questions the controller already
ruled on, and no scope creep. One non-blocking note on the implementer's
report (not the code) regarding an inaccurate claim about pets' comment
wording.
