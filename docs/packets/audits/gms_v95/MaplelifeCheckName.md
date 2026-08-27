# MaplelifeCheckName (← `CUICharacterSaleDlg::SendCheckDuplicateIDPacket`)

- **IDA:** 0x777d20
- **Atlas file:** `libs/atlas-packet/maplelife/serverbound/check_name.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sCharName; MAPLELIFE_CHECK_NAME opcode 311` | ✅ |  |

