# NoteOperationSend (← `CCashShop::OnCashItemResLoadGiftDone`)

- **IDA:** 0x47c73c
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation_send.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |

