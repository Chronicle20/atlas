# FieldStalkResult (← `CField::OnStalkResult`)

- **IDA:** 0x51bce9
- **Atlas file:** `libs/atlas-packet/field/clientbound/stalk_result.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `count @0x51bcfa` | ✅ |  |
| 1 | int32 | int32 `charId @0x51bd11` | ✅ |  |
| 2 | byte | byte `flag @0x51bd13` | ✅ |  |
| 3 | string | string `name @0x51bd25` | ✅ |  |
| 4 | int32 | int32 `x @0x51bd37` | ✅ |  |
| 5 | int32 | int32 `y @0x51bd39` | ✅ |  |

