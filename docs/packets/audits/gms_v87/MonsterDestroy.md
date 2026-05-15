# MonsterDestroy (← `CMobPool::OnMobLeaveField`)

- **IDA:** 0x6b5169
- **Atlas file:** `libs/atlas-packet/monster/clientbound/destroy.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobID (uniqueId)` | ✅ |  |
| 1 | byte | byte `destroyType` | ✅ |  |
| 2 | byte | int32 `dwSwallowCharacterID — only if destroyType == 4` | ❌ | atlas: short — missing trailing field |

