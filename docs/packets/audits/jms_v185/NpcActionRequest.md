# NpcActionRequest (← `CNpc::GenerateMovePath`)

- **IDA:** 0x7199ce
- **Atlas file:** `../../libs/atlas-packet/npc/serverbound/action.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId (m_dwNpcId @0x719a7e)` | ✅ |  |
| 1 | byte | byte `nAction (@0x719a89)` | ✅ |  |
| 2 | byte | byte `nChatIdx (@0x719a94)` | ✅ |  |
| 3 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual verdict (JMS v185, `CNpc::GenerateMovePath` @0x7199ce)

Rows 3-6 (❌ "atlas: extra") are an analyzer-flattening artifact, NOT a wire bug. JMS185
`CNpc::GenerateMovePath` builds `COutPacket(0xD0) + Encode4(npcId) + Encode1(nAction) +
Encode1(nChatIdx)` then a `CMovePath::Flush` body. Atlas serverbound `action.go` emits the
same `npcId(int) + action(byte) + chatIdx(byte)` + move-path; the trailing rows are the
conditional move-path tail collected unconditionally by the analyzer (rows 0-2 ✅).

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
