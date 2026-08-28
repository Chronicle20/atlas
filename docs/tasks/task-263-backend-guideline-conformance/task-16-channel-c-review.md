# Task 16 review — batch `channel-c`

Commit under review: `623378a` (single commit, `feat(atlas-channel): add Transform and round-trip tests for door/guild/monster packages`).

Brief: `.superpowers/sdd/plan/task-16-brief-channel-c.md` — add `Transform` (`Model` -> `RestModel`)
plus `TestTransformRoundTrip` to five `atlas-channel` tier-B2 packages: `door`, `guild`,
`guild/thread`, `monster`, `monster/information`.

## Scope

`git diff --stat 623378a^ 623378a` touches exactly 10 non-doc files, all under
`services/atlas-channel/atlas.com/channel/{door,guild,guild/thread,monster,monster/information}`,
plus the handwork-notes.md doc entry. 376 insertions, 1 deletion (the deleted line is a
single-line `import "testing"` replaced by a multi-line import block in
`monster/information/rest_test.go` — not a removal of behavior). Purely additive, confined to the
five named packages, no overlap with `channel-a`/`channel-b` package sets (checked via
`git diff --stat`; no other package paths appear in the diff).

## Primary charge — the two claimed carve-outs

### 1. `guild/thread` — `Model.tenantId`/`Model.guildId` unused

Confirmed. `model.go:11-12` declares `tenantId uuid.UUID` and `guildId uint32` on `Model`.
`rest.go:11-19`'s `RestModel` has no corresponding field. `Extract` (`rest.go:57-72`, specifically
the returned struct literal `rest.go:63-72`) never assigns `tenantId` or `guildId`. A full-package
`grep -rn "tenantId\|guildId"` sweep of `services/atlas-channel/atlas.com/channel/guild/thread`
turns up no other assignment site — every other `guildId` hit in the package is an unrelated local
parameter name (`processor.go`, `producer.go`, `requests.go`, `mock/processor.go`), not the
`Model.guildId` field. Claim verified. `Transform` (`rest.go:39-55`) correctly omits both fields,
and the test comment (`rest_test.go:11-15`) documents the omission honestly.

### 2. `monster` — `RestModel.DamageEntries` unread

Confirmed. `rest.go:33` declares `DamageEntries []DamageEntry` on `RestModel`. `Extract`
(`rest.go:95-127`) never reads `m.DamageEntries`. `Model` (`model.go:26-42`) has no field that
could hold it. `Transform` (`rest.go:62-93`) correctly does not populate it (there is nothing to
populate it from). Claim verified — and correctly framed as *not* lossy in the round-trip sense,
since every `Model` field is carried through.

### 3. Sweep for excluded restorable fields (independent inventory, not the implementer's list)

Field-by-field, `Model` vs `RestModel` vs what the test fixture sets, derived independently from
`model.go`/`rest.go` in each package:

- **`door`**: `Model` (`model.go:14-32`) has 16 scalar fields + `field.Model`. `RestModel`
  (`rest.go:14-35`) mirrors every one (the composite `field.Model` unpacks to `WorldId`,
  `ChannelId`, `MapId`, `Instance`). `Transform` (`rest.go:52-75`) and `Extract` (`rest.go:77-97`)
  both touch all 19 corresponding fields. Fixture (`rest_test.go:15-36`) sets all 19 to distinct
  non-zero values (`Id:"1"`, `AreaDoorId:100`, …, `ExpiresAt: 2026-01-02T03:04:05Z`). No carve-out,
  none needed. PASS.
- **`guild`**: `Model` (`model.go:11-25`) has 11 scalars + `members []member.Model` +
  `titles []title.Model`. `RestModel` (`rest.go:12-26`) mirrors all 13. `Transform`/`Extract`
  (`rest.go:45-95`) touch every field, including recursive `member.Transform`/`title.Transform`
  (pre-existing, unmodified by this commit — confirmed via `git diff 623378a^ 623378a` touching
  only `guild/rest.go` and `guild/rest_test.go`, not `guild/member/rest.go` or `guild/title/rest.go`).
  Fixture (`rest_test.go:15-33`) sets all top-level fields plus one non-zero member and one
  non-zero title. PASS.
- **`guild/thread`**: `Model` (`model.go:10-21`) has 8 fields besides `tenantId`/`guildId` (see
  above). All 8 are mirrored by `RestModel` (`rest.go:11-19`) and touched by both
  `Transform`/`Extract`. Fixture (`rest_test.go:17-28`) sets all 8 to distinct non-zero values,
  including one non-zero nested `reply.RestModel`. PASS with the documented, verified carve-out.
- **`monster`**: `Model` (`model.go:26-42`) has 13 scalars + `field.Model` + `statusEffects`.
  `RestModel` (`rest.go:15-35`) mirrors all of these plus the unread `DamageEntries` (see above).
  `Transform`/`Extract` (`rest.go:62-127`) touch every `Model`-side field. Fixture
  (`rest_test.go:15-40`) sets all fields to distinct non-zero values, including one non-zero
  `StatusEffectRestModel` with a populated `Statuses` map. PASS with the documented, verified
  carve-out.
- **`monster/information`**: `Model` (`model.go:9-12`) has `monsterId` + `attacks`. `RestModel`
  (`rest.go:5-8`) mirrors both. `Transform`/`Extract` (`rest.go:29-62`) touch both. Fixture
  (`rest_test.go:12-17`) sets both to non-zero values. PASS.

No field was excluded from a test assertion to make a narrowed test pass; every field the
implementer's notes did not carve out is exercised, and the two carve-outs both check out against
the actual `Extract` body.

## Fixture non-zero check

All five fixtures use distinct, non-zero values for every field they populate (spot-checked above
per package). No zero-value tautology.

## Live mutation check

Performed a live mutation the implementer did not exercise: in `door/rest.go:70`, changed
`TownY: m.TownY()` to `TownY: 0` (dropping a field `Transform` should carry). Re-ran
`go test ./door/... -run TestTransformRoundTrip -v`:

```
--- FAIL: TestTransformRoundTrip (0.00s)
    rest_test.go:54: round trip mismatch. Expected {... townX:12 townY:13 ...},
    got {... townX:12 townY:0 ...}
```

Test failed as expected, with `townY` isolated in the diff. Reverted with `git checkout --
door/rest.go`; `git diff --stat` confirms a clean working tree afterward. This confirms the test
is not tautological and would catch a real regression in any single field.

`go test ./door/... ./guild/... ./monster/...` (all five packages, cached) passes: `ok` for
`door`, `guild`, `guild/member`, `guild/thread`, `guild/thread/reply`, `guild/title`, `monster`,
`monster/information`.

## Structural checks

- `Transform` is defined in `rest.go` in all five packages (`door/rest.go:52`,
  `guild/rest.go:45`, `guild/thread/rest.go:39`, `monster/information/rest.go:29`,
  `monster/rest.go:62`).
- Field access style is inconsistent across the batch but not incorrect: `guild/rest.go` and
  `guild/thread/rest.go` build the `RestModel` literal from unexported fields directly (`m.id`,
  `m.worldId`, …), matching `Extract`'s style. `door/rest.go`, `monster/rest.go`, and
  `monster/information/rest.go` instead call the package's own exported getters (`m.Id()`,
  `m.WorldId()`, `m.MonsterId()`, …). Both forms are same-package legal and functionally
  identical here — none of the getters touched have side effects (unlike, e.g., `guild.Members()`,
  which sorts; that getter is correctly avoided in `guild/rest.go`'s `Transform`, which iterates
  `m.members` directly via `model.SliceMap`). Non-blocking style observation, not a defect.

## Pre-existing observations (out of scope, not blocking)

- `guild/member` and `guild/title` already had `Transform`/`Extract` before this commit
  (confirmed via `git diff 623378a^ 623378a --stat`, which does not touch either file); this
  commit correctly reuses them rather than duplicating logic.
- `guild/thread/reply` likewise already had `Transform`/`Extract` before this commit, reused
  correctly by `guild/thread/rest.go:40,58`.
- `monster/information/rest.go:44-49`'s `Extract` silently defaults `id` to `0` on a
  `strconv.ParseUint` failure ("id may be empty in tests; tolerate") — pre-existing behavior
  (confirmed: this code is outside the diff range added by `623378a`), not introduced or modified
  by this commit. Flagged as an observation only.

## Not evaluable

None. The full review surface (5 packages' `model.go`, `rest.go`, `rest_test.go`, plus the
`member`/`title`/`reply` sub-packages whose pre-existing `Transform`/`Extract` the new code calls)
was read and verified against a live mutation.

## Verdict

APPROVED. No blocking findings. One non-blocking style observation (getter vs. direct-field
access split across the batch) and pre-existing, out-of-scope observations noted above.
