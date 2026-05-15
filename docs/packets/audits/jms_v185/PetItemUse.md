# PetItemUse (← `CWvsContext::SendStatChangeItemUseRequestByPetQ`)

- **IDA:** 0xaee1d4
- **Atlas file:** `libs/atlas-packet/pet/serverbound/item_use.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int64 | bytes `liPetSN (8 bytes)` | ❌ | width mismatch |
| 1 | byte | byte `bBuffSkill` | ✅ |  |
| 2 | int32 | int32 `get_update_time()` | ✅ |  |
| 3 | int16 | int16 `nPOS` | ✅ |  |
| 4 | int32 | int32 `nItemID` | ✅ |  |

