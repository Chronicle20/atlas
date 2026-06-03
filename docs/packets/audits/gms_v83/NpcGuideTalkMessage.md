# NpcGuideTalkMessage (← `CUserLocal::OnTutorMsg#Message`)

- **IDA:** 0x960239
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/guide_talk.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bByMessage = 0 (false) selects the string/message arm` | ✅ |  |
| 1 | string | string `message` | ✅ |  |
| 2 | int32 | int32 `width` | ✅ |  |
| 3 | int32 | int32 `duration` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
