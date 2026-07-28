# StatusMessageDropPickUpInventoryFull (← `CWvsContext::OnMessage#DropPickUpInventoryFull`)

- **IDA:** 0x8438b5
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** ⚠️

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `drop type -1 (inventory full, no further read) @0x9192f4` | ✅ |  |
| 1 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

