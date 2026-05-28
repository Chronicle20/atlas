# GuildWithdraw (← `CField::SendWithdrawGuildMsg`)

- **IDA:** 0x56dcc7
- **Atlas file:** `../../libs/atlas-packet/guild/serverbound/operation_withdraw.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `sub-op = 7 (WITHDRAW)` | ❌ | width mismatch |
| 1 | string | byte `` | ❌ | atlas: extra — client never reads this field |

