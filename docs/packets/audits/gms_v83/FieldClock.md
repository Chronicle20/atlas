# FieldClock (← `CField::OnClock`)

- **IDA:** 0x5361bd
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/clock.go`
- **Variant:** GMS/v83
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `clockType (mode discriminator; switch arms 0/1/2/3)` | ✅ |  |
| 1 | int32 | byte `flag1 (EventTimerClock enable byte; representative arm = case 3)` | ❌ | width mismatch |
| 2 | byte | int32 `seconds` | ❌ | width mismatch |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual per-mode verdict (tool limitation)

`Clock` is a BAD-FORM single struct whose `clockType` mode byte is set at
construction and whose `Encode` switch emits a *different* payload per mode. The
flat analyzer cannot model mutually-exclusive switch arms, so it concatenates
every arm and reports ❌ above. **This is a TOOL LIMITATION, not a wire bug.**

IDA reference: `CField::OnClock` v83 @0x5361bd — `Decode1(clockType)` then a
`switch (clockType)` with arms 0/1/2/3 (no 0x64 CakePie arm in v83; that mode
was added in a later client).

| clockType | atlas constructor | v83 IDA case | atlas Encode payload (after mode byte) | v83 IDA reads (after mode byte) | verdict |
|---|---|---|---|---|---|
| 0x00 EventClock | `NewEventClock(seconds)` | case 0 (line 116) | `WriteInt(seconds)` (4) | `Decode4 seconds` → SetEventTimer | ✅ |
| 0x01 TownClock | `NewTownClock(h,m,s)` | case 1 (lines 107-109) | 3×`WriteByte` (3) | `Decode1 h + Decode1 m + Decode1 s` → CClock::SetClock | ✅ |
| 0x02 TimerClock | `NewTimerClock(seconds)` | case 2 (lines 80-99, default-timer arm) | `WriteInt(seconds)` (4) | `Decode4 seconds` → SetTimer | ✅ |
| 0x03 EventTimerClock | `NewEventTimerClock(seconds)` | case 3 (lines 50/67) | `WriteBool(flag1)+WriteInt(seconds)` (1+4) | `Decode1 enable + Decode4 seconds` (clock create gated on enable byte) | ✅ |

**Per-mode verdict: ⚠️ (tool-limitation, all v83 modes manually verified ✅
against `CField::OnClock` v83 @0x5361bd).** Identical per-mode shapes to GMS v95
(0/1/2/3). The only cross-version note: v83 has no 0x64 CakePieEventTimerClock
arm, so a v83 client would ignore that atlas mode — but atlas only emits 0x64
when the server constructs it, and the v83 server domain does not. No wire bug.

Ack: world-audit Phase 3 v83 on 2026-05-28
