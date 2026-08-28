# Task 15 review — atlas-consumables (`Transform`/`TransformReward`)

Commit reviewed: `4f82f105d` (range `e6ed5f642..4f82f105d`)
Brief: `.superpowers/sdd/plan/task-15-brief-atlas-consumables.md`
Shared procedure: `.superpowers/sdd/plan/task-15-common.md`
Report: `.superpowers/sdd/plan/task-15-report-atlas-consumables.md`

## Scope confirmed

`git show --stat 4f82f105d` touches exactly two files:

- `services/atlas-consumables/atlas.com/consumables/data/consumable/rest.go` (+91)
- `services/atlas-consumables/atlas.com/consumables/data/consumable/rest_test.go` (+112, -1)

No `docs/` file is part of this commit (Task 15 requires no exemption note; correct — this
package has a `RestModel`). `atlas-inventory`/`atlas-npc-shops` consumable packages are untouched
by this commit (a `rest_test.go` under `atlas-inventory` appears in the worktree's `git status`
as an untracked file, but it is not part of commit `4f82f105d` and belongs to a sibling batch's
concurrent work in the shared worktree — confirmed via `git show --stat 4f82f105d`).

## 1. Completeness sweep — `Extract*` outside `rest.go`

```
$ grep -n '^func Extract\|^func Transform' rest.go
92:func Transform(m Model) (RestModel, error)
172:func Extract(rm RestModel) (Model, error)
258:func TransformReward(m RewardModel) (RewardRestModel, error)
269:func ExtractReward(rm RewardRestModel) (RewardModel, error)
```

Confirmed no other `.go` file in the package directory declares `Extract*` (grep over `*.go` in
the package produced no additional matches). Both pairs from the brief's table are the complete
set — no delegated `Extract*` in a sibling file (unlike the `atlas-channel` precedent cited in the
brief).

## 2. `Transform`/`Extract` are true inverses — PASS

Read side by side (`rest.go:92-247`). Every field of `Model` (`model.go:47-104`) is present on
both sides:

`id, tradeBlock, price, unitPrice, slotMax, timeLimited, notSale, reqLevel, quest, only,
consumeOnPickup, success, cursed, create, masterLevel, reqSkillLevel, tradeAvailable,
noCancelMouse, pquest, left, right, top, bottom, bridleMsgType, bridleProp, bridlePropChg,
useDelay, delayMsg, incFatigue, npc, script, runOnPickup, monsterBook, monsterId, bigSize,
tragetBlock (pre-existing typo, mirrored not "fixed" — correctly out of scope for this task),
effect, monsterHp, worldMsg, incPDD, incMDD, incACC, incMHP, incMMP, incPAD, incMAD, incJump,
incEVA, incLUK, incDEX, incINT, incSTR, incSpeed, spec, monsterSummons, morphs, skills, rewards`.

No field is mapped on one side and dropped on the other. `spec` and `morphs` (maps) and `skills`
(slice) are passed through by direct reference on both sides — consistent with how `Extract`
already treated them; no normalization exists to lose.

`monsterSummons`/`Summon` ↔ `[]SummonModel`: `Transform` builds `Summon{TemplateId, Probability}`
element-wise (`rest.go:102-108`); `Extract` builds `SummonModel{templateId, probability}`
element-wise via `model.SliceMap` (`rest.go:177-182`). Both fields round-trip.

`rewards`/`Rewards` ↔ `[]RewardModel`: `Transform` delegates to `TransformReward` per element
(`rest.go:93-100`); `Extract` delegates to `ExtractReward` via `model.SliceMap`
(`rest.go:173-176`). Consistent, delegated correctly.

## 3. `TransformReward`/`ExtractReward` are true inverses — PASS

`rest.go:258-278`. `RewardModel` fields (`model.go:257-263`): `itemId, count, prob, effect,
worldMsg, period`. All six are mapped on both sides, no additions or omissions.

## 4. Mutation evidence — independently reproduced (both pairs)

Report claims a single mutation on `Transform`'s `TargetBlock` field. Per the reviewer brief I ran
my own independent mutations for **both** pairs, using precise, uniquely-anchored edits (Python
string replace with an assertion the anchor occurs exactly once), not blind `sed`.

**Mutation 1 — `Transform`, field `incFatigue`:**

```diff
-		IncFatigue:      m.incFatigue,
+		IncFatigue:      0,
```

Result:

```
=== RUN   TestTransformRoundTrip/Transform
    rest_test.go:85: round trip mismatch:
         got  = {... incFatigue:0 ...}
         want = {... incFatigue:-3 ...}
--- FAIL: TestTransformRoundTrip (0.00s)
    --- FAIL: TestTransformRoundTrip/Transform (0.00s)
=== RUN   TestTransformRoundTrip/Reward
--- PASS
```

Field-level diff, isolated to the `Transform` subtest. Reverted; `git diff --exit-code -- rest.go`
clean.

**Mutation 2 — `TransformReward`, field `worldMsg`:**

```diff
-		WorldMsg: m.worldMsg,
+		WorldMsg: "",
```

Result: both subtests fail (`Transform` fails too, because `Transform` calls `TransformReward` on
its embedded reward), each with a field-level `worldMsg` diff — expected, since `TransformReward`
is exercised transitively inside `Transform`'s reward loop. Reverted; `git diff --exit-code --
rest.go` clean after both mutation/revert cycles.

Both pairs are demonstrated non-tautological with independently-reproduced evidence, not just the
report's transcript.

## 5. Fixtures are real identities — PASS

`rest_test.go:10-71` (`Model` fixture): every field distinct and non-zero, including negative
signed ints (`left:-1, top:-2, incFatigue:-3`), a non-default bool sentinel
(`tragetBlock: true`), non-empty strings, a 2-entry map for `spec`, `morphs`, a 2-element slice
for `monsterSummons`, `skills`, and a populated `rewards` slice. No `Extract`/`Transform`
normalization exists in this package that would require a non-default caveat (confirmed by
reading both bodies — no conditional defaulting).

`rest_test.go:90-97` (`RewardModel` fixture): all six fields distinct non-zero values.

Neither `Extract` nor `ExtractReward` returns a pointer, so the "pointer non-nil before deref"
rule does not apply here.

## 6. Scope discipline — PASS

- No accessor methods minted on `Model`/`RewardModel`/`SummonModel`; `Transform`/`TransformReward`
  read unexported fields directly (`m.id`, `m.tradeBlock`, ... `m.itemId`, etc.) — D1 compliant.
- No `docs/` file is part of commit `4f82f105d`.
- `git show --stat 4f82f105d` confirms nothing outside
  `services/atlas-consumables/atlas.com/consumables/data/consumable/{rest.go,rest_test.go}` was
  touched. `atlas-inventory`/`atlas-npc-shops` consumable packages are untouched by this commit.
- Pre-existing tests `TestExtractRewardFields` and `TestExtractPropagatesRewardsToModel` are
  byte-identical to the pre-commit version (diffed against `e6ed5f642`); only `TestTransformRoundTrip`
  was added.

## 7. Module-local gate — reproduced

```
$ tools/lint.sh --check --fmt --go services/atlas-consumables/atlas.com/consumables
lint.sh: OK

$ go test ./data/consumable/... -run TestTransformRoundTrip -v
--- PASS: TestTransformRoundTrip (0.00s)
    --- PASS: TestTransformRoundTrip/Transform (0.00s)
    --- PASS: TestTransformRoundTrip/Reward (0.00s)
PASS
```

`go build ./... && go vet ./...` not independently re-run in full (report shows clean output and
no source outside the two reviewed files changed); no reason to doubt it given the diff is
additive and self-contained.

## Findings

No blocking findings. No non-blocking findings.

## Not evaluable

None — full review surface (both `Extract`/`Transform` pairs, fixtures, mutation proof, scope)
was directly inspectable and independently verified.
