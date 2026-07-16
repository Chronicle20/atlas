# StatusMessageForfeitQuestRecord (← `CWvsContext::OnMessage#ForfeitQuestRecord`)

- **IDA:** 0x919604
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v72
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | int16 `questId @0x919627` | ❌ | width mismatch |
| 1 | int16 | byte `subtype 0 (forfeit, no further read) @0x919638` | ❌ | width mismatch |
| 2 | byte | byte `` | ⚠️ | atlas: trailing padding byte — client stops reading (harmless over-write) |

