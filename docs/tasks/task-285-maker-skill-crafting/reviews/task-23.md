# Task 23 review — `atlas-maker` craft validation, consumption plan, and saga emission

Range reviewed: `a8864d7..534212bfc` (commit `534212bfc`).
Scope confirmed: exactly the files the brief named
(`craft/plan.go`, `craft/processor.go`, `craft/inflight.go`, their `_test.go`
counterparts) plus the mechanical extension of `craft/eligibility.go`
(`Processor.Create`/`ReleaseInFlight`, widened `NewProcessor`) and its two
call-site updates in `eligibility_test.go`/`snapshot_test.go`. No drive-by
changes outside this surface.

## 1. Saga step sequences — verified against design §4.5.2 and payloads.go

**Mode 1|2** (`processor.go:199-262`): `AwardMesos` (`Amount: -int32(r.Meso())`,
`processor.go:232`) → `appendDestroySteps` (one `DestroyAssetFromSlot` per
`plan.Consumptions` entry, `processor.go:433-443`) → `AwardCraftedAsset` for an
equip output (`processor.go:246`) or `AwardAsset` otherwise (`processor.go:248`).
Matches design exactly. `TestCreateModeOneBuildsSequence` and
`TestCreateModeOneNonEquipUsesAwardAsset` assert this order and would fail
without it (they assert `steps[0].Action`/`steps[len-1].Action` and payload
fields, not just step count).

**Mode 3** (`processor.go:267-334`): `AwardMesos` (negative, `processor.go:315`)
→ `appendDestroySteps` for the leftover plan → `AwardAsset` for exactly one
`Draw` result (`processor.go:321-325`). Matches design.
`TestCreateModeThreeConsumesOneHundredLeftover` sums `DestroyAssetFromSlot`
quantities and asserts the total is `LeftoverConsumeQuantity` (100), not the
recipe's `count` (1) — a real regression test (deleting the
`LeftoverConsumeQuantity` override in `BuildCrystalPlan` in favor of
`r.Materials()[0].Count` would fail it).

**Mode 4** (`processor.go:341-429`): slot verification (`processor.go:359-368`,
scans `snap.Slots(req.EquipItemId)` for a stored slot equal to
`req.SlotPos`, independent of the client-declared slot) → `DestroyAssetFromSlot`
(`processor.go:399`) → `AwardAsset` per crystal (`processor.go:407`) →
`AwardMesos` for the charge (`processor.go:413`). Matches design order
exactly. `TestCreateModeFourVerifiesSlotBeforeDestroying` proves the reject
path emits nothing; `TestCreateModeFourChargesMeso` proves the sequence and
that `crystalband.CrystalForLevel` is invoked with the equip's `ReqLevel()`.

**Meso sign.** All three `AwardMesos` steps use `-int32(...)`
(`processor.go:232`, `:315`, `:418`) — genuinely negative, confirmed by
`TestCreateModeOneBuildsSequence`'s `assert.EqualValues(t, -1200, mesos.Amount)`.
No free-mesos sign-error found.

**`TemplateId` always set.** `resolveConsumption` (`plan.go:74`) sets
`TemplateId: itemId` on every `Consumption` it returns, unconditionally;
`appendDestroySteps` (`processor.go:440`) copies it straight into
`DestroyAssetFromSlotPayload.TemplateId` on every step, and the mode-4 path
sets it explicitly too (`processor.go:404`). `TestEveryStepUsesACompensableAction`
walks all three built sagas and asserts every action is in
`CompensableActions` (`AwardMesos`, `AwardAsset`, `AwardCraftedAsset`,
`DestroyAssetFromSlot` only). `grep` over `craft/*.go` confirms `DestroyAsset`
and `DestroyAllAssets` appear nowhere except in doc comments explaining why
they were rejected (`processor.go:145-147`, `snapshot.go:14`). PASS.

## 2. Multi-stack material consumption — verified

`resolveConsumption` (`plan.go:60-78`) walks `snap.Slots(itemId)` (already
ascending, `snapshot.go:60-64`) and emits one `Consumption` per slot actually
touched, capping each entry's `Quantity` at that slot's holding and
decrementing `remaining` until it reaches 0. `TestPlanResolvesMaterialAcrossMultipleSlots`
(`plan_test.go:37-65`) is a genuine red-without-the-fix test: it asserts two
entries — `(slot 1, qty 3)` and `(slot 7, qty 2)` — summing to exactly 5, and
would fail against a naive "first slot only" or "single entry claiming 5"
implementation. `TestPlanConsumesFromLowestSlotFirst` pins the ascending
order (1, 4, 7) independently of `TestSnapshotSumsQuantityAcrossSlots`.
PASS — this is the strongest-verified part of the unit.

## 3. Client input never trusted — verified

- `resolveConsumption`/`BuildCreatePlan`/`BuildCrystalPlan` read quantities
  only from `Snapshot` (`snap.Slots`, `snap.Held`), never from `Request`
  (`TestPlanNeverTrustsClientQuantities`, `plan_test.go:137-152`, proves a
  named-twice gem produces exactly one destroy step and an unheld gem
  produces none).
- Mode 4's slot check (`processor.go:359-368`) verifies the claimed
  `(EquipItemId, SlotPos)` pair against the snapshot before any mutation;
  `TestCreateModeFourVerifiesSlotBeforeDestroying` proves the mismatch case
  emits zero sagas.
- Gem/catalyst frequency is computed from `gemFrequency` over the request's
  named ids, but only ever used to bound how many of an *already-held* item
  are destroyed (`plan.go:86-92`, `:108-120`) — never to fabricate a count
  beyond what `Snapshot` proves is held.

PASS.

## 4. Concern 1 — `DisassembleMesoCharge = 0`

Independently re-checked, not merely trusted:

- `reagent-derivation.md` §5.7 item 2: "Crystal count per craft — UNKNOWN...
  Any per-craft quantity must come from `ItemMake.img` or the server, not
  from here," and item 5: `price = 1` is present on every archived
  `0426.img` info node but explicitly **not read** by `Load_MonsterCrystalLevel`
  — the doc's own author warns "do not attribute maker semantics to them."
- `design.md:449-465` (§4.3.2 mode-4 wire body) confirms `nMesoCost` is a
  server-computed clientbound field the client merely renders
  (`Format(SP_292_YOU_HAVE_LOST_MESOS_D, -v37)`); it is not something the
  server decodes from client-owned or archive data, so there is no wire or
  archive source to derive it from even in principle.
- `grep`/`find` across `docs/` and the repo for a reference-server
  crystallization or disassembly-charge implementation turned up nothing
  beyond what the report already cites.

**Ruling: this is a genuine unavailable-data blocker, not a missed value.**
The named constant with its doc comment (`processor.go:33-46`) is the correct
way to surface it under CLAUDE.md's "ambiguous design decision... surface it
and ask" carve-out — it is not a silently-landed placeholder; it is flagged
in the report, in a code comment, and the saga shape does not need to change
when a real formula is supplied. Non-blocking, but the controller should
treat "confirm a real meso-charge formula before players can disassemble
gear for free" as an open production item, not something this review can
close.

## 5. Concern 3 — judgment call A (ratified, no post-draw check)

Verified in code: `eligibility.go:217-226` (`awardsOf`) builds the
`CanAccommodate` request from *every* `RandomRewards()` entry when present;
`compartment.Processor.CanAccommodate`'s own doc
(`compartment/processor.go:20-25`) confirms the semantics are "would it
accept a grant of every item in items" — strictly stronger than "would the
one drawn item fit." `crystal()` (`processor.go:267-334`) performs no
second, narrower accommodation check after `Draw` — it relies solely on the
`Evaluate` call already run. Since mode 1|2 recipes in this dataset carry no
`randomReward` (confirmed: `createOrUpgrade` never calls `Draw` at all), the
only path this matters for is mode 3, and Task 21's reviewer already flagged
exactly this trade-off as "fail-safe... not blocking" (`task-21-review.md:61-71`).
Task 23's ratification is consistent and does not introduce a new
disagreement between the two layers — eligibility and execution genuinely
cannot diverge, because execution never re-checks. **Assessment: acceptable.**
The cost (a recipe with many random rewards can report `inventory_full` for a
player who would fit the one item actually drawn) is real but pre-existing
and already surfaced; this review does not treat it as newly introduced by
Task 23.

## 6. Concern 4 — OQ-7 scope extension to eligibility

`crystal()` (`processor.go:289-297`) adds `snap.Held(req.LeftoverItemId) <
LeftoverConsumeQuantity` as an explicit rejection (`insufficient_materials`)
layered on top of the generic eligibility check (which only proves the
archive's `count: 1` is held). The reasoning is sound: without this, a
character holding, say, 40 leftovers would pass `Evaluate` (which only
checks against the recipe's literal `count`), then `BuildCrystalPlan` would
destroy only the 40 actually held (via `resolveConsumption`'s shortfall-safe
behavior) while the client's hardcoded chat line still says "-100" —
reproducing the exact mismatch OQ-7 exists to prevent, just moved from
"accepted, wrong constant" to "accepted, silently under-consumed." Extending
the gate to require the full 100 is the correct fix and is in scope for
Task 23 to make (it is enforcing OQ-7's own resolution, not overriding it).

**Reference-server check:** the report says a targeted search for a
reference-server crystallization implementation found nothing beyond the
decompiled-client evidence design.md §5 already cites. Independently
confirmed: `reagent-derivation.md` and the rest of `docs/tasks/task-285-maker-skill-crafting/`
contain no such source, and no reference-server tree exists in this
checkout. This matches CLAUDE.md's "genuine external blocker" carve-out —
the check was performed, not skipped, and the outcome (no reversal) is
recorded in `plan.go:19-27`'s doc comment as the brief required.

**Gap found:** the extension itself (`processor.go:295-297`, the "held
between 1 and 99" rejection branch) has **no test**. `crystalHarness`
(`processor_test.go:165-184`) always seeds exactly 100 held, so no existing
test exercises a held count in `[1, 99]` that eligibility's generic check
(`count: 1`) would pass but this new guard must still reject. Deleting
`processor.go:295-297` would not fail any test in this package. This is a
real test-coverage gap in exactly the code path Concern 4 is about — see
Non-blocking findings below (not blocking because the guard's logic is a
single, simple comparison, correctly placed before any saga step is built,
and its absence would only be caught by a test that does not currently
exist; it is not a regression in already-tested behavior).

## 7. Concern 2 — extending `craft.Processor`

`Processor` now carries `NewSnapshot`/`Evaluate` (Task 21) plus
`Create`/`ReleaseInFlight` (Task 23). The brief's literal text
("`craft.Processor.Create`") is satisfied by construction once Task 21 had
already claimed the name; splitting into two types would force every real
caller (Task 24's channel handler, which needs both eligibility-for-display
and execution) to wire and hold two objects built from overlapping
dependency sets. `NewProcessor`'s signature grew to 11 positional
parameters (`eligibility.go:130`), which is a lot, but every one is a real
collaborator (7 upstream processors, ctx, logger, emitter) — not padding.
grep confirms no production caller outside `craft/` calls `NewProcessor` yet,
so the widened signature has zero blast radius today; Task 24 is the first
real caller and will need this exact surface anyway. **Ruling: acceptable.**
This is a legitimate scope note for the controller (worth confirming Task 24
doesn't need `Create`/`Evaluate` split for testability reasons), not a
defect.

## 8. Replay suppression

`inflightGuard` (`inflight.go:28-64`) is a `sync.Mutex`-guarded map;
`TryAcquire` is check-and-set under the same lock (no TOCTOU window). `Create`
(`processor.go:162-174`) acquires before any validation work and releases on
every error return (`craft.go:170`); on success the guard is deliberately
left held for `ReleaseInFlight` (Task 24's terminal-event consumer) to clear.
Traced every return path in `createOrUpgrade`/`crystal`/`disassemble`: all
of them return through `create()`'s single call site, so `Create`'s own
`if err != nil { craftGuard.Release(...); return }` covers every rejection
uniformly — there is no bypass path that returns an error without going
through `p.create(...)`. **No leak found.**

`go test ./craft/... -race -count=1 -v` (re-run independently, not just
trusted from the report):

```
ok  	atlas-maker/craft	1.056s
```

All 18 top-level tests + 8 `TestEveryRejectionEmitsNoSaga` subtests pass,
race-clean. `TestGuardIsConcurrencySafe` fires 50 goroutines at one key and
asserts exactly one `TryAcquire` succeeds — a genuine concurrency test, not
a single-goroutine formality. Cross-test isolation is handled correctly:
`testContext(t)` (`eligibility_test.go`) mints a fresh tenant UUID per call,
so the process-wide `craftGuard` singleton cannot leak state between test
cases even though the same `characterId` literal (1001) is reused across
many of them — verified this is deliberate, not accidental, by checking that
`TestEveryRejectionEmitsNoSaga`'s "craft in progress" subtest (which leaves
its guard held on purpose, `processor_test.go:397-412`) runs under its own
`buildCreateProcessor` call with its own fresh tenant context. PASS.

## 9. Carried constraint — `recipe.Model` backing-slice hazard

Checked every call site: `plan.go:103` (`for _, mat := range r.Materials()`),
`eligibility.go:222` (`for _, rw := range rewards` from `r.RandomRewards()`),
`processor.go:299` (`Draw(r.RandomRewards())`), `draw.go:52-71` (`Draw`'s own
doc: "rewards is read only; it is never sorted, reordered, or mutated, since
callers may pass the recipe cache's own backing slice" — and the body only
indexes into it, never calls `sort.Slice` or appends in place). No
mutation, sort, or reorder found anywhere Task 23 touches a `recipe.Model`
slice accessor. PASS — the hazard the carried constraint warned about does
not bite here.

## 10. Other disclosed decisions

- **`Request.WorldId`/`ChannelId`**: reasonable and consistent with the
  existing pattern of other Atlas services carrying session scope on a
  request the domain processor itself doesn't track; confirmed
  `AwardMesosPayload` genuinely requires both fields
  (`libs/atlas-saga/payloads.go`) and that the orchestrator's
  `handleAwardMesos` builds a channel model from them
  (`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`).
  Not evaluable further within this unit's scope — Task 24 is where the
  channel handler actually supplies these, and that wiring is out of scope
  here.
- **`reagentStats` dropping `incReqLevel`/`randOption`/`randStat`**: verified
  `AwardCraftedAssetPayload` (`libs/atlas-saga/payloads.go:1078-1093`) has no
  field for any of the three — the twelve stat cases in `reagentStats`
  (`processor.go:472-497`) exhaust the reagent stat vocabulary
  `reagent/builder.go` actually allows (`grep` confirms the 12 string
  literals match exactly). Correct as implemented.

## 11. Test-quality findings (spot-checked, not a full sweep of all ~26 tests)

- **`reagentStats`/`addStat` have no dedicated test.** No test in
  `processor_test.go` or `plan_test.go` exercises the 12-case stat-mapping
  switch or the negative-delta-clamps-at-zero behavior in `addStat`
  (`processor.go:447-500`). `TestCreateModeOneBuildsSequence` passes no
  `GemItemIds` and never inspects the `AwardCraftedAssetPayload`'s stat
  fields, only `Slots`/`TemplateId`. A wrong stat-string mapping (e.g.
  `"incDEX"` accidentally wired to `Strength`) would ship silently. Given
  this directly implements FR-3.1/3.2's reagent-adjusted stats — a
  money-path requirement — this is a real coverage gap, not a nice-to-have.
- **`TestCreateModeOneBuildsSequence`'s own comment is misleading.**
  `processor_test.go:117-119` says "AwardMesos (negative) -> 2 destroy steps
  (material + catalyst) -> AwardCraftedAsset," but `newEligibleHarness`
  (`eligibility_test.go:153-167`) never seeds the catalyst (4130000) into
  `h.etc`, so `BuildCreatePlan`'s catalyst branch resolves to zero
  consumptions (FR-3.2's "omit when not held" — correctly implemented) and
  the actual sequence is mesos + 1 destroy (material only) + award = 3
  steps, matching `require.Len(t, steps, 3)`. The assertion is correct; the
  comment describing it is not. This also means **no processor-level test
  exercises a held-and-consumed catalyst in a built saga** — that behavior
  is only unit-tested at the `BuildCreatePlan` level
  (`TestPlanIncludesCatalystWhenFlagSetAndHeld`), which is adequate but
  worth knowing when reading this test file.
- **Concern-4 guard is untested** — see §6 above; restated here because it
  belongs in the coverage tally.

None of these three are blocking: the underlying logic in each case is
either simple enough to verify by direct inspection against a closed
whitelist (`reagentStats`), or already covered at a different layer
(`BuildCreatePlan`'s catalyst path), or a single comparison whose absence
would be caught by the very OQ-7 concern the report itself flags for
controller attention. They are flagged as `APPROVED_WITH_FINDINGS` material
because a controller deciding whether Task 24 can safely build on this
should know these three specific gaps exist before doing so.

## 12. Independent verification run

```
cd services/atlas-maker/atlas.com/maker
go build ./...                        # clean
go vet ./craft/...                    # clean
gofmt -l craft/                       # no output
go test ./... -count=1                # all packages pass
go test ./craft/... -race -count=1 -v # all 18 tests + 8 subtests pass, race-clean
```

Matches the report's claimed output exactly; no discrepancy found.

## Not evaluable

- The cross-service seam this unit depends on (`AwardCraftedAsset` wiring
  into `atlas-saga-orchestrator`, `atlas-inventory`'s creation-command
  extension per design §4.5.1) is **not** part of this commit — `libs/atlas-saga/payloads.go`
  already carries `AwardCraftedAssetPayload` (pre-existing, read-only
  dependency for this unit), but whether the orchestrator's handler,
  compensator, and `atlas-inventory`'s command actually implement the new
  contract is a different task's surface and was not re-verified here.
  Flagging so it is not silently assumed done by virtue of this review
  passing.
- `Request.WorldId`/`ChannelId` being correctly populated by the calling
  channel handler is Task 24's responsibility and not evaluable from this
  diff alone.

## Verdict rationale

No blocking defect found in the money path: all three saga sequences match
design exactly, the meso sign is correct in every case, multi-stack
consumption is correct and genuinely tested, client input is never trusted,
`TemplateId` is always set, and the in-flight guard has no leak and is
race-clean under an independently-run `-race` pass. The four disclosed
concerns are all legitimate, correctly surfaced, and either genuinely
unresolvable with current evidence (Concern 1) or a defensible, previously-flagged
trade-off (Concern 3) or a correct scope decision (Concerns 2, 4). The
findings that keep this out of a bare `APPROVED` are test-coverage gaps
(reagent stat mapping, the OQ-7 eligibility-extension guard) and one
misleading test comment — real, but none of them a shipped defect in this
commit.

---

verdict: APPROVED_WITH_FINDINGS
artifact: .superpowers/sdd/plan/task-23-review.md
scope_confirmed: commit 534212bfc — craft/plan.go, craft/processor.go, craft/inflight.go, their _test.go files, and the mechanical Processor-interface extension in craft/eligibility.go plus its two test call-site updates. No files touched outside this surface.
blocking: 0
non_blocking: 3
  - services/atlas-maker/atlas.com/maker/craft/processor.go:464-500 — reagentStats/addStat (the 12-case reagent-stat mapping and zero-clamp) has no dedicated test; TestCreateModeOneBuildsSequence never passes gems or inspects stat fields, so a wrong stat-to-field mapping would ship silently.
  - services/atlas-maker/atlas.com/maker/craft/processor.go:295-297 — the OQ-7 eligibility-extension guard (reject mode 3 when held < LeftoverConsumeQuantity but >= the recipe's literal count) is untested; crystalHarness always seeds exactly 100 held, so deleting this line would not fail any current test.
  - services/atlas-maker/atlas.com/maker/craft/processor_test.go:117-119 — comment claims "2 destroy steps (material + catalyst)" but the harness never holds the catalyst, so the test actually exercises only 1 destroy step; misleading documentation, and no processor-level test covers a held-and-consumed catalyst in a built saga (only unit-tested via BuildCreatePlan).
not_evaluable: 2
  - The atlas-saga-orchestrator/atlas-inventory side of the AwardCraftedAsset cross-service seam (design §4.5.1) is a different task's surface, not part of this commit.
  - Request.WorldId/ChannelId being correctly populated by the calling channel handler is Task 24's responsibility.
