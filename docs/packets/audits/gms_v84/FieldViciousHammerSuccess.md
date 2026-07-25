# FieldViciousHammerSuccess (← `CField::OnItemUpgrade#Success`)

- **IDA:** 0x5443af
- **Atlas file:** `libs/atlas-packet/field/clientbound/vicious_hammer.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (= 61; success) — sub_85676C a1[34]=v6` | ✅ |  |
| 1 | int32 | int32 `flag (0 = success -> "Increased available upgrade by 1. N upgrades are left" StringPool 5059 where N = 2-a1[36]; non-0 -> "Unknown error %d" StringPool 5757 using a1[35])` | ✅ |  |

