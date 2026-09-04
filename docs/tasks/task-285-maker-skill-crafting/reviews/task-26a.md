# Task 26a review — the craft consumption manifest carrier

## Verdict

APPROVED

## Scope reviewed

Commit `61ff8cbd8` (range `331b0c0b5..61ff8cbd8`), against
`.superpowers/sdd/plan/task-26a-brief.md` including the ADDENDUM (Step 5b), and
`docs/tasks/task-285-maker-skill-crafting/manifest-carrier-derivation.md`. Diff
touches exactly the three modules named in the brief:

```
libs/atlas-saga/{model,payloads,unmarshal,payloads_test}.go
services/atlas-maker/.../craft/{plan,plan_test,processor,processor_test}.go
services/atlas-saga-orchestrator/.../saga/{compensator,event_acceptance,
  event_acceptance_test,handler,model,producer,producer_test}.go
```

No file in `services/atlas-channel` is touched (`git diff --stat ... --
services/atlas-channel` is empty) — Task 26's territory is untouched, per
scope. `rest.go`'s `payloadUnmarshalers` and `compensator.go`'s
`lateCompensableActions` are both untouched, as the brief explicitly requires.

Ran independently (not just trusting the report):

```
libs/atlas-saga: go build ./... && go test ./... -count=1        → ok
atlas-maker:     go build ./... && go test ./craft/... -count=1  → ok
saga-orchestrator: go build ./... && go test ./saga/... -count=1 → ok (saga, saga/mock)
tools/lint.sh --go <all three modules>                           → 0 issues each
gofmt -l <all three module trees>                                → no output
```

`TestStepUnmarshal_EveryActionRepresented` (orchestrator,
`unmarshal_completeness_test.go:24`) passes with a `record_craft_manifest`
subtest; `TestAcceptanceTable_EveryActionRepresented` passes.

## Point-by-point findings

### 1. `InventoryTransaction` is not maker-exclusive — PASS

`producer.go:243` guards the new `EmitSagaFailed` arm on
`craftManifestCharacterId(s) != 0`, i.e. the manifest step's presence, and is
placed **before** the `ExtractCharacterCreationIds` fallthrough — never on
`s.SagaType() == InventoryTransaction`. `craftManifestCharacterId`
(`compensator.go:2318-2333`) returns `0` when no `RecordCraftManifest` step
exists, matching the doc comment's stated contract.

Regression coverage is real, not just present:
`TestFailedNonCraftInventoryTransactionUnchanged`
(`producer_test.go`) builds an `InventoryTransaction` saga with **no**
`RecordCraftManifest` step and asserts `characterId == 0` — i.e. it still
falls through to `ExtractCharacterCreationIds`, unchanged. If the new arm
had instead matched on `SagaType`, this test would fail (the saga's type is
`InventoryTransaction`), so the test is honest, not a tautology.

The `timer.go` timeout entry point is covered, not just the compensator:
both `timer.go:195` and every `compensator.go` call site invoke the same
top-level `EmitSagaFailed` function with no per-caller branching inside it.
`TestFailedCraftSagaCarriesCharacterId` and
`TestFailedNonCraftInventoryTransactionUnchanged` call `EmitSagaFailed`
directly and swap the package-level `emitSagaFailedByIdsFn` var to observe
the emitted arguments — since the function under test is the literal shared
call target of both real entry points, exercising it directly is a valid
substitute for calling through each entry point separately (confirmed by
`grep`: both `timer.go:195` and every `compensator.go` `EmitSagaFailed(...)`
call site route through the identical function).

### 2. `record_craft_manifest` is self-completing — PASS

- `event_acceptance.go:164` — `sharedsaga.RecordCraftManifest: {}`, empty
  slice, matching `IncubatorResult`/`ValidateCharacterState`.
- `handler.go:1338-1355` — `handleRecordCraftManifest` validates the payload
  type then calls `StepCompleted` directly, no produce. Matches
  `handleIncubatorResult`'s shape minus the produce, as specified.
- Compensator: `RecordCraftManifest` has no `case` arm in
  `CompensateFailedStep`'s per-action switch (`compensator.go:495-539`), so a
  failure on that step (which can only occur from the payload-type guard
  inside the handler, since the step never dispatches externally) falls to
  the `default:` arm, which marks the step `Pending` and re-persists the
  saga — i.e. a no-op re-queue, not an error path that halts the saga.
  Verified `IncubatorResult` has the identical absence of a `case` (grep
  found none), so this is the established precedent's exact behavior, not a
  new gap introduced by this change.

### 3. The Kafka round trip — PASS

`extractMakerCraftResults` (`producer.go:151-172`) returns
`{"kind": MakerCraftResultKind, "characterId": p.CharacterId, "craftManifest": p}`
— `characterId` is a top-level scalar, `craftManifest` is the payload struct
nested as one object. `TestCompletedCraftSagaEchoesManifest` marshals the
whole `StatusEventCompletedBody` to bytes, unmarshals it back through
`encoding/json` into a generic map (the same untyped path Kafka delivery
takes), asserts `Results["characterId"].(float64)` round-trips to the
expected `uint32`, and separately re-marshals `Results["craftManifest"]` and
unmarshals it into a `CraftManifestPayload`, asserting field equality
against the original including a `Materials` slice entry — proving a nested
list survives the round trip, which a flat/scalar-only shape could not
express. `TestManifestSurvivesSagaRehydration` additionally proves the
payload stays a typed `CraftManifestPayload` (not `map[string]any`) after
the saga itself round-trips through `Step[any].UnmarshalJSON`, the actual
jsonb-rehydration path a pod restart exercises.

### 4. All four modes — PASS

`TestManifestStepIsFirstInEveryMode` is table-driven over modes 1–4 and
asserts `steps[0].Action == RecordCraftManifest` for every one, including
mode 4 (disassemble), which the brief specifically calls out as having no
`AwardCraftedAsset` step. Verified in the actual diff:
`processor.go:169` (createOrUpgrade, modes 1/2), `:254-261` (crystal, mode
3), `:351-358` (disassemble, mode 4) — each prepends `record_manifest`
before any other `AddStep` call in its function.
`TestModeOneAndTwoManifestsDifferOnlyByMode` independently proves modes 1
and 2 build byte-identical manifests except for `Mode`, which is the exact
justification (derivation §2 F2) for choosing this carrier over deriving
`nMode` from the saga.

### 5. FR-3.2 (unheld gem dropped) — PASS

`craftManifest` (`processor.go:397-414`) derives `GemItemIds` from
`AppliedGems(snap, gemItemIds)` — the same hold-filtered, deduplicated list
`BuildCreatePlan` itself uses — never from `req.GemItemIds` directly.
`craftManifestCatalyst` (`:441-449`) reports a catalyst only when the
`Plan` actually holds a `RoleCatalyst` consumption, matching the fact that
`BuildCreatePlan` resolves none when unheld. `TestUnheldGemIsDroppedFromManifest`
covers both the fully-unheld-gem case and the duplicate-named/held-once case
(`gemFrequency` in `plan.go`), and asserts the craft is not rejected (no
`require.Error`). `TestUnheldCatalystLeavesCatalystUnused` covers the
catalyst symmetrically.

### 6. `DisassembleMesoCharge == 0` and `noItemAwarded == false` — correctly not flagged

`processor.go:355` sets `MesoCost: uint32(DisassembleMesoCharge)` (the named
constant, not a literal `0`), and
`TestDisassembleManifestCarriesCrystalsAndZeroMeso` asserts against
`craft.DisassembleMesoCharge` rather than a literal, so a future retune
would not silently pass. `NoItemAwarded` has no setter anywhere in the diff
and defaults to Go's zero value (`false`) in every manifest constructed —
consistent with the addendum's U3 correction; no test asserts a `true` case,
which is correct given nothing on this branch can produce one.

### 7. No `atlas-channel` file touched — PASS

`git diff --stat 331b0c0b5..61ff8cbd8 -- services/atlas-channel` is empty.

## Other checks performed

- `CompensableActions` (maker, `processor.go:86-91`) does **not** include
  `RecordCraftManifest` — correct, since the map's own doc comment scopes it
  to actions that commit a mutation. `TestEveryStepUsesACompensableAction`
  explicitly exempts `RecordCraftManifest` with a comment
  (`processor_test.go:751-756`) rather than weakening the assertion for any
  other action — verified the diff adds only that one `continue`, nothing
  else in the loop changed.
- `TestEveryRejectionStillEmitsNoSaga` confirms a rejected craft (level too
  low) never emits any saga, including the manifest step
  (`d.em.calls` empty).
- `TestCreateManifestMatchesActualConsumption` cross-checks the manifest's
  aggregated `Materials` against the sum of the actual
  `DestroyAssetFromSlot` steps' quantities by template id — a genuine
  consistency check, not a restatement of the production code.
- `TestCrystalManifestCarriesDrawnRewardAndLeftover` derives its expected
  `CrystalItemId` from the saga's own `award_crystal`/`AwardAsset` step
  rather than a hardcoded literal, and asserts `Materials` is empty for mode
  3 per derivation §5.
- Payload doc comments (`payloads.go:1104-1122`) correctly explain the
  no-`omitempty` rule for `MesoCost`/`CatalystUsed`/`NoItemAwarded`, matching
  the `AwardCraftedAssetPayload` precedent cited in the brief.

## Not evaluable

None — the unit's full surface (all three touched modules, the seam into
`timer.go`/`compensator.go` call sites, and the Kafka round-trip contract
Task 26 depends on) was traced and verified against real test execution and
lint output, not from the implementer's report alone.

## Non-blocking notes

None.
