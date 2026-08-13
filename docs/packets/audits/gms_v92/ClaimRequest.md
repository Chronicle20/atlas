# ClaimRequest (← `CWvsContext::SendClaimRequest`)

- **IDA:** 0x9d9c30
- **Atlas file:** `libs/atlas-packet/report/serverbound/claim_request.go`
- **Variant:** GMS/v92
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bChatClaim (flag; also gates the trailing chatLog string below) @0x9da26e` | ✅ |  |
| 1 | string | string `sTargetCharacterName @0x9da28d` | ✅ |  |
| 2 | byte | byte `nType @0x9da29b` | ✅ |  |
| 3 | string | string `sContext @0x9da2ba` | ✅ |  |
| 4 | string | string `chatLog (only when bChatClaim) @0x9da375` | ✅ |  |

