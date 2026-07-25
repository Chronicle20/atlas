# ChatWorldMessageItemMegaphone (← `CWvsContext::OnBroadcastMsg#ItemMegaphone`)

- **IDA:** 0xa04160
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | byte | byte `channelId - case 8` | ✅ |  |
| 3 | byte | byte `whispersOn` | ✅ |  |
| 4 | byte | byte `hasItem` | ✅ |  |
| 5 | byte | bytes `GW_ItemSlotBase::Decode(item) - opaque item block, guarded by hasItem, no slotPos byte` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |

