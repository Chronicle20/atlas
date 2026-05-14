# CharacterDespawn (← `CUserPool::OnUserLeaveField`)

- **IDA:** 0x94d4c0
- **Atlas file:** `libs/atlas-packet/character/clientbound/despawn.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterID — user leaving the field` | ✅ |  |

