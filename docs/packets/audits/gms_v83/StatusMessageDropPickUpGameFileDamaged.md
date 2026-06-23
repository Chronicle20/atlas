# StatusMessageDropPickUpGameFileDamaged (← `CWvsContext::OnMessage#DropPickUpGameFileDamaged`)

- **IDA:** 0xa20ad9
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (0 = drop pick-up)` | ✅ |  |
| 1 | byte | byte `inner disc int8 = -3 (game file damaged → StringPool 5317 + chat 5311)` | ✅ |  |

