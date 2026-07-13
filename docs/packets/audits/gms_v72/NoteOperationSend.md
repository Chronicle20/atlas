# NoteOperationSend (← `CCashShop::OnCashItemResLoadGiftDone`)

- **IDA:** 0x47122e
- **Atlas file:** `libs/atlas-packet/note/serverbound/operation_send.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | int16 `` | ❌ | width mismatch |
| 1 | string | bytes `` | ❌ | width mismatch |

