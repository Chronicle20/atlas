# PetFood (← `CWvsContext::SendPetFoodItemUseRequest`)

- **IDA:** 0xaee58f
- **Atlas file:** `../../libs/atlas-packet/pet/serverbound/food.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `get_update_time()` | ✅ |  |
| 1 | int16 | int16 `nPOS` | ✅ |  |
| 2 | int32 | int32 `nItemID` | ✅ |  |

