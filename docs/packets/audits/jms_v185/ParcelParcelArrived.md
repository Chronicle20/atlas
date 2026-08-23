# ParcelParcelArrived (← `CParcelDlg::OnPacket#ParcelArrived`)

- **IDA:** 0x755cdd
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (25 PARCEL_ARRIVED; dispatch byte, jms_v185 shift from v83's mode 24)` | ✅ |  |
| 1 | bytes | bytes `one parcel entry (PARCEL::Decode via sub_510691: DecodeBuffer(0xEA=234) fixed block + Decode1 hasItem + optional GW_ItemSlotBase::Decode item)` | ✅ |  |

