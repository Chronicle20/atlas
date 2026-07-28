# ChatWorldMessageItemMegaphone (← `CWvsContext::OnBroadcastMsg#ItemMegaphone`)

- **IDA:** 0xa6dc97
- **Atlas file:** `libs/atlas-packet/chat/clientbound/world_message.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (=8, ITEM_MEGAPHONE)` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | byte | byte `channelId` | ✅ |  |
| 3 | byte | byte `whispersOn` | ✅ |  |
| 4 | byte | byte `hasItem bool` | ✅ |  |
| 5 | byte | bytes `GW_ItemSlotBase::Decode(item) — only when hasItem true; no separate slotPos byte before/around the block` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |

