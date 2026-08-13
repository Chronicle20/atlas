# FieldRequestFootholdInfo (← `CField::OnRequestFootHoldInfo`)

- **IDA:** 0x5282e0
- **Atlas file:** `libs/atlas-packet/field/serverbound/request_foothold_info.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | unresolved `resolved address/decompile but zero recognized Decode/Encode calls found; see notes` | 🚫 | IDA read-order unresolved: resolved address/decompile but zero recognized Decode/Encode calls found; see notes |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

