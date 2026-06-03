# InteractionInteractionUpdateMerchant (← `CPersonalShopDlg::OnRefresh#UpdateMerchant`)

- **IDA:** 0x51cc30
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (25; dispatch byte)` | ✅ |  |
| 1 | int32 | int32 `meso (m_nMoney; CEntrustedShopDlg::OnRefresh prefix)` | ✅ |  |
| 2 | byte | byte `count (m_nItem)` | ✅ |  |
| 3 | int16 | bytes `items (count x: perBundle short, quantity short, price int, GW_ItemSlotBase substruct)` | ❌ | width mismatch |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

