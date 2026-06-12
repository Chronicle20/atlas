# Divergent Off-By-One Characterization

**Date:** 2026-06-10. Produced by the new `diff-shape` diagnostic run live against all four IDBs
(ports 13337–13340), reports in `/tmp/ds/<version>.md`.

## Note on counts

`diff-shape` reports a **superset** of `validate`'s divergent bucket: it runs
`ExtractShape`+`ValidateShape` for every entry but does NOT replicate `validate`'s leaf /
multi-way branching, so empty-dispatch multi-way `#Mode` entries (which `validate` marks
`unverifiable`) show here as large-delta "divergent" noise. The off-by-one signal is the small
deltas; the large deltas are filtered out as not-this-lever's-concern.

## The systematic finding — leading `Decode1` omission (109 entries)

The ±1 deltas break down by divergence position:

| position | count | meaning |
|---|---|---|
| **leading** | **109** | the extra read is at the FRONT |
| interior | 59 | extra read sandwiched between matched prefix+suffix |
| trailing | 46 | extra read at the back |

**Every `leading +1` entry has the identical signature:** `prefix 0`, `suffix = full hand
length` — i.e. `hand == live[1:]`, and the extra leading read is a **`Decode1`**. Samples (v83):

```
CCashShop::OnBuyCouple      hand [Decode4,Decode4,Decode4,Str,Str]   live [Decode1, …same…]
CCashShop::OnRebateLockerItem hand [Decode4,DecodeBuf]               live [Decode1,Decode4,DecodeBuf]
CMiniRoomBaseDlg::CheckAndSendChat hand [Str]                        live [Decode1,Str]
CPersonalShopDlg::DeliverBlackList hand [Decode2,Str]                live [Decode1,Decode2,Str]
```

The 109 span the dialog/shop/trade families: `CCashShop`, `CShopDlg`, `CPersonalShopDlg`,
`CTrunkDlg`, `CTradingRoomDlg`, `CMiniRoomBaseDlg`, `CUIMessenger`. The client reads a leading
op/action byte first; the hand baselines were traced from the body and **omitted it**. This is a
genuine systematic representation gap (the hand shapes are each short by one real leading field),
not a tool artifact — `diff-shape` shows the byte in the live (client-decompiled) read sequence.

## Remediation category

Per the design blend, the 109 leading-`Decode1` entries are a **shared-prefix omission**. The
two honest ways to complete them:
- **Baseline `calls` correction:** prepend `{op: Decode1, comment: "leading op/action byte"}` to
  each entry's `calls`. Direct; treats it as the per-leaf omission it is (these are independent
  dialog handlers, not one shared wrapper).
- **Dispatcher annotation:** add a `dispatcher` kind that prepends one `Decode1` and tag the 109.
  Reuses the mechanism, but implies a single shared wrapper that does not really exist as one
  entity across these distinct dialogs.

Recommended: **baseline `calls` correction** — it is the most faithful (each leaf genuinely reads
the byte) and avoids over-abstracting unrelated dialogs into a fake shared dispatcher.

## The interior (59) and trailing (46) ±1

Not part of the systematic leading cluster — these are mixed: width regrouping, trailing optional
fields, and some genuine Atlas-vs-client differences. Per option 3 they stay **honest divergent**
unless a clear sub-cluster emerges; any that are genuine Atlas-vs-client field differences go to
`divergent-findings.md` as encoder work, not auto-fixed.

## Scale note

109 baseline edits is a substantial batch mutation on committed data. The mechanism and scope
were checkpointed with the user before applying (see remediation task).
