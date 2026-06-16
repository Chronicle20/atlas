# InteractionInteractionUpdateMerchant (← `CEntrustedShopDlg::OnRefresh#UpdateMerchant`)

- **IDA:** 0x518852
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (25; OnPacketBase dispatch byte)` | ✅ |  |
| 1 | int32 | int32 `meso (m_nMoney; this[481]; CEntrustedShopDlg::OnRefresh @0x518852)` | ✅ |  |
| 2 | byte | byte `count (this[100]; CPersonalShopDlg::OnRefresh @0x6fcc4e)` | ✅ |  |
| 3 | int16 | bytes `items (count x: Decode2 perBundle, Decode2 quantity, Decode4 price, GW_ItemSlotBase::Decode asset)` | ✅ |  |
| 4 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

