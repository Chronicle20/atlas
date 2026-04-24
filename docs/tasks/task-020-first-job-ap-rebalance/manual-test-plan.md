# First-Job AP Rebalance — Manual Test Plan

Exercise each row against a running dev deployment after implementation lands. Each row creates a Level-10 beginner on a vanilla v83 client, lets auto-allocation run to completion (expected starting stats STR 53 / DEX 9 / INT 4 / LUK 4, unallocated AP 0), then attempts first-job advancement.

| # | Class | NPC / Quest | Expected post-advancement | Acceptance criterion |
|---|---|---|---|---|
| 1 | Pirate | Kyrin (NPC 1090000) | STR 4, DEX 20, INT 4, LUK 4, unallocated 38 | PRD §10.7 video match |
| 2 | Warrior | Dances with Balrog (NPC 1022000) | STR 35, DEX 4, INT 4, LUK 4, unallocated 23 | PRD §10.2 surplus-return boundary |
| 3 | Bowman | Athena Pierce (NPC 1012100) | STR 4, DEX 25, INT 4, LUK 4, unallocated 33 | Representative DEX-25 class |
| 4 | Magician | Grendel (NPC 1032001) | STR 4, DEX 4, INT 20, LUK 4, unallocated 38 | INT path exercised |
| 5 | Thunder Breaker | quest_20105 (Cygnus) | STR 20, DEX 20, INT 4, LUK 4, unallocated 22 | Multi-target `rebalance_ap` |

For each row, also verify:

- atlas-character logs contain an info line of the form `Rebalanced character [N] AP. Before STR=... -> after STR=... targets=[...]`.
- atlas-character emits exactly one `STAT_CHANGED` event containing `updates` with all five stat types (`AVAILABLE_AP`, `STRENGTH`, `DEXTERITY`, `INTELLIGENCE`, `LUCK`).
- atlas-character emits exactly one `JOB_CHANGED` event *after* the `STAT_CHANGED` (ordering observable in Kafka topic offsets).
- Character row in DB has the expected STR/DEX/INT/LUK/AP values.
- No regression: pre-existing 2nd/3rd/4th job advancements are unaffected.

Open Question 3 (client auto-opens stat window): observe client behavior on each run. If the stat window does not auto-open, file a follow-up task per PRD §9 Q3.
