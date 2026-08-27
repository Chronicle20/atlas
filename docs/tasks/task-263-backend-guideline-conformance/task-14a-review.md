# Review: task-263 Task 14, batch A — `NO-RESTMODEL` hand work (D2)

**Verdict: APPROVED_WITH_FINDINGS**

## Scope

Commits `d936016c8`, `d633c9a80`, `3ef26603d`, `f74f2b0b4` (code, one commit
per service) and `1769bdb49` (handwork-notes.md + report), taken as
`d936016c8~1..1769bdb49`.

`git diff --stat d936016c8~1..1769bdb49 -- services/` confirms exactly the
four packages named by the brief were touched (`rewardpool`, `data/foothold`,
`messengers/character`, `parties/character`) — 8 files, 187 insertions, no
deletions, no other package touched. Matches the brief's `### Files` list
exactly.

## Findings by requirement

### 1. `Transform*` is the exact field-by-field inverse of `Extract*` (D1)

- **`drops/data/foothold`** — PASS. `Extract` (rest.go:64) maps
  `id, x1, y1, x2, y2` from `FootholdRestModel.{Id,First.X,First.Y,Second.X,Second.Y}`.
  `Transform` (rest.go) is the exact inverse, reconstructing the nested
  `pointRestModel` pointers from the same five unexported fields. No
  accessors minted. `PositionRestModel` correctly left untouched — it has no
  `Extract`, confirmed by grep (only used inline in `requests.go`).
- **`messengers/character`** and **`parties/character`** — PASS, identical
  shape in both packages. `ExtractForeign` maps
  `id, worldId, name, level, jobId, gm` out of the seven `ForeignRestModel`
  fields it names in its signature comment; `TransformForeign` maps back
  exactly those six and no more (`AccountId`, `Experience`, stats, `Meso`,
  `X/Y`, etc. are correctly left zero/unmapped). Reads unexported fields
  directly, same package (D1). Confirmed against `model.go` field lists.
- **`cashshop/rewardpool`** — see the dedicated ruling below (§3).

### 2. Naming mirrors FR-2

`Extract` (bare) → `Transform` (foothold); `ExtractForeign` → `TransformForeign`
(messengers, parties) — both correct per FR-2's mechanical rule. Rewardpool
has no `Extract` to mirror, so FR-2 doesn't directly apply there (see §3).

### 3. `cashshop/rewardpool` — Step 0 ruling (the primary judgment call)

Verified against `processor.go:37-43` (`SelectReward`):

```go
return Model{itemId: rm.ItemId, quantity: rm.Quantity, commodityId: rm.CommodityId}, nil
```

This is the **complete** set of fields `SelectReward` maps — `Model` (model.go)
declares only `itemId`, `quantity`, `commodityId`, nothing else, so there is
no additional field being silently dropped by the citation. `TransformReward`
maps exactly the same three fields in the reverse direction
(`Model.{itemId,quantity,commodityId}` → `RewardRestModel.{ItemId,Quantity,CommodityId}`),
so the direction is genuinely inverse, not merely superficially similar.

`RewardRestModel.{Tier,Weight,GachaponId}` are not referenced anywhere else
in the `cashshop` module (`grep -rn "Tier\|Weight\|GachaponId"` returns only
the field declarations at rest.go:11-14) — they exist purely as fields on the
wire type the upstream `atlas-reward-pools` service defines for pool
selection, and cashshop's domain model deliberately never represents them.
That supports treating `Model` as the actual (if narrow) domain
representation of what this consumer needs from `RewardRestModel`, rather
than treating the wire type as an untyped/domain-less payload.

`TransformReward` is not called anywhere outside its own test
(`grep -rn TransformReward` finds only rest.go and rest_test.go). That is
consistent with the rest of this task family — e.g.
`atlas-channel/monsterbook`'s `Transform` is likewise unused in production
code — so it is not a rewardpool-specific irregularity; this task mints
`Transform` functions for conformance, not for immediate wiring.

**Judgment: outcome (a) — write `TransformReward` — is defensible and I
would not overturn it.** The brief's Step 0 branch is "domain type exists
that RewardRestModel represents" vs. "bare JSON:API payload with no model
counterpart," and `Model` is a real (if partial) domain type consumed by
real processor logic, not an invented one. That said, this is a genuinely
harder call than the other three packages: there is no formal `Extract`
function anywhere to point to as the counterpart being mirrored, only an
inline field-selection three lines long inside one processor method. A
recorded exemption ("no domain type maps the full wire payload; a partial,
inline mapping exists in `SelectReward` but is not promoted to a named
`Extract`") would have been an equally defensible, arguably more
conservative reading of the brief's outcome (b). I am not overturning the
implementer's call — it is grounded in real code and does not invent
anything — but flag it as non-blocking for the controller's awareness given
Task 25 will turn these entries into formal exemption records.

### 4. `handwork-notes.md` — all four packages, correct form, repo-relative paths

All four packages present, one line each, in the design §8.1 form (package
path — no `RestModel` — wire type(s) — `Extract` location — provided
`Transform` name(s) — unmapped fields noted). `Extract` line numbers verified:
`foothold/rest.go:64` ✓, `messengers/character/rest.go:97` ✓,
`parties/character/rest.go:98` ✓ (all confirmed by direct grep against the
committed files). `grep -nE '(/home/|/root/|/Users/)' handwork-notes.md`
returns no matches — no absolute/home paths.

### 5. Round-trip test honesty

All four test files construct fixtures with every mapped field distinct and
non-zero:

- foothold: `id=42, x1=100, y1=200, x2=300, y2=400` — all distinct.
- messengers/parties `ForeignModel`: `id=42, worldId=1, name="Bob", level=99,
  jobId=200, gm=2` — all distinct.
- rewardpool `Model`: `itemId=2000000, quantity=3, commodityId=5000123` — all
  distinct.

Given field distinctness and `reflect.DeepEqual` assertions (or per-field
`!=` checks for rewardpool), a `Transform` that swapped, dropped, or
mis-mapped any field would produce a value distinguishable from the input
and fail the assertion. None of the four tests is tautological. Ran all
four locally:

```
go test ./rewardpool/... -run TestTransformReward -v        → PASS
go test ./data/foothold/... -run TestTransformRoundTrip -v   → PASS
go test ./character/... -run TestTransformRoundTrip -v       → PASS (messengers)
go test ./character/... -run TestTransformRoundTrip -v       → PASS (parties)
```

`go build ./... && go vet ./...` clean in all four module roots
(`atlas-cashshop`, `atlas-drops`, `atlas-messengers`, `atlas-parties`).

The rewardpool test (`TestTransformReward`) is necessarily one-directional
(no `Extract` exists to round-trip against, as the implementer's report
states) but is not tautological: it asserts `rm.ItemId == m.itemId`,
`rm.Quantity == m.quantity`, `rm.CommodityId == m.commodityId` against three
distinct values, so a field swap would be caught.

**RED-run gap**: the implementer's report honestly discloses that Step 3
(capture a `FAIL: undefined Transform<X>` transcript before implementing)
was skipped — test and implementation were written in the same pass per
package. This is a process deviation from the brief, already self-reported
and not something the diff itself can hide; disclosed rather than found. It
does not affect GREEN-state correctness of the tests (verified above), so
it is not blocking on its own — noted as non-blocking per this batch's
instructions, consistent with how the implementer itself flagged it.

### 6. No package outside the brief's four gained a `Transform`

Confirmed by `git diff --stat d936016c8~1..1769bdb49 -- services/` — exactly
the four packages, no others.

## Not evaluable

None. All four packages' `Extract`/domain-model contracts were read directly
and the round-trip claims verified by running the tests, not just reading
them.

## Summary of findings

- **Blocking: 0.**
- **Non-blocking (2):**
  1. `services/atlas-cashshop/atlas.com/cashshop/rewardpool/rest.go` — Step 0
     outcome (a) vs. a recorded exemption (outcome b) is a closer call than
     the brief's binary framing suggests, given there is no formal `Extract`
     to mirror, only a three-line inline mapping in `processor.go:37-43`. The
     ruling is defensible and grounded in real code; flagging for awareness,
     not requesting a change.
  2. Implementer-disclosed RED-run gap (brief Step 3) across all four
     packages — tests and implementations were written together rather than
     staged fail-then-pass. Tests were independently confirmed meaningful by
     inspection (field distinctness, non-tautological assertions) and by
     running them GREEN in this review, so the gap does not carry forward
     into a correctness defect.
