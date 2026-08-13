# CharacterShowCombo (← `CUserLocal::OnIncComboResponse`)

- **IDA:** 0x99f31e
- **Atlas file:** `libs/atlas-packet/character/clientbound/show_combo.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `count (CUserLocal::OnIncComboResponse: Decode4 store to the m_nCombo mirror, then get_update_time() timestamp, then DrawCombo(this); task-217, design.md §2.2)` | ✅ |  |

