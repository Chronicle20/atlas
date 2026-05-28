# BuffCancelRequest (← `CUserLocal::SendSkillCancelRequest`)

- **IDA:** 0x9f22b8
- **Atlas file:** `../../libs/atlas-packet/character/serverbound/buff_cancel.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `nSkillID (skill to cancel)` | ✅ |  |

