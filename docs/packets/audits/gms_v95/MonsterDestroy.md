# MonsterDestroy (← `CMobPool::OnMobLeaveField`)

- **IDA:** 0x658b90
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/destroy.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobID (uniqueId)` | ✅ |  |
| 1 | byte | byte `destroyType (v4)` | ✅ |  |
| 2 | int32 | int32 `dwSwallowCharacterID — conditional: only if destroyType == 4 (swallowed by character/eater)` | ✅ |  |

