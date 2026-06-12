# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x7c6536
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcTemplateId (m_dwNpcTemplateID @0x7c655a)` | ✅ |  |
| 1 | int16 | int16 `commodity count (loop bound; no max cap @0x7c6560)` | ✅ |  |
| 2 | int32 | int32 `[item] itemId / TemplateId (@0x7c658d)` | ✅ |  |
| 3 | int32 | int32 `[item] price / mesoPrice (@0x7c65a7) -- NO discountRate byte, NO tokenItemId int (JMS lacks the GMS>=87/>=95 fields)` | ✅ |  |
| 4 | int32 | int32 `[item] tokenPrice (@0x7c65b1)` | ✅ |  |
| 5 | int32 | int32 `[item] itemPeriod (@0x7c65bb)` | ✅ |  |
| 6 | int32 | int32 `[item] levelLimited (@0x7c65be)` | ✅ |  |
| 7 | int16 | int16 `[item] quantity (non-ammo arm; ammo itemId/10000 in {207,233} uses DecodeBuffer(8) unitPrice instead @0x7c6605/0x7c65f4)` | ✅ |  |
| 8 | int64 | int16 `[item] maxPerSlot (@0x7c661e)` | ❌ | width mismatch |
| 9 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

