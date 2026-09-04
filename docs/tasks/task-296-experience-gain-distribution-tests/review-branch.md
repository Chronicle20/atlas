# task-296 experience-gain-distribution-tests — Branch-level review (Task 6, Step 2)

Range reviewed: `25ba8d1cb..HEAD` (HEAD = `c4fc10ab7`), covering implementation
commits `dc9c7707e`, `fffe4fdde`, `0159742ad`, `bdfb6c239`, `c4fc10ab7`.

Scope: the seam view across the whole branch — behavioral equivalence of the
final switch against the original if/else chain, whether the test table
covers every switch arm and every `AllExperienceDistributionTypes` entry, and
whether the extract-then-convert sequence left dead code or stale comments.
Per-task reviews (Tasks 1-5) already passed and are not re-litigated here
except where the seam view surfaces something they could not see.

## 1. Behavioral equivalence: switch vs. original if/else chain

Compared `git show 25ba8d1cb:.../consumer.go` (the pre-branch
`announceExperienceGain` inline chain, lines 369-404) against the final
`buildIncreaseExperienceConfig` switch in `consumer.go:367-433` on `HEAD`.

Arm-by-arm (type → right-hand side):

| Type | Original (`else if`) | Final (`switch` case) | Match |
|---|---|---|---|
| WHITE | `White=true; Amount=int32(Amount)` | same | yes |
| YELLOW | `White=false; Amount=int32(Amount)` | same | yes |
| CHAT | `InChat=true; Amount=int32(Amount)` | same | yes |
| MONSTER_BOOK | `MonsterBookBonus=int32(Amount)` | same | yes |
| MONSTER_EVENT | `MobEventBonusPercentage=byte(Amount)` | same | yes |
| PLAY_TIME | `MobEventBonusPercentage=byte(Amount); PlayTimeHour=byte(Attr1)` | same, both assignments retained in one case | yes |
| WEDDING | `WeddingBonusEXP=int32(Amount)` | same | yes |
| SPIRIT_WEEK | `QuestBonusRate=byte(Amount)` | same | yes |
| PARTY | `PartyBonusExp=int32(Amount); PartyBonusEventRate=byte(Attr1)` | same | yes |
| ITEM | `ItemBonusEXP=int32(Amount)` only — **no** `Amount` write | same — only `ItemBonusEXP`, `Amount` untouched | yes, and this is exactly the task-277 trap the switch's comment documents |
| INTERNET_CAFE | `PremiumIPExp=int32(Amount)` | same | yes |
| RAINBOW_WEEK | `RainbowWeekEventEXP=int32(Amount)` | same | yes |
| PARTY_RING | `PartyEXPRingEXP=int32(Amount)` | same | yes |
| CAKE_PIE | `CakePieEventBonus=int32(Amount)` | same | yes |
| unknown type | falls through all `else if`s, no assignment | no `case` matches, no `default` clause, no assignment | yes — silent-ignore preserved (FR-6), confirmed in code: no `default:` exists in the switch (`consumer.go:370-430`) |

All 14 arms are semantically identical, same order, same conversions
(`int32(...)`/`byte(...)`), same operand (`d.Amount`/`d.Attr1`). Arm order
also matches (WHITE, YELLOW, CHAT, MONSTER_BOOK, MONSTER_EVENT, PLAY_TIME,
WEDDING, SPIRIT_WEEK, PARTY, ITEM, INTERNET_CAFE, RAINBOW_WEEK, PARTY_RING,
CAKE_PIE) — matches the plan's ordering contract (design §4.7) and the
`if/else` chain's original order. The `c := model2.IncreaseExperienceConfig{}`
initialization and `return c` semantics are preserved (extracted into
`buildIncreaseExperienceConfig`, called once from `announceExperienceGain`,
`consumer.go:440`). The 17-argument `session.Announce` splat immediately
following is untouched, char for char, vs. the pre-branch version.

**PASS.** No behavioral drift introduced by the extract-then-convert
sequence. The ITEM-vs-Amount handling that task-277 got wrong is correctly
preserved (ITEM never touches `Amount`), and the mutation-testing record in
`progress.md` (Task 5, Step 3) independently demonstrates the test suite
would catch a regression of exactly that shape.

## 2. Test table coverage vs. final switch and `AllExperienceDistributionTypes`

`AllExperienceDistributionTypes` (`kafka/message/character/kafka.go:163-177`)
has exactly 14 entries, one per constant, in declaration order. The switch in
`buildIncreaseExperienceConfig` has exactly 14 `case` arms, one per constant,
same order, no `default`.

`distributionMappingCases` (`consumer_test.go:215-385`) has 14 single-type
cases (one `types` entry each, one arm per constant, verified against the
value-scheme table) plus 5 multi-distribution/unknown-type cases with
`types: nil`. Ran the actual suite (not just read it):

```
go test ./kafka/consumer/character/... -run 'TestBuildIncreaseExperienceConfig|TestExperienceDistributionTypeExhaustiveness' -v
```

All 19 `TestBuildIncreaseExperienceConfig` subtests PASS, and
`TestExperienceDistributionTypeExhaustiveness` PASSes — confirming (at test
run time, not just static reading) that every entry in
`AllExperienceDistributionTypes` has exactly one covering case and vice
versa. Also ran `go vet ./kafka/...` (clean) and `gofmt -l` over all three
changed files (no output — no formatting).

**PASS.** Table coverage is exhaustive and verified live, both directions.

## 3. Dead code / stale comments from the extract-then-convert sequence

- `announceExperienceGain` retains its four-level curried signature and now
  calls `buildIncreaseExperienceConfig(distributions)` once
  (`consumer.go:440`); no leftover inline loop, no unused local.
- `grep` for `announceExperienceGain`/`buildIncreaseExperienceConfig` across
  the package shows exactly the expected two definition sites and two call
  sites (`handleStatusEventExperienceChanged` → `announceExperienceGain`;
  `announceExperienceGain` → `buildIncreaseExperienceConfig`; test →
  `buildIncreaseExperienceConfig`). No orphaned symbol.
- No stale doc comment: the function doc on `buildIncreaseExperienceConfig`
  (`consumer.go:362-366`) accurately describes the current pure,
  no-default-clause behavior; the per-case client-line comments were added
  fresh in the Task 4 conversion commit and match
  `socket/model/experience_status.go`'s field comments (spot-checked ITEM,
  PLAY_TIME, PARTY — all consistent with the plan's sourced wording).
- `git diff 25ba8d1cb..HEAD -- consumer.go` shows only the expected shape:
  new function inserted, old inline block replaced by a single call. No
  incidental changes to `handleStatusEventExperienceChanged`,
  `InitHandlers`, or any unrelated handler.
- `git status --porcelain` in the worktree shows no leftover mutation from
  Task 5's temporary edits — only pre-existing untracked review-task
  artifact files, none of which are `kafka.go` or `consumer.go`.

**PASS.** No dead code or unused symbols; no stale comments.

## Not evaluable

- Live client rendering of the per-case comment text ("right side, yellow:
  ...") was not and could not be verified from source alone — these are
  transcribed from `socket/model/experience_status.go`'s existing field
  comments (pre-branch, unmodified by this branch) and from the plan; nothing
  in this branch changes that mapping's truth value, so it is out of this
  unit's scope, not a defect.
- The `progress.md` mutation-testing capture (Task 5) was read and its
  content cross-checked against the current code's actual behavior (ITEM's
  no-`Amount`-write is confirmed live above), but the capture itself was
  produced by the implementer in an earlier session and not independently
  re-executed by this review beyond the two `-run` invocations shown above.
  Not a finding — the mutations were reverted and the working tree confirmed
  clean.

## Verdict rationale

The seam this review exists to check — the if/else-to-switch conversion
preserving behavior arm for arm, including the ITEM trap — holds. Test
coverage is exhaustive in both directions and independently verified by
running the suite, not just reading it. No dead code or stale artifacts from
the multi-commit extract-then-convert sequence. No blocking findings.
