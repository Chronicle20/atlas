# ParcelActionSend (← `CTabSend::SendParcel`)

- **IDA:** 0x682da0
- **Atlas file:** `libs/atlas-packet/parcel/serverbound/action_send.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `inventoryType` | ✅ |  |
| 1 | int16 | int16 `slot` | ✅ |  |
| 2 | int16 | int16 `quantity` | ✅ |  |
| 3 | int32 | int32 `mesos` | ✅ |  |
| 4 | string | string `recipientName` | ✅ |  |
| 5 | byte | byte `quick` | ✅ |  |
| 6 | string | string `message -- quick-send path only (sub_682DA0 @0x682da0); the NPC path (CTabSend::SendParcel @0x683c20) stops after the quick flag` | ✅ |  |
| 7 | int32 | int32 `ticketRef -- quick-send path only` | ✅ |  |

