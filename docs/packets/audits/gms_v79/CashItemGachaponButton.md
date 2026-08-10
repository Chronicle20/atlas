# CashItemGachaponButton (← `CUICashItemGachapon::OnButtonClicked`)

- **IDA:** 0x8efda6
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_gachapon_button.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `cashId: 8 bytes LARGE_INTEGER (int64), the cash serial (m_liItemSN)` | ✅ |  |

