# ParcelOpen (← `CParcelDlg::OnPacket#Open`)

- **IDA:** 0x70dfe7
- **Atlas file:** `libs/atlas-packet/parcel/clientbound/parcel.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (8 OPEN; dispatch byte)` | ✅ |  |
| 1 | byte | byte `quickEnabled` | ✅ |  |
| 2 | byte | byte `mailbox count` | ✅ |  |
| 3 | bytes | bytes `one parcel entry (PARCEL::Decode: 234-byte fixed id/senderName/mesos/expiresAt/message block + hasItem + optional GW_ItemSlotBase item) -- loop body` | ✅ |  |
| 4 | byte | byte `arrived count` | ✅ |  |
| 5 | bytes | bytes `one parcel entry (PARCEL::Decode, same shape as mailbox) -- loop body` | ✅ |  |

