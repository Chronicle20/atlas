# Action (← `CWvsContext::ResignQuest#Action`)

- **IDA:** 0x9f3cf0
- **Atlas file:** `libs/atlas-packet/quest/serverbound/action.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `action type byte` | ✅ |  |
| 1 | int16 | int16 `questId uint16` | ✅ |  |

## Manual analysis

**IDA function:** `CWvsContext::ResignQuest` @ 0x9f3cf0 (action type 3, forfeit/resign quest)

The unified quest-action packet uses opcode 119 (0x77). All quest actions share the same
opcode; the action byte distinguishes the action type. For action 3 (forfeit):

```
COutPacket::COutPacket(&oPacket, 119)
COutPacket::Encode1(&oPacket, 3u)          // action byte = 3 (QuestActionForfeit)
COutPacket::Encode2(&oPacket, questId)     // questId uint16 LE
CClientSocket::SendPacket(...)
```

Total sub-struct: 3 bytes (1 action + 2 questId). The `Action` struct encodes exactly this.

### Wire comparison

| Field | IDA width | Atlas width | Match? |
|---|---|---|---|
| action | Encode1 (1 byte) | WriteByte (1 byte) | ✅ |
| questId | Encode2 (uint16 LE) | WriteShort (uint16 LE) | ✅ |

**SUMMARY row collision check:** `type Action struct` could collide with other domains.
`locateAtlasFile` walks alphabetically; `quest` sorts after `channel`/`buddy` directories
that contain no `Action` struct. The tool correctly resolves to
`libs/atlas-packet/quest/serverbound/action.go`.

### No bug — already correct

`Action.Encode/Decode` matches v95 exactly. The ✅ static-diff verdict is accurate.

Wire shape verified by `TestActionWireShape` in
`libs/atlas-packet/quest/serverbound/action_test.go`:
all four variants produce exactly 3 bytes (1 action byte + 2 questId LE).

Ack: misc-audit Phase 2g on 2026-06-03

