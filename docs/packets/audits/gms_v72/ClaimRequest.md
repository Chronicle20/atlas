# ClaimRequest (← `CWvsContext::SendClaimRequest`)

- **IDA:** 0x91f2b4
- **Atlas file:** `libs/atlas-packet/report/serverbound/claim_request.go`
- **Variant:** GMS/v72
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bChatClaim (flag; also gates the trailing chatLog string below) @0x91f758` | ✅ |  |
| 1 | string | string `targetCharacterName @0x91f771` | ✅ |  |
| 2 | byte | byte `reasonType @0x91f77c` | ✅ |  |
| 3 | string | string `description @0x91f795` | ✅ |  |
| 4 | string | string `chatLog (only when bChatClaim) @0x91f80d` | ✅ |  |

