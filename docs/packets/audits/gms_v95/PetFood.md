# PetFood (← `CWvsContext::SendPetFoodItemUseRequest`)

- **IDA:** 0x9d9f20
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/food.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `get_update_time() (tick)` | ✅ |  |
| 1 | int16 | int16 `nPOS (item slot)` | ✅ |  |
| 2 | int32 | int32 `nItemID` | ✅ |  |

