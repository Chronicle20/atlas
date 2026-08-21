# MaplelifeCheckName (← `CUICharacterSaleDlg::SendCheckDuplicateIDPacket`)

- **IDA:** 0x756250
- **Atlas file:** `libs/atlas-packet/maplelife/serverbound/check_name.go`
- **Variant:** GMS/v92
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | string | string `sCharName; MAPLELIFE_CHECK_NAME opcode 301 (0x12D); function renamed from sub_756250 to the mangled CUICharacterSaleDlg::SendCheckDuplicateIDPacket symbol this pass` | ✅ |  |

