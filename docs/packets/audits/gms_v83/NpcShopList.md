# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x7529ad
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcTemplateId (this[47])` | ✅ |  |
| 1 | int16 | int16 `commodity count (loop bound; no max cap)` | ✅ |  |
| 2 | int32 | int32 `[item] itemId / TemplateId` | ✅ |  |
| 3 | int32 | int32 `[item] mesoPrice (v99)` | ✅ |  |
| 4 | int32 | int32 `[item] tokenPrice (v100)` | ✅ |  |
| 5 | int32 | int32 `[item] period (v101)` | ✅ |  |
| 6 | int32 | int32 `[item] levelLimit (v102)` | ✅ |  |
| 7 | int16 | int16 `[item] quantity (non-ammo branch; ammo categories 207/233 read DecodeBuffer(8) unitPrice double instead)` | ✅ |  |
| 8 | int64 | int16 `[item] slotMax` | ❌ | width mismatch |
| 9 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

