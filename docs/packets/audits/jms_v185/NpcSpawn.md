# NpcSpawn (← `CNpcPool::OnNpcEnterField`)

- **IDA:** 0x72068f
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (object id @0x7206a1)` | ✅ |  |
| 1 | int32 | int32 `templateId (new-entry path @0x7206d8)` | ✅ |  |
| 2 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual verdict (JMS v185, `CNpcPool::OnNpcEnterField` @0x72068f)

Rows 2-8 (❌ "atlas: extra") are an analyzer-flattening artifact, NOT a wire bug. JMS185
reads `Decode4(npcId)` and (new-entry path) `Decode4(templateId)` then delegates the
position/foothold/range fields to `CNpc::Init` (a sub-call the analyzer cannot follow, so
atlas's `spawn.go` body tail is flagged as extra). Rows 0-1 (✅) match; the remaining
position fields are emitted by atlas inside the spawn body matching `CNpc::Init`'s reads
(x/cy/moveAction/foothold/rx0/rx1) — same shape as GMS v95. Carry-forward manual-verify.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
