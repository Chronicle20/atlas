# CharacterMobBanishPlayer (← `CUserLocal::SendBanMapByMobRequest`)

- **IDA:** 0x99b16a
- **Atlas file:** `libs/atlas-packet/character/serverbound/mob_banish_player.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

