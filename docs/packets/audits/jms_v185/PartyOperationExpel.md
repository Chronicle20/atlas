# PartyOperationExpel (← `CField::SendKickPartyMsg`)

- **IDA:** 0x56cf23
- **Atlas file:** `libs/atlas-packet/party/serverbound/operation_expel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `sub-op = 3 (KICK)` | ❌ | width mismatch |
| 1 | byte | string `target character name` | ❌ | atlas: short — missing trailing field |

