# CharacterMobBanishPlayer (← `CUserLocal::SendBanMapByMobRequest`)

- **IDA:** 0xa28621
- **Atlas file:** `libs/atlas-packet/character/serverbound/mob_banish_player.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

