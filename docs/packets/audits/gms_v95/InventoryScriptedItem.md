# InventoryScriptedItem (← `CWvsContext::SendScriptRunItemRequest`)

- **IDA:** 0x9de7a0
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/scripted_item.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `update_time (get_update_time(); COutPacket::Encode4 at 0x9de857; task-230 verify gms_v95)` | ✅ |  |
| 1 | int16 | int16 `nPOS source slot (COutPacket::Encode2 at 0x9de865; task-230 verify gms_v95)` | ✅ |  |
| 2 | int32 | int32 `nItemID (COutPacket::Encode4 at 0x9de86f; opcode 84/0x54 from COutPacket::COutPacket(&oPacket, 84) at 0x9de840; task-230 verify gms_v95)` | ✅ |  |

