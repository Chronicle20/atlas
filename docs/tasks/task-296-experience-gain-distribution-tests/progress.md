# task-296: Mutation-testing proof (Task 5)

Design §4.6. Each mutation below was applied to a working file, run against
the relevant test, the failure output captured verbatim, then reverted with
`git checkout -- <path>` before the next mutation. All commands were run from
`services/atlas-channel/atlas.com/channel`.

## Step 1: Registry mutation A — added type with no case

Mutation: appended `"FAKE_TYPE"` to `AllExperienceDistributionTypes` in
`kafka/message/character/kafka.go`.

Command:

```
go test ./kafka/consumer/character/... -run TestExperienceDistributionTypeExhaustiveness -v
```

Captured output (verbatim):

```
=== RUN   TestExperienceDistributionTypeExhaustiveness
    consumer_test.go:413: distribution type "FAKE_TYPE" is in AllExperienceDistributionTypes but has no case in distributionMappingCases
--- FAIL: TestExperienceDistributionTypeExhaustiveness (0.00s)
FAIL
FAIL	atlas-channel/kafka/consumer/character	0.010s
```

Result: matches the brief's prediction — the test fails and names
`"FAKE_TYPE"` as registered-but-uncovered.

Reverted with `git checkout -- kafka/message/character/kafka.go`.

## Step 2: Registry mutation B — removed type still covered

Mutation: deleted `ExperienceDistributionTypeCakePie` from
`AllExperienceDistributionTypes` in `kafka/message/character/kafka.go`.

Command:

```
go test ./kafka/consumer/character/... -run TestExperienceDistributionTypeExhaustiveness -v
```

Captured output (verbatim):

```
=== RUN   TestExperienceDistributionTypeExhaustiveness
    consumer_test.go:419: case "CakePie_EventBonus_v95Plus" covers distribution type "CAKE_PIE", which is not in AllExperienceDistributionTypes
--- FAIL: TestExperienceDistributionTypeExhaustiveness (0.00s)
FAIL
FAIL	atlas-channel/kafka/consumer/character	0.009s
```

Result: matches the brief's prediction — the test fails and names
`"CAKE_PIE"` as covered-but-unregistered.

Reverted with `git checkout -- kafka/message/character/kafka.go`.

## Step 3: Mapping mutation — the task-277 bug, reintroduced

Mutation: in `kafka/consumer/character/consumer.go`, the
`ExperienceDistributionTypeItem` case was changed to also write
`c.Amount = int32(d.Amount)` in addition to `c.ItemBonusEXP = int32(d.Amount)`
(the task-277 bug: ITEM incorrectly touching the primary amount).

Command:

```
go test ./kafka/consumer/character/... -run TestBuildIncreaseExperienceConfig -v
```

Captured output (verbatim, full run):

```
=== RUN   TestBuildIncreaseExperienceConfig
=== RUN   TestBuildIncreaseExperienceConfig/White_PrimaryWhiteText
=== RUN   TestBuildIncreaseExperienceConfig/Yellow_PrimaryYellowText
=== RUN   TestBuildIncreaseExperienceConfig/Chat_PrimaryInChat
=== RUN   TestBuildIncreaseExperienceConfig/MonsterBook_BonusEventExp
=== RUN   TestBuildIncreaseExperienceConfig/MonsterEvent_MobEventPercentage
=== RUN   TestBuildIncreaseExperienceConfig/PlayTime_MobEventPercentageAndHours
=== RUN   TestBuildIncreaseExperienceConfig/Wedding_BonusWeddingExp
=== RUN   TestBuildIncreaseExperienceConfig/SpiritWeek_QuestBonusRate
=== RUN   TestBuildIncreaseExperienceConfig/Party_BonusExpAndEventRate
=== RUN   TestBuildIncreaseExperienceConfig/Item_EquipItemBonusExpNotPrimary
    consumer_test.go:392: config mismatch
         got: {White:false Amount:7000 InChat:false MonsterBookBonus:0 MobEventBonusPercentage:0 PlayTimeHour:0 PartyBonusPercentage:0 WeddingBonusEXP:0 QuestBonusRate:0 QuestBonusRemainCount:0 PartyBonusEventRate:0 PartyBonusExp:0 ItemBonusEXP:7000 PremiumIPExp:0 RainbowWeekEventEXP:0 PartyEXPRingEXP:0 CakePieEventBonus:0}
        want: {White:false Amount:0 InChat:false MonsterBookBonus:0 MobEventBonusPercentage:0 PlayTimeHour:0 PartyBonusPercentage:0 WeddingBonusEXP:0 QuestBonusRate:0 QuestBonusRemainCount:0 PartyBonusEventRate:0 PartyBonusExp:0 ItemBonusEXP:7000 PremiumIPExp:0 RainbowWeekEventEXP:0 PartyEXPRingEXP:0 CakePieEventBonus:0}
=== RUN   TestBuildIncreaseExperienceConfig/InternetCafe_PremiumIpExp
=== RUN   TestBuildIncreaseExperienceConfig/RainbowWeek_BonusEventExp
=== RUN   TestBuildIncreaseExperienceConfig/PartyRing_ExpRingExp_v95Plus
=== RUN   TestBuildIncreaseExperienceConfig/CakePie_EventBonus_v95Plus
=== RUN   TestBuildIncreaseExperienceConfig/WhiteAndChat_PrimaryAwardShape
=== RUN   TestBuildIncreaseExperienceConfig/PrimaryPlusBonuses_Accumulate
    consumer_test.go:392: config mismatch
         got: {White:true Amount:770 InChat:false MonsterBookBonus:0 MobEventBonusPercentage:0 PlayTimeHour:0 PartyBonusPercentage:0 WeddingBonusEXP:0 QuestBonusRate:0 QuestBonusRemainCount:0 PartyBonusEventRate:66 PartyBonusExp:600 ItemBonusEXP:770 PremiumIPExp:0 RainbowWeekEventEXP:0 PartyEXPRingEXP:0 CakePieEventBonus:0}
        want: {White:true Amount:2500 InChat:false MonsterBookBonus:0 MobEventBonusPercentage:0 PlayTimeHour:0 PartyBonusPercentage:0 WeddingBonusEXP:0 QuestBonusRate:0 QuestBonusRemainCount:0 PartyBonusEventRate:66 PartyBonusExp:600 ItemBonusEXP:770 PremiumIPExp:0 RainbowWeekEventEXP:0 PartyEXPRingEXP:0 CakePieEventBonus:0}
=== RUN   TestBuildIncreaseExperienceConfig/WhiteThenYellow_LastWins
=== RUN   TestBuildIncreaseExperienceConfig/EmptySlice_ZeroConfig
=== RUN   TestBuildIncreaseExperienceConfig/UnknownType_DeathIgnored
--- FAIL: TestBuildIncreaseExperienceConfig (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/White_PrimaryWhiteText (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/Yellow_PrimaryYellowText (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/Chat_PrimaryInChat (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/MonsterBook_BonusEventExp (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/MonsterEvent_MobEventPercentage (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/PlayTime_MobEventPercentageAndHours (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/Wedding_BonusWeddingExp (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/SpiritWeek_QuestBonusRate (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/Party_BonusExpAndEventRate (0.00s)
    --- FAIL: TestBuildIncreaseExperienceConfig/Item_EquipItemBonusExpNotPrimary (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/InternetCafe_PremiumIpExp (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/RainbowWeek_BonusEventExp (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/PartyRing_ExpRingExp_v95Plus (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/CakePie_EventBonus_v95Plus (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/WhiteAndChat_PrimaryAwardShape (0.00s)
    --- FAIL: TestBuildIncreaseExperienceConfig/PrimaryPlusBonuses_Accumulate (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/WhiteThenYellow_LastWins (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/EmptySlice_ZeroConfig (0.00s)
    --- PASS: TestBuildIncreaseExperienceConfig/UnknownType_DeathIgnored (0.00s)
FAIL
FAIL	atlas-channel/kafka/consumer/character	0.009s
FAIL
```

Result: matches the brief's prediction on both counts —
`Item_EquipItemBonusExpNotPrimary` fails with `Amount:7000` in `got` vs
`Amount:0` in `want`, and `PrimaryPlusBonuses_Accumulate` fails too, because
its `ITEM` entry (Amount 770) clobbers the accumulated `WHITE` primary amount
(want 2500, got 770).

Reverted with `git checkout -- kafka/consumer/character/consumer.go`.

## Step 4: Confirm the tree is clean and green

```
git status --porcelain
```

Output: only the pre-existing untracked task-tracking files (present before
this task started, not touched by it):

```
?? docs/tasks/task-296-experience-gain-distribution-tests/agent-ledger.tsv
?? docs/tasks/task-296-experience-gain-distribution-tests/review-task-1.md
?? docs/tasks/task-296-experience-gain-distribution-tests/review-task-2.md
?? docs/tasks/task-296-experience-gain-distribution-tests/review-task-3.md
?? docs/tasks/task-296-experience-gain-distribution-tests/review-task-4.md
```

No mutated file (`kafka/message/character/kafka.go`,
`kafka/consumer/character/consumer.go`) appears — both were reverted cleanly.

```
go test ./...
```

Run from `services/atlas-channel/atlas.com/channel`. Exit code: `0`. All
packages report `ok` or `[no test files]`; no `FAIL` lines. Full log tail
(final section, confirming a clean pass through to the end of the package
list):

```
?   	atlas-channel/party_quest	[no test files]
?   	atlas-channel/party_quest/mock	[no test files]
ok  	atlas-channel/pendingchange	(cached)
ok  	atlas-channel/pet	(cached)
?   	atlas-channel/pet/exclude	[no test files]
?   	atlas-channel/pet/mock	[no test files]
ok  	atlas-channel/playernpc	(cached)
ok  	atlas-channel/pointreset	(cached)
ok  	atlas-channel/portal	(cached)
?   	atlas-channel/portal/mock	[no test files]
ok  	atlas-channel/position	(cached)
ok  	atlas-channel/quest	(cached)
ok  	atlas-channel/reactor	(cached)
?   	atlas-channel/reactor/mock	[no test files]
ok  	atlas-channel/remotemerchant	(cached)
ok  	atlas-channel/report	(cached)
ok  	atlas-channel/respawn	(cached)
ok  	atlas-channel/ring	(cached)
?   	atlas-channel/rps	[no test files]
?   	atlas-channel/saga	[no test files]
ok  	atlas-channel/server	(cached)
?   	atlas-channel/server/mock	[no test files]
ok  	atlas-channel/session	(cached)
?   	atlas-channel/session/mock	[no test files]
ok  	atlas-channel/shopscanner	(cached)
ok  	atlas-channel/skill/handler	(cached)
ok  	atlas-channel/skill/handler/chakra	(cached)
ok  	atlas-channel/skill/handler/dispel	(cached)
ok  	atlas-channel/skill/handler/echoofhero	(cached)
ok  	atlas-channel/skill/handler/flamegear	(cached)
ok  	atlas-channel/skill/handler/heal	(cached)
ok  	atlas-channel/skill/handler/healdispel	(cached)
ok  	atlas-channel/skill/handler/hide	(cached)
ok  	atlas-channel/skill/handler/mistcast	(cached)
ok  	atlas-channel/skill/handler/monstermagnet	(cached)
ok  	atlas-channel/skill/handler/mprecovery	(cached)
ok  	atlas-channel/skill/handler/mysticdoor	(cached)
ok  	atlas-channel/skill/handler/poisonbomb	(cached)
ok  	atlas-channel/skill/handler/poisonmist	(cached)
ok  	atlas-channel/skill/handler/recoveryaura	(cached)
ok  	atlas-channel/skill/handler/registrations	(cached)
ok  	atlas-channel/skill/handler/resurrection	(cached)
ok  	atlas-channel/skill/handler/smokescreen	(cached)
ok  	atlas-channel/skill/handler/timeleap	(cached)
ok  	atlas-channel/socket	(cached)
ok  	atlas-channel/socket/handler	(cached)
ok  	atlas-channel/socket/model	(cached)
ok  	atlas-channel/socket/writer	(cached)
ok  	atlas-channel/storage	(cached)
ok  	atlas-channel/summon	(cached)
?   	atlas-channel/summon/mock	[no test files]
?   	atlas-channel/tasks	[no test files]
ok  	atlas-channel/teleportrock	(cached)
?   	atlas-channel/test	[no test files]
ok  	atlas-channel/trade	(cached)
ok  	atlas-channel/transport/route	(cached)
?   	atlas-channel/weather	[no test files]
?   	atlas-channel/weather/mock	[no test files]
ok  	atlas-channel/world	(cached)
ok  	atlas-channel/worldbroadcast	(cached)
```

## Conclusion

All three mutations produced exactly the failures the brief predicted, and
the tree returns to a clean, fully green state after each revert. The
`AllExperienceDistributionTypes` exhaustiveness test and the
`buildIncreaseExperienceConfig` table-driven test would both have caught the
task-277 bug had they existed at the time.
