# CashItemUseTeleportRock (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0x9eb3e0
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_teleport_rock.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `update_time = get_update_time() — common header, leading (task-124 v95: MajorVersion()>=87 updateTimeFirst)` | ❌ | width mismatch |
| 1 | string | int16 `nPOS — common header` | ❌ | width mismatch |
| 2 | int32 | int32 `nItemID — common header; switches via get_consume_cash_item_type(nItemID) to a per-case body (opaque to this flat harvest, same class of gap as RunMapTransferItem)` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

