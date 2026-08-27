# Task 16 batch `channel-b` review

verdict: APPROVED

## Scope

Reviewed exactly two commits:
- `d2227f582` — `Transform` added in five `rest.go` files (`data/map`, `data/npc/template`, `data/portal`, `data/skill`, `data/skill/effect`).
- `f0a756762` — five new `rest_test.go` files, one per package.

Confirmed via `git show --stat` on both commits: no file outside these five packages is touched, no deletions in either commit, and no overlap with the ten packages already landed in batch `channel-a` (`aea948a4a`: `account`, `buddylist`, `character/buff`, `character/teleportrock`, `data/item`). The two-commit split is the expected/documented artifact of `git commit -- <pathspec>` not staging untracked files — not itself a finding, per the brief's note.

## Charge 1 — field coverage, derived independently

For each package, `Model` fields (from that package's own `model.go`) were enumerated and checked against `Transform`'s output field-by-field, and against `Extract`'s existing field-by-field mapping in the same file, without reference to any sibling package.

- **`data/map`** (`model.go:9-15`, 5 fields: `clock`, `returnMapId`, `fieldLimit`, `town`, `footholds`) — all five present in `Transform` (`rest.go:92-117`). `footholds` is flattened back into a single-level `FootholdTreeRestModel`; `flattenFootholds` (`rest.go:133-153`) walks any depth and re-derives the same flat map, verified by the round-trip test with two footholds.
- **`data/npc/template`** (`model.go:3-9`, 5 fields) — all five present, `Id` round-tripped through `strconv.Itoa`/`strconv.Atoi` (`rest.go:29`, `:38`).
- **`data/portal`** (`model.go:5-14`, 8 fields) — all eight present in `Transform` (`rest.go:34-45`), matching this package's own `Model`, not the same-named twins in atlas-character/atlas-messages/atlas-summons/atlas-doors/atlas-saga-orchestrator/atlas-transports. Field set (`id`, `name`, `target`, `portalType`, `x`, `y`, `targetMapId`, `scriptName`) confirmed against local `model.go`.
- **`data/skill`** (`model.go:7-13`, 5 fields) — all five present, nested `effects` mapped via `model.SliceMap(effect.Transform)` (`rest.go:37`), mirroring `Extract`'s `model.SliceMap(effect.Extract)` (`rest.go:52`).
- **`data/skill/effect`** (`model.go:9-68`, 58 fields) — all 58 present in `Transform` (`rest.go:78-153`), enumerated and diffed line-by-line against `model.go` and against `Extract`'s existing field list (`rest.go:170-229`). Not borrowed from any sibling; verified independently.

## Charge 2 — `LT`/`RB` ambiguity claim (verified in full)

- **(a) Collapse is real.** `point.Model` (`libs/atlas-constants/point/model.go:3-6`) has only `x X, y Y`; `point.NewModel(0,0)` and `point.Model{}` are value-equal. `Extract` (`rest.go:161-168`) converts nil `*PointRestModel` to the zero-value `point.Model{}`, and an explicit `&PointRestModel{0,0}` also converts to `point.Model{X:0,Y:0}` — indistinguishable at the `Model` level. Confirmed.
- **(b) Genuinely pre-existing.** `git show d2227f582^:.../data/skill/effect/rest.go` shows `Extract`'s nil-check logic for `LT`/`RB` (lines matching current `rest.go:161-168`) already present, byte-for-byte, before this commit. This commit only added `Transform`; `Extract` is untouched. Confirmed pre-existing, not introduced.
- **(c) Round trip genuinely holds; test does not mask the ambiguity.** `rest_test.go:69-70` uses `lt: point.NewModel(45,46)`, `rb: point.NewModel(47,48)` — non-zero, non-ambiguous values, so `TestTransformRoundTrip` is not passing merely because both branches collapse identically. The ambiguity only exists at the RestModel→Model direction for the zero case, which the test does not exercise (correctly, since it is a pre-existing RestModel-side asymmetry, not a `Model` field drop — `handwork-notes.md` is for dropped `Model` fields, and none is dropped here). Implementer's characterization in the report (lines 30-32) matches what I independently confirmed.

## Charge 3 — faithfulness to `Extract`, not the JSON surface

- `data/map`: `RestModel.Name, StreetName, MonsterRate, OnFirstUserEnter, OnUserEnter, MobInterval, Seats, EverLast, DecHP, ProtectItem, ForcedReturnMapId, Boat, TimeLimit, FieldType, MobCapacity, Recovery` are never read by `Extract` (`rest.go:119-129`) and are correctly left unpopulated by `Transform`.
- `data/skill/effect`: `RestModel.CardStats` (`rest.go:72`) is never read by `Extract` (`rest.go:155-229`) and is correctly left unpopulated by `Transform` (`rest.go:78-153`, no `CardStats:` key).
- `data/npc/template`, `data/portal`, `data/skill`: `RestModel` field sets equal what `Extract` reads exactly; no extraneous fields to check.

## Charge 4 — narrowing accessors (D1)

All five `Transform` functions read unexported fields directly off `m` (`m.clock`, `m.id`, `m.name`, `m.hp`, `m.x`, `m.y`, etc.) rather than through a narrowing getter. Specifically checked `data/skill/effect/rest.go:94-151`: every field access is `m.<unexported field>`, including `m.hp` (not the `HP() uint16` getter — though same width here, direct access is still used per D1) and `m.x`/`m.y` (not `X()`/`Y()`). No narrowing-accessor pattern found.

## Charge 5 — lossy fields (no `handwork-notes.md` entry)

Checked `data/map` (`Extract` at `rest.go:119`) and `data/skill/effect` (`Extract` at `rest.go:155`), the two heaviest packages, plus the other three:

- Every `Model` field in all five packages either round-trips through `Transform`→`Extract` or is genuinely absent from the `Model` (in which case it's a `RestModel`-only field, not a `Model` field, and correctly not carried by `Transform` — resolution #3, not resolution #4). No `Model` field exists that `Extract` cannot restore. The `handwork-notes.md` omission is correct — none of these five belong there.

## Charge 6 — no behavior change outside the addition

`git diff d2227f582^ d2227f582 -- <the five rest.go files>` contains zero removed lines (`grep "^-"` after excluding file headers returned nothing) — purely additive. No `Build()` present in any of the five packages (`Model` is a plain struct in all five, no builder), so FR-17 is not implicated. No `RestModel` field was added (confirmed by inspection of each `RestModel` struct against pre-commit state, and by the additive-only diff).

## Charge 7 — no overwritten file, no leakage

`git show --stat` on both commits shows only the ten expected files (five `rest.go` + five `rest_test.go`), all under the five batch packages. `git show f0a756762 | grep "^-"` (excluding diff headers) returned nothing — the new test files are pure additions, no pre-existing test content destroyed. No file under batch `channel-a`'s ten packages or any other atlas-channel package is touched.

## Charge 8 — tests actually constrain the code (mutation test)

Performed an independent mutation on `data/portal/rest.go`: changed `Transform`'s `ScriptName: m.scriptName` to `ScriptName: ""`. Re-ran:
```
go test ./data/portal/... -run TestTransformRoundTrip -v
```
Result: `FAIL — round trip mismatch ... scriptName:script1 ... got ... scriptName:`. Confirmed the test fails at field level on a dropped field. Reverted cleanly with `git checkout -- data/portal/rest.go`; `git status` confirmed clean afterward.

## Test run (scoped)

```
go test ./data/map/... ./data/npc/template/... ./data/portal/... ./data/skill/... ./data/skill/effect/... -run TestTransformRoundTrip -v
```
All five `TestTransformRoundTrip` PASS. No build errors originated in these five packages (confirmed by scoping the run away from concurrently-modified `door`/`guild`/`monster` packages per the concurrency notice).

## Not evaluable

None. All eight charges were fully evaluable within this batch's two-commit diff plus the `point.Model` and `statup` files the diff's correctness genuinely depends on.

## Findings

None blocking. None non-blocking beyond what is already surfaced (and correctly not actioned) by the implementer in the report's Concerns section — the `LT`/`RB` ambiguity is pre-existing, out of this batch's scope to fix (PRD §5 forbids the `RestModel` API change that would resolve it), and correctly left undocumented since no `Model` field is dropped.
