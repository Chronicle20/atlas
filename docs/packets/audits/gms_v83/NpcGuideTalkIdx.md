# NpcGuideTalkIdx (← `CUserLocal::OnTutorMsg#Idx`)

- **IDA:** 0x960239
- **Atlas file:** `../../libs/atlas-packet/npc/clientbound/guide_talk.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bByMessage = 1 (true) selects the hint-index arm` | ✅ |  |
| 1 | int32 | int32 `hintId / balloon type` | ✅ |  |
| 2 | int32 | int32 `duration` | ✅ |  |


Ack: world-audit Phase 3 v83 (12b npc) on 2026-05-28
