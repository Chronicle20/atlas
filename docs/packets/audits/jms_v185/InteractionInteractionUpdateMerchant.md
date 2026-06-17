# InteractionInteractionUpdateMerchant (← `CEntrustedShopDlg::OnRefresh#UpdateMerchant`)

- **IDA:** 0x54adb9
- **Atlas file:** `libs/atlas-packet/interaction/clientbound/interaction.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (22; OnPacketBase dispatch byte; jms personal-shop refresh = 22, gms = 25)` | ✅ |  |
| 1 | int32 | int32 `meso (this[500]=m_nMoney; CEntrustedShopDlg::OnRefresh @0x54adb9)` | ✅ |  |
| 2 | byte | byte `count (this[3].m_rcInvalidated.bottom; CPersonalShopDlg::OnRefresh sub_761DBA @0x761dba)` | ✅ |  |
| 3 | int16 | bytes `items (count x: Decode2 perBundle, Decode2 quantity, Decode4 price, GW_ItemSlotBase::Decode asset @0x50f611)` | ✅ |  |
| 4 | int16 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 5 | int32 | byte `` | ✅ | absorbed by trailing opaque buffer |
| 6 | byte | byte `` | ✅ | absorbed by trailing opaque buffer |

