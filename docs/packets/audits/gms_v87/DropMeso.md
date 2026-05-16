# DropMeso (← `CWvsContext::SendDropMoneyRequest`)

- **IDA:** 0xabb8b3
- **Atlas file:** `libs/atlas-packet/character/serverbound/drop_meso.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time())` | ✅ |  |
| 1 | int32 | int32 `nAmount (meso amount to drop)` | ✅ |  |

