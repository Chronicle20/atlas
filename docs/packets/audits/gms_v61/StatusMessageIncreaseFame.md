# StatusMessageIncreaseFame (← `CWvsContext::OnMessage#IncreaseFame`)

- **IDA:** 0x84471b
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int32 `amount @0x91a471` | ❌ | width mismatch |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

