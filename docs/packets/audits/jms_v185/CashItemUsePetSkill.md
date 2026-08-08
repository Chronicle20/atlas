# CashItemUsePetSkill (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xaef2f5
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_pet_skill.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | int32 `update_time = get_update_time() — common header, leading (task-124 jms: MajorVersion()>=87 updateTimeFirst)` | ❌ | width mismatch |
| 1 | byte | int16 `nPOS — common header` | ❌ | atlas: short — missing trailing field |
| 2 | byte | int32 `nItemID — common header; switches via get_consume_cash_item_type(nItemID) to a per-case body (opaque to this flat harvest, same class of gap as RunMapTransferItem)` | ❌ | atlas: short — missing trailing field |

