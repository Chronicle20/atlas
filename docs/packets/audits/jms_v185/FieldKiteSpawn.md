# FieldKiteSpawn (← `CMessageBoxPool::OnMessageBoxEnterField`)

- **IDA:** 0x6d5978
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/kite_spawn.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `id (message-box object id @line67)` | ✅ |  |
| 1 | int32 | int32 `itemId / templateId (@line78)` | ✅ |  |
| 2 | string | string `string #1 (@line79; atlas message)` | ✅ |  |
| 3 | string | string `string #2 (@line85; atlas name)` | ✅ |  |
| 4 | int16 | int16 `x (@line92)` | ✅ |  |
| 5 | int16 | int16 `y / kiteType (@line94)` | ✅ |  |

