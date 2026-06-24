# CharacterSelectRegisterPic (← `CLogin::SendSelectCharPacket#CharacterSelectRegisterPic`)

- **IDA:** 0x66ddac
- **Atlas file:** `libs/atlas-packet/login/serverbound/character_select_register_pic.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (m_bLoginOpt == 0 branch; v10 flag)` | ✅ |  |
| 1 | int32 | int32 `dwCharacterID` | ✅ |  |
| 2 | string | string `sSPW (PIC); written when flag != 0` | ✅ |  |

