# ReceiveResponse (← `CWvsContext::OnGivePopularityResult#ReceiveResponse`)

- **IDA:** 0xa223dc
- **Atlas file:** `../../libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 5 = RECEIVE — v8==1 sub-branch)` | ✅ |  |
| 1 | string | string `fromName (character who gave fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |


## Manual analysis

**v83 IDA:** `CWvsContext::OnGivePopularityResult` @ 0xa223dc, v8==1 sub-branch (mode=5, RECEIVE) — DecodeStr(fromName), Decode1(bInc). No Decode4 for nPOP (consistent with v95). Matches v95 exactly.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
