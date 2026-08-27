# Plan-adherence audit — task-263, plan Tasks 21-27

Agent: `plan-adherence-reviewer` (sonnet), range shard 21-27 of 27.
Branch `task-263-backend-guideline-conformance`, HEAD `f8de7249c`, merge base `eaa5ce6f7`.

**Verdict: APPROVED_WITH_FINDINGS. Blocking: 0. Non-blocking: 2. Not evaluable: 0.**

> Transcribed by the controller from the agent's return. This shard reported inline and did not
> write its own artifact file; the other two shards' artifacts are verbatim agent output.

All seven tasks are substantively implemented, with contemporaneous evidence in `progress.md`,
`exemptions.md`, and `reviews/`. Every blocking finding raised during the branch's own review cycle
was traced through to a fix and confirmed against current code.

## Per task

- **Task 21** (relocate, second half) — IMPLEMENTED. `builder.go` verified present in every named
  target: `atlas-quest`, `atlas-tenants`, `atlas-channel` (incl. the later-added
  `account/builder.go`), `libs/atlas-constants`, `libs/atlas-database`, `atlas-storage`,
  `atlas-saga-orchestrator`, `atlas-notes`, `atlas-messages`, `atlas-marriages`, `atlas-inventory`,
  `atlas-families`, `atlas-drops`, `atlas-doors`, `atlas-character-factory`, `atlas-character`, and
  all named `atlas-npc-conversations` subpackages.
- **Task 22** (cashshop `reference_data.go` split) — IMPLEMENTED. All 7 `*ReferenceDataBuilder` sets
  confirmed in `services/atlas-cashshop/atlas.com/cashshop/asset/builder.go`; `reference_data.go`
  has zero remaining `Builder` matches.
- **Task 23** (npc-conversations 20-builder split) — IMPLEMENTED. `builder.go` declares exactly 20
  `type *Builder struct`; `model.go` has no declaration-level residue.
- **Task 24** (FILE-05 closeout) — IMPLEMENTED. Regenerated `inventory-file05-builders.txt` equalled
  exactly the 5 documented dispositions; independently re-verified (`review-task-24.md`).
- **Task 25** (`exemptions.md`) — IMPLEMENTED. 664 lines, all required sections present, no bare
  "out of scope" (every hit paired with `file:line`), sibling-builder section deduped to 25 packages.
- **Task 26** (FR-17/FR-18 behavior preservation) — IMPLEMENTED. Full 6-step audit; one reviewer
  round (`review-task-26.md`) found an unreproducible number in Step 6, fixed in `109d2c474`;
  converged on 3 itemized benign residuals.
- **Task 27** (final gate, scaffolding removal, review) — IMPLEMENTED. Flagless `tools/verify.sh`
  PASS at `ea15a4477` (91 modules, `-race`, all guards). Scaffolding removed in `f8de7249c` with
  durable artifacts kept; tree clean. The Step-1 sweep's 2-of-3 failing counts were correctly
  disposed of: one genuine missed FR-14 rename (`atlas-quest/quest/builder.go`
  `NewModelBuilder`→`NewBuilder`, fixed `e867279bc`) and six DOM-04 "has-model" rows that were
  detector artifacts, now documented in `exemptions.md`. The separate `backend-guidelines-reviewer`
  audit (`reviews/audit.md`) found two further genuine FILE-05/DOM-01 misses outside the sweep's
  regex reach — `atlas-cashshop/cashshop/asset`'s generic `Builder[E any]` and
  `atlas-channel/channel/account`'s lowercase `builder` — both fixed in `ed4fbb94d`.

## Non-blocking findings

1. `services/atlas-npc-conversations/atlas.com/npc/conversation/model.go:444` — doc comment
   referencing `CraftActionBuilder`. **Controller disposition: no change.** On inspection this is not
   stale: the comment on `NewCraftActionModelDirect` reads "…cannot be produced through the validated
   `CraftActionBuilder`", a correct cross-reference to the type at `builder.go:722` explaining what
   the direct constructor deliberately bypasses.
2. Plan Task 27 Step 4 called for two reviewer dispatches; only the guideline audit ran, because the
   plan's named `atlas-reviewer` agent type does not exist in this repo. This plan-adherence audit
   closes that gap.
