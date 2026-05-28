# NoteOperationSend (← `CCashShop::OnCashItemResLoadGiftDone`)

- **IDA:** 0x496520
- **Atlas file:** `../../libs/atlas-packet/note/serverbound/operation_send.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `senderName (gift sender name — from gift list entry)` | ✅ |  |
| 1 | string | string `message (acceptance note from CUIReceiveGift::GetResult)` | ✅ |  |

