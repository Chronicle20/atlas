# ActionScriptStart (← `CQuest::StartQuest#ActionScriptStart`)

- **IDA:** 0x6b40a0
- **Atlas file:** `libs/atlas-packet/quest/serverbound/action_script_start.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcId uint32` | ✅ |  |
| 1 | int16 | int16 `x int16` | ✅ |  |
| 2 | int16 | int16 `y int16` | ✅ |  |

## Manual analysis

**IDA function:** `CQuest::StartQuest` @ 0x6b40a0, action=4 (`IsStartScriptLinkedQuest` branch)

When the quest has a start script linked, the client sends action 4 via opcode 119. After
the `Action` header (action byte + questId), the script-start sub-fields are:

```
COutPacket::Encode1(&oPacket, 4u)            // action byte = 4 (in Action header)
COutPacket::Encode2(&oPacket, questId)        // questId (in Action header)
COutPacket::Encode4(&oPacket, npcId)          // npcId uint32 LE
COutPacket::Encode2(&oPacket, ptUserPos)      // x int16 LE
COutPacket::Encode2(&oPacket, n[0])           // y int16 LE
CClientSocket::SendPacket(...)
```

Total sub-struct (after Action header): 8 bytes.

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| npcId | Encode4 (uint32 LE) | WriteInt (uint32 LE) | ✅ |
| x | Encode2 (int16 LE) | WriteInt16 (int16 LE) | ✅ |
| y | Encode2 (int16 LE) | WriteInt16 (int16 LE) | ✅ |

### No bug — already correct

`ActionScriptStart.Encode/Decode` matches v95 exactly. The ✅ static-diff verdict is accurate.
Note: unlike `ActionStart` (action=1), the script-start branch does NOT write a delivery-item
slot field. See `_pending.md` for the `ActionStart`/`ActionComplete` delivery-item gap.

Wire shape verified by `TestActionScriptStartWireShape` in
`libs/atlas-packet/quest/serverbound/action_script_start_test.go`:
all four variants produce exactly 8 bytes (4 npcId + 2 x + 2 y).

Ack: misc-audit Phase 2g on 2026-06-03

