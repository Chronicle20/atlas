# AllCharacterListPong (← `CLogin::MakeVACDlg`)

- **IDA:** 0x632d3a
- **Atlas file:** `libs/atlas-packet/login/serverbound/all_character_list_pong.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `render flag (literal 1u in MakeVACDlg; ResetVAC sends 0; opcode 0x0F)` | ✅ |  |

