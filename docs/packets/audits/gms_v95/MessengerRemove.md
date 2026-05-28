# MessengerRemove (← `CUIMessenger::OnPacket#Remove`)

- **IDA:** 0x7f5e20
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/remove.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=2, REMOVE) — dispatcher switch byte consumed by CUIMessenger::OnPacket` | ✅ |  |
| 1 | byte | byte `position — slot index of the character that left` | ✅ |  |

