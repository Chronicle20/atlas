# StatusMessageDropPickUpInventoryFull (← `CWvsContext::OnMessage`)

- **IDA:** 0xa06c90
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode/sub-op byte (0=drop pick-up, 1=quest record, 2=cash item expire, 3=inc EXP, 4=inc SP, 5=inc fame/POP, 6=inc meso, 7=inc GP, 8=give buff, 9=general item expire, 10=system message, 11=quest record ex, 12=item protect expire, 13=item expire replace, 14=skill expire)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

