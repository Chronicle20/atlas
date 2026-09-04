# Completeness critic — task-281-map-back-effects

Branch: `task-281-map-back-effects` (`git branch --show-current` confirmed).
Diff base: `bda6566f3...797d1e0cf` (HEAD).
Manifest: `docs/tasks/task-281-map-back-effects/coverage-manifest.yaml` (present, commit `797d1e0cf`).

## Verdict: CLEAN — 0 findings

0 CHANGED-BUT-UNCLAIMED, 0 CLAIMED-BUT-UNVERIFIED. The manifest matches the
branch's actual codec, gate, and matrix deltas cell-for-cell.

## Step 2 — CHANGED-BUT-UNCLAIMED scan

**Touched codecs** (`git diff --name-only bda6566f3...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v _test`):

```
libs/atlas-packet/field/clientbound/clear_back_effect.go
libs/atlas-packet/field/clientbound/set_back_effect.go
```

Both map to dir `field/clientbound` and match manifest `ops` entries
`field/clientbound/SetBackEffect` and `field/clientbound/ClearBackEffect`. No
other file under `libs/atlas-packet` was touched by this branch. CLAIMED.

**Touched version gates**
(`git diff bda6566f3...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'`):

```
+			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
+			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
```

Both hits are in the two codecs' `_test.go` files (test-table version-context
construction), not a `MajorVersion()`/`MajorAtLeast()` gate inside `Decode`/
`Encode`. Confirmed by reading both non-test files in full: neither
`set_back_effect.go` nor `clear_back_effect.go` contains any version
conditional — a single wire shape covers all present versions, matching the
manifest's `fields` note ("No MajorAtLeast gate in the codec... six GMS
versions... derived layout-IDENTICAL"). No gate-class finding.

**Matrix delta** (`git show <rev>:docs/packets/audits/status.json | jq` diff
of every `kind:"op"` row's cell states, base vs HEAD):

| op | base cells | HEAD cells | manifest disposition |
|---|---|---|---|
| `SET_BACK_EFFECT` | gms_v48 n-a, gms_v61 n-a, gms_v72–jms_v185 incomplete | gms_v48 n-a, gms_v61–jms_v185 **verified** | claimed in `ops`, versions list matches exactly |
| `CLEAR_BACK_EFFECT` | gms_v48–v79 n-a, gms_v83–jms_v185 incomplete | gms_v48 n-a, gms_v61–jms_v185 **verified** | claimed in `ops`, versions list matches exactly |
| `SET_MAP_OBJECT_VISIBLE` | gms_v72/v79 incomplete | gms_v72/v79 reclassified to **n-a** (VERSION-ABSENT); v83–jms_v185 unchanged `incomplete` | declared `out_of_scope`; manifest's residual note ("VERSION-ABSENT on gms_v48/gms_v61/gms_v72/gms_v79... present-but-unimplemented on gms_v83/v84/v87/v92/v95/jms_v185") matches this delta exactly |
| `IDA_0X05F`, `IDA_0X060` | present (gms_v61 unnamed-opcode placeholders, opcode 0x5F/0x60) | rows removed | mechanical fname-doc consequence of naming gms_v61 opcodes 95/96 as `SET_BACK_EFFECT`/`CLEAR_BACK_EFFECT` — not an independent coverage claim |

No row outside the manifest's `ops`/`out_of_scope` sets changed state. Ran
`go run ./tools/packet-audit matrix --check` (repo root) — exit 0, only two
unrelated `n-a evidence consumed` notes (`CASHSHOP_CASH_ITEM_GACHAPON_RESULT`,
`USE_TELEPORT_ROCK`), nothing back-effect related. Also ran `gate-check
--check` (exit 0, "21 gate(s) have verified byte-fixtures... 1 partial-by-design",
none touching back-effect) and `dispatcher-lint` (exit 0, "clean").

## Step 3 — CLAIMED-BUT-UNVERIFIED scan

Manifest `ops × versions`:
- `field/clientbound/SetBackEffect`: gms_v61, gms_v72, gms_v79, gms_v83,
  gms_v84, gms_v87, gms_v92, gms_v95, jms_v185 claimed verified; gms_v48
  explicitly declared `(n-a)`, not a verified claim.
- `field/clientbound/ClearBackEffect`: same pattern.

HEAD `status.json` cells for both ops: all nine listed versions = `verified`;
gms_v48 = `n-a` (matches the manifest's own disclosure, not a silent gap).
0 CLAIMED-BUT-UNVERIFIED.

## Corrections carried and reconfirmed against repo files

- **VERSION-ABSENT (⬜) applies to gms_v48 only, not jms_v185.**
  `docs/packets/audits/STATUS.md:187` (SET_BACK_EFFECT) and `:190`
  (CLEAR_BACK_EFFECT) both render `⬜` only in the v48 column; the JMS185
  column pair is `0x07E | ✅` (SET) and `0x080 | ✅` (CLEAR). Cross-checked
  against `status.json`: `jms_v185` state `verified`, opcode 126 (0x7E) for
  SET_BACK_EFFECT and matching CLEAR_BACK_EFFECT opcode — consistent.
  `structures/jms_v185.md` and the manifest itself correctly list jms_v185
  among the nine verified columns, not as n-a. The manifest is correct on
  this point.
- **gms_v48's two n-a cells rest on `feature-na-evidence.yaml`, not `evidence
  pin`.** Confirmed two matching entries at
  `docs/packets/feature-na-evidence.yaml:252` (`op: SET_BACK_EFFECT, version:
  gms_v48`) and `:285` (`op: CLEAR_BACK_EFFECT, version: gms_v48`), each
  citing the exhaustive `CField::OnPacket` router enumeration and the absence
  of `CMapLoadable::OnPacket` on that binary. `tools/packet-audit/cmd/
  na_consistency.go` defines `loadNAEvidence`/`naConsistencyCheck`, which
  `matrix --check` invokes; that check exited 0 with these two cells present
  as n-a, confirming the mechanism still accepts them without an `evidence
  pin` record. Not re-litigated beyond re-running the gate — result matches
  the prior review's conclusion.
- **`SET_MAP_OBJECT_VISIBLE` (case 145) is out of scope and not flagged.**
  Present in the manifest's `out_of_scope` list, and cross-checked against
  `docs/tasks/task-281-map-back-effects/follow-ups.md`'s VERSION-ABSENT /
  incomplete disposition, which matches the observed matrix delta exactly
  (see table above).

## Manifest quality assessment

The manifest was authored after the fact (Task 13 note in its own header),
but it is not loosely back-fitted to just whatever the diff happened to
touch — it is falsifiable and it survives the check:

- Its `versions` block states the *exact* nine-verified/one-n-a disposition
  per op, and both status.json and STATUS.md match it column-for-column,
  including the specific opcodes cited in the accompanying prose (not just
  states).
- Its `out_of_scope` entry doesn't just name the packet — it predicts the
  *exact* state transition (`SET_MAP_OBJECT_VISIBLE` v72/v79 incomplete→n-a,
  v83–jms_v185 staying incomplete), which the matrix diff confirms precisely.
  A rubber-stamped manifest would not risk a falsifiable per-cell claim like
  this.
- Its `residual` section names the specific gate function
  (`tools/packet-audit/cmd/na_consistency.go`) and evidence file backing the
  two n-a cells, which independently re-ran clean.
- No touched file, gate, or matrix row exists outside what the manifest
  declares (`ops` + `out_of_scope`) — there is no leftover diff surface for a
  looser manifest to have needed to hide.

No scope hole found in this branch.
