# MessengerJoin (← `CUIMessenger::OnPacket#Join`)

- **IDA:** 0x7f5e00
- **Atlas file:** `../../libs/atlas-packet/messenger/clientbound/join.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=1, JOIN) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | byte | byte `position — self slot index in messenger room (Decode1 → OnSelfEnter)` | ✅ |  |

