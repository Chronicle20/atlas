# Task 15 review — `atlas-npc-shops` (tier B1, `data/consumable`)

Commit reviewed: `38810baa9` (range `056cd3b..38810baa9`)
Brief: `.superpowers/sdd/plan/task-15-brief-atlas-npc-shops.md` +
`.superpowers/sdd/plan/task-15-common.md`
Report: `.superpowers/sdd/plan/task-15-report-atlas-npc-shops.md`

## Scope

`git show --stat 38810baa9`:

```
services/atlas-npc-shops/atlas.com/npc/data/consumable/rest.go      | 88 +++++
services/atlas-npc-shops/atlas.com/npc/data/consumable/rest_test.go | 110 ++++
 2 files changed, 198 insertions(+)
```

Matches the brief exactly: one new file (`rest_test.go`) plus additions to
the existing `rest.go`. No sibling package
(`atlas-consumables`/`atlas-inventory`) touched by this commit (confirmed
`git diff 056cd3b..38810baa9 --stat -- services/atlas-consumables
services/atlas-inventory` is empty). No `docs/` file touched by this commit
(the `docs/tasks/...` entries visible in `git status` are controller
tracking artifacts from other in-flight work, not part of `38810baa9`).

## 1. "Byte-identical to atlas-inventory" claim — VERIFIED

Extracted and sorted the `Model`, `RewardModel`, and `RestModel` field lists
from both packages and diffed:

```
services/atlas-npc-shops/.../model.go      Model fields (sorted)
services/atlas-inventory/.../model.go      Model fields (sorted)
→ diff exit 0, "MODEL FIELDS IDENTICAL"

RewardModel struct bodies: byte-for-byte identical (itemId/count/prob,
same types, no tags — it's an unexported model, not a rest model).

RestModel fields (sorted): diff exit 0, "RESTMODEL FIELDS IDENTICAL"

RewardRestModel struct bodies: byte-for-byte identical
(ItemId/Count/Prob uint32, same json tags).
```

Confirmed field-for-field, not just count — 57 `Model` fields, 3
`RewardModel` fields, both packages match exactly. No divergence like the
`atlas-consumables` case (extra `effect`/`worldMsg`/`period` on
`RewardModel`) exists here. The implementer's claim is correct.

## 2. Transform*/Extract* are true inverses

`services/atlas-npc-shops/atlas.com/npc/data/consumable/rest.go:92-171`
(`Transform`) and `:172-253` (`Extract`), `:254-261`
(`TransformReward`)/`:262-269` (`ExtractReward`).

Programmatically enumerated all 57 `Model` struct fields and confirmed each
one is referenced (`m.<field>`) inside `Transform`'s composite literal — 57/57
present, none missing, none extra (no field emitted that `Extract` doesn't
also consume). `Model.HandsIncrease()` (a hardcoded `return 0` accessor, not
a backing field) is correctly absent from both `Transform` and `Extract`,
confirming the deliberate non-field distinction requested in the brief.

`TransformReward`/`ExtractReward` are simple 3-field (`itemId`/`count`/`prob`)
mirror mappings, confirmed inverse by inspection and by live mutation
(below).

`Summon`/`SummonModel` sub-mapping: `Transform` builds `[]Summon` from
`m.monsterSummons` via `templateId`/`probability`; `Extract` builds
`[]SummonModel` the same way via an inline closure passed to
`model.SliceMap`. Symmetric.

## 3. Test honesty — live mutation performed independently

Ran my own mutation (not the implementer's), anchored at
`rest.go:258` inside `TransformReward`:

```
-		Prob:   m.prob,
+		Prob:   m.count,
```

Result:

```
--- FAIL: TestTransformRoundTrip (0.00s)
    --- FAIL: TestTransformRoundTrip/Transform (0.00s)
        rest_test.go:85: round trip mismatch: ... rewards:[{... prob:2}]
                                             vs ... rewards:[{... prob:100}]
    --- FAIL: TestTransformRoundTrip/Reward (0.00s)
        rest_test.go:107: round trip mismatch: got={... prob:2} want={... prob:100}
```

Both subtests failed with a precise field-level diff, as expected — the
`Reward` subtest is not tautological, and the top-level `Transform` subtest
transitively exercises `TransformReward` through the `rewards` slice.
Reverted with `git checkout --`, confirmed `git diff --exit-code` on
`rest.go` clean, and re-ran the suite green (`PASS` x3: parent + 2 subtests
— this fully explains the report's "3 passed," there is no undisclosed
third pair, `go test -v` reports the parent `TestTransformRoundTrip` plus
its two `t.Run` subtests as 3 lines).

`go build ./...` and `go vet ./data/consumable/...` both clean after
revert.

## 4. Delegated `Extract*` outside `rest.go`

```
grep -rn '^func Extract' services/atlas-npc-shops/atlas.com/npc/data/consumable/
→ rest.go:172: func Extract(...)
→ rest.go:262: func ExtractReward(...)
```

Both `Extract*` functions are declared and defined in `rest.go` itself —
no delegation to another file, no undiscovered third `Extract*`. Matches
the brief's 2-pair table exactly.

## 5. Fixture quality

`rest_test.go:11-73` (`Model` fixture): every one of the 57 fields set to a
distinct, non-zero-default value — all booleans `true` (not the zero
value `false`), all numeric fields distinct non-zero integers/floats
(`unitPrice: 3`, `bridlePropChg: 1.5`), strings non-empty
(`delayMsg: "delay-msg"`, `script: "some_script"`), `spec`/`morphs` maps
populated with 2 entries each, `monsterSummons` a 2-element slice,
`skills` a 2-element slice, `rewards` a 1-element slice referencing a
distinct `RewardModel`. `left`/`top`/`incFatigue` deliberately negative to
exercise the signed `int32` fields.

`rest_test.go:88-92` (`RewardModel` fixture, `Reward` subtest): `itemId:
1132010, count: 2, prob: 100` — all distinct, non-zero, non-default.

No pointer-returning `Extract*` in this pair set (`Extract`/`ExtractReward`
both return value types), so the pointer-nil check is not applicable here.

## 6. Scope discipline

- No accessor methods minted on `Model`/`RewardModel` — `Transform`/
  `TransformReward` read unexported fields directly (`m.id`, `m.itemId`,
  etc.), consistent with D1 and the existing `Extract`/`ExtractReward`
  style already in the file.
- No `docs/` file touched by this commit.
- No changes outside `services/atlas-npc-shops`; siblings
  `atlas-consumables` and `atlas-inventory` are untouched (verified via
  `git diff` scoped to those paths across the range, empty output).

## Verdict rationale

All six review-brief items check out with direct, reproduced evidence:
field-identity claim confirmed by independent diff, both `Transform*`
functions verified as complete/symmetric inverses by field enumeration, one
live mutation reproduced and reverted cleanly, no rogue `Extract*`, fixtures
are non-default identities, and scope is exactly the two files the brief
specified.

No blocking or non-blocking findings.

## Not evaluable

None — the full review surface (both file diffs, the sibling package used
for the identity comparison) was inspected directly.
