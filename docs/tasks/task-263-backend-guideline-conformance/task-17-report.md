# Task 17 report — W3 hand work, tier C

## Summary

Implemented the controller's four-group re-sizing of tier C exactly as specified. No
re-derivation of the partition; no group reassignment.

### Group 1 — genuine tier C (`atlas-channel/maps/location`)

- `services/atlas-channel/atlas.com/channel/maps/location/rest.go` — added
  `func Transform(m Model) (RestModel, error)`, mapping `WorldId`, `ChannelId`, `MapId`,
  `Instance`, `State` (via `string(m.State())`, since `characterconst.PresenceState` is a
  defined `string` type).
- **Field-pairing finding:** `RestModel.Id` has no `Model` counterpart by case-insensitive
  name match — `Model` exposes the character identifier only as `CharacterId()`, not `Id()`.
  Conversely `Model.characterId` has no `RestModel` counterpart. Per rule 8, `Id` is left at
  its zero value in `Transform` rather than invented; recorded in `handwork-notes.md`.
- `services/atlas-channel/atlas.com/channel/maps/location/rest_test.go` — new `TestTransform`,
  field-by-field, using `NewModelForTest` with distinct non-zero values for every field.

### Group 2 — misnamed converter (`atlas-character/session/history`)

- `services/atlas-character/atlas.com/character/session/history/rest.go` — renamed
  `TransformToRest(m Model) RestModel` to `Transform(m Model) (RestModel, error)` (returns
  `nil` error; the function stays total). `TransformSliceToRest` now calls `Transform` and
  discards the (always-nil) error. Field mapping unchanged.
- Verified `resource.go:66` calls `TransformSliceToRest`, not `TransformToRest`, so no caller
  update was needed. Confirmed with
  `grep -rn 'TransformToRest\|TransformSliceToRest' services/atlas-character` — only
  `TransformSliceToRest` remains referenced, at `rest.go:44` (def) and `resource.go:66` (call).
- `services/atlas-character/atlas.com/character/session/history/rest_test.go` — new
  `TestTransform`, field-by-field, over a `Model` built via `modelFromEntity(entity{...})`
  with distinct non-zero values including a non-nil `LogoutTime`.

### Group 3 — already conformant, test only (`atlas-dragons/dragon`, `atlas-summons/summon`)

No code moved, no new `Transform` written — both already had the correct name/signature in
`resource.go`. Added only:

- `services/atlas-dragons/atlas.com/dragons/dragon/resource_test.go` — new `TestTransform`,
  field-by-field over `Id`, `OwnerCharacterId`, `X`, `Y`, `Stance`, `JobId`, `WorldId`,
  `ChannelId`, `MapId`, `Instance`, built via `NewBuilder(100).SetField(...)...Build()`.
- `services/atlas-summons/atlas.com/summons/summon/resource_test.go` — new `TestTransform`,
  field-by-field over `Id`, `OwnerCharacterId`, `SkillId`, `SkillLevel`, `SummonType`,
  `MovementType`, `X`, `Y`, `Hp`, `MaxHp`, `ExpiresAt`, `WorldId`, `ChannelId`, `MapId`,
  `Instance`, built via `NewBuilder().Set*...Build()`.

### Group 4 — structurally exempt, documentation only

Confirmed absence of `type Model` with one grep per package (all returned zero hits):

- `grep -n "type Model" services/atlas-consumables/atlas.com/consumables/monster/*.go` — no hits
- `grep -n "type Model" services/atlas-data/atlas.com/data/skill/*.go` — no hits
- `grep -n "type Model" services/atlas-maps/atlas.com/maps/character/*.go` — no hits

Recorded all three under a new `## Task 17 tier C — NO-MODEL exemptions` heading in
`handwork-notes.md`, with each `RestModel`'s `file:line`, the grep, and one line on what the
DTO is deserialized from. Wrote no Go code for this group.

## TDD Evidence

### Group 1 (`atlas-channel/maps/location`) — genuine RED

RED:
```
$ go test ./... -run TestTransform -v
./rest_test.go:18:13: undefined: Transform
FAIL	atlas-channel/maps/location [build failed]
```
GREEN (after implementing `Transform`):
```
=== RUN   TestTransform
--- PASS: TestTransform (0.00s)
PASS
ok  	atlas-channel/maps/location	0.006s
```
Mutation proof: changed `MapId: m.MapId()` to `m.MapId() + 1` →
`rest_test.go:32: MapId mismatch. Expected 3, got 4` (FAIL). Reverted → PASS again.

### Group 2 (`atlas-character/session/history`) — genuine RED

Because the rename happened before the test was written, RED was reproduced by stashing
`rest.go` (restoring the pre-rename `TransformToRest`), confirming the test fails, then
popping the stash to restore the rename:
```
$ git stash push -- rest.go && go test ./... -run TestTransform -v
./rest_test.go:23:13: undefined: Transform
FAIL	atlas-character/session/history [build failed]
$ git stash pop
```
GREEN (rename applied):
```
=== RUN   TestTransform
--- PASS: TestTransform (0.00s)
PASS
ok  	atlas-character/session/history	0.007s
```
Mutation proof: changed `CharacterId: m.CharacterId()` to `m.CharacterId() + 1` →
`rest_test.go:33: CharacterId mismatch. Expected 200, got 201` (FAIL). Reverted → PASS again.

### Group 3 (`atlas-dragons/dragon`, `atlas-summons/summon`) — no RED, as expected

`Transform` already existed with the correct name and signature in both packages before this
task, so there is no RED to show — the test passed the first time it was run (both PASS
immediately). Per resolution #5, this is stated explicitly rather than fabricated.

Mutation proof (`atlas-dragons/dragon`): changed `Stance: m.Stance()` to `m.Stance() + 1` →
`resource_test.go:48: Stance mismatch. Expected 5, got 6` (FAIL). Reverted → PASS.

Mutation proof (`atlas-summons/summon`): changed `Hp: m.Hp()` to `m.Hp() + 1` →
`resource_test.go:72: Hp mismatch. Expected 30, got 31` (FAIL). Reverted → PASS.

## Module-local verification

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test ./...
```
All packages `ok` (or `no test files`); no failures.

```
cd services/atlas-character/atlas.com/character && go build ./... && go vet ./... && go test ./...
```
All packages `ok` (or `no test files`); no failures.

```
cd services/atlas-dragons/atlas.com/dragons && go build ./... && go vet ./... && go test ./...
```
`ok atlas-dragons/dragon 3.428s`; all others `ok` or `no test files`.

```
cd services/atlas-summons/atlas.com/summons && go build ./... && go vet ./... && go test ./...
```
`ok atlas-summons/summon 0.094s`; all others `ok` or `no test files`.

Lint (all four module roots):
```
tools/lint.sh --check --fmt --go services/atlas-channel/atlas.com/channel      -> lint.sh: OK
tools/lint.sh --check --fmt --go services/atlas-character/atlas.com/character  -> lint.sh: OK
tools/lint.sh --check --fmt --go services/atlas-dragons/atlas.com/dragons      -> lint.sh: OK
tools/lint.sh --check --fmt --go services/atlas-summons/atlas.com/summons      -> lint.sh: OK
```

## Files changed

- `services/atlas-channel/atlas.com/channel/maps/location/rest.go` (Transform added)
- `services/atlas-channel/atlas.com/channel/maps/location/rest_test.go` (new)
- `services/atlas-character/atlas.com/character/session/history/rest.go` (rename + widen)
- `services/atlas-character/atlas.com/character/session/history/rest_test.go` (new)
- `services/atlas-dragons/atlas.com/dragons/dragon/resource_test.go` (new)
- `services/atlas-summons/atlas.com/summons/summon/resource_test.go` (new)
- `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md` (Group 1 finding +
  Group 4 NO-MODEL exemptions) — **left uncommitted**, per brief's git-discipline note.

## Commits (under `services/`)

- `7d81d43` feat(atlas-channel): add Transform and test for maps/location
- `9680019` feat(atlas-character): rename TransformToRest to Transform in session/history
- `a71607d` test(atlas-dragons): add TestTransform for dragon package
- `dc1eab8` test(atlas-summons): add TestTransform for summon package

Verified after each commit: `git rev-parse --show-toplevel` resolves to the worktree root
(`.worktrees/task-263-backend-guideline-conformance`); `git branch --show-current` ->
`task-263-backend-guideline-conformance`.

## Self-review findings

- Every new `Transform` returns a `nil` error unconditionally — none of the four field sets
  involve a conversion that can genuinely fail (FR-1 compliance).
- Group 2's field mapping was left untouched, per resolution #3 — only the name and return
  signature changed, plus the one-line adaptation in `TransformSliceToRest`.
- Group 3 required no file moves and no new `Transform` — confirmed both `resource.go`
  functions already matched `^func Transform(` exactly before this task.
- Group 4 required no Go code — confirmed via grep, not by inspection alone.
- Fixtures for all four `TestTransform`s use distinct, non-zero values per field (rule 4);
  none is a zero-value tautology.
- No `git add -A`/`git add .` used; each commit added only the relevant explicit file paths.

## Issues or concerns

None. All four module-local gates (`go build`, `go vet`, `go test`, `tools/lint.sh --check
--fmt --go`) pass cleanly for every affected module.

## Fix round 1 (review findings)

The reviewer correctly rejected the Group 1 "no counterpart" reasoning above: the wire
shape's producer, `services/atlas-maps/atlas.com/maps/character/location/rest.go:56-63`, maps
`Id: m.CharacterId()` for the identical field set, so `atlas-channel`'s `Transform` leaving
`Id` at zero was a real bug (a zero JSON:API resource id on the consumer side), not a
justified carve-out.

**Blocking — `maps/location/rest.go:44` `Transform` leaves `RestModel.Id` unmapped.**

RED — added the missing assertion to `TestTransform` first and ran it against the
still-unfixed `Transform`:

```
$ go test ./maps/location/... -run TestTransform -v
=== RUN   TestTransform
    rest_test.go:24: Id mismatch. Expected 100, got 0
--- FAIL: TestTransform (0.00s)
FAIL	atlas-channel/maps/location	0.005s
```

Fix — added `Id: m.CharacterId()` to `Transform` and deleted the now-false comment explaining
why `Id` was unmapped.

GREEN:

```
$ go build ./... && go vet ./... && go test ./maps/location/... -run TestTransform -v
=== RUN   TestTransform
--- PASS: TestTransform (0.00s)
PASS
ok  	atlas-channel/maps/location	0.005s
```

Full module test suite, same command set:

```
$ go test ./...
```

All packages `ok` or `no test files`; no failures.

Lint, from worktree root:

```
$ tools/lint.sh --check --fmt --go services/atlas-channel/atlas.com/channel
lint.sh: OK
```

**Non-blocking — missing `rm.Id` assertion in `rest_test.go`.** Fixed as part of the RED step
above (the new assertion is what proved the bug).

**Fixture check.** `NewModelForTest(100, ...)` in `rest_test.go` already used `characterId =
100`, distinct and non-zero, so the new assertion (`rm.Id != 100`) is meaningful with no
fixture change needed.

### Files changed (fix round 1)

- `services/atlas-channel/atlas.com/channel/maps/location/rest.go` — `Transform` now sets
  `Id: m.CharacterId()`; removed the stale comment justifying the omission.
- `services/atlas-channel/atlas.com/channel/maps/location/rest_test.go` — `TestTransform` now
  asserts `rm.Id == 100`.
- `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md` — corrected the
  `maps/location` tier-C entry: `Id` maps to `CharacterId()`, matching the `atlas-maps`
  producer, rather than being unpairable. Left uncommitted per the fix-round brief.

### Constraints honored

- Only `Id`'s mapping changed in `Transform`; no other field mapping touched, no struct field
  added/removed/reordered.
- `atlas-maps` producer package untouched (read-only reference, confirmed by grep only).
- Groups 2/3/4 from Task 17 untouched.

### Git

Committed `services/atlas-channel` changes only:

```
$ git commit -m "fix(atlas-channel): map RestModel.Id from CharacterId in maps/location Transform" -- services/atlas-channel
ok e3faf58
```

`git show --stat HEAD` confirms exactly the two intended files (`rest.go` +1/-3,
`rest_test.go` +4). `git rev-parse --abbrev-ref HEAD` confirms
`task-263-backend-guideline-conformance`. `handwork-notes.md` and this report edit are left
uncommitted per instructions.
