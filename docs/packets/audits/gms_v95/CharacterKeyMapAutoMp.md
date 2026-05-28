# CharacterKeyMapAutoMp (← `CFuncKeyMappedMan::OnPetConsumeMPItemInit`)

- **IDA:** 0x5688f0
- **Atlas file:** `libs/atlas-packet/character/clientbound/keymap_auto_mp.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nPetConsumeMPItemID (MP auto-pot item ID; 0 = use config default)` | ✅ |  |

