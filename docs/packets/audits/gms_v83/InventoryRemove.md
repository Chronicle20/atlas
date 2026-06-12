# InventoryRemove (← `CWvsContext::OnInventoryOperation#Remove`)

- **IDA:** 0xa1ead9
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v83
- **Branch depth:** 2
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest/update_time flag` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | byte | byte `action (3 = Remove)` | ✅ |  |
| 3 | byte | byte `invType` | ✅ |  |
| 4 | int16 | int16 `slot` | ✅ |  |
| 5 | byte | int32 `(equip path) cash/serial; addMov byte if invType==1 && slot<0` | ❌ | width mismatch |

