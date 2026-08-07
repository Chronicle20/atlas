# Completeness critic — task-165-mist-writer-template-wiring

**Verdict: CLEAN — 0 findings.**

## Manifest note

This task folder has no `coverage-manifest.yaml` (the plan predates that
convention). The claim set was derived instead from `plan.md`'s "Reference
facts" opcode tables (lines ~40–95) and the "Task → requirement traceability"
table (lines 844–858), corroborated by `discovery.md`'s Task 8 outcome
records:

- **Claimed ops:** `SPAWN_MIST` → `field/clientbound/FieldAffectedAreaCreated`,
  `REMOVE_MIST` → `field/clientbound/FieldAffectedAreaRemoved`.
- **Claimed matrix versions (9):** gms_v48, gms_v61, gms_v72, gms_v79,
  gms_v83, gms_v84, gms_v87, gms_v95, jms_v185 — all expected to end
  `verified`.
- **Claimed template-only versions (2, no matrix column by design):**
  gms_92, gms_12 — provenance recorded in `discovery.md` only, per Task 8
  Step 4's explicit statement that "no registry exists and standing one up is
  out of scope."
- **Claimed codec:** only `affected_area_created.go` was expected to gain
  version gates (Task 8 Step 3, Global Constraints: "Tier B may add an
  *additive* version gate only ... using the `MajorAtLeast` idiom").
  `affected_area_removed.go` was expected to stay byte-identical across all
  eleven versions (both Tier B outcomes for REMOVE_MIST record "Matches the
  current codec exactly").

A `coverage-manifest.yaml` should still be added retroactively so future
runs of this critic don't have to re-derive the claim set from prose; that is
a process-hygiene recommendation, not a finding against this branch.

## Step 2 — CHANGED-BUT-UNCLAIMED

**Touched codecs.**
```
$ git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v '_test\.go$'
libs/atlas-packet/field/clientbound/affected_area_created.go
```
Exactly one non-test codec file changed, and it is the declared
`field/clientbound/FieldAffectedAreaCreated` codec. `affected_area_removed.go`
was not modified, consistent with both Tier B outcome records ("Matches the
current codec exactly" for Removed on all three versions). No unclaimed
codec touched.

**Touched version gates.**
```
$ git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'
-	v95Plus := t.Region() == "GMS" && t.MajorVersion() >= 95
+	twoTimeWords := t.IsRegion("GMS") && t.MajorAtLeast(92)
+	wideNType := !t.IsRegion("GMS") || t.MajorAtLeast(61)
+	hasOwnerId := !t.IsRegion("GMS") || t.MajorAtLeast(48)
+	if t.IsRegion("GMS") && !t.MajorAtLeast(48) {
+	} else if t.IsRegion("GMS") && !t.MajorAtLeast(61) {
```
All four gates live in `affected_area_created.go` (the sole declared codec)
and all use the `MajorAtLeast` idiom, per the Global Constraints ban on raw
`> N` comparisons. Each gate's threshold is traceable to a claimed version's
disassembly evidence in `discovery.md`:

| gate | threshold | evidence |
|---|---|---|
| `twoTimeWords` (renamed from `v95Plus`) | `MajorAtLeast(92)` | discovery.md gms_92 section: "Gate lowered to `IsRegion(\"GMS\") \&\& MajorAtLeast(92)`" |
| `wideNType` | `!GMS \|\| MajorAtLeast(61)` | discovery.md gms_48/gms_12: "nType widened from Decode1 to Decode4 between v48 and v61" |
| `hasOwnerId` | `!GMS \|\| MajorAtLeast(48)` | discovery.md gms_12: "dwOwnerId does not exist before v48" |
| `trailing` width switch | `MajorAtLeast(48)` / `MajorAtLeast(61)` bands | discovery.md gms_48 ("trailing +0x30 slot ... Decode1") / gms_12 ("no trailing time word") |

Because every gate is `IsRegion("GMS") && MajorAtLeast(N)` (or its negation),
the only GMS majors affected are 12, 48, 61, 72, 79, 83, 84, 87, 92, 95 — i.e.
exactly the eleven claimed versions (9 matrix + 2 template-only). No
non-region-GMS version (e.g. jms_v185, which the gate deliberately excludes
via `!t.IsRegion("GMS")` short-circuiting) and no unclaimed GMS version is
touched by any threshold. This is the highest-risk area named in the
dispatch brief and it checks out clean.

**Matrix delta.**
```
$ git diff $BASE...HEAD -- docs/packets/audits/status.json | grep -v toolSha
```
Two rows moved, both fully consistent with the claim:

- `SPAWN_MIST` / `field/clientbound/FieldAffectedAreaCreated`: `gms_v48`
  `n-a` → `verified` (opcode 202); `gms_v83`/`gms_v84`/`gms_v87`/`gms_v95`/
  `jms_v185` `incomplete` ("tier-1 without fixture") → `verified`; `gms_v61`/
  `gms_v72`/`gms_v79` were already `verified` and stay `verified`. Matches
  plan.md's "FR-3 SPAWN_MIST ❌→✅ ×5" line exactly (the 5 incomplete→verified
  transitions) plus the Task 8 gms_48 promotion.
- `REMOVE_MIST` / `field/clientbound/FieldAffectedAreaRemoved`: same
  transition pattern (n-a→verified for gms_v48, plus consistency with "FR-3
  REMOVE_MIST 🟡ᶠ→✅ ×3").

Both rows are in `claimedPackets`; no other row in the diff changed state.
No unclaimed matrix movement.

**Templates.** For completeness (not requested as a separate check but
corroborating the gate finding above): all eleven declared templates
(`template_gms_{12,48,61,72,79,83,84,87,92,95}_1.json`,
`template_jms_185_1.json`) were touched, each with a 10-line insertion (two
opcode entries), and no other template file changed.

## Step 3 — CLAIMED-BUT-UNVERIFIED

All 18 claimed op×version cells (`SPAWN_MIST` and `REMOVE_MIST` ×
{gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95,
jms_v185}) read `"state": "verified"` in the HEAD `status.json`. None found
`partial` or `incomplete`.

gms_92 and gms_12 have no matrix column (by design — they are template-only
seeds absent from `tools/packet-audit/internal/matrix/model.go`'s version
set), so they cannot be claimed against the matrix; their provenance is
correctly confined to `discovery.md` rather than asserted as a matrix
`verified` state anywhere. No mismatch between manifest-equivalent claim and
matrix `n-a` handling.

**No CLAIMED-BUT-UNVERIFIED findings.**

---

**Post-review correction (added by the controller after these audits ran).**
These audits were produced against the tree at commit `a97540c8b`, before the mist
consumer fix landed. Any statement in this document that the `phase = 0` defect in
`services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go` is a
pre-existing, out-of-scope follow-up is now **obsolete**: it was fixed in commit
`9b471311f` at the user's explicit direction, overriding the plan's original
"no consumer/producer changes" constraint. `phase` now carries `Duration / 100`
(clamped to int16 max, floored at 1) via the `mistPhase` helper. No wire byte moved.
