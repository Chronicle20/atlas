# FieldAriantArenaUserScore (← `CField_AriantArena::OnUserScore`)

- **IDA:** 0x528799
- **Atlas file:** `libs/atlas-packet/field/clientbound/ariant_arena_user_score.go`
- **Variant:** GMS/v79
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `count` | ✅ |  |
| 1 | string | string `sName` | ✅ |  |
| 2 | int32 | int32 `nScore` | ✅ |  |

