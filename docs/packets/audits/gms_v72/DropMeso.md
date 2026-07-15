# DropMeso (← `CWvsContext::SendDropMoneyRequest`)

- **IDA:** 0x91bf9f
- **Atlas file:** `libs/atlas-packet/character/serverbound/drop_meso.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime Encode4 @0x91c023` | ✅ |  |
| 1 | int32 | int32 `amount Encode4 @0x91c02e` | ✅ |  |

