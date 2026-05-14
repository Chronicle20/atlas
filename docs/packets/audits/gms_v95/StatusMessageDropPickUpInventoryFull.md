# StatusMessageDropPickUpInventoryFull (← `CWvsContext::OnMessage`)

- **IDA:** 0xa06c90
- **Atlas file:** `libs/atlas-packet/character/clientbound/status_message.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `mode/sub-op byte (0=drop pick-up, 1=quest record, 2=cash item expire, 3=inc EXP, 4=inc SP, 5=inc fame/POP, 6=inc meso, 7=inc GP, 8=give buff, 9=general item expire, 10=system message, 11=quest record ex, 12=item protect expire, 13=item expire replace, 14=skill expire)` | ✅ |  |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

## ack

❌ verdict is a **tool-limitation false positive (sub-op enum drift)** — not a real wire bug.

`CWvsContext::OnMessage` dispatches on a leading mode byte (Decode1) to 15 sub-handlers
(case 0–14). The IDA export models only the outermost Decode1 (mode byte). Each atlas struct
for this opcode correctly writes the mode byte first, then sub-op-specific fields. The tool
sees the second atlas write as "extra" because the IDA export only has 1 call entry.

The representative struct chosen for this report is `StatusMessageDropPickUpInventoryFull`
(mode=0, sub-op=-1). The second byte (−1 as int8) is the pick-up status code read by
`CWvsContext::OnDropPickUpMessage` after the outer mode dispatch. This byte IS read by the
client sub-handler — it is NOT a spurious write.

Sub-op enum drift for all 15 modes is deferred to `_pending.md`
`## Sub-op enum drift — character domain` heading.

Causes: (1) loop/sub-dispatch not modelable in flat IDA export; (2) sub-op body bytes attributed
as "extra" by the flat diff.
