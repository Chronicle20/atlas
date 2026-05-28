# NpcShopList (← `CShopDlg::SetShopDlg`)

- **IDA:** 0x79e4d0
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/shop_list.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `npcTemplateId (this+51)` | ✅ |  |
| 1 | int16 | int16 `commodity count (loop bound; no max cap)` | ✅ |  |
| 2 | int32 | int32 `[item] itemId / TemplateId` | ✅ |  |
| 3 | int32 | int32 `[item] mesoPrice (nPrice)` | ✅ |  |
| 4 | byte | byte `[item] discountRate (nDiscountRate; GMS>=87, present in v87 @0x79e54e; absent in v83)` | ✅ |  |
| 5 | int32 | int32 `[item] token field (single int @0x79e558; IDA labels nTokenItemID; atlas TokenPrice fills this slot for v87 — v95 splits into tokenItemId+tokenPrice, v83 has none)` | ✅ |  |
| 6 | int32 | int32 `[item] period (nItemPeriod)` | ✅ |  |
| 7 | int32 | int32 `[item] levelLimit (v106)` | ✅ |  |
| 8 | int16 | int16 `[item] quantity (non-ammo branch; ammo categories 207/233 read DecodeBuffer(8) unitPrice double instead)` | ✅ |  |
| 9 | int64 | int16 `[item] slotMax (v108)` | ❌ | width mismatch |
| 10 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |


## Loop bounds (tool limitation)

Rows 0–8 match the v87 client exactly. Rows 9–10 are artifacts of the flat
analyzer: atlas `ShopList.Encode` (shop_list.go) emits a **per-commodity loop**
whose body contains a **mutually-exclusive branch** (`if !IsAmmo { WriteShort
Quantity } else { WriteLong UnitPrice }`) followed by `WriteShort(SlotMax)`. The
analyzer cannot model either the loop repetition or the branch, so it inlines the
non-ammo `Quantity`(int16) AND the ammo `UnitPrice`(int64) consecutively and
misaligns `SlotMax`, producing the spurious row-9 width mismatch and row-10
"extra".

### Per-item shape verified against IDA `CShopDlg::SetShopDlg@0x79e4d0`

Loop header (0x79e4f4–0x79e4fa): `Decode4(npcTemplateId)` + `Decode2(count)`;
the loop runs `while ( ++idx >= count ) break` — **no maximum-commodity cap** is
enforced by the client (no `min(count, MAX)` guard exists).

Per-item body (0x79e527–0x79e5b1), one iteration:

| Offset | IDA call | Atlas `ShopCommodity` field | Width |
|---|---|---|---|
| 0x79e527 | `Decode4` | `TemplateId` | 4 |
| 0x79e541 | `Decode4` | `MesoPrice` | 4 |
| 0x79e54e | `Decode1` | `DiscountRate` (GMS≥87) | 1 |
| 0x79e558 | `Decode4` | `TokenPrice` (single token int; v95 splits into TokenTemplateId+TokenPrice) | 4 |
| 0x79e562 | `Decode4` | `Period` | 4 |
| 0x79e565 | `Decode4` | `LevelLimit` | 4 |
| 0x79e59b | `DecodeBuffer(8)` (ammo arm: `itemId/10000 ∈ {207,233}`) | `UnitPrice` (float64) | 8 |
| 0x79e5ac | `Decode2` (non-ammo arm) | `Quantity` | 2 |
| 0x79e5b1 | `Decode2` | `SlotMax` | 2 |

KEY CROSS-VERSION FINDING: v87 carries `DiscountRate`(byte, GMS≥87) + exactly ONE
token int. v83 has NEITHER discountRate NOR any token int (2 ints after meso); v95
has discountRate + TWO token ints (TokenTemplateId + TokenPrice, 4 ints after
meso). Atlas gates `WriteByte(DiscountRate)` on GMS≥87 and `WriteInt(TokenTemplateId)`
on GMS≥95, then unconditionally writes `TokenPrice`/`Period`/`LevelLimit`. For v87
atlas emits meso + discountRate + [no TokenTemplateId, <95] + TokenPrice + Period +
LevelLimit = byte + 3 ints, matching the v87 client byte+3-int read exactly (the
single v87 token slot is filled by atlas `TokenPrice`). The ammo/non-ammo branch
key differs in form only (client `itemId/10000 ∈ {207,233}` vs atlas `IsAmmo`
bool); the emitted byte shape (8-byte double vs 2-byte short) is identical.

**Verdict: ⚠️ (tool-limitation, manually verified — per-item wire is correct).**

Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
