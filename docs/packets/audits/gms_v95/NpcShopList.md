# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x6eab00
- **Atlas file:** `libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

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

## Loop bounds (tool limitation)

Rows 0–9 match the v95 client exactly. Rows 10–11 are artifacts of the flat
analyzer: atlas `ShopList.Encode` (shop_list.go) emits a **per-commodity loop**
whose body contains a **mutually-exclusive branch** (`if !IsAmmo { WriteShort
Quantity } else { WriteLong UnitPrice }`) followed by `WriteShort(SlotMax)`. The
analyzer cannot model either the loop repetition or the branch, so it inlines the
non-ammo `Quantity`(int16) AND the ammo `UnitPrice`(int64) consecutively and
misaligns `SlotMax`, producing the spurious row-10 width mismatch and row-11
"extra".

### Per-item shape verified against IDA `CShopDlg::SetShopDlg@0x6eab00`

Loop header (0x6eab44–0x6eab5e): `Decode4(npcTemplateId)` + `Decode2(count)`;
the loop runs `do { ... } while (idx < count)` — **no maximum-commodity cap** is
enforced by the client (no `min(count, MAX)` guard exists).

Per-item body (0x6eabc3–0x6eac62), one iteration:

| Offset | IDA call | Atlas `ShopCommodity` field | Width |
|---|---|---|---|
| 0x6eabc3 | `Decode4` | `TemplateId` | 4 |
| 0x6eabda | `Decode4` | `MesoPrice` | 4 |
| 0x6eabe5 | `Decode1` | `DiscountRate` (GMS≥87) | 1 |
| 0x6eabf1 | `Decode4` | `TokenTemplateId` (GMS≥95) | 4 |
| 0x6eabfb | `Decode4` | `TokenPrice` | 4 |
| 0x6eac05 | `Decode4` | `Period` | 4 |
| 0x6eac08 | `Decode4` | `LevelLimit` | 4 |
| 0x6eac46 | `DecodeBuffer(8)` (ammo arm: `itemId/10000 ∈ {207,233}`) | `UnitPrice` (float64) | 8 |
| 0x6eac55 | `Decode2` (non-ammo arm) | `Quantity` | 2 |
| 0x6eac62 | `Decode2` | `SlotMax` | 2 |

The `DiscountRate` byte is read for every version this binary serves (GMS≥87)
and `TokenTemplateId` for GMS≥95 — both atlas version gates fire for v95 and
match the client. The ammo/non-ammo branch key differs in form only: the client
branches on the item-category `itemId/10000 ∈ {207, 233}` (throwing-star/bullet
recharge categories) while atlas branches on the server-set `IsAmmo` bool; the
two are equivalent as long as the server sets `IsAmmo` for category 207/233
items (a server-data concern, not a wire-shape bug). The emitted byte shape
(8-byte double vs 2-byte short) is identical.

**Verdict: ⚠️ (tool-limitation, manually verified — per-item wire is correct).**

Ack: world-audit Phase 2e on 2026-05-28
