# ParcelActionReceive (← `CTabReceive::ReceiveParcel`)

- **IDA:** 0x6f0ca3
- **Atlas file:** `libs/atlas-packet/parcel/serverbound/action_parcel_id.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `parcelId` | ✅ |  |

