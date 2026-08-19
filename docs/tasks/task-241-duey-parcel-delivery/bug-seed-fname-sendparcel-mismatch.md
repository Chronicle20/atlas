# bug: seed-template DUEY_ACTION fname disagrees with the regenerator (all 8 versions)

**Status:** open, blocking for branch end. Not yet fixed.
**Found by:** controller, checking a claim in the v92 fix report before handoff.

## The failure

`tools/packet-audit/cmd` → `TestSeedFName_RealTemplatesInsertionCoverage` FAILS
on the current branch head (`a8adafb12`), on **7 of 8 versions**:

```
gms_72_1: socket.handlers[44] fname mismatch:
    committed="CTabSend::SendParcel" regenerated-from-virgin="CTabReceive::ReceiveParcel"
gms_83_1: socket.handlers[50]  committed="CTabSend::SendParcel" regenerated="CParcelDlg::CloseParcelDlg"
gms_84_1: socket.handlers[50]  (same)
gms_87_1: socket.handlers[47]  (same)
gms_92_1: socket.handlers[36]  (same)
gms_95_1: socket.handlers[48]  (same)
jms_185_1: socket.handlers[34] (same)
```

(`gms_79_1` resolves 351/351 and is the only clean one.)

## It is NOT pre-existing — this branch introduced it

The v92 fix report called this "one pre-existing, unrelated failure ... confirmed
pre-existing by stash/re-run on the unmodified tree." That baseline is wrong: it
compared against *current HEAD minus that agent's own change*, not against the
branch's merge base.

Checked properly:

```
git log --oneline d9ec287b8..HEAD -S'CTabSend::SendParcel'
    → 6bb465f86 feat(packet): DUEY_ACTION serverbound codecs
git grep -c 'CTabSend::SendParcel' d9ec287b8
    → (no output — the string does not exist at the merge base)
```

`CTabSend::SendParcel` enters the tree for the first time in this branch's own
commit `6bb465f86`. The failure is branch-introduced and parcel-owned — the exact
opposite of unrelated.

## Why the per-batch gates stayed green

Every `--quick --base` gate in this campaign passed at exit 0 while this test
fails. Whatever the reason (module selection, or `--quick` skipping the
`tools/` module), **the iteration gate has not been covering this test**, so it
will surface at the flagless `tools/verify.sh` branch-end run or in CI. Confirm
the selection with `tools/verify.sh --facts` rather than reasoning from the
script's source.

## What to work out before fixing

DUEY_ACTION legitimately has several call sites (`CTabSend::SendParcel`,
`CTabReceive::ReceiveParcel`, `CTabReceive::DiscardParcel`,
`CParcelDlg::CloseParcelDlg`, `CTabQuickSend::SendQuickDelivery`,
`CUIFadeYesNo::OnButtonClicked`). The seed template records ONE fname per
handler, and the regenerator picks a different member of that roster than the
committed templates do. So the question is which side is wrong:

- If the regenerator's choice is canonical → the committed templates need
  re-seeding for all 7 versions.
- If `CTabSend::SendParcel` is the intended representative → the regenerator's
  selection rule (or the roster order it reads) needs fixing.

Do not simply re-seed to silence the test until that is decided — picking the
side that makes the test green is how a wrong fname gets frozen into all eight
templates. Note also that v72's regenerated value (`CTabReceive::ReceiveParcel`)
differs from the other six (`CParcelDlg::CloseParcelDlg`), which is itself a clue
about the selection rule.

Related known defect (already in the ledger): `duey_action.yaml`'s call-site
roster omits `CTabQuickSend::SendQuickDelivery` on every version — six batches
have each re-derived it by hand. The roster and this selection rule likely want
fixing together.
