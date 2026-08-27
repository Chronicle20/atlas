# Review — Task 15 batch `atlas-party-quests`

Commit reviewed: `4d3ab3b308d61dd840f52a35b1d1361288527096` — "feat(atlas-party-quests): add
Transform inverses for party with round-trip tests"

Brief: `.superpowers/sdd/plan/task-15-brief-atlas-party-quests.md` +
`.superpowers/sdd/plan/task-15-common.md`
Report: `.superpowers/sdd/plan/task-15-report-atlas-party-quests.md`

## Scope

`git show --stat 4d3ab3b` confirms exactly two files touched:

```
services/atlas-party-quests/atlas.com/party-quests/party/rest.go       | 21 +++++++++
services/atlas-party-quests/atlas.com/party-quests/party/rest_test.go  | 54 ++++++++++++++++++++++
```

Matches the brief's `### Files` inventory exactly. No sibling `party` package (checked all 9:
`atlas-channel`, `atlas-doors`, `atlas-drops`, `atlas-guilds`, `atlas-monster-death`,
`atlas-parties`, `atlas-party-quests`, `atlas-query-aggregator`, `atlas-saga-orchestrator`) was
modified. No `docs/` file was touched by this commit.

## Findings

### 1. Field inventory — PASS

`Model` (`party/model.go`) has three fields: `id`, `leaderId`, `members []MemberModel`. All three
are read directly (D1, no minted accessors) in `Transform` (`rest.go:93-104`) and map onto
`RestModel{Id, LeaderId, Members}`.

`MemberModel` has three fields: `id`, `worldId`, `channelId`. All three are mapped in
`TransformMember` (`rest.go:142-148`) onto `MemberRestModel{Id, WorldId, ChannelId}`.

`MemberRestModel` additionally carries `Name`, `Level`, `JobId`, `MapId`, `Online` — but
`MemberModel` has no backing fields for these at all, and the paired `ExtractMember`
(`rest.go:134-140`, pre-existing, unmodified) never read them either. Per the brief's mapped-fields
rule ("a field the `Extract` does not map must not be emitted by the `Transform`"), their absence
from `TransformMember` is correct, not a gap — confirmed against `ExtractMember`'s body, not
assumed by equivalence with a sibling package (per the review brief's warning about the disproved
atlas-consumables equivalence assumption).

No embedded `field.Model` or similar composite exists in this package — `WorldId()`/`ChannelId()`
here are plain scalar accessors on `MemberModel` itself, not a shared embedded type.

### 2. Hardcoded accessor values — PASS

`model.go` has no accessor returning a constant; all three (`Id()`, `LeaderId()`, `Members()`) on
`Model`, and `Id()`, `WorldId()`, `ChannelId()` on `MemberModel`, return the corresponding stored
field. `Transform`/`TransformMember`/`Extract`/`ExtractMember` mirror this — nothing hardcoded on
either side.

### 3. Live mutation (independent of the implementer's own proof) — PASS

The implementer's own mutation proof (in their report) dropped `LeaderId` from `Transform`. I
independently mutated a field their proof did not touch — `WorldId` in `TransformMember`
(`rest.go:142-148`), dropping the `WorldId: m.worldId,` line — and re-ran:

```
$ go test ./party/... -run TestTransformRoundTrip -v
=== RUN   TestTransformRoundTrip/Party
    rest_test.go:32: round trip mismatch: got {... members:[{id:3003 worldId:0 channelId:2}]}, want {... members:[{id:3003 worldId:1 channelId:2}]}
=== RUN   TestTransformRoundTrip/Member
    rest_test.go:51: round trip mismatch: got {id:4004 worldId:0 channelId:6}, want {id:4004 worldId:5 channelId:6}
--- FAIL: TestTransformRoundTrip (0.00s)
```

Field-level diff confirmed (`worldId:0` vs `1`/`5`) — the test is not tautological. Reverted with a
precise, uniquely-anchored replace (full `TransformMember` function body) and confirmed:

```
$ git diff --exit-code -- services/atlas-party-quests/atlas.com/party-quests/party/rest.go services/atlas-party-quests/atlas.com/party-quests/party/rest_test.go
$ echo $?
0
```

Both touched files are byte-identical to the committed state after mutate/revert.

### 4. Fixtures use distinct non-default values — PASS

`Party` subtest: `id:1001, leaderId:2002, members[0]{id:3003, worldId:1, channelId:2}` — all
distinct, all non-zero.
`Member` subtest: `id:4004, worldId:5, channelId:6` — all distinct, all non-zero.

### 5. `docs/` untouched, no exemption note, no sibling package touched — PASS

Verified in Scope section above.

### 6. `Extract*` inventory completeness — PASS

```
$ grep -rn '^func Extract' services/atlas-party-quests/atlas.com/party-quests/party/
rest.go:77:func Extract(rm RestModel) (Model, error) {
rest.go:134:func ExtractMember(rm MemberRestModel) (MemberModel, error) {

$ grep -rn '^func Transform' services/atlas-party-quests/atlas.com/party-quests/party/
rest.go:93:func Transform(m Model) (RestModel, error) {
rest.go:142:func TransformMember(m MemberModel) MemberRestModel {
```

Both `Extract*` functions have an inverse; no `Extract*` was found outside `rest.go`.

### Naming / signature conventions — PASS

`Transform(Model) (RestModel, error)` mirrors `Extract`'s error-returning signature even though the
current body never errors, consistent with the pattern established in the sibling batches (e.g.
`atlas-ban`). `TransformMember(MemberModel) MemberRestModel` is declared error-free since
`ExtractMember`'s mapping is trivial (unmarshal-only) and the brief explicitly permits an
error-free `Transform<X>` "where a mapping has no failure path" — correctly not adding a
permanently-nil error return.

### Report honesty — PASS

The implementer's report's RED transcript, mutation proof, and final verification section all
match what I independently reproduced (mutation on a different field, same outcome pattern:
field-level failure, clean revert).

## Anomaly (not part of this review's scope)

During review, `git status` showed `services/atlas-trades/atlas.com/trades/data/inventory/rest_test.go`
modified in the working tree. This is **not** touched by commit `4d3ab3b` and was not modified by
any action I took (my only file operations were scoped to
`services/atlas-party-quests/.../party/rest.go`, mutated and reverted with a verified clean diff).
It appears to be either pre-existing worktree state from unrelated concurrent work or a residual
artifact of another agent's session sharing this worktree. Flagging for the controller's awareness
— it is outside this unit's review surface and does not affect the verdict below.

## Not evaluable

None. The full review surface (both touched files, the `Model`/`RestModel`/`MemberModel`/
`MemberRestModel` field inventory, and the sibling-package/docs scope check) was directly
verified.

## Verdict

APPROVED. All six required checks pass with direct evidence; the round-trip tests are proven
non-tautological by an independently-chosen field mutation, and no scope, naming, or convention
defect was found.
