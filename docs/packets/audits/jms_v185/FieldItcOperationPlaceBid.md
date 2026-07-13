# FieldItcOperationPlaceBid (← `CITCBidAuctionDlg::OnButtonClicked`)

- **IDA:** 0x62b063
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x13) @0x62b211` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x62b222` | ✅ |  |
| 2 | int32 | int32 `bidPrice @0x62b230` | ✅ |  |
| 3 | int32 | int32 `bidRange @0x62b23e` | ✅ |  |

