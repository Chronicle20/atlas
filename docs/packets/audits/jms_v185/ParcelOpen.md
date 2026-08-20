# ParcelOpen (← `CParcelDlg::OnPacket#Open`)

- **IDA:** 0x755e7b
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (10 OPEN; dispatch byte, jms_v185 shift from v83's mode 8)` | ✅ |  |
| 1 | byte | byte `quickEnabled` | ✅ |  |
| 2 | byte | byte `mailbox count` | ✅ |  |
| 3 | bytes | bytes `one parcel entry (PARCEL::Decode via sub_510691: DecodeBuffer(0xEA=234) fixed block + Decode1 hasItem + optional GW_ItemSlotBase::Decode item) -- loop body` | ✅ |  |
| 4 | byte | byte `arrived count` | ✅ |  |
| 5 | bytes | bytes `one parcel entry (PARCEL::Decode, same shape as mailbox) -- loop body` | ✅ |  |

