# ChatAvatarMegaphoneResult (← `CWvsContext::OnAvatarMegaphoneRes`)

- **IDA:** 0xac2061
- **Atlas file:** `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- **Variant:** GMS/v87
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code — v = code - 88; v==0 (code 88 WAITING_LINE) and v==1 (code 89 LEVEL_GATE) skip the DecodeStr below; any other code reads it` | ✅ |  |
| 1 | string | string `message — only when code is neither 88 nor 89` | ✅ |  |

