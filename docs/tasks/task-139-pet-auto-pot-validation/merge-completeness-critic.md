# Merge completeness critic — task-139-pet-auto-pot-validation

Branch: `task-139-pet-auto-pot-validation`, HEAD `1bc731215ead061f1c7bcb91ab6323b6b7cc66a6`.
Merge commit: `55cf0a30f` (main → task-139). Post-merge fixes: `ac832acaf` (matrix regen),
`f7fc16631` (gms_92 skillGate), `1bc731215` (gms_48/gms_92 petSkill table).
Parents compared: branch-tip-pre-merge `c97ccf6f2`, main-tip `8a9f70301`.

**Verdict: 2 findings — both CHANGED-BUT-UNCLAIMED (scope-declaration gaps), no
functional regression and no CLAIMED-BUT-UNVERIFIED cell.** The regenerated
matrix is mechanically clean and correctly unions both parents' work. The
template routing and petSkill wire-table sweeps requested in items 2–3 are
each 11/11 correct except for the one already-declared, pre-existing gms_12
exclusion. The `autoSpeaking` documentation (item 4) is adequate in the commit
message but not propagated to the manifest.

---

## 1. Matrix regeneration — self-consistent, correctly unions both parents

- `go run ./tools/packet-audit matrix --check` → exit 0.
- `go run ./tools/packet-audit matrix` (full regen, in place) produced **zero
  diff** against the committed `status.json`/`STATUS.md` — confirms the
  committed `toolSha` (`fdc0c767fb39d377e985f0e4c7936c22bb113e2503b24cf49dc9146bdb6c6ae1`,
  `docs/packets/audits/status.json:2`) is the live HEAD-derived value, not
  stale.
- Full-matrix state diff `8a9f70301...HEAD` (main-tip → HEAD) touches **only**
  the two declared rows:
  - `PET_AUTO_POT`/`pet/serverbound/PetItemUse`: `gms_v48 n-a→verified`,
    `gms_v61/v72/v79 partial→verified` (branch's own legacy fixture work,
    carried through the merge).
  - `cash/serverbound/CashItemUsePetSkill` (sub-struct row): new row, all 8
    GMS cells `n-a`, `jms_v185 verified`, `gms_v92 incomplete` (see §1a).
  No other row's state changed relative to main-tip — the merge introduced no
  collateral matrix drift.
- Full-matrix diff `c97ccf6f2...HEAD` (branch-tip → HEAD) shows many rows
  that look like disappearances/renames (`SKILL_LEARN_ITEM_RESULT`,
  `CASHSHOP_OPERATION` split, a `character/serverbound/Move` sub-struct cell
  `gms_v95 verified→incomplete`, etc.). Traced each one: HEAD's state for
  every one of these rows is **byte-identical to main-tip's own state**
  (verified directly, e.g. the `Move` sub-struct row: `main-tip == HEAD`
  exactly, including the `gms_v95 incomplete` value). These are pre-existing
  main-branch changes absorbed by the merge, not regressions caused by this
  task. Confirmed not a task-139 regression.

### 1a. `gms_v92` cells are new, not regressed
`CashItemUsePetSkill` didn't exist as a row on main (it's task-139's own
sub-struct), so its `gms_v92: incomplete` cell has no prior state to regress
from — it's the default "newly-unioned column, unaudited for this row" state,
consistent with the manifest's own note (`coverage-manifest.yaml:89`) that the
n-a sweep covered only the 8 pre-v92 GMS majors. Not a defect by itself, but
see §2 below — the coverage-manifest never declares `gms_v92` as in scope at
all, despite the branch's own post-merge commits touching v92 config for this
exact feature.

---

## 2. CHANGED-BUT-UNCLAIMED — `gms_v92` template fixes outside the declared version set

`coverage-manifest.yaml:74-83` declares exactly 9 versions (`gms_v48/61/72/
79/83/84/87/95`, `jms_v185`) — **`gms_v92` is not among them.**

Yet the branch's own post-merge follow-up commits directly modify
`template_gms_92_1.json` to fix a real functional bug in this feature:

- `f7fc16631` (`services/atlas-configurations/seed-data/templates/template_gms_92_1.json`,
  hunk at the `CWvsContext::SendStatChangeItemUseRequestByPetQ` handler entry):
  adds `"options": {"skillGate": "equipAbility"}`. Without it,
  `pet_item_use.go:67-70` fails closed with `skill_gate_unconfigured` — every
  v92 auto-pot request would have been silently rejected post-merge.
- `1bc731215` (same file, two hunks at the `CWvsContext::OnInventoryOperation`
  and `CStage::OnSetField` writer entries): adds
  `"options": {"petSkill": {"autoSpeaking": "0x100"}}`. Without it,
  `resolvePetSkillWireMask` (`libs/atlas-packet/model/asset.go:368` call site)
  encodes the flag as 0 for every v92 client.

Both commit messages state explicitly *why* this happened: task-139's design
(`design.md:17`, `design.md:305`) recorded gms_92 as **out of scope by
evidence** — "partial bring-up templates carrying no pet handlers at all."
Main's own independent work (merged in via `55cf0a30f`) added a
`PetItemUseHandle` entry to `template_gms_92_1.json`, invalidating that
premise. The two follow-up commits are a correct, well-evidenced reaction to
that invalidation — but three scope artifacts were never updated to match:

| artifact | current text | status |
|---|---|---|
| `coverage-manifest.yaml:74-83` (`versions:`) | 9 versions, no `gms_v92` | stale — branch demonstrably touched v92 config |
| `design.md:17`, `design.md:305` | "Out of scope by evidence: the gms_12 / gms_92 templates... no pet handlers at all" | **wrong** for gms_92 as of the merge; still true for gms_12 |
| `plan.md:24`, `plan.md:2331` | same gms_12/gms_92 out-of-scope claim | same staleness |

Matrix evidence for the resulting v92 state: `PET_AUTO_POT` × `gms_v92` =
`partial` (not `verified`) and `cash/serverbound/CashItemUsePetSkill` ×
`gms_v92` = `incomplete` (not `n-a`) — i.e., the feature is live-wired on v92
(both fixes landed) but the version was never taken through
`/verify-packet`, and the manifest gives no signal that it should be.

**Recommendation:** either (a) add `gms_v92` to `coverage-manifest.yaml`'s
`versions:` list and drive `PET_AUTO_POT`/`gms_v92` to `verified` via
`/verify-packet`, correcting `design.md`/`plan.md`'s stale exclusion note; or
(b) if v92 is deliberately being left partial for this task, add it to
`out_of_scope:` with a note explaining the fixes were a minimal
merge-conflict repair (not full v92 coverage) and that a follow-up task owns
promoting it to `verified`. Leaving it silently absent from `versions:` while
the branch's own commits functionally changed v92 behavior is the exact
scope hole this critic exists to catch.

No other CHANGED-BUT-UNCLAIMED codec or gate findings: the full task-139 delta
over main (`git diff --name-only 8a9f70301...HEAD`) touches only
`libs/atlas-packet/model/asset.go`, `resolve.go`, `pet/serverbound/item_use.go`,
`cash/serverbound/item_use_pet_skill.go` — all declared by
`coverage-manifest.yaml`'s `ops:` list — and the version-gate diff
(`git diff 8a9f70301...HEAD -- libs/atlas-packet | grep -E 'MajorVersion|MajorAtLeast|IsRegion|Region\('`)
shows only the two declared gates (`MajorAtLeast(72)`/`MajorAtLeast(79)` for
`remainLife`/trailing attribute) plus the petId gate matching
`coverage-manifest.yaml:13-14`'s prose. No undeclared gate motion.

---

## 3. Template routing table — `PetItemUseHandle` × `skillGate` (all 11 templates)

| template | `PetItemUseHandle` present | `opCode` | `options.skillGate` | valid? |
|---|---|---|---|---|
| `template_gms_12_1.json` | **no** | — | — | n/a — declared out-of-scope, `design.md:305`, still true |
| `template_gms_48_1.json` | yes | `0x75` | `equipAbility` | valid |
| `template_gms_61_1.json` | yes | `0x8E` | `equipAbility` | valid |
| `template_gms_72_1.json` | yes | `0xA5` | `equipAbility` | valid |
| `template_gms_79_1.json` | yes | `0xA7` | `equipAbility` | valid |
| `template_gms_83_1.json` | yes | `0xAB` | `equipAbility` | valid |
| `template_gms_84_1.json` | yes | `0xB0` | `equipAbility` | valid |
| `template_gms_87_1.json` | yes | `0xB7` | `equipAbility` | valid |
| `template_gms_92_1.json` | yes | `0xC8` | `equipAbility` | valid (fixed by `f7fc16631`, see §2) |
| `template_gms_95_1.json` | yes | `0xCB` | `equipAbility` | valid |
| `template_jms_185_1.json` | yes | `0xAE` | `petSkillFlag` | valid |

`pet_item_use.go:32-33,67-70` recognizes exactly `skillGateEquipAbility` and
`skillGatePetSkillFlag` as the valid set — every routed template's value is
one of these two. **10/11 templates route the handler with a valid gate; the
11th (`gms_12`) doesn't route it at all, which is a declared, unchanged
exclusion, not a live break.** No template routes the handler with a missing
or unrecognized `skillGate` — the specific "fails closed silently" failure
mode this check exists for does not occur anywhere in the current tree.

---

## 4. `petSkill` writer table — pet-encoding writers × table presence (all 11 templates)

| template | `CharacterInventoryChange` | `SetField` | notes |
|---|---|---|---|
| `template_gms_12_1.json` | present, **no `petSkill` table** | present, **no `petSkill` table** | pre-existing, untouched by this branch — no `PetItemUseHandle` either, consistent with the declared gms_12 exclusion |
| `template_gms_48_1.json` | present, `petSkill` table (`autoSpeaking:0x100`) | **absent entirely** (64 writers, none `SetField` — `1bc731215` commit msg, confirmed) | one table is correct: no second writer exists to table |
| `template_gms_61_1.json` | table | table | — |
| `template_gms_72_1.json` | table | table | — |
| `template_gms_79_1.json` | table | table | — |
| `template_gms_83_1.json` | table | table | — |
| `template_gms_84_1.json` | table | table | — |
| `template_gms_87_1.json` | table | table | — |
| `template_gms_92_1.json` | table (fixed by `1bc731215`) | table (fixed by `1bc731215`) | both writers were untabled pre-fix, silently encoding `0` |
| `template_gms_95_1.json` | table | table | — |
| `template_jms_185_1.json` | table (`consumeHP`, `consumeMP`, `autoSpeaking`) | table (same) | richer table matches design (JMS reads the pouch flag; GMS gates on worn equip) |

**Flag:** `gms_12`'s two pet-encoding writers (`CharacterInventoryChange`
`0x14`, `SetField` `0x29`) carry no `petSkill` table at all, so any pet cash
asset encoded through them silently writes `usPetSkill = 0`. This is the
exact "writer with no table" failure mode item 3 asks to flag. It is **not** a
regression introduced by this branch or its merge — `gms_12` was untouched by
`c97ccf6f2...HEAD` in the templates dir (confirmed: `template_gms_12_1.json`
does not appear in `git diff --name-only 8a9f70301...HEAD`) and is consistent
with the declared, unrevised gms_12 exclusion (`1bc731215`'s own commit
message: "gms_12 remains untabled — it still routes no pet handler"). Flagging
for completeness per the task's instructions, not as a task-139 defect.

---

## 5. `autoSpeaking: 0x100` family-wide application — documentation adequacy

Per the task framing: this bit is IDA-verified on v83 only
(`CPet::AutoSpeakingByEvent` @ `0x70761f`) and applied family-wide to all 10
routed GMS/JMS templates, explicitly *not* independently verified for
v48/v92. Judged as a declared assumption, not a hidden defect:

- The `1bc731215` commit message states the provenance precisely: names the
  one verified site, names the two newly-tabled versions as unverified,
  explains *why* absence-of-evidence couldn't be treated as evidence-of-
  absence for v48/v92 (`insn_query` can't match displacements; the control
  test against a known-positive site returned 0 hits, so the search is
  inconclusive, not negative). This is a well-reasoned, honestly-hedged
  assumption — it does not overclaim "verified" language for v48/v92.
- `design.md:292`'s table already recorded `autoSpeaking | 0x100 (v83) |
  v83 0x70761f` as v83-only evidence before this branch, so the family-wide
  application was always understood as an extrapolation, not a fresh claim.

**Gap:** none of this — the v48/v92-unverified caveat specifically — is
recorded anywhere in `coverage-manifest.yaml`'s `fields:` section, which is
the artifact `PROCESS.md` designates as the durable, git-log-independent
scope record. Today the caveat exists only in one commit message and in a
generic (not per-version-caveated) design.md table row. A future reader
running `packet-completeness-critic` or `/verify-packet` against
`gms_v48`/`gms_v92` for this op has no way to discover "the wire value here
is an unverified family-wide extrapolation" without `git log -p` archaeology.

**Recommendation:** append one sentence to `coverage-manifest.yaml:85-89`
(`fields:`) stating the `autoSpeaking=0x100` bit is v83-verified only and
applied family-wide by extrapolation, unverified for v48/v92 specifically —
mirroring the commit-message language. This is a documentation-placement
fix, not a re-verification requirement; the evidence standard in the commit
message itself is sufficient, it just needs to live in the canonical spot.

---

## Summary tables

### CHANGED-BUT-UNCLAIMED

| kind | file/packet | evidence | recommendation |
|---|---|---|---|
| version-scope (config) | `template_gms_92_1.json` (`skillGate`, `f7fc16631`) and (`petSkill` table, `1bc731215`) | `coverage-manifest.yaml:74-83` `versions:` list has no `gms_v92`; `design.md:305`/`plan.md:2331` still assert gms_92 is out of scope "no pet handlers at all" | add `gms_v92` to `versions:` + verify the cell, or move it to `out_of_scope:` with an explicit "merge-conflict repair only" note; correct `design.md`/`plan.md`'s stale exclusion text |
| documentation placement | `autoSpeaking=0x100` v48/v92 caveat | stated only in `1bc731215` commit message; not in `coverage-manifest.yaml` `fields:` | add one sentence to the manifest's `fields:` section |

### CLAIMED-BUT-UNVERIFIED

None. All 9 declared `versions` × `PET_AUTO_POT` cells are `verified`
(`gms_v48/61/72/79/83/84/87/95`, `jms_v185`), and `cash/serverbound/
CashItemUsePetSkill`'s 8 declared-n-a GMS cells + `jms_v185 verified` exactly
match `coverage-manifest.yaml:89`'s documented sweep result.
