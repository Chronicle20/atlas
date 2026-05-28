# ShopOperationMoveFromCashInventory (← `CCashShop::OnMoveCashItemLtoS`)

- **IDA:** 0x4828e0
- **Atlas file:** `../../libs/atlas-packet/cash/serverbound/shop_operation_move_from_cash_inventory.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `liSN 8 bytes (serialNumber uint64)` | ❌ | width mismatch |
| 1 | byte | byte `nTI (inventoryType)` | ✅ |  |
| 2 | int16 | int16 `nPOS (slot)` | ✅ |  |


> ack: tool limitation — WriteLong int64 vs v95 EncodeBuffer(8) is a representation-only mismatch; both 8 bytes on the wire. Wire-correct. See _pending.md "Cash tool-limitation false positives".
