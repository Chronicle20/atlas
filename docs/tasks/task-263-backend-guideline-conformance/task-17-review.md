# Task 17 review — W3 hand work, tier C

Commits reviewed: `7d81d43`, `9680019`, `a71607d`, `dc1eab8` (all under `services/`).
Brief: `.superpowers/sdd/plan/task-17-brief.md` `## CONTROLLER RE-SIZING` section.
Report: `docs/tasks/task-263-backend-guideline-conformance/task-17-report.md`.

## Verdict

CHANGES_REQUIRED — one blocking finding in Group 1.

## Group 1 — `atlas-channel/.../maps/location` (`7d81d43`)

`services/atlas-channel/atlas.com/channel/maps/location/rest.go:44-56` adds:

```go
func Transform(m Model) (RestModel, error) {
	return RestModel{
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		Instance:  m.Instance(),
		State:     string(m.State()),
	}, nil
}
```

**BLOCKING — `RestModel.Id` is silently left at zero even though a legitimate mapping
exists (`file:line` `services/atlas-channel/atlas.com/channel/maps/location/rest.go:44`,
finding recorded at `docs/tasks/task-263-backend-guideline-conformance/handwork-notes.md`
under "Task 17 tier C — field-pairing findings").**

The report and handwork-notes justify the omission with a case-insensitive name match: `Model`
has no field/method literally named `Id`, only `CharacterId()`, so "there is no counterpart."
That heuristic is wrong for this package. `services/atlas-channel/atlas.com/channel/maps/location/rest.go:16-19`
states in its own doc comment that `RestModel` "mirrors the JSON:API shape returned by
atlas-maps's GET /characters/{id}/location endpoint" — i.e. this is the client-side mirror of
`services/atlas-maps/atlas.com/maps/character/location/rest.go`, the producer of that exact wire
shape. The producer's own `Transform` (`services/atlas-maps/atlas.com/maps/character/location/rest.go:56-67`)
does exactly this mapping:

```go
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:        m.CharacterId(),
		...
```

with the doc comment "GetID returns the JSON:API resource id (the character id)" at
`services/atlas-maps/.../location/rest.go:30`. The same pattern recurs at Group 3
(`services/atlas-dragons/atlas.com/dragons/dragon/resource.go:41-51`, `Id: strconv.Itoa(int(m.OwnerCharacterId()))`
where "the dragon has no identity of its own") and `services/atlas-summons/atlas.com/summons/summon/resource.go:44-62`.
Across every sibling package in this same tier, the JSON:API `Id` is derived from the entity's
natural key even when the `Model` accessor is not literally named `Id()`. `Model.CharacterId()`
(`services/atlas-channel/atlas.com/channel/maps/location/model.go:24`, backed by unexported
`characterId` at `model.go:16`) is that natural key here — the resource is
`characters/{id}/location`, per `Resource = "characters/%d/location"` at `rest.go:14`.

So the premise check requested by the brief ("is this genuinely an unpairable field, or was a
legitimate mapping omitted") resolves to: **omitted**. `Transform` should set `Id:
m.CharacterId()`. As written, any future caller of this `Transform` (there is none today — see
below) gets `Id: 0` for every location, which is a real defect, not a documented exemption.

Consequence for test honesty: `rest_test.go` (new, `services/atlas-channel/atlas.com/channel/maps/location/rest_test.go`)
asserts every other field but never asserts on `rm.Id` — the missing assertion is consistent
with, and masks, the missing mapping. All five assertions that *are* present are field-by-field
with distinct non-zero fixture values (`NewModelForTest(100, world.Id(1), channel.Id(2),
_map.Id(3), instance, characterconst.PresenceStateInField)`) and I independently mutated `State`
(`string(m.State())` → `string(m.State())+"x"`) and reproduced a failing assertion
(`rest_test.go:40: State mismatch. Expected IN_FIELD, got IN_FIELDx`), then reverted cleanly
(`git checkout -- maps/location/rest.go`; `git status --short` clean afterward).

Non-blocking observation, not required to fix but worth recording: `Transform` here has no
current caller in production code (`grep -rln "location.RestModel\|location\".*Transform"
services/atlas-channel/atlas.com/channel` returns nothing beyond the definition; the package is
client-only — `Get`/`GetField` in `requests.go` build `Model` from the wire `RestModel` inline,
without ever going through an `Extract`, and there is no outbound endpoint in atlas-channel that
serializes `location.RestModel`). This means the missing `Id` mapping has no runtime blast radius
today, but that does not change the field-completeness verdict — DOM-04 tier C requires the
converter to map every derivable field, and `Id` is derivable.

## Group 2 — `atlas-character/.../session/history` (`9680019`) — PASS

- `services/atlas-character/atlas.com/character/session/history/rest.go:33-42`: `TransformToRest`
  renamed to `Transform`, signature widened to `(RestModel, error)`, return is unconditionally
  `nil` for the error. Diffed the six-field mapping (`Id`, `CharacterId`, `WorldId`, `ChannelId`,
  `LoginTime`, `LogoutTime`) against `git show 9680019^:.../rest.go` — byte-identical field
  assignments, only the trailing `}` → `}, nil` changed. No field added, removed, or reordered.
- `TransformSliceToRest` (`rest.go:44-51`) now calls `Transform` and discards the always-nil
  error — behavior-preserving.
- `resource.go:66` (`result := TransformSliceToRest(paged.Items)`) is untouched and still
  compiles against the adapted `TransformSliceToRest`; confirmed with
  `grep -rn 'TransformToRest\|TransformSliceToRest\|Transform(' services/atlas-character` — no
  stray `TransformToRest` reference remains anywhere in the module.
- New `rest_test.go` asserts all six fields (`Id`, `CharacterId`, `WorldId`, `ChannelId`,
  `LoginTime`, `LogoutTime`) with distinct non-zero fixture values, including a non-nil
  `LogoutTime`. I independently mutated `WorldId: m.WorldId()` → `m.WorldId() + 1`, reproduced
  `rest_test.go:37: WorldId mismatch. Expected 1, got 2`, then reverted cleanly (`git checkout --
  session/history/rest.go`; verified PASS and clean `git status`).
- Field mapping report claim ("field mapping unchanged") is correct — verified directly, not
  taken on faith.

## Group 3 — `atlas-dragons/dragon`, `atlas-summons/summon` (`a71607d`, `dc1eab8`) — PASS

- `git show --stat a71607d dc1eab8` confirms each commit adds exactly one new
  `resource_test.go` and touches no other file — no `resource.go`/`rest.go` change, no file
  move, matching the brief's "do not move any file" instruction exactly.
- Pre-existing `Transform` was left in place at `services/atlas-dragons/atlas.com/dragons/dragon/resource.go:41`
  and `services/atlas-summons/atlas.com/summons/summon/resource.go:44` — confirmed unchanged by
  the stat output (zero lines touched in those files).
- `dragon/resource_test.go` covers all ten `RestModel` fields (`Id`, `OwnerCharacterId`, `X`,
  `Y`, `Stance`, `JobId`, `WorldId`, `ChannelId`, `MapId`, `Instance`) with distinct non-zero
  values built via `NewBuilder(100).SetField(f).SetX(10).SetY(20).SetStance(5).SetJobId(...)`.
  `Id` is correctly asserted as `"100"` (`strconv.Itoa(int(m.OwnerCharacterId()))`) — this
  package gets the `Id`-derivation right, unlike Group 1.
- `summon/resource_test.go` covers all fourteen `RestModel` fields with distinct non-zero
  values, including `ExpiresAt` compared via `expiresAt.UnixMilli()` to match the `Transform`'s
  `m.ExpiresAt().UnixMilli()` conversion.
- No production Go file changed anywhere in either commit.

## Group 4 — `atlas-consumables/monster`, `atlas-data/skill`, `atlas-maps/character` — PASS

- `git show --stat` on all four commits shows zero touches to any file under
  `services/atlas-consumables`, `services/atlas-data`, or `services/atlas-maps` — documentation
  only, exactly as the brief required.
- Independently re-ran the NO-MODEL grep per package (not just re-reading the implementer's
  claim):
  - `grep -n "type Model\|Model struct\|Model interface" services/atlas-consumables/atlas.com/consumables/monster/*.go` — zero hits (only `RestModel struct` at `rest.go:11`).
  - `grep -n "type Model\|Model struct\|Model interface" services/atlas-data/atlas.com/data/skill/*.go` — zero hits (only `RestModel struct` at `rest.go:8`).
  - `grep -n "type Model\|Model struct\|Model interface" services/atlas-maps/atlas.com/maps/character/*.go` — zero hits (only `RestModel struct` at `rest.go:11`).
- `handwork-notes.md` records each with `file:line`, the grep, and a one-line DTO provenance
  note, under `## Task 17 tier C — NO-MODEL exemptions`, as the brief required.

## Lint / gofumpt

Ran `tools/lint.sh --check --fmt --go <module-root>` for all four touched module roots (not
`verify.sh`):

- `services/atlas-channel/atlas.com/channel` → `lint.sh: OK`
- `services/atlas-character/atlas.com/character` → `lint.sh: OK`
- `services/atlas-dragons/atlas.com/dragons` → `lint.sh: OK`
- `services/atlas-summons/atlas.com/summons` → `lint.sh: OK`

`gofmt -l` on the four new/changed `_test.go` files also returned no output (clean).

## Fixture and struct-shape checks

- No fixture in any of the four new/changed test files uses an all-zero-value `Model` — every
  one supplies distinct non-zero values per field (verified by reading each fixture literal
  above).
- No `RestModel` or `Model` struct field was added, removed, or reordered in any of the four
  commits — confirmed by diff review of each `rest.go`/`resource.go`/`model.go` touched (Group 1
  and 2 only add a function; Groups 3/4 add zero production-code lines).

## Not evaluable

- None. All four commits and all cross-referenced files (`atlas-maps` location package, the
  `atlas-character` history `resource.go` call site) were within reach of this review's surface
  and were read directly.

## Scope confirmation

The four commits under review match the four-group partition described in the brief and the
report exactly: one `feat` per genuine-Transform group (channel location, character history
rename) and one `test`-only commit per already-conformant group (dragons, summons), plus
documentation-only handling of the NO-MODEL group. No scope mismatch.
