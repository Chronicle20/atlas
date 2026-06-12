# InventoryAdd (← `CWvsContext::OnInventoryOperation#Add`)

- **IDA:** 0xa1ead9
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest/update_time flag (if !=0 reset excl + get_update_time)` | ✅ |  |
| 1 | byte | byte `count (number of operation entries)` | ✅ |  |
| 2 | byte | byte `action (0 = Add)` | ✅ |  |
| 3 | byte | byte `invType` | ✅ |  |
| 4 | int16 | int16 `slot` | ✅ |  |
| 5 | byte | bytes `asset GW_ItemSlotBase::Decode (case 0) — sub-struct, tool-opaque` | 🔍 | opaque type: model.Asset — register boundary (see opaque registry) |
| 6 | byte | byte `trailing addMov byte — ONLY if any entry set nCurItemPos (equip move/remove)` | ❌ | atlas: short — missing trailing field |

