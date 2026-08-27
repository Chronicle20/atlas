# Task 15 review — atlas-guilds (`party`)

Commit reviewed: `69d96c7ce` — "feat(atlas-guilds): add Transform inverses for party with round-trip tests"

Brief: `.superpowers/sdd/plan/task-15-brief-atlas-guilds.md` + `.superpowers/sdd/plan/task-15-common.md`
Report: `.superpowers/sdd/plan/task-15-report-atlas-guilds.md`

## Scope

Diff is exactly two files, both under `services/atlas-guilds/atlas.com/guilds/party/`:

```
services/atlas-guilds/atlas.com/guilds/party/rest.go      | 31 +++++++++
services/atlas-guilds/atlas.com/guilds/party/rest_test.go | 73 +++++++++++++++
2 files changed, 104 insertions(+)
```

No other file changed by this commit. No sibling `party` package touched, no `docs/` file touched.

## Findings

### 1. "3 passed" vs. 2 brief pairs — resolved, not a defect

The brief lists exactly 2 `Extract*`/`Transform*` pairs. The report's "3 passed in 2 packages" is
Go's own summary counting the parent `TestTransformRoundTrip` plus its two subtests (`Member`,
`Party`) — not an undisclosed third pair. Verified directly:

```
$ go test ./party/... -run TestTransformRoundTrip -v
=== RUN   TestTransformRoundTrip
=== RUN   TestTransformRoundTrip/Member
=== RUN   TestTransformRoundTrip/Party
--- PASS: TestTransformRoundTrip (0.00s)
    --- PASS: TestTransformRoundTrip/Member (0.00s)
    --- PASS: TestTransformRoundTrip/Party (0.00s)
PASS
```

`grep -rn '^func Extract\|^func Transform' services/atlas-guilds/atlas.com/guilds/party/` returns
exactly `Extract`, `ExtractMember`, `Transform`, `TransformMember` — 2 `Extract*`, 2 `Transform*`,
no extra pair hiding in a sibling file (unlike the `atlas-channel` `ExtractListing` case the common
brief warns about). PASS.

### 2. Field enumeration — direct read of `model.go` and `rest.go` (not an equivalence claim)

`Model` (`party/model.go:11-27`): `id`, `leaderId`, `members` — all 3 mapped by `Transform`
(`rest.go:130-134`).

`MemberModel` (`party/model.go:29-72`): `id`, `name`, `level`, `jobId`, `field` (`field.Model`,
embedded), `online` — all mapped by `TransformMember` (`rest.go:138-148`). `field` is decomposed via
`m.WorldId()`/`m.ChannelId()`/`m.MapId()`/`m.Field().Instance()`, which are the only accessors that
read `field.Model`'s components — this is the structural inverse of `ExtractMember`'s
`field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build()`
(`rest.go:115`).

`RestModel` (`rest.go:16-20`): `Id`, `LeaderId`, `Members` — all 3 consumed by `Extract`
(`rest.go:102-106`) and produced by `Transform`.

`MemberRestModel` (`rest.go:151-161`): `Id`, `Name`, `Level`, `JobId`, `WorldId`, `ChannelId`,
`MapId`, `Instance`, `Online` — all 9 consumed by `ExtractMember` and produced by `TransformMember`.

Every field on every one of the four types is covered in both directions. This was checked by
reading `party/model.go` and `party/rest.go` in full in this worktree, not by re-using an
equivalence claim from a sibling service's batch. PASS.

### 3. Hardcoded-value accessors

`MemberModel.WorldId()`, `.ChannelId()`, `.MapId()` (`model.go:62-72`) are pure delegations to
`m.Field()` — no hardcoded/fake values. No accessor on `Model` or `MemberModel` returns a constant.
Neither `Transform` nor `Extract` emits anything not backed by a real field read. PASS.

### 4. Independent live mutation

Mutated `TransformMember`'s `Level:` field (a field the implementer's own mutation proof did not
touch — implementer mutated `Instance`) from `m.Level()` to a hardcoded `0`:

```
Level:     0,
```

Re-ran `go test ./party/... -run TestTransformRoundTrip -v`: both subtests failed with a
field-level diff (`level:120` expected vs. `level:0` got, all other fields matching). Reverted with
a precise string replace anchored on unique surrounding lines, then confirmed:

```
$ git diff --exit-code -- services/atlas-guilds/atlas.com/guilds/party/rest.go
$ (no output, exit 0)
```

File is byte-identical to pre-mutation state. PASS — confirms the tests are not tautological on a
field the implementer didn't already probe.

### 5. Fixture distinctness

`MemberModel{id:42, name:"Bishop", level:120, jobId:2000, field: world=1/channel=2/map=100000000
+ uuid.New(), online:true}` and `Model{id:7, leaderId:42, members:[mm]}` — every mapped field is
non-zero, non-default, and each numeric field takes a distinct value (42 vs 7 vs 120 vs 2000 vs 1
vs 2 vs 100000000), so no accidental cross-field coincidence could mask a swapped mapping. `Instance`
uses `uuid.New()` (non-nil), matching the common brief's requirement to avoid a zero-value
false-positive. PASS.

### 6. No docs, no exemption note, no sibling package touched

`git show 69d96c7ce --name-only` lists only the two `atlas-guilds/.../party/` files. No `docs/`
path, no `handwork-notes.md` entry (correctly — B1 packages have a `RestModel`, no exemption
needed), no other service's `party` package. PASS.

### 7. No orphan `Extract*` without inverse

`grep -rn '^func Extract' services/atlas-guilds/atlas.com/guilds/party/` → `Extract`, `ExtractMember`
only; both have a paired `Transform`/`TransformMember`. PASS.

## Gate reproduction

```
$ go test ./party/... -run TestTransformRoundTrip -v      → PASS (3/3)
$ tools/lint.sh --check --fmt --go services/atlas-guilds/atlas.com/guilds   → lint.sh: OK
$ go build ./... && go vet ./... && go test ./...          → all packages ok (no failures)
```

`tools/verify.sh` correctly not run by the implementer (out of scope per the plan's fact block);
not run by this review either, consistent with per-unit review scope.

## Not evaluable

None — every check in the review brief was directly verifiable within this unit's diff and its
immediate package files (`model.go`, `rest.go`).

## Verdict

APPROVED. Both `Transform`/`TransformMember` are exact, fully-covering inverses of `Extract`/
`ExtractMember`; the round-trip tests are non-tautological (proven independently on a different
field than the implementer's own proof); fixtures are distinct and non-default; no scope creep into
docs or sibling packages; naming matches FR-2 exactly.
