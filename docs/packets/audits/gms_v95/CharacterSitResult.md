# CharacterSitResult (← `CUserLocal::OnSitResult`)

- **IDA:** 0x905e70
- **Atlas file:** `libs/atlas-packet/character/clientbound/sit_result.go`
- **Variant:** GMS/v95
- **Branch depth:** 1
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `sitting flag (0=cancel sit / stand up, 1=sit in chair)` | ✅ |  |
| 1 | int16 | int16 `chairId / nSeat (only if sitting flag == 1)` | ✅ |  |
| 2 | byte | byte `` | ❌ | atlas: extra — client never reads this field |

ack: tool-limitation false positive — branch-flattening unions the if-branch writes (byte+short) and
else-branch writes (byte) sequentially instead of treating them as alternatives. At runtime only one
branch fires. IDA CUserLocal::OnSitResult reads Decode1 (flag) then conditionally Decode2 (chairId),
which exactly matches the atlas encoder's two runtime paths. No wire bug present. See _pending.md
"Known false positives — character misc-state bucket (Task 10)".
