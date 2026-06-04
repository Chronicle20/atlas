# InventoryChangeBatch (← `CWvsContext::OnInventoryOperation#ChangeBatch`)

- **IDA:** 0xa1ead9
- **Atlas file:** `../../libs/atlas-packet/inventory/clientbound/change_batch.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `exclRequest/update_time flag` | ✅ |  |
| 1 | byte | byte `count` | ✅ |  |
| 2 | bytes | byte `per-entry action switch (0/1/2/3) repeated count times` | ✅ |  |
| 3 | byte | byte `trailing addMov byte ONLY if any entry set nCurItemPos` | ✅ |  |

