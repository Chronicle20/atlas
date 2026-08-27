# Task 15 review — `atlas-channel` tier-B1 batch (merchant, messenger, party)

Commit reviewed: `e6ed5f642` (range `4f6252b..e6ed5f642`)
Brief: `.superpowers/sdd/plan/task-15-brief-atlas-channel.md` + `.superpowers/sdd/plan/task-15-common.md`
Report: `.superpowers/sdd/plan/task-15-report-atlas-channel.md`

## Scope

`git diff --stat 4f6252b..e6ed5f642` touches exactly 6 files, all under
`services/atlas-channel/atlas.com/channel/{merchant,messenger,party}/`:

```
merchant/rest.go        | 67 +++
merchant/rest_test.go   | 91 +++
messenger/rest.go       | 27 +++
messenger/rest_test.go  | 68 +++
party/rest.go           | 31 +++
party/rest_test.go      | 75 +++
```

Matches the brief's inventory (3 pairs merchant, 2 messenger, 2 party = 7
`Transform*`). No `docs/` file in the commit. Scope confirmed.

## 1. Each `Transform<X>` is a true inverse of its paired `Extract<X>`

Read side by side for all 7 pairs.

- **merchant `Extract`/`Transform`** (`rest.go:116` / new `Transform`) — every
  field in `Model`/`RestModel` round-trips: `id.String()` <-> `uuid.Parse`,
  `world.Id(rm.WorldId)` <-> `byte(m.worldId)`, `instanceId` similarly,
  `messages`/`extractMessages`<->`transformMessages`, `listings` via
  `model.SliceMap` in both directions. PASS.
- **merchant `transformListing`/`ExtractListing`** (`merchant/listing.go:53`)
  — all 10 `ListingModel`/`ListingRestModel` fields mapped both ways
  including `Id`/`id` (unlike Blacklist/Visit, `ExtractListing` does map
  `Id`, and `transformListing` correctly emits it). PASS.
- **merchant `ExtractBlacklistName`/`TransformBlacklistName`**
  (`rest.go:309` / new) — `BlacklistRestModel{Id, Name}`; `Extract` reads
  only `Name`; `Transform` emits `BlacklistRestModel{Name: name}` and leaves
  `Id` zero-valued, matching the brief's explicit resolution and the
  self-review note (`rest.go:369-373` region). PASS — see item 2 below.
- **merchant `ExtractVisitEntry`/`TransformVisitEntry`** (`rest.go:314` /
  new) — same pattern (`VisitRestModel{Id, Name, Count}`, `Id` ignored by
  both). PASS.
- **messenger `Extract`/`Transform`** (`rest.go:106` / new) — `Model{id,
  members}` <-> `RestModel{Id, Members}`, member list transformed via
  `TransformMember` per element. PASS.
- **messenger `ExtractMember`/`TransformMember`** (`rest.go:122` / new) —
  all 6 `MemberModel`/`MemberRestModel` fields (`id, name, worldId,
  channelId, online, slot`) mapped both directions. PASS.
- **party `Extract`/`Transform`** (`rest.go:113` / new) — `Model{id,
  leaderId, members}` <-> `RestModel{Id, LeaderId, Members}`. PASS.
- **party `ExtractMember`/`TransformMember`** (`rest.go:130` / new) — Extract
  builds `field` via `field.NewBuilder(rm.WorldId, rm.ChannelId,
  rm.MapId).SetInstance(rm.Instance).Build()`; Transform reconstitutes via
  `m.field.WorldId()/ChannelId()/MapId()/Instance()`. All 9
  `MemberRestModel` fields (`Id, Name, Level, JobId, WorldId, ChannelId,
  MapId, Instance, Online`) covered. PASS.

No field found mapped by `Extract` and omitted by `Transform`, or vice versa,
across all 7 pairs.

## 2. `ExtractBlacklistName` bare-string round trip

`BlacklistRestModel` (`merchant/rest.go:337-340`) carries `Id` and `Name`.
`ExtractBlacklistName` (`rest.go:364-366`) reads only `rm.Name`.
`TransformBlacklistName` (new, `rest.go:369-373`) returns
`BlacklistRestModel{Name: name}` — `Id` is left as the zero value, not
invented. Same pattern confirmed for `VisitRestModel`/`VisitEntry`
(`Id` ignored on both sides). PASS — round trip is over the mapped fields
only, no extra `BlacklistRestModel`/`VisitRestModel` field leaks into the
comparison since the test round-trips only the bare `string`/`VisitEntry`.

## 3. Live mutation proofs (independent, not taken on the report's word)

Ran my own mutations, one per package, reverted via file restore from a
pre-mutation copy, confirmed `git diff --exit-code` clean afterward, and
confirmed the round-trip tests pass again post-revert.

- **merchant** `TransformBlacklistName`: changed `BlacklistRestModel{Name:
  name}` -> `BlacklistRestModel{Name: "mutated"}`. Result:
  `--- FAIL: TestTransformRoundTrip/BlacklistName` with
  `rest_test.go:89: round trip mismatch: got = mutated, want =
  nefarious-trader`. Sibling subtests `Transform`/`VisitEntry` still passed
  — proves the subtest is isolated and field-specific, not tautological.
- **messenger** `Transform`: changed `RestModel{Id: m.id, ...}` ->
  `RestModel{Id: m.id + 1, ...}`. Result: `--- FAIL:
  TestTransformRoundTrip/Transform` with `got = {id:1001 ...} want =
  {id:1000 ...}`; sibling `Member` subtest still passed.
- **party** `TransformMember`: changed `Name: m.name` -> `Name: "mutated"`.
  Result: both `Transform` and `Member` subtests failed (expected, since
  `Transform`'s member list goes through `TransformMember`), each showing a
  field-level diff (`name:mutated` vs `name:Leader`/`name:Member`/`name:Solo`
  as appropriate).

After each mutation: restored the file from a pre-mutation copy,
`git diff --exit-code -- merchant/rest.go messenger/rest.go party/rest.go`
returned clean (exit 0, output `CLEAN`), and
`go test ./merchant/... ./messenger/... ./party/... -run
TestTransformRoundTrip` passed again. All 3 sampled pairs are genuine,
non-tautological tests.

## 4. Working-tree cleanliness after the reported near-miss (priority item)

Read the **full** diff of all three `rest.go` files against `4f6252b`
(not just the stat). Every hunk in all three files is a pure insertion (`+`
lines only, 67/27/31 lines respectively matching the stat exactly) —
`git diff 4f6252b..e6ed5f642 -- .../merchant/rest.go
.../messenger/rest.go .../party/rest.go` shows zero `-` lines anywhere. In
particular the `party/rest.go` diff hunk starts cleanly at line 138 (right
after `ExtractMember`) and every added line is part of the new
`Transform`/`TransformMember` functions; the pre-existing
`SetToManyReferenceIDs` stub-member constructor (which contains the
`Level: 0,` literal the report's `sed` briefly corrupted) shows **no diff at
all** against `4f6252b`. The reported near-miss left no trace.

`git status --porcelain -- services/atlas-channel` returns empty. PASS —
independently confirmed clean.

## 5. Fixtures are real identities

- **merchant** `Transform` subtest (`merchant/rest_test.go:20-67`): every
  field distinct/non-zero — `worldId=3`, `channelId=2`, `shopType=1`,
  `y=-200` (negative, per the report's note), `characterId=30001`,
  `permitItemId=5060000`, `mesoBalance=123456`, `listingCount=3`,
  `visitors=[1,2,3]`, one message and one listing with distinct nested
  fields (`itemSnapshot` sub-struct populated with
  `Quantity/Owner/Flag`). No `Extract` in this batch normalizes/defaults a
  value (confirmed by reading all 7 `Extract*` bodies — none has an
  empty-to-constant or nil-to-true pattern), so the "no default match"
  requirement is trivially satisfied; the report's field-default note is
  accurate.
- **messenger**: two distinct members (`worldId 3/5`, `channelId 2/4`,
  `slot 1/2`, `online true/false`) plus a third distinct member in the
  `Member` subtest. Fields distinct across members; `online:false` is a
  legitimate value (not a defaulted omission, since `Extract`/`Transform`
  map booleans unconditionally either way).
- **party**: `level 120/95/200`, `jobId 112/200/412`, distinct
  `world/channel/map` ids across the `Member` subtest's fixture vs the
  `Transform` subtest's members, and a random `uuid.New()` instance in both.
  No pointer-returning `Extract*` in this batch (confirmed by signatures in
  the brief table — all three pairs are value types), so the
  nil-pointer-dereference concern in item 5 of the review brief does not
  apply to this batch.

## 6. Scope discipline

- No new accessor methods minted: all `Transform*` bodies read unexported
  struct fields directly within the same package (`m.shopType`, `m.name`,
  `m.field.WorldId()` — note `field.Field` is a value from
  `libs/atlas-constants/field`, not a new accessor added by this commit; its
  getters pre-exist and are used by `ExtractMember` already via the
  builder). No `D1` violation.
- No `docs/` file touched by commit `e6ed5f642` itself (the working tree's
  other pending `docs/` edits — `agent-ledger.tsv`, `progress.md`, plus two
  untracked review files — are batch-tracking artifacts from prior/later
  review turns, not part of this commit, and outside this batch's task
  scope; confirmed via `git show --stat e6ed5f642`).
- `git diff-tree --no-commit-id --name-only -r e6ed5f642` lists exactly the
  6 expected files under `services/atlas-channel/...`. Nothing outside
  `atlas-channel`.

## Build/test gate (informational, not the approval basis)

`go build ./... && go vet ./...` from
`services/atlas-channel/atlas.com/channel` exits 0 with no output. Full
`go test ./merchant/... ./messenger/... ./party/...` passes, including the
new `TestTransformRoundTrip` in each package. (Re-run independently, not
just quoted from the report.)

## Not evaluable

None — all checklist items in the review brief were directly verifiable
within the diff and its immediate callees (`merchant/listing.go` for
`ExtractListing`/`ListingModel`/`ListingRestModel`, read as a called
contract, not a scope expansion).

## Verdict rationale

All 7 `Transform*`/`Extract*` pairs are exact field-for-field inverses. The
bare-string `BlacklistRestModel` round trip is scoped correctly. Three
independently-run mutation proofs (spread across all three packages) each
produced a specific, field-level test failure and reverted to a
byte-identical file. The reported near-miss left no trace in the final
diff — confirmed by reading the complete diff of all three `rest.go` files,
not just the stat. Fixtures are non-default, distinct, non-zero. No scope
creep, no minted accessors, no docs touched.

No blocking or non-blocking findings.
