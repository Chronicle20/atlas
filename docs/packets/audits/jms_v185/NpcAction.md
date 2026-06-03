# NpcAction (← `CNpc::OnMove`)

- **IDA:** 0x7194d1
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/action.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (dispatcher prefix via CNpcPool::OnNpcPacket)` | ✅ |  |
| 1 | byte | byte `nAction / oneTimeAction (@0x7194e6)` | ✅ |  |
| 2 | byte | byte `nChatIdx (@0x7194f4)` | ✅ |  |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual verdict (JMS v185, `CNpc::OnMove` @0x7194d1)

Rows 3-6 (❌ "atlas: extra") are an analyzer-flattening artifact, NOT a wire bug. Atlas
`action.go` writes the trailing rx/movement fields only inside conditional branches; the
analyzer collects those tails unconditionally. At runtime atlas emits `objectId(int) +
action(byte) + chatIdx(byte)` then the move-path body, which is exactly what JMS185
`CNpc::OnMove` reads (rows 0-2 ✅) — `Decode1(nAction)` + `Decode1(nChatIdx)` after the
dispatcher-prefix `Decode4(npcId)` from `CNpcPool::OnNpcPacket`, then `CMovePath::OnMovePacket`.

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
