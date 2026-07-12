# FieldItcOperationPlaceBid (← `CITCBidAuctionDlg::OnButtonClicked`)

- **IDA:** 0x58eb50
- **Atlas file:** `libs/atlas-packet/field/serverbound/itc_operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode byte (0x13) @0x58edb4` | ✅ |  |
| 1 | int32 | int32 `nITCSN @0x58edc7` | ✅ |  |
| 2 | int32 | int32 `m_nMyBidPrice @0x58edd7` | ✅ |  |
| 3 | int32 | int32 `m_nMyBidRange @0x58ede7` | ✅ |  |

