# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x7529ad
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

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


## Loop bounds (tool limitation)

Rows 0–7 match the v83 client exactly. Rows 8–9 are artifacts of the flat
analyzer: atlas `ShopList.Encode` (shop_list.go) emits a **per-commodity loop**
whose body contains a **mutually-exclusive branch** (`if !IsAmmo { WriteShort
Quantity } else { WriteLong UnitPrice }`) followed by `WriteShort(SlotMax)`. The
analyzer cannot model the loop repetition or the branch, so it inlines the
non-ammo `Quantity`(int16) and the ammo `UnitPrice`(int64) consecutively and
misaligns `SlotMax`, producing the spurious row-8 width mismatch and row-9
"extra".

### Per-item shape verified against IDA `CShopDlg::SetShopDlg@0x7529ad`

Loop header (0x7529d1–0x7529e3): `Decode4(npcTemplateId)` + `Decode2(count)`;
the loop runs `while (idx < count)` — **no maximum-commodity cap**.

Per-item body (0x752a01–0x752a78), one iteration:

| Offset | IDA call | Atlas `ShopCommodity` field | Width |
|---|---|---|---|
| 0x752a01 | `Decode4` | `TemplateId` | 4 |
| 0x752a18 | `Decode4` | `MesoPrice` | 4 |
| 0x752a22 | `Decode4` | `TokenPrice` | 4 |
| 0x752a2c | `Decode4` | `Period` | 4 |
| 0x752a2f | `Decode4` | `LevelLimit` | 4 |
| 0x752a62 | `DecodeBuffer(8)` (ammo arm: `itemId/10000 ∈ {207,233}`) | `UnitPrice` (float64) | 8 |
| 0x752a73 | `Decode2` (non-ammo arm) | `Quantity` | 2 |
| 0x752a78 | `Decode2` | `SlotMax` | 2 |

v83 reads NO `DiscountRate` byte (atlas gates it GMS≥87) and NO `TokenTemplateId`
int (atlas gates it GMS≥95) — both gates are correctly OFF for v83, so the v83
per-item shape is `TemplateId + 4×int32 + (ammo?int64:int16) + int16(SlotMax)`,
exactly what the client reads. The ammo/non-ammo branch key (client:
`itemId/10000 ∈ {207,233}`; atlas: server-set `IsAmmo`) is equivalent.

**Verdict: ⚠️ (tool-limitation, manually verified — per-item wire is correct for v83).**

Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
