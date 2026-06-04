# GiveResponse (← `CWvsContext::OnGivePopularityResult#GiveResponse`)

- **IDA:** 0xb094aa
- **Atlas file:** `libs/atlas-packet/fame/clientbound/response.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode (case 0 = GIVE)` | ✅ |  |
| 1 | string | string `toName (recipient)` | ✅ |  |
| 2 | byte | byte `bInc (1=fame, 0=defame)` | ✅ |  |
| 3 | int16 | int32 `nPOP (new total fame — CUIUserInfo::NotifyGivePopResult uses Decode4)` | ❌ | width mismatch |
| 4 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual analysis

**The auto-generated ❌ at rows 3-4 is a static-tool artifact** — identical to GMS v95 `GiveResponse` (see `gms_v95/GiveResponse.md`). Atlas writes `WriteInt16(total) + WriteShort(0)` = 4 bytes LE, which is wire-identical to `CInPacket::Decode4` reading `int32(total)` for any value in the int16 range.

JMS v185 `OnGivePopularityResult` (@ 0xb094aa, mode 0): reads `Decode4` for nPOP (@ 0xb0973a `v33 = CInPacket::Decode4(v3)`), same as GMS v95.

**JMS vs GMS: gate confirmed ✅.** No region/version gate required; atlas wire (int16+short) produces the same 4 bytes as the client's Decode4.

Ack: misc-audit Phase 3 JMS185 on 2026-06-03
