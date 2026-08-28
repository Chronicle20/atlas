# Task 16 batch `doors-summons-reactors` — review

verdict: APPROVED

## Scope

Commits reviewed:
- `5a292ffba` feat(atlas-doors): add Transform and round-trip tests
- `2bac98f26` feat(atlas-summons): add Transform and round-trip tests
- `e4d483d9b` feat(atlas-reactors): add Transform and round-trip tests

Brief: `.superpowers/sdd/plan/task-16-brief-doors-summons-reactors.md`
Report: `.superpowers/sdd/plan/task-16-doors-summons-reactors-report.md`

Seven packages: `atlas-doors/data/skill`, `atlas-doors/party`,
`atlas-summons/data/skill`, `atlas-summons/data/skill/effect`,
`atlas-reactors/reactor/data`, `atlas-reactors/reactor/data/area`,
`atlas-reactors/reactor/data/state`.

## Charge 1 — Commit contents match declared scope

`git show --stat` on each commit:

- `5a292ffba`: touches only `services/atlas-doors/.../data/skill/{rest.go,rest_test.go}`
  and `services/atlas-doors/.../party/{rest.go,rest_test.go}`. 4 files, +87/-0.
- `2bac98f26`: touches only `services/atlas-summons/.../data/skill/{rest.go,rest_test.go}`
  and `services/atlas-summons/.../data/skill/effect/{rest.go,rest_test.go}`. 4 files, +131/-0.
  Both atlas-summons packages (`data/skill`, `data/skill/effect`) fully present.
  No foreign paths, no atlas-summons work lost to the concurrent agent's
  stage/unstage incident.
- `e4d483d9b`: touches only `services/atlas-reactors/.../reactor/data/{rest.go,rest_test.go}`,
  `.../reactor/data/area/{rest.go,rest_test.go}`, `.../reactor/data/state/{rest.go,rest_test.go}`.
  6 files, +200/-0.

No commit touches a path outside its own declared service. All three are
purely additive (`+N/-0`), satisfying charges 1 and 6 together.

## Charge 2/3 — Field coverage and faithfulness to Extract (per package)

### `atlas-doors/data/skill` (`services/atlas-doors/atlas.com/doors/data/skill/`)

`Model` (model.go): `id skill.Id`, `effects []effect.Model`.
`Extract` (rest.go): `id: rm.Id`, `effects: es` (via `effect.Extract`).
`Transform` (rest.go): `Id: m.id`, `Effects: es` (via `effect.Transform`).
Both fields covered, no borrowed shape — matches this package's own `Extract`, not the look-alike `atlas-summons/data/skill`. PASS.

### `atlas-doors/party` (`services/atlas-doors/atlas.com/doors/party/`)

`Model`: `id uint32`, `leaderId character.Id`, `members []character.Id`.
`Extract` (rest.go, largest in batch): reads `rm.Id`, `rm.LeaderId`, and loops
`rm.Members` extracting only `m.Id` from each `MemberRestModel` (which itself
carries only an `Id` field — no other attribute exists to lose). `Transform`
mirrors this exactly: `Id: m.id`, `LeaderId: m.leaderId`, and a loop rebuilding
`[]MemberRestModel{Id: id}` in the same (unsorted) order. No field is invented;
`MemberRestModel` has nothing beyond `Id` for `Extract` to have dropped, so
there is no lossy field here despite this being the largest `Extract` in the
batch. PASS.

### `atlas-summons/data/skill` (`services/atlas-summons/atlas.com/summons/data/skill/`)

`Model`: `id uint32`, `action bool`, `element string`, `animationTime uint32`,
`effects []effect.Model`. `Extract`/`Transform` both cover all five fields
symmetrically. Distinct field set from `atlas-doors/data/skill` (no borrowing
evident — this package's `Transform` has no `skill.Id` typed field or any
doors-specific shape). PASS.

### `atlas-summons/data/skill/effect` (`services/atlas-summons/atlas.com/summons/data/skill/effect/`)

`Model`: `weaponAttack`, `magicAttack`, `hp`, `duration`, `x`, `y`, `prop`,
`monsterStatus`, `statups []StatChange`. `Extract`/`Transform` cover all nine
fields. `Transform` reads the unexported `m.hp` (`uint16`) directly rather
than through the narrowing `Hp() int16` getter, so no truncation is
introduced by the new code — confirmed at `effect/rest.go` (Transform assigns
`Hp: m.hp`) vs `effect/model.go:39` (`Hp() int16` getter, unused by
`Transform`). Distinct shape from `atlas-doors/data/skill/effect` (different
field set, not checked here since out of this batch's scope) and from
`atlas-reactors` — no evidence of borrowing. PASS.

### `atlas-reactors/reactor/data/area` (`services/atlas-reactors/atlas.com/reactors/reactor/data/area/`)

`Model`: `tl point.Model`, `br point.Model`. `Extract`/`Transform` both call
`point.Extract`/`point.Transform` on both fields. `point.Transform` predates
this batch and is used, not touched. PASS.

### `atlas-reactors/reactor/data/state` (`services/atlas-reactors/atlas.com/reactors/reactor/data/state/`)

`Model`: `theType int32`, `reactorItem *item.Model` (nilable), `activeSkills
[]uint32`, `nextState int8`. `Extract` guards `rm.ReactorItem != nil` before
calling `item.Extract`; `Transform` mirrors with `m.reactorItem != nil` before
`item.Transform`. All four fields covered symmetrically, nil-pointer case
correctly handled in both directions. PASS.

### `atlas-reactors/reactor/data` (top-level, `services/atlas-reactors/atlas.com/reactors/reactor/data/`)

`Model`: `name`, `tl`, `br`, `activateByTouch`, `touchAreaInfo
map[int8]area.Model`, `stateInfo map[int8][]state.Model`, `timeoutInfo
map[int8]int32`, `timeoutNextStateInfo map[int8]int8`. `Extract` guards
`rm.TouchAreaInfo != nil` before building the map (an absent map stays nil,
not becoming `{}`); `Transform` mirrors with `if m.touchAreaInfo != nil`.
`stateInfo` and the two int-keyed maps pass straight through in both
directions. `RestModel.Id` is a REST-only field that `Extract` never reads
into `Model` (there is no `Model.id`); `Transform` correctly leaves
`RestModel.Id` unset rather than inventing a value — consistent with
`Extract`'s behavior, not a gap. PASS.

No sibling-package analogy was used to derive any of the above; each field
list was read from the package's own `model.go`.

## Charge 4 — The lossy claim

Checked each package's `Model` fields against what `Extract` can restore from
`RestModel`. In all seven packages, every `Model` field is populated from a
`RestModel` field that `Extract` reads, and `Transform` reconstructs that
`RestModel` field from the same `Model` field. No `Model` field is dropped
by `Extract`. `atlas-doors/party`'s `Extract` (the largest, flagged in the
brief for likely nested-member loss) does not, in fact, lose anything: the
only member attribute in play (`character.Id`) round-trips fully. The
implementer's "no lossy fields, no handwork-notes.md entry" claim holds for
all seven packages. Confirmed no `handwork-notes.md` entries were added for
this batch (`git show --stat` above lists no such file in any of the three
commits).

## Charge 5 — No behavior change outside the addition

All three commits are `+N/-0` (pure additions, confirmed by `git show
--stat`). No `Extract` body, `Build()` validation rule, or `RestModel` field
set was touched — every `Extract`/`RestModel` definition shown above is the
pre-existing shape with only a new `Transform` function added beside it.
None of these seven packages use a validating builder chain for `Extract`
(all are plain struct construction / loops), so FR-17 was never in play, as
the report states.

## Charge 6 — No file overwritten

Covered above: all three commits show `+N/-0`, i.e. zero deletions anywhere.
For the four packages whose `rest_test.go` pre-existed (`atlas-doors/data/skill`,
`atlas-doors/party`, `atlas-reactors/reactor/data`), the diff stat shows only
line additions to those files and the pre-existing tests
(`TestGetEffect_Served`, `TestGetEffectLevelIndexing`,
`TestGetEffectDurationSentinel`, `TestGetByMemberId_ServedPreservesOrder`,
`TestExtractPreservesMemberOrder`, `TestExtractTouchFields`,
`TestModelJSONRoundTripTouchFields`) are still present in the files as read
directly. No pre-existing test was destroyed.

## Charge 7 — Tests actually constrain the code

Mutation test performed on `atlas-reactors/reactor/data/state`: changed
`Transform`'s `NextState: m.nextState` to `NextState: m.nextState + 1` in
`services/atlas-reactors/atlas.com/reactors/reactor/data/state/rest.go`, ran
`go test ./reactor/data/state/... -run TestTransformRoundTrip -v`:

```
--- FAIL: TestTransformRoundTrip (0.00s)
    rest_test.go:35: round trip mismatch. want {... nextState:2}, got {... nextState:3}
FAIL
```

Confirmed the test fails at field level on a single-field mutation, then
reverted with `git checkout -- reactor/data/state/rest.go` and confirmed
clean with `git status --short` — no residual diff in any
`atlas-doors`/`atlas-summons`/`atlas-reactors` path (only unrelated
in-progress files from other concurrent agents in other services, plus the
expected `handwork-notes.md`/`progress.md`/`agent-ledger.tsv` modifications,
appear in `git status --short`).

## Module gates

Ran `go build ./... && go vet ./... && go test ./...` from each of the three
module roots after the mutation-revert; all three exited clean with no
failures (output matches the report's claimed clean gates).

## Not evaluable

None — all seven packages, their `Extract`/`Transform` pairs, and their
round-trip tests were read directly and independently verified against each
package's own `model.go`.

## Verdict

All seven `Transform` implementations are faithful, field-complete inverses
of their own package's `Extract`, derived independently (no sibling-package
borrowing found). Commits are scoped correctly, purely additive, no file
overwritten, atlas-summons fully present despite the concurrent-agent
stage/unstage incident noted in the task. Round-trip tests genuinely
constrain the code (confirmed by mutation). No `handwork-notes.md` entry
needed; the "no lossy fields" claim is correct for this batch.
