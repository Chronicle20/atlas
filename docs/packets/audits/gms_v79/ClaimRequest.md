# ClaimRequest (← `CWvsContext::SendClaimRequest`)

- **IDA:** 0x9711ff
- **Atlas file:** `libs/atlas-packet/report/serverbound/claim_request.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bChatClaim (flag; also gates the trailing chatLog string below) @0x9716a3` | ✅ |  |
| 1 | string | string `targetCharacterName @0x9716bc` | ✅ |  |
| 2 | byte | byte `reasonType @0x9716c7` | ✅ |  |
| 3 | string | string `description @0x9716e0` | ✅ |  |
| 4 | string | string `chatLog (only when bChatClaim) @0x971758` | ✅ |  |

