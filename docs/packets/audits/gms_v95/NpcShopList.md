# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x6eab00
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcTemplateId (m_dwNpcTemplateID)` | ✅ |  |
| 1 | int16 | int16 `commodity count (loop bound; no max cap)` | ✅ |  |
| 2 | int32 | int32 `[item] itemId / TemplateId` | ✅ |  |
| 3 | int32 | int32 `[item] mesoPrice (v95)` | ✅ |  |
| 4 | byte | byte `[item] discountRate (GMS>=87; v96)` | ✅ |  |
| 5 | int32 | int32 `[item] tokenItemId (GMS>=95; v97)` | ✅ |  |
| 6 | int32 | int32 `[item] tokenPrice (v98)` | ✅ |  |
| 7 | int32 | int32 `[item] period (v99)` | ✅ |  |
| 8 | int32 | int32 `[item] levelLimit (v100)` | ✅ |  |
| 9 | int16 | int16 `[item] quantity (non-ammo arm; ammo itemId/10000 in {207,233} uses DecodeBuffer(8) unitPrice instead)` | ✅ |  |
| 10 | int64 | int16 `[item] slotMax (v102)` | ❌ | width mismatch |
| 11 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

