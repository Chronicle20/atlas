# MonsterCarnival (← `CUIMonsterCarnival::RequestSend`)

- **IDA:** 0x80b4a0
- **Atlas file:** `libs/atlas-packet/monster/carnival/serverbound/monster_carnival.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

