# ChatAvatarMegaphoneResult (← `CWvsContext::OnAvatarMegaphoneRes`)

- **IDA:** 0xa016c0
- **Atlas file:** `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code - v4 = code - 96; v4==0 (code 96 WAITING_LINE) and v4==1 (code 97 LEVEL_GATE) skip the DecodeStr below; any other code reads it. Base offset SHIFTED from 83 (gms_v83) to 96 (gms_v95).` | ✅ |  |
| 1 | string | string `message - only when code is neither 96 nor 97` | ✅ |  |

