# FieldItcOperationPlaceBid (← `CITCBidAuctionDlg::OnButtonClicked`)

- **IDA:** 0x549672
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x13) @0x5c38ec` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5c38fd` | ✅ |  |
| 2 | int32 | int32 `bidPrice @0x5c390b` | ✅ |  |
| 3 | int32 | int32 `bidRange @0x5c3919` | ✅ |  |

