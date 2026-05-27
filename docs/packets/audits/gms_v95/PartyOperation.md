# PartyOperation (← `CField::SendWithdrawPartyMsg`)

- **IDA:** 0x52edb0
- **Atlas file:** `libs/atlas-packet/party/serverbound/operation.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `op=2 (LEAVE_PARTY)` | ✅ |  |
| 1 | byte | byte `trailing 0 — not modelled in atlas Operation dispatcher` | ❌ | atlas: short — missing trailing field |

