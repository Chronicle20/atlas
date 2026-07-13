# FieldViciousHammerSuccess (← `CField::OnItemUpgrade#Success`)

- **IDA:** 0x799d61
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 60; success) - this[32]=v6 read by CUIItemUpgrade::OnItemUpgradeResult 0x799d61, reached via CUIItemUpgrade::OnPacket 0x799d4b (a2==330)` | ✅ |  |
| 1 | int32 | int32 `flag (0 = success -> str 5016 'Increased available upgrade by 1. N upgrades are left' where N = 2-this[34]; non-0 -> str 5343 unknown-error using this[33])` | ✅ |  |

