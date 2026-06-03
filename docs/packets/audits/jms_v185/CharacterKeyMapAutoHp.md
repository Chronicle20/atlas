# CharacterKeyMapAutoHp (← `CFuncKeyMappedMan::OnPetConsumeItemInit`)

- **IDA:** 0x5e7a23
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/keymap_auto_hp.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `m_nPetConsumeItemID (HP auto-pot item ID)` | ✅ |  |

