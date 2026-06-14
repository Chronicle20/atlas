# MonsterCarnival (← `CUIMonsterCarnival::RequestSend`)

- **IDA:** 0x89bdda
- **Atlas file:** `libs/atlas-packet/monster/carnival/serverbound/monster_carnival.go`
- **Variant:** GMS/v84
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

