# ClaimRequest (← `CWvsContext::SendClaimRequest`)

- **IDA:** 0xa05fb0
- **Atlas file:** `libs/atlas-packet/report/serverbound/claim_request.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `bChatClaim (v33; also gates the trailing chatLog string below) @0xa065ee` | ✅ |  |
| 1 | string | string `targetCharacterName @0xa0660d` | ✅ |  |
| 2 | byte | byte `reasonType (nType) @0xa0661b` | ✅ |  |
| 3 | string | string `description (sContext) @0xa0663a` | ✅ |  |
| 4 | string | string `chatLog (only when bChatClaim) @0xa066f5` | ✅ |  |

