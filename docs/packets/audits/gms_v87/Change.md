# Change (← `CWvsContext::SendGivePopularityRequest`)

- **IDA:** 0xabb983
- **Atlas file:** `libs/atlas-packet/fame/serverbound/change.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | int32 `dwCharacterId (target character ID as uint32)` | ✅ |  |
| 1 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `CWvsContext::SendGivePopularityRequest` @ 0xabb983: Encode4(dwCharacterId) + Encode1(bInc). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
