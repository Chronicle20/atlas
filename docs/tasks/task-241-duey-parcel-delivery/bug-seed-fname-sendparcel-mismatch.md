# bug: seed-template DUEY_ACTION fname disagrees with the regenerator (all 8 versions)

**Status:** resolved. Fixed by commit `fix(packets): correct DUEY_ACTION template
fname to the registry primary`.
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

## Resolution (RULING 25 / RULING 27)

Controller ruling: the op registry is canonical; the seed templates are
derived artifacts. `indexRegistryByOpcode` groups by `(direction, opcode)` and
reads only the registry's **primary** `fname` (never `fname_alts`); DUEY_ACTION
has exactly one registry entry per version, so `pickCandidate` returns "only
candidate" — there is no selection-rule ambiguity to fix. The six
`CParcelDlg::CloseParcelDlg` registry primaries (v83/v84/v87/v92/v95/jms185)
are unchanged from the merge base (`d9ec287b8`); v72's `CTabReceive::
ReceiveParcel` and v79's `CTabSend::SendParcel` were added by this branch with
`provenance: ida-discovered` and real IDA addresses. Re-seeding is correct;
"fixing" the registry primaries or the selection rule would have been wrong.

Attempt 1 ran `go run ./tools/packet-audit seed-fname --write`, which is the
generator's own regeneration path. It correctly derived the 7 target fname
values but also rewrote key ordering (`services`/`options` pairs) across
unrelated handler entries in every file it touched, and touched 3 files with
no fname change at all (`template_gms_48_1.json`, `template_gms_61_1.json`,
`template_gms_79_1.json`). The controller confirmed by running the same
`--write` against the merge-base templates/registries in a temp dir: it
rewrote all 10 base templates too, so the reformat is a pre-existing writer
defect (real, but not this branch's, and not a prerequisite for this fix).
Rejected per RULING 27 — surfaced separately, not fixed here.

Fix applied instead (RULING 27): hand-applied exactly the generator's semantic
output — the single `fname` line at the DUEY_ACTION handler in each of the 7
affected templates — without running the writer, so no reformat noise:

| Template | `CTabSend::SendParcel` → |
|---|---|
| `template_gms_72_1.json` | `CTabReceive::ReceiveParcel` |
| `template_gms_83_1.json` | `CParcelDlg::CloseParcelDlg` |
| `template_gms_84_1.json` | `CParcelDlg::CloseParcelDlg` |
| `template_gms_87_1.json` | `CParcelDlg::CloseParcelDlg` |
| `template_gms_92_1.json` | `CParcelDlg::CloseParcelDlg` |
| `template_gms_95_1.json` | `CParcelDlg::CloseParcelDlg` |
| `template_jms_185_1.json` | `CParcelDlg::CloseParcelDlg` |

`template_gms_79_1.json` was left untouched — its `CTabSend::SendParcel`
already matches its registry primary.

Verified: `TestSeedFName_RealTemplatesInsertionCoverage` passes; `go build
./... && go test ./...` in `tools/packet-audit` passes; the four packet-audit
gates (`matrix --check`, `dispatcher-lint`, `fname-doc --check`, `operations
--check`) all exit 0. See `.superpowers/sdd/plan/ruling25-fix-report.md` for
full command output.

Secondary observation (not fixed here, per RULING 27's brief): the ledger's
claim that `duey_action.yaml`'s call-site roster omits
`CTabQuickSend::SendQuickDelivery` is still accurate — that dispatcher file
has no `fname_alts`/call-site roster field at all (checked
`docs/packets/dispatchers/duey_action.yaml`); the string only appears in
`docs/packets/registry/gms_v83.yaml:2236`.
