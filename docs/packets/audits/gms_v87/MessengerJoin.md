# MessengerJoin (← `CUIMessenger::OnPacket#Join`)

- **IDA:** 0x8b978f
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/join.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (1)` | ✅ |  |
| 1 | byte | byte `slot index` | ✅ |  |
| 2 | byte | byte `userCount` | ❌ | atlas: short — missing trailing field |
| 3 | byte | int32 `characterId per user (loop)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |
| 4 | byte | bytes `avatar look per user (loop)` | ⚠️ | loop body — atlas emits zero iterations (count==0) |

