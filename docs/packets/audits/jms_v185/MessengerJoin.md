# MessengerJoin (← `CUIMessenger::OnPacket#Join`)

- **IDA:** 0x8e447e
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/join.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | byte | byte `position` | ✅ |  |

