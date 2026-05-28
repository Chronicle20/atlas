# PartyOperation (← `CField::SendWithdrawPartyMsg`)

- **IDA:** 0x56cba7
- **Atlas file:** `libs/atlas-packet/party/serverbound/operation.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sub-op = 2 (WITHDRAW)` | ✅ |  |

