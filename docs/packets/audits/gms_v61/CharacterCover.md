# CharacterCover (← `CUserLocal::SetMonsterBookCover`)

- **IDA:** 0x6e78f2
- **Atlas file:** `libs/atlas-packet/character/serverbound/monsterbook/cover.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `coverCardId @0x6e7935 (COutPacket(53) ctor @0x6e7922)` | ✅ |  |

