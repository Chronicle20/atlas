# NpcGuideTalkMessage (← `CUserLocal::OnTutorMsg#Message`)

- **IDA:** 0x916f60
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/guide_talk.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bByMessage = 0 (false) selects the string/message arm` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | int32 | int32 `width` | ✅ |  |
| 3 | int32 | int32 `duration` | ✅ |  |

