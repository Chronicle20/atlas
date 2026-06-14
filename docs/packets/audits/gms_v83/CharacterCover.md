# CharacterCover (← `CUserLocal::SetMonsterBookCover`)

- **IDA:** 0x95fb3e
- **Atlas file:** `libs/atlas-packet/character/serverbound/monsterbook/cover.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

