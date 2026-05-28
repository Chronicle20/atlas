# MonsterDestroy (← `CMobPool::OnMobLeaveField`)

- **IDA:** 0x6f8a1f
- **Atlas file:** `../../libs/atlas-packet/monster/clientbound/destroy.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwMobID` | ✅ |  |
| 1 | byte | byte `destroyType` | ✅ |  |
| 2 | int32 | int32 `dwSwallowCharacterID — only destroyType == 4` | ✅ |  |

