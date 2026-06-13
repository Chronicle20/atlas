# CharacterDespawn (← `CUserPool::OnUserLeaveField`)

- **IDA:** 0x9f727f
- **Atlas file:** `libs/atlas-packet/character/clientbound/despawn.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterID — user leaving the field` | ✅ |  |

