# AcceptTos (← `CLogin::OnAcceptLicense`)

- **IDA:** 0x633e1a
- **Atlas file:** `../../libs/atlas-packet/account/serverbound/accept_tos.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `accepted flag (literal 1u)` | ✅ |  |

