# InventoryChangeBatch (← `CWvsContext::OnInventoryOperation#ChangeBatch`)

- **IDA:** 0x96953e
- **Atlas file:** `libs/atlas-packet/inventory/clientbound/change_batch.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest flag @0x969556; Atlas WriteBool(!silent)` | ✅ |  |
| 1 | byte | byte `count @0x96959a; Atlas WriteByte(len(entries))` | ✅ |  |
| 2 | bytes | byte `per-entry action switch (0/1/2/3) repeated count times @0x9695b8` | ✅ |  |
| 3 | byte | byte `trailing addMov byte @0x96997e — post-loop, ONLY if an entry set nCurItemPos (equip move/remove with a negative slot). For count=1 this coincides with the per-entry inline addMov Atlas writes.` | ✅ |  |

