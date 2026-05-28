# MessengerRemove (← `CUIMessenger::OnPacket#Remove`)

- **IDA:** 0x8e447e
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/remove.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 2 (Remove/OnLeave)` | ✅ |  |
| 1 | byte | byte `position (slot 0–2)` | ✅ |  |

