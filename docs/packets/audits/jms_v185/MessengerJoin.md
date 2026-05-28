# MessengerJoin (← `CUIMessenger::OnPacket#Join`)

- **IDA:** 0x8e447e
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/join.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte = 1 (Join/OnSelfEnterResult)` | ✅ |  |
| 1 | byte | byte `position (slot 0–2)` | ✅ |  |

