# ParcelActionDiscard (← `CTabReceive::DiscardParcel`)

- **IDA:** 0x68f5e0
- **Atlas file:** `libs/atlas-packet/parcel/serverbound/action_parcel_id.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `parcelId` | ✅ |  |

