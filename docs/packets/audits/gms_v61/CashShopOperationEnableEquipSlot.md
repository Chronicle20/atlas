# CashShopOperationEnableEquipSlot (← `CCashShop::OnEnableEquipSlotExt`)

- **IDA:** 0x459928
- **Atlas file:** `libs/atlas-packet/cash/serverbound/shop_operation_enable_equip_slot.go`
- **Variant:** GMS/v61
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode 6\|7` | ✅ |  |
| 1 | int32 | byte `pointType` | ❌ | width mismatch |
| 2 | byte | int32 `currency` | ❌ | width mismatch |
| 3 | int32 | byte `flag` | ❌ | width mismatch |
| 4 | byte | int32 `serialNumber` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

