# NpcGuideTalkMessage (← `CUserLocal::OnTutorMsg#Message`)

- **IDA:** 0xa2d342
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/guide_talk.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bByMessage = 0 (false) selects the string/message arm (@0xa2d35f)` | ✅ |  |
| 1 | string | string `message (@0xa2d38d)` | ✅ |  |
| 2 | int32 | int32 `width (@0xa2d39f)` | ✅ |  |
| 3 | int32 | int32 `duration (@0xa2d3a1)` | ✅ |  |


Ack: world-audit Phase 3 JMS185 npc domain on 2026-05-28
