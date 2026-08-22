# ParcelParcelArrived (← `CParcelDlg::OnPacket#ParcelArrived`)

- **IDA:** 0x6839b3
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (24/0x18 PARCEL_ARRIVED; dispatch byte)` | ✅ |  |
| 1 | bytes | bytes `one parcel entry (PARCEL::Decode: 234-byte fixed id/senderName/mesos/expiresAt/message block + hasItem + optional GW_ItemSlotBase item)` | ✅ |  |

