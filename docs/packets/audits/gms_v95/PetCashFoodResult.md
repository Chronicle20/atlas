# PetCashFoodResult (← `CWvsContext::OnCashPetFoodResult`)

- **IDA:** 0x9f7180
- **Atlas file:** `../../libs/atlas-packet/pet/clientbound/cash_food_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `result (v3: 0 = success, 1 = error notice)` | ✅ |  |
| 1 | byte | byte `petSlotIndex — only on result == 0 (success path picks pet)` | ✅ |  |

