# Review: Task 3 — The mapping and exhaustiveness tests

Commit range: `fffe4fdde..0159742ad`
Commit: `0159742ad test(atlas-channel): lock EXPERIENCE_CHANGED distribution mapping and exhaustiveness`

## Scope

`git diff --stat fffe4fdde..0159742ad` shows exactly one file touched:

```
.../kafka/consumer/character/consumer_test.go | 221 +++++++++++++++++++++
1 file changed, 221 insertions(+)
```

`consumer.go` and `kafka.go` are unmodified in this range (confirmed by
`git diff --name-only`). Matches the plan's file-scope constraint.

## Findings

### 1. Case-table values vs. brief's fixed value scheme (PASS)

Checked each of the 14 single-type rows against
`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:367-406`
(`buildIncreaseExperienceConfig`) and the brief's table (`task-3-brief.md:71-84`),
value by value:

| Case | given Amount/Attr1 | want | Source branch (consumer.go) | Match |
|---|---|---|---|---|
| White | 1000/– | `{White:true, Amount:1000}` | `c.White=true; c.Amount=int32(d.Amount)` | yes |
| Yellow | 2000/– | `{Amount:2000}` | `c.White=false; c.Amount=...` | yes |
| Chat | 3000/– | `{InChat:true, Amount:3000}` | `c.InChat=true; c.Amount=...` | yes |
| MonsterBook | 4000/– | `{MonsterBookBonus:4000}` | `c.MonsterBookBonus=int32(d.Amount)` | yes |
| MonsterEvent | 11/– | `{MobEventBonusPercentage:11}` | `c.MobEventBonusPercentage=byte(d.Amount)` | yes |
| PlayTime | 22/33 | `{MobEventBonusPercentage:22, PlayTimeHour:33}` | both fields set from Amount/Attr1 | yes |
| Wedding | 5000/– | `{WeddingBonusEXP:5000}` | `c.WeddingBonusEXP=int32(d.Amount)` | yes |
| SpiritWeek | 55/– | `{QuestBonusRate:55}` | `c.QuestBonusRate=byte(d.Amount)` | yes |
| Party | 6000/44 | `{PartyBonusExp:6000, PartyBonusEventRate:44}` | both fields set | yes |
| Item | 7000/– | `{ItemBonusEXP:7000}` | `c.ItemBonusEXP=int32(d.Amount)` (Amount untouched — task-277 trap comment present) | yes |
| InternetCafe | 8000/– | `{PremiumIPExp:8000}` | `c.PremiumIPExp=int32(d.Amount)` | yes |
| RainbowWeek | 9000/– | `{RainbowWeekEventEXP:9000}` | `c.RainbowWeekEventEXP=int32(d.Amount)` | yes |
| PartyRing | 10000/– | `{PartyEXPRingEXP:10000}` | `c.PartyEXPRingEXP=int32(d.Amount)` | yes |
| CakePie | 11000/– | `{CakePieEventBonus:11000}` | `c.CakePieEventBonus=int32(d.Amount)` | yes |

All 14 rows are verbatim per the brief and consistent with the current
implementation's field-by-field arithmetic.

The 5 multi-distribution / unknown-type cases were hand-traced against the
`if/else` chain:
- `WhiteAndChat_PrimaryAwardShape`: WHITE(1500) sets White=true,Amount=1500;
  CHAT(1500) sets InChat=true,Amount=1500 (unchanged). Final `{White:true,
  InChat:true, Amount:1500}` — matches `want`.
- `PrimaryPlusBonuses_Accumulate`: WHITE(2500) → White/Amount; PARTY(600/66) →
  PartyBonusExp/PartyBonusEventRate; ITEM(770) → ItemBonusEXP. All four fields
  accumulate independently — matches `want`.
- `WhiteThenYellow_LastWins`: WHITE(1200) sets White=true,Amount=1200; then
  YELLOW(3400) sets White=false,Amount=3400. Final `{Amount:3400}` — matches
  `want`, and correctly demonstrates FR-8/FR-9 last-writer-wins.
- `EmptySlice_ZeroConfig`: nil slice, loop body never runs, zero-value struct
  returned — matches `want: {}`.
- `UnknownType_DeathIgnored`: "DEATH" matches no branch (silently dropped);
  WHITE(1300) then sets White=true,Amount=1300 — matches `want`.

### 2. `PartyBonusPercentage` / `QuestBonusRemainCount` always zero (PASS)

Grepped `buildIncreaseExperienceConfig` (`consumer.go:367-406`): neither field
is assigned in any branch. Confirmed no `want` struct literal in the diff sets
either field (both absent everywhere → Go zero value `0`), satisfying the
constraint.

### 3. Whole-struct equality, not `reflect.DeepEqual` or field-by-field (PASS)

`consumer_test.go:226`: `if got != tc.want { t.Errorf(...) }`. No
`reflect.DeepEqual` anywhere in the diff. `IncreaseExperienceConfig`
(`socket/model/experience_status.go:3-21`) has 17 fields, all `bool`/`int32`/
`byte` — comparable, so `!=` is valid whole-struct comparison.

### 4. Exhaustiveness test — bidirectional, names offending type(s) (PASS)

`consumer_test.go:233-257`: builds `covered` from `tc.types` (`t.Fatalf` on
duplicate ownership, naming both case names), then checks
registered-but-uncovered (`for _, dt := range character2.AllExperienceDistributionTypes`,
error names the type) and covered-but-unregistered (`for dt, name := range covered`,
error names both the type and the case). Ran it against the live
`AllExperienceDistributionTypes` (`kafka/message/character/kafka.go:168-182`,
14 entries) — passes, and the 14 single-type cases' `types` fields (verified
in the diff) list exactly those 14 constants, one each, no duplicates. The 5
multi/unknown cases declare `types: nil` (verified for all five in the diff).

### 5. `"DEATH"` unknown-type case (PASS)

`kafka/message/character/kafka.go:59` in atlas-character (a **different**
service, not touched by this range) declares
`ExperienceDistributionTypeDeath = "DEATH"` — confirmed by direct grep. The
test case correctly uses the literal string (atlas-channel has no constant
for it) and asserts the zero-White/1300-Amount config that reflects DEATH
being silently ignored while a subsequent WHITE entry still applies.

### 6. Every case makes a real assertion (PASS)

All 19 cases go through the single shared `t.Run` body
(`got := buildIncreaseExperienceConfig(tc.given); if got != tc.want { t.Errorf(...) }`),
so each case is a real equality assertion, not a no-op.

### 7. Scope discipline (PASS)

Only `consumer_test.go` is in the diff. `consumer.go` and `kafka.go` are
byte-identical to the pre-range state (git diff shows no hunks against them
in this range). No chain refactor was introduced — that is correctly deferred
to Task 4.

### 8. Baseline green run (PASS)

Reproduced locally from `services/atlas-channel/atlas.com/channel`:

```
go test ./kafka/consumer/character/... -run 'TestBuildIncreaseExperienceConfig|TestExperienceDistributionTypeExhaustiveness' -v
```

All 19 `TestBuildIncreaseExperienceConfig` subtests plus
`TestExperienceDistributionTypeExhaustiveness` PASS, against the unmodified
`if/else` chain in `consumer.go` (Task 2's pre-refactor state). `gofmt -l
kafka/consumer/character/consumer_test.go` produced no output (clean).

### Non-blocking note

The brief's Step 4 asked for the `-v` output to be recorded in `progress.md`.
No `progress.md` exists in this task's docs directory, and the report notes
this and substitutes the full transcript inline in `task-3-report.md`
instead. This is a minor process deviation, not a code defect — the baseline
evidence is captured, just not in the file the brief named. Not blocking.

## Not evaluable

None. The full case table, both test functions, and every constraint in the
task brief were checked directly against the touched file and the read-only
files it depends on (`consumer.go`, `experience_status.go`, `kafka.go` in
both atlas-channel and atlas-character).

## Verdict

All binding constraints for Task 3 are satisfied: values match the fixed
value scheme exactly, assertions are whole-struct `!=`, the two
zero-always fields are confirmed zero in every case, exhaustiveness is
bidirectional and names the offending type, the DEATH case is grounded in a
real atlas-character constant, no case is a no-op assertion, and only the
test file was touched. The test run reproduces the reported green baseline
against the pre-refactor chain.
