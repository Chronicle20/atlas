# StatusMessageDropLossUnStackableItem (← `CWvsContext::OnMessage#DropLossUnStackableItem`)

- **IDA:** 0xab818c
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `outer mode (0 = drop pick-up)` | ✅ |  |
| 1 | byte | byte `inner disc int8 = 2 (unstackable / equip item)` | ✅ |  |
| 2 | int32 | int32 `itemId` | ✅ |  |

