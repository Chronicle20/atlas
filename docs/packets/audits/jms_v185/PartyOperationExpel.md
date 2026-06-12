# PartyOperationExpel (← `CField::SendKickPartyMsg`)

- **IDA:** 0x56cf23
- **Atlas file:** `../../libs/atlas-packet/party/serverbound/operation_expel.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op=5` | ✅ |  |
| 1 | int32 | int32 `charId` | ✅ |  |

