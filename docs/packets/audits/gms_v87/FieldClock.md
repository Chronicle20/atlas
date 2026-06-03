# FieldClock (← `CField::OnClock`)

- **IDA:** 0x55DA5F
- **Atlas file:** `libs/atlas-packet/field/clientbound/clock.go`
- **Variant:** GMS/v87
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `clockType (mode discriminator, v4 @0x55da76)` | ✅ |  |
| 1 | int32 | byte `flag1 (EventTimerClock enable byte; representative arm = case 3 @0x55dac2)` | ❌ | width mismatch |
| 2 | byte | int32 `seconds (@0x55dae6)` | ❌ | width mismatch |
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
every arm and reports ❌ above. **This is a TOOL LIMITATION, not a wire bug.** Per
design §8 this file is NOT refactored into per-mode structs.

v87 has NO standalone `CField::OnClock` symbol — the clock-set handler is inlined
as `sub_55DA5F`@0x55DA5F (identified via its three `CClock::SetTimer` callers).
It reads `Decode1(clockType)` (@0x55da76) then dispatches a 5-arm
`switch (clockType)`.

| clockType | atlas constructor | IDA case (addr) | atlas Encode payload (after mode byte) | IDA reads (after mode byte) | verdict |
|---|---|---|---|---|---|
| 0x00 EventClock | `NewEventClock(seconds)` | case 0 @0x55dd6d | `WriteInt(seconds)` (4) | `Decode4 seconds` | ✅ |
| 0x01 TownClock | `NewTownClock(h,m,s)` | case 1 @0x55dd40 | `WriteByte(hour)+WriteByte(minute)+WriteByte(second)` (3) | `Decode1 h + Decode1 m + Decode1 s` | ✅ |
| 0x02 TimerClock | `NewTimerClock(seconds)` | case 2 @0x55dd0f | `WriteInt(seconds)` (4) | `Decode4 seconds` | ✅ |
| 0x03 EventTimerClock | `NewEventTimerClock(seconds)` (flag1=true) | case 3 @0x55dac2 | `WriteBool(flag1)+WriteInt(seconds)` (1+4) | `Decode1 enable + Decode4 seconds` | ✅ |
| 0x64 CakePieEventTimerClock | `NewCakePieEventTimerClock(seconds)` (flag1=flag2=true) | case 0x64 @0x55dadd | `WriteBool(flag1)+WriteBool(flag2)+WriteInt(seconds)` (1+1+4) | `Decode1 enable + Decode1 flag2 + Decode4 seconds` | ✅ |

**Per-mode verdict: ⚠️ (tool-limitation, all 5 modes manually verified ✅ against
`sub_55DA5F`@0x55DA5F).** No wire bug. **v87 byte-identical to v95** — the mode
byte values (0/1/2/3/0x64) and per-mode field widths all match the IDA switch
arms exactly.

Ack: world-audit Phase 3 v87 cross-version on 2026-05-28
