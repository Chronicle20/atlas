# Review — Task 1: Document distribution constants and add exhaustiveness registry

Commit range: `e2a4fdf7e..dc9c7707e` (single commit `dc9c7707e`)
Module: `services/atlas-channel/atlas.com/channel`

## Scope

Diff touches exactly one file:

```
.../channel/kafka/message/character/kafka.go | 57 ++++++++++++++++++++++
1 file changed, 57 insertions(+)
```

Matches the brief's stated file list (`kafka.go` is the only file changed;
`experience_status.go` and the PRD were read-only references). No test files,
no `consumer.go` changes. Scope confirmed — matches the brief exactly.

## Findings

### PASS — Doc comment placed correctly, inside the const block

`kafka.go:110-144` — the FR-24/FR-25 doc comment is inserted immediately above
`ExperienceDistributionTypeWhite = "WHITE"` (`kafka.go:145`), inside the
existing `const (` block that opens at line 101. Matches brief Step 1.

### PASS — Doc comment wording agrees with `experience_status.go` field comments

Cross-checked every line of the per-type table (`kafka.go:117-129`) against
`socket/model/experience_status.go:3-21` field comments, which the brief
designates authoritative over the PRD:

- WHITE/YELLOW/CHAT → `Amount`/`White`/`InChat` fields, "You have gained
  experience" — matches `experience_status.go:4-6`.
- MONSTER_BOOK → `MonsterBookBonus`, "Right side. Yellow. Bonus Event EXP" —
  matches line 7 verbatim.
- MONSTER_EVENT → `MobEventBonusPercentage`, "In chat. Pink. ... every 3rd
  monster defeated" — matches line 8.
- PLAY_TIME → `MobEventBonusPercentage, PlayTimeHour (from Attr1)`, "Bonus EXP
  for hunting over (N) hrs" — matches line 9 (`PlayTimeHour` comment
  explicitly references `+MobEventBonusPercentage`, so citing both fields is
  correct).
- WEDDING → `WeddingBonusEXP` — matches line 11.
- SPIRIT_WEEK → `QuestBonusRate`, "Earned 'Spirit Week Event' bonus EXP" —
  matches line 12 verbatim (correctly omits "right side, yellow" since the
  source field comment doesn't have that qualifier either).
- PARTY → `PartyBonusExp, PartyBonusEventRate (Attr1)` — matches lines 14-15.
- ITEM → `ItemBonusEXP`, "Equip Item Bonus EXP" — matches line 16, and the
  "ITEM is the trap" callout (`kafka.go:132-134`) correctly cites task-277.
- INTERNET_CAFE → `PremiumIPExp` — matches line 17.
- RAINBOW_WEEK → `RainbowWeekEventEXP` — matches line 18.
- PARTY_RING → `PartyEXPRingEXP`, "v95+ only" — matches line 19 ("Available
  v95+").
- CAKE_PIE → `CakePieEventBonus`, "v95+ only" — matches line 20.

No disagreement found between the doc comment and the authoritative field
comments.

### PASS — `AllExperienceDistributionTypes` registry is complete and exact

`kafka.go:168-183` — the registry lists exactly 14 entries, one per
`ExperienceDistributionType*` constant declared at `kafka.go:145-158`, in the
same declaration order (White, Yellow, Chat, MonsterBook, MonsterEvent,
PlayTime, Wedding, SpiritWeek, Party, Item, InternetCafe, RainbowWeek,
PartyRing, CakePie). No entry missing, no extra entry, no duplicate.
Cross-checked with `grep -rn "ExperienceDistributionType"` across the module —
these 14 constants are the only ones defined anywhere in the package, and each
appears in the registry exactly once.

### PASS — `var` correctly placed outside the `const` block

`kafka.go:163-183` — the `AllExperienceDistributionTypes` var sits after the
`const` block's closing `)` (`kafka.go:160`, `StatusEventActorTypeCharacter`)
and before `type StatusEvent[E any] struct` (`kafka.go:185`), matching brief
Step 2 exactly.

### PASS — No production behavior change

`consumer.go` is untouched (confirmed via `git diff --stat`); it still
contains its own independent `if/else if` chain over the same 14 constants
(`kafka/consumer/character/consumer.go:370-401`), unaffected by this task.
Nothing in the diff reads `AllExperienceDistributionTypes` yet — it is inert
until Task 3 consumes it, matching the brief's "no behavior change" framing.

### PASS — Build/vet/format verification

Reran the brief's Step 3 commands from `services/atlas-channel/atlas.com/channel`:

```
go build ./...                                    # exit 0, no output
go vet ./kafka/message/character/...              # exit 0, no output
gofmt -l kafka/message/character/kafka.go          # no output (clean)
```

All three pass as the implementer report claims.

## Not evaluable

None — the full diff (57 lines, one file) was read in full alongside the
authoritative reference file; nothing in this unit's surface required
escalation beyond what was reviewed.

## Verdict

APPROVED. No blocking or non-blocking findings.
