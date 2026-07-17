# ChatSetAvatarMegaphone (← `CWvsContext::OnSetAvatarMegaphone`)

- **IDA:** 0xb117bb
- **Atlas file:** `libs/atlas-packet/chat/clientbound/avatar_megaphone.go`
- **Variant:** JMS/v185
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `itemId` | ✅ |  |
| 1 | string | string `name (sender name)` | ✅ |  |
| 2 | string | string `lines[0] — DIVERGENT from GMS: jms_v185 decodes only ONE message line here (raw disasm b117fc/b1180b: exactly two DecodeStr calls total before the trailing fields), not four` | ✅ |  |
| 3 | int32 | int32 `channelId` | ✅ |  |
| 4 | byte | byte `whispersOn` | ✅ |  |
| 5 | bytes | bytes `AvatarLook::Decode(look) — opaque avatar block (model.Avatar recurse)` | ✅ |  |

