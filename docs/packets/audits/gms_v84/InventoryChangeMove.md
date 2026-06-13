# InventoryChangeMove (← `CWvsContext::OnInventoryOperation#ChangeMove`)

- **IDA:** 0xa69d8f
- **Atlas file:** `libs/atlas-packet/inventory/clientbound/change.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | byte | byte `` | ✅ |  |
| 3 | byte | byte `` | ✅ |  |
| 4 | int16 | int16 `` | ✅ |  |
| 5 | int16 | byte `` | ❌ | width mismatch |
| 6 | byte | unresolved `packet var passed to unresolved/indirect call; hand-trace` | 🚫 | IDA read-order unresolved: packet var passed to unresolved/indirect call; hand-trace |
| 7 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 8 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 9 | byte | int16 `` | ❌ | atlas: short — missing trailing field |
| 10 | byte | byte `` | ❌ | atlas: short — missing trailing field |

