# ChatGeneralChat (← `CUser::OnChat`)

- **IDA:** 0x790fcb
- **Atlas file:** `libs/atlas-packet/chat/clientbound/general.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | string `` | ❌ | width mismatch |
| 2 | string | byte `` | ❌ | width mismatch |
| 3 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |

