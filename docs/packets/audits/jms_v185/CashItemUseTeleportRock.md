# CashItemUseTeleportRock (← `CWvsContext::SendConsumeCashItemUseRequest`)

- **IDA:** 0xaef2f5
- **Atlas file:** `libs/atlas-packet/cash/serverbound/item_use_teleport_rock.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `update_time = get_update_time() — common header, leading (task-124 jms: MajorVersion()>=87 updateTimeFirst)` | ❌ | width mismatch |
| 1 | string | int16 `nPOS — common header` | ❌ | width mismatch |
| 2 | int32 | int32 `nItemID — common header; switches via get_consume_cash_item_type(nItemID) to a per-case body (opaque to this flat harvest, same class of gap as RunMapTransferItem)` | ✅ |  |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

