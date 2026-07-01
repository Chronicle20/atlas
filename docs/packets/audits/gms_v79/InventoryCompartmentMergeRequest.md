# InventoryCompartmentMergeRequest (← `CWvsContext::SendGatherItemRequest`)

- **IDA:** 0x954c6b
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/compartment_merge.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `updateTime — sub_954C6B (SendGatherItemRequest twin) COutPacket(67) Encode4 get_update_time @0x954cae` | ✅ |  |
| 1 | byte | byte `compartmentType (nInvType, 1..5) — Encode1(a2) @0x954cb9` | ✅ |  |

