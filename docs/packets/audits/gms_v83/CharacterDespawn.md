# CharacterDespawn (← `CUserPool::OnUserLeaveField`)

- **IDA:** 0x9722f9
- **Atlas file:** `libs/atlas-packet/character/clientbound/despawn.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterID — user leaving the field` | ✅ |  |

