# FieldContiMove (← `CField_ContiMove::OnContiMove`)

- **IDA:** 0x5374c1
- **Atlas file:** `libs/atlas-packet/field/clientbound/conti_move.go`
- **Variant:** GMS/v79
- **Branch depth:** 1
- **Verdict:** ✅

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `state; (state-7) selects one of 6 arms (nullsub_7/sub_5375AE/nullsub_8/sub_5375D3/nullsub_9/sub_537609)` | ✅ |  |
| 1 | byte | byte `subState (state-gated: only arms 8/10/12 -- sub_5375AE/sub_5375D3/sub_537609 -- read a second byte; arms 7/9/11 are nullsubs and read nothing further)` | ✅ |  |

