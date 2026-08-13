# ClaimRequest (← `CWvsContext::SendClaimRequest`)

- **IDA:** 0xa72957
- **Atlas file:** `libs/atlas-packet/report/serverbound/claim_request.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bChatClaim (v60; also gates the trailing chatLog string below) @0xa72dfb` | ✅ |  |
| 1 | string | string `targetName @0xa72e14` | ✅ |  |
| 2 | byte | byte `reasonType (v52) @0xa72e1f` | ✅ |  |
| 3 | string | string `description @0xa72e38` | ✅ |  |
| 4 | string | string `chatLog (only when bChatClaim) @0xa72eb0` | ✅ |  |

