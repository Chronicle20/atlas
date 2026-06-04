# CharacterKeyMapAutoHp (← `CFuncKeyMappedMan::OnPetConsumeItemInit`)

- **IDA:** 0x5688c0
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/keymap_auto_hp.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nPetConsumeItemID (HP auto-pot item ID; 0 = use config default)` | ✅ |  |

