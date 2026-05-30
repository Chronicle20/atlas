# NpcSpawnRequestController (← `CNpcPool::OnNpcChangeController`)

- **IDA:** 0x720782
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/spawn_request_controller.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `localFlag (@0x720794)` | ✅ |  |
| 1 | int32 | int32 `npcId (@0x720797)` | ✅ |  |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual verdict (JMS v185, `CNpcPool::OnNpcChangeController` @0x720782)

Rows 2-9 (❌ "atlas: extra") are an analyzer-flattening artifact, NOT a wire bug. JMS185
reads `Decode1(localFlag)` + `Decode4(npcId)` (rows 0-1 ✅); when `localFlag != 0` the
new-entry path in `CNpcPool::SetLocalNpc` reads `templateId` + the `CNpc::Init` position
fields (a sub-call the analyzer cannot follow). Atlas `spawn_request_controller.go` emits
the same fields gated on the local flag — same shape as GMS v95. Carry-forward manual-verify.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
