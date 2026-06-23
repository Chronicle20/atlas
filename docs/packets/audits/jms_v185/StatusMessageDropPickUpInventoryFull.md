# StatusMessageDropPickUpInventoryFull (← `CWvsContext::OnMessage#DropPickUpInventoryFull`)

- **IDA:** 0xb07a01
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (0 = drop pick-up)` | ✅ |  |
| 1 | byte | byte `inner disc int8 = -1 (default → 'cannot pick up any more', StringPool 295)` | ✅ |  |

