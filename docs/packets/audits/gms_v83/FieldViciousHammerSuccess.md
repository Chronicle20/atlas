# FieldViciousHammerSuccess (← `CField::OnItemUpgrade#Success`)

- **IDA:** 0x537f8c
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 61; success) — read in sub_82B2C3 (this[34]=v5), reached via sub_82B2AD (gates nType==354) from the CField::OnItemUpgrade vtable forwarder` | ✅ |  |
| 1 | int32 | int32 `flag (0 = success -> "Increased available upgrade by 1. N upgrades are left" where N = 2-this[36]; non-0 -> "Unknown error %d" using this[35])` | ✅ |  |

