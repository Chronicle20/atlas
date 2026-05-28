# FieldClock (← `CField::OnClock`)

- **IDA:** 0x531510
- **Atlas file:** `libs/atlas-packet/field/clientbound/clock.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `clockType (mode discriminator)` | ✅ |  |
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
every arm and reports ❌ above. **This is a TOOL LIMITATION, not a wire bug.** Per
design §8 this file is NOT refactored into per-mode structs.

IDA reference: `CField::OnClock`@0x531510 — `Decode1(clockType)` then a
`switch (clockType)` with arms 0/1/2/3/0x64.

| clockType | atlas constructor | IDA case (addr) | atlas Encode payload (after mode byte) | IDA reads (after mode byte) | verdict |
|---|---|---|---|---|---|
| 0x00 EventClock | `NewEventClock(seconds)` | case 0 @0x53156e | `WriteInt(seconds)` (4) | `Decode4 seconds` → SetEventTimer | ✅ |
| 0x01 TownClock | `NewTownClock(h,m,s)` | case 1 @0x5315ab | `WriteByte(hour)+WriteByte(minute)+WriteByte(second)` (3) | `Decode1 h + Decode1 m + Decode1 s` → CClock::SetClock | ✅ |
| 0x02 TimerClock | `NewTimerClock(seconds)` | case 2 @0x5315d7 | `WriteInt(seconds)` (4) | `Decode4 seconds` (OnMakeTimerParam reads nothing from packet) → SetTimer | ✅ |
| 0x03 EventTimerClock | `NewEventTimerClock(seconds)` (flag1=true) | case 3 @0x5316bb | `WriteBool(flag1)+WriteInt(seconds)` (1+4) | `Decode1 enable + Decode4 seconds` (clock create gated on enable byte) | ✅ |
| 0x64 CakePieEventTimerClock | `NewCakePieEventTimerClock(seconds)` (flag1=flag2=true) | case 0x64 @0x5317c4 | `WriteBool(flag1)+WriteBool(flag2)+WriteInt(seconds)` (1+1+4) | `Decode1 enable + Decode1 v22(board-variant) + Decode4 seconds` | ✅ |

**Per-mode verdict: ⚠️ (tool-limitation, all 5 modes manually verified ✅ against
`CField::OnClock`@0x531510).** No wire bug. The mode-byte values
(0/1/2/3/0x64) and per-mode field widths all match the IDA switch arms exactly.

Note on case 3 vs 0x64 flag bytes: in case 3 the single `Decode1` before
`Decode4(seconds)` is the clock-create enable byte (atlas `flag1`); the client
only builds the clock when it is non-zero. In case 0x64 there are two leading
bytes — an enable byte (atlas `flag1`) gating creation, then `v22` (atlas
`flag2`) selecting the board pixel size (391×83 vs 279×88) before
`Decode4(seconds)`.

Ack: world-audit Phase 2d on 2026-05-28
