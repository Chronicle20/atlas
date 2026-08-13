# MessengerRemove (← `CUIMessenger::OnPacket#Remove`)

- **IDA:** 0x61d8b8
- **Atlas file:** `libs/atlas-packet/messenger/clientbound/remove.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | byte | byte `position` | ✅ |  |

