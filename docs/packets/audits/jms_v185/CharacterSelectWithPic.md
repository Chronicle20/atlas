# CharacterSelectWithPic (← `CLogin::SendSelectCharPacket#CharacterSelectWithPic`)

- **IDA:** 0x66ddac
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select_with_pic.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sSPW (PIC, m_bLoginOpt == 1 branch)` | ✅ |  |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |

