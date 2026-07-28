# task-160-skill-cast-consumption — Execution Context

Companion to `plan.md`. Summarizes the key files, locked decisions, and dependencies an implementer needs; the plan has the step-by-step detail.

## Scope reconciliation (2026-07-27)

`task-158` (PR #1003, "Shadow Stars") landed on main and implemented the entire
Shadow Stars half of the original task-160 PRD. What was originally FR-2 (cast-time
`bulletConsume` + SHADOW_CLAW star-id rewrite), FR-3 (claw attack-path skip), and
Task 5 (the `SpiritJavelinItemId` getter) are all DONE on main. This task is now
**FR-1 only: generic `itemConNo` quantity plumbing.** See `plan.md`'s "Superseded
scope" table for exactly where each shipped item lives.

## What this task does

One atlas-channel behavior change (original PRD FR-1): skill casts consume the WZ
`itemConNo` quantity (was hardcoded `1`), drawn from the lowest-index slot that
alone holds ≥ that amount. Shortfall (no single slot has enough): warn + skip +
cast proceeds (unchanged defense-in-depth stance — an `itemCon` shortfall does not
block the cast). Absent/zero `itemConNo` floors to `1`.

Concrete driver: a skill with `itemConNo > 1` — e.g. Echo of Hero consuming 2×
Magic Rock (4006000) on a later-version tenant — currently deducts only one. v83
WZ has every `itemConNo == 1`, so v83 is unaffected; this is cross-version
correctness plumbing.

## Legacy versions: no special handling

FR-1 is entirely WZ-data-driven and server-side. `itemConNo` is a skill-effect
attribute resolved per-tenant through atlas-data (`effect.Model.ItemConsumeAmount()`
← `atlas-data/skill/reader.go:218`), NOT a wire field. The legacy version bring-up
(v48/v61/v72/v79) does not touch the itemCon path. No version branches, no gates.
(The legacy work's one intersection with this task's surface is a new
`RequestItemConsume` call site — the owl shop-scanner, `shopscanner/processor.go:79`
— which Task 2's compiler-enforced sweep covers by passing a literal `1`.)

## Key files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/compartment/model.go` | NEW method `FindFirstByItemIdWithQuantity(templateId uint32, quantity int16)` — sorts by slot ascending, single-slot-with-enough |
| `services/atlas-channel/atlas.com/channel/consumable/processor.go` | `RequestItemConsume` gains `quantity int16` (before `updateTime`) on BOTH the `Processor` interface and `ProcessorImpl`; floors `<1` to 1 |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | `UseSkill` itemCon block (lines 105-118): real `ItemConsumeAmount()`, quantity-aware slot lookup, two new seams (`loadCasterWithInventoryFunc`, `requestItemConsumeFunc`) |

`RequestItemConsume` call sites that gain a literal `1` (behavior unchanged):
`shopscanner/processor.go`, `socket/handler/character_item_use.go` (×3),
`character_cash_item_use.go`, `pet_food.go`, `pet_item_use.go`, plus
`skill/handler/common.go` (which then gets the real amount in Task 3). Eight sites,
seven files — the compiler enumerates them all on the signature change.

## Locked decisions (do not relitigate)

- **Signature change, not a sibling method**, for the quantity parameter — the compiler enforces the call-site sweep. Touches the interface AND `ProcessorImpl` (post-task-116 Gen3 processor shape).
- **Single-slot draw only.** Choose the lowest-index slot that alone holds ≥ the amount. An aggregate-enough-but-split inventory (e.g. 1+1 for a cost of 2) is a shortfall: warn + skip + cast proceeds. No multi-slot draw.
- **Shortfall permits the cast** (defense-in-depth stance) — an `itemCon` shortfall never rejects the cast. (This differs from Shadow Stars' `bulletConsume`, which task-158 already rejects on shortfall; that is a separate, shipped path.)
- **Floor `< 1` to `1` in TWO places** — the processor layer (guards every caller) and the cast-path layer (self-documents intent). Absent/`"0"`-string `itemConNo` means one item.
- **Amount is WZ-data-driven** — `effect.Model.ItemConsumeAmount()`, never a skill-id or version branch.

## Test strategy

- Package-level var-func seams restored via `t.Cleanup` (existing `common.go` convention: `loadCasterFunc`, `loadCasterInventoryFunc`, `rectQueryFunc`). New seams: `loadCasterWithInventoryFunc` (full `character.Model`, distinct from task-158's USE-only `loadCasterInventoryFunc`) and `requestItemConsumeFunc`.
- Builder-pattern setup only (`character.NewModelBuilder`, `inventory.NewBuilder`, `compartment.NewBuilder`, `asset.NewModelBuilder`, `effect.Extract(effect.RestModel{...})`, `packetmodel.NewSkillUsageInfoBuilder`); no `*_testhelpers.go`.
- `UseSkill` is testable offline with those seams: the test skill id is not `NightLordShadowStarsId` (Shadow Stars pre-flight inert), HP/MP/cooldown gate on zero-valued effect fields, mob path early-returns on empty `AffectedMobIds`, dispatcher lookup misses the arbitrary id, mount guards don't match it.
- `github.com/pkg/errors` is NOT in atlas-channel's go.mod — use stdlib `errors` in tests.
- Verify builder/RestModel field names against the current tree before running (`effect.RestModel` field for `itemConNo` — the reader setter is `SetItemConsumeNumber`; `inventory.NewBuilder(...).SetCompartment/MustBuild`; `compartment.NewBuilder(...).Build()` return shape). The assertions are the contract; builder mechanics may need small adjustment.

## Task order & dependencies

```
Task 1 (compartment helper) ─┐
                             ├─→ Task 3 (UseSkill itemCon amount + seams) ─→ Task 4 (verification)
Task 2 (quantity signature) ─┘
```

Task 1 and Task 2 are independent. Task 3 consumes both. Task 4 is last.

## Verification (Task 4)

`go test -race`, `go vet`, `go build` in `services/atlas-channel/atlas.com/channel` (the only touched module — no atlas-constants or atlas-packet change in the descoped plan); `docker buildx bake atlas-channel` from the worktree root; then repo-root guards `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` (run `tools/lint.sh` to auto-fix formatting). Code review (`superpowers:requesting-code-review`) before any PR.

## Reference points in the current tree (post-merge line numbers)

- Hardcoded quantity: `consumable/processor.go:43`; interface method: `processor.go:17`
- itemCon block: `skill/handler/common.go:105-118`; existing caster seams: `common.go:32` (`loadCasterFunc`), `shadow_stars.go:122` (`loadCasterInventoryFunc`)
- `FindFirstByItemId` (pattern to mirror): `compartment/model.go:58`
- New call site from legacy work: `shopscanner/processor.go:79`
- `effect.Model.ItemConsumeAmount()`: `data/skill/effect/model.go:93`
- atlas-data `itemConNo` read: `atlas-data/skill/reader.go:218`
