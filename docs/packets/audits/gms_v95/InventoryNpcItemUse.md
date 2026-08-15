# InventoryNpcItemUse (← `CWvsContext::SendSelectNpcItemUseRequest`)

- **IDA:** 0x9da430
- **Atlas file:** `libs/atlas-packet/inventory/serverbound/npc_item_use.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int16 | int16 `nPOS source slot (COutPacket::Encode2 at 0x9da5cd; task-230 verify gms_v95)` | ✅ |  |
| 1 | int32 | int32 `nItemID (v4/nItemID; COutPacket::Encode4 at 0x9da5d7; opcode 123/0x7B from COutPacket::COutPacket(&oPacket, 123) at 0x9da5b7; task-230 verify gms_v95)` | ✅ |  |

