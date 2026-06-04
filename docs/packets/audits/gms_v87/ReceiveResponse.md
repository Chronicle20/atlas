# ReceiveResponse (← `CWvsContext::OnGivePopularityResult#ReceiveResponse`)

- **IDA:** 0xab9c24
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 5 = RECEIVE)` | ✅ |  |
| 1 | string | string `fromName (character who gave fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |


## Manual analysis

v87 vs v95/v83: gate confirmed ✅. `OnGivePopularityResult` @ 0xab9c24, mode 5 (case `v8==1`): reads DecodeStr(fromName) + Decode1(bInc). Atlas matches.

Ack: misc-audit Phase 3 v87 on 2026-06-03
