# MaplelifeCheckName (← `CUICharacterSaleDlg::SendCheckDuplicateIDPacket`)

- **IDA:** 0x82e04d
- **Atlas file:** `libs/atlas-packet/maplelife/serverbound/check_name.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sCharName; MAPLELIFE_CHECK_NAME opcode 270 (0x10E)` | ✅ |  |

