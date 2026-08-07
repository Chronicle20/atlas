# FieldItcOperationPlaceBid (← `CITCBidAuctionDlg::OnButtonClicked`)

- **IDA:** 0x5f45b1
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x13) @0x5f475f` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x5f4770` | ✅ |  |
| 2 | int32 | int32 `bidPrice (this[44]) @0x5f477e` | ✅ |  |
| 3 | int32 | int32 `bidRange (this[43]) @0x5f478c` | ✅ |  |

