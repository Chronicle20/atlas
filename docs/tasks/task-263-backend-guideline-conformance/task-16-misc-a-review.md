# Task 16 batch `misc-a` review

Scope: commits `749366aaa` (atlas-cashshop), `a07cefc18` (atlas-dragons),
`c30eb446a` (atlas-inventory), `159dd280a` (atlas-kites), `87dfe8a`
(atlas-maps), `1d4d9ed` (atlas-merchant), `8f1e02e` (docs).

Brief: `.superpowers/sdd/plan/task-16-brief-misc-a.md`. Task: DOM-04 — add
hand-written `Transform` (Model -> RestModel) to six tier-B2 packages whose
`Extract` is non-flat or lives outside `rest.go`.

## Charge 1 — Transform placement where Extract is displaced

- `atlas-cashshop/.../commodity`: `Extract` unchanged at `model.go:37`.
  `Transform` added at `rest.go:35-46`. Extract NOT moved. PASS.
- `atlas-inventory/.../data/setup`: `Extract` unchanged at `processor.go:40`.
  `Transform` added at `rest.go` (new file content, package `setup`). Extract
  NOT moved. PASS.

## Charge 2 — documented field-gap premises

- `atlas-maps/.../reactor`: verified `RestModel` (`rest.go:15-27`) has no
  field for `time.Time` / `updateTime`; `Extract` (`rest.go`, bottom of file)
  never sets `updateTime`, leaving it Go zero value. `Transform` emits 13
  fields (`Id, WorldId, ChannelId, MapId, Instance, Classification, Name,
  State, EventState, X, Y, Delay, Direction`); `TestTransformRoundTrip`
  (`rest_test.go:43-81`) asserts exactly those 13, field by field, matching
  every field `Transform` populates — nothing was quietly dropped from the
  assertion beyond `UpdateTime` itself. Cross-checked against `Model`
  (`model.go:13-24`): all 10 non-`updateTime`, non-`f`-composite fields are
  present, and `f` (`field.Model`) is represented via `WorldId`/`ChannelId`/
  `MapId`/`Instance`. Confirmed by mutation test (see Charge 8). PASS.
- `atlas-merchant/.../data/portal`: verified `Extract` (`rest.go`, bottom of
  file) reads only `rm.Id, rm.Name, rm.Type, rm.X, rm.Y, rm.TargetMapId` —
  `rm.Target` and `rm.ScriptName` are never referenced anywhere in `Extract`'s
  body. `Transform` correctly omits both. PASS.

## Charge 3 — field coverage, derived independently per package

- `atlas-cashshop/.../commodity`: `Model` (`model.go:3-12`) has 8 fields
  (id, itemId, count, price, period, priority, gender, onSale). `Transform`
  (`rest.go:35-46`) emits all 8. PASS.
- `atlas-dragons/.../character`: `Model` (`model.go:10-16`) has 5 fields
  (id, jobId, x, y, stance). `Transform` (`rest.go:34-42`) emits all 5. PASS.
- `atlas-inventory/.../data/setup`: `Model` (`model.go:3-15`) has 11 fields
  (id, price, slotMax, recoveryHP, tradeBlock, notSale, reqLevel, distanceX,
  distanceY, maxDiff, direction). `Transform` (`rest.go`) emits all 11.
  Verified independently against this package's own model.go, not by
  analogy to atlas-npc-shops/atlas-storage twins. PASS.
- `atlas-kites/.../configuration`: `Model` (`model.go:14-18`) has 3 fields
  (maxPerMap, maxMessageLength, blockedMapPrefixes). `Transform` (`rest.go`)
  emits all 3. PASS.
- `atlas-maps/.../reactor`: see Charge 2 — 12 of 13 model fields (excluding
  `updateTime`) all covered. PASS.
- `atlas-merchant/.../data/portal`: `Model` (defined in `rest.go`, ~line 30)
  has 6 fields (id, name, portalType, x, y, targetMapId). `Transform`
  emits all 6, mapped to `Id/Name/Type/X/Y/TargetMapId`. Verified
  independently against this package's own type, not the six sibling
  `portal` packages elsewhere in the repo. PASS.

## Charge 4 — faithfulness to Extract, not the JSON surface

All six `Transform` functions populate only `RestModel` fields that the
package's own `Extract` reads back out. Confirmed for merchant's
`Target`/`ScriptName` gap (Charge 2) and maps's `UpdateTime` gap (Charge 2);
no other package has an unread `RestModel` field. PASS.

## Charge 5 — narrowing getters

- `atlas-cashshop/.../commodity`, `atlas-inventory/.../data/setup`: direct
  unexported-field access (`m.id`, `m.price`, ...). No getters used.
- `atlas-dragons/.../character`, `atlas-maps/.../reactor`: use accessor
  methods (`m.Id()`, `m.X()`, ...), consistent with the brief's named
  pattern `atlas-ban/.../ban/rest.go:36-62`, which also uses accessors. All
  accessors checked return the same type as the underlying field (e.g.
  `X() int16` returns `m.x int16`, `Id() uint32` returns `m.id uint32`) — no
  narrowing.
- `atlas-kites/.../configuration`: direct field access.
- `atlas-merchant/.../data/portal`: direct field access
  (`m.id, m.name, m.portalType, m.x, m.y, m.targetMapId`).
No narrowing getter routed a wider field through a truncating accessor in
any of the six packages. PASS.

## Charge 6 — no behavior change outside the addition

`git show --stat` for all six service commits shows only `rest.go` (or
`rest.go`+`rest_test.go`) touched, and the per-file diffs are purely
additive (new `Transform` function + new/appended test). `Extract` bodies
are byte-identical to pre-commit versions in the two displaced-Extract
packages (verified by reading `processor.go`/`model.go`, which the diffs do
not touch). No `Build()` validation rule changed (dragons/maps `Build()`/
builder logic untouched — diffs don't touch `builder.go`). No `RestModel`
field added in any of the six packages. PASS.

## Charge 7 — no overwritten file, no commit leaked

`git diff-tree --no-commit-id --name-only -r <sha>` for each of the seven
commits:
- `749366aaa`: `cashshop/commodity/{rest.go,rest_test.go}` only.
- `a07cefc18`: `dragons/character/{rest.go,rest_test.go}` only.
- `c30eb446a`: `inventory/data/setup/{rest.go,rest_test.go}` only.
- `159dd280a`: `kites/configuration/{rest.go,rest_test.go}` only. Diff on
  `rest_test.go` is a clean append (`+import "reflect"`, `+TestTransformRoundTrip`
  at end of file) — all five pre-existing tests
  (`TestExtractFoldsZeroKnobsToDefaults`, `TestExtractKeepsProvidedKnobs`,
  `TestIsMapBlockedMirrorsClientArithmetic`, plus tenant-fetcher tests in the
  same package) still present and passing. No `Write`-style destruction.
- `87dfe8a`: `maps/reactor/{rest.go,rest_test.go}` only.
- `1d4d9ed`: `merchant/data/portal/{rest.go,rest_test.go}` only.
- `8f1e02e`: `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md`
  only — docs-only as required.
No cross-service leakage, no unexpected deletions. PASS.

## Charge 8 — tests actually constrain the code

Ran mutation on `atlas-maps/.../reactor/rest.go`: changed
`Direction: m.Direction()` to `Direction: 0` in `Transform`. Result:

```
rest_test.go:74: Direction mismatch. Expected 11, got 0
--- FAIL: TestTransformRoundTrip (0.00s)
```

Reverted with `git checkout -- reactor/rest.go`; `git status --short` shows
the file no longer modified (only unrelated concurrent-agent files remain
dirty, not touched by this review). PASS.

## Test execution

All six packages' `TestTransformRoundTrip` (plus pre-existing kites tests)
run and pass:

```
ok atlas-cashshop/cashshop/commodity
ok atlas-dragons/character
ok atlas-inventory/data/setup
ok atlas-kites/configuration (+ 5 pre-existing tests)
ok atlas-maps/reactor
ok atlas-merchant/data/portal
```

## Not evaluable

None — all charges were directly checkable within the seven-commit diff and
each package's own model/rest source.

## Verdict

APPROVED. All eight review charges pass. No blocking or non-blocking
findings.
