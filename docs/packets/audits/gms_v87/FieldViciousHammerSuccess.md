# FieldViciousHammerSuccess (← `CField::OnItemUpgrade#Success`)

- **IDA:** 0x55fa12
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 63; success) — read in sub_88F348 (this[38]=v5), reached via sub_88F332 (gates a2==375) from the CField::OnItemUpgrade vtable forwarder` | ✅ |  |
| 1 | int32 | int32 `flag (0 = success -> "Increased available upgrade by 1. N upgrades are left" where N = 2-this[40]; non-0 -> "Unknown error %d" using this[39])` | ✅ |  |

