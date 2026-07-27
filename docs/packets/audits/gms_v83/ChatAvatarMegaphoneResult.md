# ChatAvatarMegaphoneResult (← `CWvsContext::OnAvatarMegaphoneRes`)

- **IDA:** 0xa2a3bc
- **Atlas file:** `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- **Variant:** GMS/v83
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code — v3 = code - 83; v3==0 (code 83 WAITING_LINE) and v3==1 (code 84 LEVEL_GATE) skip the DecodeStr below; any other code reads it` | ✅ |  |
| 1 | string | string `message — only when code is neither 83 nor 84` | ✅ |  |

