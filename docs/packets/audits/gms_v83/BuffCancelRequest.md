# BuffCancelRequest (← `CUserLocal::SendSkillCancelRequest`)

- **IDA:** 0x96d873
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/buff_cancel.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nSkillID (skill to cancel; remapped for Aran/Evan aliases)` | ✅ |  |

