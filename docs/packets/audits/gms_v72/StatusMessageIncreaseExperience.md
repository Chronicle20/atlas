# StatusMessageIncreaseExperience (← `CWvsContext::OnMessage#IncreaseExperience`)

- **IDA:** 0x919e04
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v72
- **Branch depth:** 4
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `white @0x919e1c` | ✅ |  |
| 1 | byte | int32 `amount @0x919e29` | ❌ | width mismatch |
| 2 | int32 | byte `inChat @0x919e32` | ❌ | width mismatch |
| 3 | byte | int32 `monsterBookBonus @0x919e3f` | ❌ | width mismatch |
| 4 | int32 | byte `mobEventBonusPct @0x919e49` | ❌ | width mismatch |
| 5 | byte | byte `partyBonusPct @0x919e56` | ✅ |  |
| 6 | byte | int32 `weddingBonusEXP @0x919e63` | ❌ | width mismatch |
| 7 | int32 | byte `playTimeHour (mob>0) @0x919e74` | ❌ | width mismatch |
| 8 | byte | byte `questBonusRate (inChat) @0x919e8b` | ✅ |  |
| 9 | byte | byte `questBonusRemain (rate>0) @0x919ea2` | ✅ |  |
| 10 | byte | byte `partyBonusEventRate @0x919eb4` | ✅ |  |
| 11 | byte | int32 `partyBonusExp (sole trailing int) @0x919ec1` | ❌ | width mismatch |
| 12 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |

