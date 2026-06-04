# Spike Verdict — Quest serverbound ActionStart / ActionComplete (task-080 B1.3)

**Verdict: NO-OP — plan premise disproven. Existing Atlas decode is byte-correct against all four client versions. No code change.**

## Premise under test (plan B1.3)
The plan claimed `CQuest::StartQuest@0x6b40a0` (actions 1 Start / 2 Complete) reads
`Encode4(nItemPos)` (a delivery-item slot, 0 for normal quests) **between** `npcId`
and the conditional `x,y`, and that the Atlas `ActionStart`/`ActionComplete` decoders
were missing it.

## Evidence (IDA decompile + four-version export cross-check)

### Live decompile (loaded JMS185 IDB)
`CQuest::StartQuest` resolved at JMS185 `0x77d065` (the plan's `0x6b40a0` is the GMS-v95
address). The function builds the serverbound `COutPacket` for all five quest actions:
- **Action 1 (Start):** `Encode1(1)` + `Encode2(questId)` + `Encode4(npcId)` + `if(!IsAutoAlertQuest(questId)){ Encode2(x); Encode2(y); }`
- **Action 2 (Complete):** as Action 1, plus a trailing `Encode4(selection)`.

There is **no `nItemPos`** field in the read sequence — not gated, not between npcId and x/y, nowhere.

### Four-version export cross-check (`docs/packets/ida-exports/*.json`)
`CQuest::StartQuest#ActionScriptStart` / `#ActionScriptEnd` read-order is identical in
**all four** baselines (v83 `0x716fe1`, v87 `0x75bf04`, v95 `0x6b40a0`, JMS185 `0x77d065`):
`Decode4 npcId → Decode2 x → Decode2 y` (x,y conditional on `!IsAutoAlertQuest` in v95/JMS).
No `nItemPos` in any version. The JMS export note states verbatim:
*"No extra nItemPos field in JMS for this action. Same as GMS v95."*

## autoStart gate
**Correct, not inverted.** IDA writes `x,y` when `!IsAutoAlertQuest(questId)`; Atlas writes
`x,y` when `m.autoStart == true`. So Atlas `autoStart` ↔ IDA `!IsAutoAlertQuest`, matching
the plan's expectation. No caller fix required.

## Current Atlas code (already correct)
`libs/atlas-packet/quest/serverbound/action_start.go` and `action_complete.go` already decode
`npcId(4) + [x(2),y(2) if autoStart]` (+ trailing `selection(4)` for complete) — byte-identical
to the client wire shape. Inserting an `itemPos int32` after `npcId` would shift x/y/selection
4 bytes downstream and **corrupt every quest start/complete packet**. No change made.

## Disposition
- B1.3 closed as **audited / no fix warranted**. The blocker that motivated B1.3 was a faulty
  premise, not a real wire divergence in the four-version baseline.
- The "delivery-item slot" concept (if it exists in some client) is not in `CQuest::StartQuest`
  for v83/v87/v95/v185 and is out of this baseline's scope.
- Feeds Phase E (§4.8) as a resolved/non-actionable item.
