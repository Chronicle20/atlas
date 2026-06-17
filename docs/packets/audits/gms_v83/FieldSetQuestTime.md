# FieldSetQuestTime (← `CField::OnSetQuestTime`)

- **IDA:** 0x5378cd
- **Atlas file:** `libs/atlas-packet/field/clientbound/set_quest_time.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | int32 | int32 `` | ✅ |  |
| 2 | int64 | bytes `` | ✅ |  |
| 3 | int64 | bytes `` | ✅ |  |

