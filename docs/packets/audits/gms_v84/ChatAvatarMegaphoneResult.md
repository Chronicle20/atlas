# ChatAvatarMegaphoneResult (← `CWvsContext::OnAvatarMegaphoneRes`)

- **IDA:** 0xa75b7f
- **Atlas file:** `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- **Variant:** GMS/v84
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `code — v = code - 86; v==0 (code 86 WAITING_LINE) and v==1 (code 87 LEVEL_GATE) skip the DecodeStr below; any other code reads it` | ✅ |  |
| 1 | string | string `message — only when code is neither 86 nor 87` | ✅ |  |

