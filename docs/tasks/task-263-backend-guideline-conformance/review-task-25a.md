# Review — Task 25-A (`chore(task-263): record DOM-04 guideline exemptions`, commit `4ed1cef25`)

Scope: commit `4ed1cef25`, which adds
`docs/tasks/task-263-backend-guideline-conformance/exemptions.md` (369 lines, new file) — the three
DOM-04 sections only. Brief: `.superpowers/sdd/plan/task-25a-brief.md`. Report:
`.superpowers/sdd/plan/task-25a-report.md`. No `.go` files changed; build/test not applicable.

## 1. Citation sampling (the highest-value check)

Sampled 30+ distinct `file:line` citations spread across all three DOM-04 sections, deliberately
including citations into files this branch rewrote in Tasks 19–23B (`builder.go`, `model.go`,
`rest.go`), plus every citation into a builder-heavy file (`atlas-npc-conversations/conversation`,
`atlas-tenants/configuration`, `atlas-transports/instance`, `atlas-merchant/frederick`).

**Every sampled citation resolved exactly at HEAD.** Verified with `grep -n`/`sed -n` against the
live tree (not the frozen inventory/TSV sources):

- Section 1: `services/atlas-account/atlas.com/account/ban/rest.go:8`, `.../cashshop/configuration/tenant/cashshop/commodities/rest.go:3`, `services/atlas-data/atlas.com/data/map/model.go:9`/`:139`, `services/atlas-merchant/atlas.com/merchant/frederick/model.go:9`/`:25`, `.../frederick/rest.go:10` (`TransformStatus`, confirmed zero `ItemModel`/`MesoModel` hits in `rest.go`), `services/atlas-npc-conversations/atlas.com/npc/cosmetic/model.go:20`/`rest.go:61`, `services/atlas-transports/atlas.com/transports/instance/model.go:15`/`:106`/`rest.go:51`/`:99`, `services/atlas-channel/atlas.com/channel/party_quest/model.go:5`/`rest.go:56`, `services/atlas-maps/atlas.com/maps/data/map/monster/model.go:5`/`rest.go:39`.
- Section 2 (all 12 `NO-RESTMODEL` packages, verified independently against `classify-dom04.tsv`'s own 12 `NO-RESTMODEL` rows — package-path list is an exact match): `rewardpool/model.go:3`, `processor.go:37`/`:42`, `rest.go:43`; `channel/data/tradeability/rest.go:17`,`:29`,`:162`,`:166`,`:170`,`:174`,`:178`; `inventory/data/tradeability` (same shape, spot-checked function set matches); `monsterbook/model.go:11`, `rest.go:102/107/112/125` (via `grep -n`, not sed-offset counting); `messengers/character/rest.go:97/112`; `parties/character/rest.go:98/113`; `drops/data/foothold/rest.go:64/76`; `saga-orchestrator/rates/rest.go:24/35`; `saga-orchestrator/reactor/drop/rest.go:131/142`; `pets/data/position/rest.go:31/43`; `npc/conversation/model.go:21`, `rest.go:720/756/764/973` (`Extract*`) and `362/419/427/555` (`Transform*`); `tenants/configuration/model.go:11`, `rest.go:126/221/348/503/635/785/860/938/1070` (`Extract*`) and `50/191/280/440/617/708/843/904/1006` (`Transform*`) — all 19 line numbers matched exactly.
- Section 3: `login/character/model.go:44,47-53`; `npc/petdata/model.go:8`; `channel/monster/rest.go:33`; `channel/pet/rest.go:21`, `model.go:77-79`; `channel/reactor/builder.go:26-33` (`NewBuilder`); `messages/character/model.go:225` (`Stance` stub); `messages/data/map/model.go` (bare `struct{}`) and `messages/map/processor.go:40` (`Exists`, confirmed it is the *parent* `map` package, correctly distinguished from `data/map`); all 18 `handwork-notes.md` line citations (`:30,34,38,39,44,45,50,51,68,69,77,78,79,85,86,88,89,90`) match their content exactly.

**Hit rate: 100% (0 stale citations found in the sample).** No citation into a rewritten
`model.go`/`rest.go`/`builder.go` was stale; the implementer's claim to have re-derived every
citation against HEAD is substantiated by direct verification, not merely asserted.

## 2. Substantive claim checks

**Claim 1 — all 12 `NO-RESTMODEL` packages now have named `Transform*` coverage.** Independently
re-derived the 12-row list from `classify-dom04.tsv` (`grep NO-RESTMODEL`) — it matches the
document's 12 package paths exactly, in the same order. For each, `grep -n "^func Transform"` on
the live `rest.go` confirms the named `Transform*`/`TransformX` functions the document cites all
exist at the cited lines. The override of the brief's stale 9-of-12 pre-read is correct: Task 14's
four batches did close all remaining `NO-RESTMODEL` packages, and this document does not repeat the
stale split. **Verified true.**

**Claim 2 — 171 of 176 DOM-04-no-model rows genuinely lack `type Model`, 5 excluded as false
positives.** Confirmed `inventory-dom04-no-model.txt` has exactly 176 rows. Took a spread sample of
14 directories (every 12th row, `awk 'NR%12==1'`) including one of the five excluded rows
(`atlas-maps/data/map/monster`, correctly absent from the `atlas-maps` cluster and present only in
the "Excluded" subsection) and ran `grep -rln "^type Model struct"` across all 14 — zero hits, as
claimed. The five exclusions (`party_quest`, `data/map/monster`, `cosmetic`, `transports/instance`,
`marriages/marriage`) were each independently spot-checked for their cited domain-type declaration
and `Extract`/`Transform` asymmetry — all confirmed accurate at `file:line`. **Verified true**, and
the exclusion of these five from the cluster (rather than folding them in as false exemptions) is
the correct call — this is genuinely new information the brief did not anticipate, and it is
flagged for the controller rather than silently ruled on, consistent with the controller's
after-the-fact note that this falls to Task 25-B.

**Claim 3 — the 19 lossy-`Extract` findings trace to `handwork-notes.md`.** All 14 heading-level
findings plus the 4 reference-type-copy bullets are present in `handwork-notes.md` at the cited
lines, content-matched verbatim (dropped fields, pointer-copy vs. aliasing, map/slice copy
rationale). The `19` figure reconciles: 14 single-finding package headings + `channel/data/consumable`(1) + `channel/data/equipment`(1) + `consumables/cash`(2, since it separately names both a
`map[SpecType]int32` and a `[]string` reference-copy finding) + `monsters/monster/mobskill`(1) = 19.
Not immediately obvious from the section's own prose, but arithmetically sound on inspection.
**Verified true**, non-blocking note only (the reconciliation could have been spelled out).

## 3. Acceptance criteria

- **No literal absolute/home path**: `grep -n "/home/\|/Users/" exemptions.md` — zero hits. Pass.
- **No "out of scope" phrase without a `file:line` on the same entry**: all 4 occurrences (lines
  263, 275, 281, 293) sit in bullets that carry a `.go:line` citation earlier in the same bullet.
  Pass.
- **Every `### ` heading followed, before the next `### `, by at least one backtick-quoted
  `<path>.go:<line>`**: mechanically checked all 62 headings. **6 headings fail this criterion** —
  see Blocking findings below.

## Blocking findings

1. `docs/tasks/task-263-backend-guideline-conformance/exemptions.md:87` — the `### services/atlas-maps` heading's block (lines 88–89) cites only `monster/rest.go` (no line number) and a package path; no `<path>.go:<line>` appears anywhere before the next `### ` heading at line 91. Violates the brief's explicit acceptance criterion.
2. `docs/tasks/task-263-backend-guideline-conformance/exemptions.md:318` — `### services/atlas-merchant/atlas.com/merchant/data/portal` cites bare `rest.go` (no line) and `handwork-notes.md:39` (not a `.go` file); no `<path>.go:<line>` before the next heading at line 322.
3. `docs/tasks/task-263-backend-guideline-conformance/exemptions.md:338` — `### services/atlas-channel/atlas.com/channel/mts/listing` cites bare `rest.go` and `handwork-notes.md:77`; no `<path>.go:<line>` before the next heading at line 342.
4. `docs/tasks/task-263-backend-guideline-conformance/exemptions.md:342` — `### services/atlas-channel/atlas.com/channel/mts/wish` cites no file at all with a line number, only `handwork-notes.md:78`; no `<path>.go:<line>` before the next heading at line 346.
5. `docs/tasks/task-263-backend-guideline-conformance/exemptions.md:350` — `### services/atlas-npc-shops/atlas.com/npc/character` cites bare `rest.go`, bare `model.go`, and `handwork-notes.md:90`; no `<path>.go:<line>` before the next heading at line 354.
6. `docs/tasks/task-263-backend-guideline-conformance/exemptions.md:358` — `### services/atlas-messages/atlas.com/messages/data/map` cites bare `model.go`, `map/processor.go` (no line — the actual `Exists` function is at `map/processor.go:40`, which the document does not cite), and `handwork-notes.md:86`; no `<path>.go:<line>` before the next heading at line 362 (start of the reference-type-copy sub-list, which is itself not a `### ` heading).

All 6 are in Section 3 (lossy-`Extract`), where the source material (`handwork-notes.md`) uses bare
filenames without line numbers for several entries and the implementer carried that omission
forward instead of adding the missing line numbers it had already re-derived elsewhere in the same
document (e.g. `messages/character/model.go:225` two headings later does cite a line correctly,
proving the convention was known and achievable). This is a mechanical, easily-fixed defect, but it
is a hard acceptance criterion stated in the brief, and 6 of 62 headings (~10%) fail it.

## Non-blocking

- Section 3's "19" count is correct but not self-evidently reconciled in the prose; a reader has to
  do the arithmetic in §2 above by hand. Not required by the brief, but would strengthen the
  artifact.
- `exemptions.md:22` ("`Extract`'s existing mapping does alias `rm.Summons`...out of this task's
  scope to change", quoted from `handwork-notes.md:88`) repeats a decision that is a `handwork-notes.md`
  quote, not new "out of scope" language coined by this task — fine, just noting it is a citation, not
  an independent scope claim.

## Not evaluable

- None. The full DOM-04 surface (all three sections, 369 lines) was reviewed; no part of the unit
  fell outside a checkable surface.

## Scope confirmation

The commit matches its stated purpose exactly: it adds only `exemptions.md`'s three DOM-04 sections
(§12–370), touches no other file, and does not encroach on 25-B's DOM-01/FILE-05 sections or the
"Excluded from this section" subsection's unresolved ruling (correctly left open per the
controller's note). No scope mismatch.

---

verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-263-backend-guideline-conformance/review-task-25a.md
scope_confirmed: commit 4ed1cef25 — exemptions.md DOM-04 sections 1–3 (lines 1–370), all citations sampled and substantive claims independently verified against HEAD
blocking: 6
  - docs/tasks/task-263-backend-guideline-conformance/exemptions.md:87 — `### services/atlas-maps` heading has no backtick-quoted `<path>.go:<line>` before the next heading (bare `monster/rest.go`, no line number)
  - docs/tasks/task-263-backend-guideline-conformance/exemptions.md:318 — `### .../merchant/data/portal` heading has no `<path>.go:<line>` (bare `rest.go`; only `handwork-notes.md:39`, not a `.go` file)
  - docs/tasks/task-263-backend-guideline-conformance/exemptions.md:338 — `### .../channel/mts/listing` heading has no `<path>.go:<line>` (bare `rest.go`; only `handwork-notes.md:77`)
  - docs/tasks/task-263-backend-guideline-conformance/exemptions.md:342 — `### .../channel/mts/wish` heading has no `<path>.go:<line>` at all (only `handwork-notes.md:78`)
  - docs/tasks/task-263-backend-guideline-conformance/exemptions.md:350 — `### .../npc-shops/npc/character` heading has no `<path>.go:<line>` (bare `rest.go`/`model.go`; only `handwork-notes.md:90`)
  - docs/tasks/task-263-backend-guideline-conformance/exemptions.md:358 — `### .../messages/data/map` heading has no `<path>.go:<line>` (bare `model.go`, uncited `map/processor.go`; the real citation `map/processor.go:40` exists but was not written)
non_blocking: 2
not_evaluable: 0
