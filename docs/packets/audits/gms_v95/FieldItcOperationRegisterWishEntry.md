# FieldItcOperationRegisterWishEntry (← `CITC::OnRegisterWishEntry`)

- **IDA:** 0x573c10
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (4) @0x573cb5` | ✅ |  |
| 1 | int32 | int32 `m_nWishItemID @0x573cc5` | ✅ |  |
| 2 | int32 | int32 `m_nWishPrice @0x573cd5` | ✅ |  |
| 3 | int32 | int32 `m_nWishCount @0x573ce5` | ✅ |  |
| 4 | byte | byte `m_bWishDuration @0x573cf6` | ✅ |  |
| 5 | byte | byte `m_bWishRegistrationFeeOption @0x573d07` | ✅ |  |
| 6 | string | string `m_sWishDesc @0x573d23` | ✅ |  |

