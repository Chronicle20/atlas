# InventoryCompartmentSortRequest (← `CWvsContext::SendSortItemRequest`)

- **IDA:** 0x954cfd
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/compartment_sort.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime — sub_954CFD (SendSortItemRequest twin) COutPacket(68) Encode4 get_update_time @0x954d40` | ✅ |  |
| 1 | byte | byte `compartmentType (nInvType, 1..5) — Encode1(a2) @0x954d4b` | ✅ |  |

