# completeness-critic — task-252-jukebox-cash-item

Diff base: `git merge-base origin/main HEAD` = `d17404dbc23588202d2dae89173832f5cab96984`
Branch head: `a3f2519e9` (task-252-jukebox-cash-item)
Manifest: `docs/tasks/task-252-jukebox-cash-item/coverage-manifest.yaml` (present)

## Verdict

**CLEAN — 0 findings.**

The manifest declares exactly one op (`cash/serverbound/CashItemUseSongPlayer`),
one out-of-scope codec (`field/clientbound/PlayJukebox`), four out-of-scope
sibling packets, and one scoped `legacyConsumedSiblingWriters` entry. The
diff matches the declared scope exactly: no unclaimed codec touched, no
unclaimed version gate touched, no unclaimed matrix cell changed, and every
claimed op × version is `verified` (or `n-a` with a matching manifest note)
in the final `status.json`/`STATUS.md`.

## Step 1 — resolved scope

- `claimedPackets`: `cash/serverbound/CashItemUseSongPlayer` (dir
  `cash/serverbound`)
- `claimedOps × versions`: `CashItemUseSongPlayer` × {gms_v48 (n-a),
  gms_v61 (n-a), gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92,
  gms_v95, jms_v185 (all claimed verified)}
- `outOfScope`: `field/clientbound/PlayJukebox`,
  `cash/serverbound/CashItemUseSuperMegaphone`,
  `cash/serverbound/CashItemUseMapleTV`, `cash/serverbound/CashItemUseMegaphone`,
  `cash/serverbound/CashItemUseTripleMegaphone`

## Step 2 — CHANGED-BUT-UNCLAIMED

**Touched codecs.** `git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v '_test\.go$'`:
```
libs/atlas-packet/cash/serverbound/item_use_song_player.go
```
This is the sole non-test codec file touched, and its dir (`cash/serverbound`)
matches the claimed packet. No other `libs/atlas-packet` file — including
`libs/atlas-packet/field/clientbound/play_jukebox.go` — was touched (`git diff
$BASE...HEAD -- libs/atlas-packet/field/clientbound/play_jukebox.go` is empty),
consistent with its `out_of_scope` declaration.

| kind | file/packet | evidence | recommendation |
|---|---|---|---|
| (none) | — | — | — |

**Touched version gates.**
`git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'`
returns only two `+` lines, both `ctx := pt.CreateContext(v.Region,
v.MajorVersion, v.MinorVersion)` inside
`libs/atlas-packet/cash/serverbound/item_use_song_player.go` and
`item_use_song_player_test.go` (test-harness context construction, not a
gate branch). Both files belong to the claimed packet. No gate change in any
unclaimed file.

| kind | file/packet | evidence | recommendation |
|---|---|---|---|
| (none) | — | — | — |

**Matrix delta.** `git diff $BASE...HEAD -- docs/packets/audits/status.json`
shows exactly one new `packet` block added
(`cash/serverbound/CashItemUseSongPlayer`, all ten version cells: `n-a` for
gms_v48/gms_v61, `verified` for the remaining eight) plus the `toolSha`
bump. `git diff ... | grep '"packet"'` confirms only that one packet name
appears in the diff (the `CashItemUseSuperMegaphone` line printed by grep is
unchanged context from the following unmodified block, not a diff hunk).
No existing row's cell `state` was modified — confirmed by `git diff
$BASE...HEAD -- docs/packets/audits/status.json | grep -E '^-'` returning
only the `toolSha` line, i.e. zero `state` removals anywhere in the file.
This matches the manifest's explicit claim that the four sibling writers
sharing the fname collision are deliberately not force-promoted.

| kind | packet | evidence | recommendation |
|---|---|---|---|
| (none) | — | — | — |

**Supporting tool changes** (`tools/packet-audit/cmd/run.go`,
`tools/packet-audit/internal/matrix/build.go`) are infrastructure the
manifest explicitly documents as Task 9 work: the `candidatesFromFName`
linkage addition and the single `legacyConsumedSiblingWriters[USE_CASH_ITEM]
-> {"CashItemUseSongPlayer": true}` entry, verified by reading both diffs —
they add exactly the one candidate and exactly the one allow-list key the
manifest describes, nothing broader.

## Step 3 — CLAIMED-BUT-UNVERIFIED

Final (HEAD) `docs/packets/audits/status.json` cells for
`cash/serverbound/CashItemUseSongPlayer`:

| op | version | actual state | recommendation |
|---|---|---|---|
| CashItemUseSongPlayer | gms_v48 | n-a (manifest claims n-a, with `_unimplemented.json` proof matching manifest's summary verbatim) | none — matches |
| CashItemUseSongPlayer | gms_v61 | n-a (manifest claims n-a, with `_unimplemented.json` proof matching manifest's summary verbatim) | none — matches |
| CashItemUseSongPlayer | gms_v72 | verified | none |
| CashItemUseSongPlayer | gms_v79 | verified | none |
| CashItemUseSongPlayer | gms_v83 | verified | none |
| CashItemUseSongPlayer | gms_v84 | verified | none |
| CashItemUseSongPlayer | gms_v87 | verified | none |
| CashItemUseSongPlayer | gms_v92 | verified | none |
| CashItemUseSongPlayer | gms_v95 | verified | none |
| CashItemUseSongPlayer | jms_v185 | verified | none |

No claimed cell is `partial`/`incomplete`. `STATUS.md`'s new row
(`| cash/serverbound/CashItemUseSongPlayer (T1) | | | ⬜ | | ⬜ | | ✅ | | ✅
| | ✅ | | ✅ | | ✅ | | ✅ | | ✅ | | ✅ |`) is consistent with the same
n-a/verified pattern.

**CLAIMED-BUT-UNVERIFIED: none.**
