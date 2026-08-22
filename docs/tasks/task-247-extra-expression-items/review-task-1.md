# Review: Task 1 — Emote → cash-item mapping constants

Range reviewed: `922da325f..4f4b32b9c` (single commit `4f4b32b9c`)
Module root: `libs/atlas-constants`

## Scope confirmed

`git log --oneline 922da325f..4f4b32b9c` shows exactly one commit, matching the
implementer report's claimed commit hash. `git diff --stat` shows only two new
files touched:

```
libs/atlas-constants/item/expression.go      | 37 ++++++++++++++++
libs/atlas-constants/item/expression_test.go | 64 ++++++++++++++++++++++++++++
```

No other files in the tree were modified (`git status --short` after checkout
shows only an untracked ledger file, unrelated to this task's diff). This
matches the Task 1 Files block exactly — no scope creep into `constants.go`,
no wire-format files touched, no `character_cash_item_use.go` touched, no
`*_testhelpers.go` files created.

## Requirement-by-requirement (brief `task-1-brief.md`)

1. **`MaxEmoteId = uint32(23)`, `MaxBaseEmoteId = uint32(7)`** — present at
   `libs/atlas-constants/item/expression.go:28,32` (diff line numbers), exact
   binding constants match the global constraint stated in the task prompt.
2. **`IsExtraExpressionEmote(emote uint32) bool`** — `expression.go:36-38`:
   `return emote > MaxBaseEmoteId && emote <= MaxEmoteId`. Correctly derives
   from the two constants; no new constant/type invented.
3. **`ExtraExpressionItemId(emote uint32) (Id, bool)`** — `expression.go:49-54`.
   Formula: `Id(uint32(ClassificationExpression)*10000 + emote - MaxBaseEmoteId - 1)`,
   guarded by `IsExtraExpressionEmote`. Reuses the existing
   `ClassificationExpression = Classification(516)` from `constants.go:96` (did
   not redefine it) — satisfies the "reuse atlas-constants" convention.
4. **Boundary table for `TestIsExtraExpressionEmote`** — verified all six rows
   present verbatim and passing (ran `go test ./item/... -v -run
   'TestIsExtraExpressionEmote|TestExtraExpressionItemId'`, all subtests PASS):
   - `zero`(0)→false, `base emote upper bound`(7)→false, `first extra`(8)→true,
     `last extra in v83 data`(22)→true, `gated upper bound`(23)→true,
     `above client cap`(24)→false. All match brief exactly.
5. **Boundary table for `TestExtraExpressionItemId`** — all five rows present
   and correct:
   - emote 8 → `Id(5160000)`, true: `516*10000 + 8-7-1 = 5160000+0`. Correct.
   - emote 22 → `Id(5160014)`, true: `5160000+22-8=5160014`. Correct.
   - emote 23 → `Id(5160015)`, true: `5160000+23-8=5160015`. Correct.
   - emote 7 → `Id(0)`, false (not gated). Correct.
   - emote 24 → `Id(0)`, false (above cap). Correct.
6. **`TestExtraExpressionItemIdClassification`** — asserts
   `item.GetClassification(id) == item.ClassificationExpression` for ids
   returned at emote 8 and 22. Verified against `GetClassification`'s actual
   implementation at `libs/atlas-constants/item/constants.go:144-146`
   (`Classification(math.Floor(float64(itemId) / 10000))`): `5160000/10000 =
   516`, `5160014/10000 = 516.0014 → floor 516`. Both resolve to
   `ClassificationExpression`. Test is not tautological — it exercises the real
   `GetClassification` function, not a hand-rolled duplicate.

## File-shape convention

Compared to `libs/atlas-constants/item/vegas_spell.go` /
`vegas_spell_test.go`:
- Doc comment cites the wz path (`Item.wz/Cash/0516.img`) and IDA offsets
  (`CWvsContext::SendEmotionChange@0x9f9386`, `CAvatar::SetEmotion@0x466b00`,
  `CWvsContext::SendEtcCashItemUseRequest@0xa02c86`,
  `CUserLocal::UseFuncKeyMapped case 3u@0x933874`), consistent with the
  vegas_spell.go pattern of citing source.
- Test file is package `item_test`, table-driven, one `t.Run` per case for
  both new tests — matches `vegas_spell_test.go` shape.
- The new file uses two functions plus two constants rather than
  `vegas_spell.go`'s "named ids + one predicate" shape, but this is inherent
  to the requirement (a formulaic emote→id mapping, not an enumerated set of
  ids) and is exactly what the brief's Step 3 code dictated verbatim — not an
  implementer deviation.

## Test honesty

The tests exercise real production code (`item.IsExtraExpressionEmote`,
`item.ExtraExpressionItemId`, `item.GetClassification`) with no stub/duplicate
logic; before `expression.go` existed the package would not compile
(`undefined: item.IsExtraExpressionEmote` per the report), so these are
genuine RED→GREEN tests, confirmed independently by re-running them.

## IDA/offset citations

I did not independently re-verify the cited IDA offsets
(`CWvsContext::SendEmotionChange@0x9f9386`, `CAvatar::SetEmotion@0x466b00`,
`CWvsContext::SendEtcCashItemUseRequest@0xa02c86`,
`CUserLocal::UseFuncKeyMapped case 3u@0x933874`) against the binary — this is
outside this unit's review surface (a pure constants/math change with no
runtime dependency on the offsets being correct beyond documentation value)
and is reported under Not evaluable rather than silently accepted.

## Verification run

```
cd libs/atlas-constants && go build ./... && go test ./item/... -v -run 'TestIsExtraExpressionEmote|TestExtraExpressionItemId'
```
All subtests PASS (quoted output captured above). Consistent with the
implementer's report; no discrepancy found, so I did not re-run the full
`go test ./...` for the whole module (not required — diff is isolated to the
`item` package with no other package touched).

## Not evaluable

- The IDA/offset citations in the doc comments
  (`CWvsContext::SendEmotionChange@0x9f9386`, `CAvatar::SetEmotion@0x466b00`,
  `CWvsContext::SendEtcCashItemUseRequest@0xa02c86`,
  `CUserLocal::UseFuncKeyMapped case 3u@0x933874`) were not independently
  re-verified against the client binary; accuracy of the *documentation prose*
  is unconfirmed, though it has no bearing on runtime correctness of the pure
  Go arithmetic, which was verified.

## Verdict

APPROVED. Every brief requirement, boundary-table row, and repo convention
constraint checked out with cited evidence; the two files are additive and
in-scope; the one gap (offset citation accuracy) is documentation-only and
does not affect correctness of the shipped constants/functions.
