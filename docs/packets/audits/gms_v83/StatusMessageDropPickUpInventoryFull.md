# StatusMessageDropPickUpInventoryFull (← `CWvsContext::OnMessage`)

- **IDA:** 0xa209d4
- **Atlas file:** `../../libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode/sub-op byte (0=drop pick-up, 1=quest record, 2=cash item expire, 3=inc EXP, 4=inc SP/POP, 5=inc meso, 6=inc GP, 7=give buff, 8=general item expire, 9=system message, 0xA=quest record ex, 0xB=item protect expire, 0xC=item expire replace, 0xD=skill expire; v83 has 14 modes, v95 added mode 14=skill expire as 0xE)` | ✅ |  |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

