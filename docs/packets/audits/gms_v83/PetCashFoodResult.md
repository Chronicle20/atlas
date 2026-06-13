# PetCashFoodResult (← `CWvsContext::OnCashPetFoodResult`)

- **IDA:** 0xa29049
- **Atlas file:** `libs/atlas-packet/pet/clientbound/cash_food_result.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `result` | ✅ |  |
| 1 | byte | byte `petSlotIndex — only on result == 0` | ✅ |  |

