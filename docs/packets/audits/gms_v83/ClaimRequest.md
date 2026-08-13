# ClaimRequest (← `CWvsContext::SendClaimRequest`)

- **IDA:** 0xa2719c
- **Atlas file:** `libs/atlas-packet/report/serverbound/claim_request.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bChatClaim (flag; also gates the trailing chatLog string below) @0xa27640` | ✅ |  |
| 1 | string | string `sTargetCharacterName @0xa27659` | ✅ |  |
| 2 | byte | byte `nType @0xa27664` | ✅ |  |
| 3 | string | string `sContext (description) @0xa2767d` | ✅ |  |
| 4 | string | string `chatLog (only when bChatClaim) @0xa276f5` | ✅ |  |

