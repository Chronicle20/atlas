# BuffGiveForeign (← `CUserRemote::OnSetTemporaryStat`)

- **IDA:** 0xb13200
- **Atlas file:** `libs/atlas-packet/character/clientbound/buff_give.go`
- **Variant:** GMS/v95
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

---

ack: stub + dispatcher-layer gap — CUserRemote::OnSetTemporaryStat@0xb13200 is a stub that only clears a window list; real foreign stat decoding happens in CWvsContext path. IDA export has 0 direct calls so every atlas field appears as extra. Atlas BuffGiveForeign.Encode round-trips cleanly; no wire bug detected.
