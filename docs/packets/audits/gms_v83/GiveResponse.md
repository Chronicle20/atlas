# GiveResponse (← `CWvsContext::OnGivePopularityResult#GiveResponse`)

- **IDA:** 0xa223dc
- **Atlas file:** `../../libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (switch dispatch; case 0 = GIVE — else-branch in v83 subtraction chain)` | ✅ |  |
| 1 | string | string `fromName / toName (recipient of fame)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |
| 3 | int16 | int32 `nPOP (new total fame as int32; passed to CUIUserInfo::NotifyGivePopResult @0xa22663)` | ❌ | width mismatch |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual analysis

**v83 IDA:** `CWvsContext::OnGivePopularityResult` @ 0xa223dc, else-branch (mode=0, GIVE) — DecodeStr(name), Decode1(bInc), Decode4(nPOP). Matches v95 exactly (including Decode4 for nPOP — fame total is int32 in v83 too).

**Static-diff ❌ is a known false positive** — same artifact as v95: atlas writes `WriteInt16(total)+WriteShort(0)` which produces the same 4 bytes as the client's `Decode4`. No real wire discrepancy.

**Gate:** None needed — version-agnostic. Gate confirmed correct (✅).


Ack: misc-audit Phase 3 v83 on 2026-06-03
