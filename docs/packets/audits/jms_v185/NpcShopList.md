# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x7c6536
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

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


## Manual verdict (JMS v185, `CShopDlg::SetShopDlg` @0x7c6536)

Rows 8-9 (❌) are a loop-flattening + ammo-branch-selection artifact, NOT a wire bug. The
per-commodity loop body matches field-for-field (rows 0-7 ✅): `Decode4 itemId + Decode4
price + Decode4 tokenPrice + Decode4 itemPeriod + Decode4 levelLimited + (ammo ?
DecodeBuffer(8) : Decode2 quantity) + Decode2 maxPerSlot`. The analyzer flattens the loop to
a single iteration and picks atlas's `WriteLong` ammo branch (int64) against the IDA's
non-ammo `Decode2 maxPerSlot`, producing the row-8 width mismatch and a row-9 trailing
"extra". JMS185 has NO discountRate byte and NO tokenItemId int (the GMS>=87/>=95 fields);
atlas gates both on `Region==GMS`, so for JMS it emits exactly the 5-int layout JMS185 reads.
Carry-forward manual-verify (matches GMS v95 loop-bound handling).

Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
