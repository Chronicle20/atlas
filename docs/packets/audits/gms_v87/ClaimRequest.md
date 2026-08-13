# ClaimRequest (← `CWvsContext::SendClaimRequest`)

- **IDA:** 0xabee09
- **Atlas file:** `libs/atlas-packet/report/serverbound/claim_request.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bChatClaim (a2[0]; also gates the trailing chatLog string below) @0xabf2ad` | ✅ |  |
| 1 | string | string `targetName @0xabf2c6` | ✅ |  |
| 2 | byte | byte `reasonType (v53[0]) @0xabf2d1` | ✅ |  |
| 3 | string | string `description @0xabf2ea` | ✅ |  |
| 4 | string | string `chatLog (only when bChatClaim) @0xabf362` | ✅ |  |

