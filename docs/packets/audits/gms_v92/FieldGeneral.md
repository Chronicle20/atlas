# FieldGeneral (← `CField::SendChatMsg`)

- **IDA:** 0x52e400
- **Atlas file:** `libs/atlas-packet/field/serverbound/general.go`
- **Variant:** GMS/v92
- **Branch depth:** 2
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `` | ✅ |  |
| 1 | string | string `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |

