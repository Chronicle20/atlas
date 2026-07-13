# ChatGeneralChat (← `CUser::OnChat`)

- **IDA:** 0x890a8b
- **Atlas file:** `libs/atlas-packet/chat/clientbound/general.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | width mismatch |
| 1 | byte | string `` | ❌ | width mismatch |
| 2 | string | byte `` | ❌ | width mismatch |
| 3 | byte | string `` | ❌ | width mismatch |
| 4 | byte | unresolved `function not found in IDB` | ❌ | atlas: short — missing trailing field |

