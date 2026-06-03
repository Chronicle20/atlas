# FieldChange (← `CField::SendTransferFieldRequest`)

- **IDA:** 0x56d75a
- **Atlas file:** `../../libs/atlas-packet/field/serverbound/change.go`
- **Variant:** JMS/v185
- **Branch depth:** 2
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `fieldKey (get_field()->m_bFieldKey @line33)` | ✅ |  |
| 1 | int32 | int32 `dwTargetField / targetId (@line34)` | ✅ |  |
| 2 | string | string `sPortal / portalName (@line40; empty ZXString when NULL)` | ✅ |  |
| 3 | int16 | int16 `x — only when sPortal!=NULL (@line44)` | ✅ |  |
| 4 | int16 | int16 `y — only when sPortal!=NULL (@line46)` | ✅ |  |
| 5 | byte | byte `unused (constant 0 @line48)` | ✅ |  |
| 6 | byte | byte `bPremium / premium (@line49)` | ✅ |  |
| 7 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |


## Manual verdict (JMS v185, `CField::SendTransferFieldRequest` @0x56d75a)

Rows 7-8 (❌ "atlas: extra") are an analyzer-flattening artifact, NOT a wire bug. Atlas
`change.go` writes `targetX/targetY` only inside `if m.chase { ... }`; the analyzer
collects that conditional tail unconditionally. For JMS, `m.chase` is never set (the chase
byte itself is gated `GMS && >=83`, skipped for JMS), so at runtime atlas emits exactly
`fieldKey + targetId + portal + [x,y if portal] + unused + premium` — matching JMS185
SendTransferFieldRequest field-for-field (rows 0-6 ✅). JMS185 has NO chase byte and NO
targetX/targetY pair (Encode1 unused @line48, Encode1 bPremium @line49 are the tail).

Ack: world-audit Phase 3 JMS185 field+portal on 2026-05-28
