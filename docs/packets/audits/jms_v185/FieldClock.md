# FieldClock (← `CField::OnClock`)

- **IDA:** 0x56e849
- **Atlas file:** `../../libs/atlas-packet/field/clientbound/clock.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `clockType (mode discriminator; CField::OnClock virtual @vtable+0x2C)` | ✅ |  |
| 1 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual verdict (JMS v185, CLOCK op 0x090/144 → `CField::OnClock` virtual)

The flat ❌ is the documented BAD-FORM mode-switch tool limitation (same as GMS v95):
`Clock.Encode` emits a different payload per `clockType` arm, and the analyzer
concatenates all mutually-exclusive arms. **NOT a wire bug.** JMS185 dispatches CLOCK via
`CField::OnPacket` case 0x90 → a virtual call `[vtable+0x2C]` (`CField::OnClock` is
inlined/unnamed in the JMS185 IDB — no standalone symbol; `OnDestroyClock`@0x56ec69
confirms the clock family lives on CField). The clockType-discriminated per-mode decode
matches the standard MapleStory clock layout, but the full per-mode shape is not
resolvable from this IDB without the concrete vtable target. ⚠️ **manual-verify carried
forward** (same disposition as the GMS v95 FieldClock report).

Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
