# Shared Script Operation Implementations — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-04
Source: `docs/architectural-improvements.md` item M3
---

## 1. Overview

Four services evaluate rule- or state-machine-driven scripts and turn the resulting
operations into saga steps: `atlas-npc-conversations`, `atlas-portal-actions`,
`atlas-reactor-actions`, and `atlas-map-actions`. All four already share the
condition/operation data model, the arithmetic evaluator, and the context replacer via
`libs/atlas-script-core`. What they do **not** share is the operation *implementations*.
Each service hand-rolls its own executor, and the same operation is implemented up to
four separate times:

| Saga action | Implemented in |
|---|---|
| `SendMessage` | map (`drop_message`), reactor (`drop_message`), portal (`drop_message`), npc (`send_message`) |
| `SpawnMonster` | map, reactor, npc |
| `MoveEnvironment` | map, reactor |
| `ResetEnvironment` | map, reactor |
| `ShowIntro` | map, npc |
| `StageClearAttemptPq` | reactor (`stage_clear_attempt`), npc (`stage_clear_attempt_pq`) |
| `PlayPortalSound` | portal, npc |
| `CreateSkill` | portal, npc |
| `UpdateSkill` | portal, npc |
| `StartInstanceTransport` | portal, npc |
| `ApplyConsumableEffect` | portal, npc |
| `SaveLocation` | portal, npc |
| `WarpToSavedLocation` | portal, npc |
| `StartQuest` | portal, npc |
| `ShowHint` | portal, npc |
| `WarpToPortal` | portal (`warp`), npc (`warp_to_map`) |

Sixteen saga actions have more than one implementation. Because the link between the
copies is textual rather than compile-time, they have already drifted:
`drop_message` reads the param `messageType` in map-actions but `type` in
reactor-actions (with a `"5"→PINK_TEXT` / `"6"→BLUE_TEXT` numeric mapping that the other
services don't have); `spawn_monster` hard-errors on a bad `x`/`y`/`count` in
map-actions but silently swallows the parse error and keeps the default in
reactor-actions; map-actions never populates `Instance` or `Team` on the
`SpawnMonsterPayload` while reactor-actions and npc-conversations do.

This task extracts the shared operation *implementations* into a new package under
`libs/atlas-script-core`, converges the drifted semantics onto a single strict superset,
and rewires all four executors to delegate. It does **not** merge the services: their
trigger surfaces genuinely differ (a multi-step conversation backed by Redis state vs.
single-shot rule evaluation), and their dispatch and lifecycle concerns stay local.

## 2. Goals

Primary goals:

- One implementation per shared operation, referenced by all four services, so a
  parameter or payload change is a single edit with a compile-time link to every caller.
- Converge the four `SendMessage` param contracts and the three `SpawnMonster`
  implementations onto one documented, strict contract.
- Give the shared implementations a testable, I/O-free shape so each operation is
  covered by table-driven unit tests in the library rather than four sets of
  service-level tests.
- Preserve every service's existing dispatch behaviour that is genuinely local:
  portal's `opClassMoving` classification and `movedCharacter` return, npc's batched
  single-saga assembly, reactor's `ReactorContext` positional defaults.

Non-goals:

- Merging any of the four services, or introducing a new "scripting" service.
- Moving dialogue-specific or cosmetic npc operations (hair/face/skin generation, pet
  evolution, storage/Duey, `select_random_*`, quest progress reads) into the library.
- Moving operations implemented in exactly one service (`field_effect`, `lock_ui`,
  `unlock_ui`, `block_portal`, `cancel_consumable_effect`, `hit_reactor`,
  `broadcast_pq_message`, `update_pq_state`, `drop_items`, `spray_items`).
- Changing the on-disk JSON rule/conversation schemas, the seeder data, or the
  `atlas-ui` script editors.
- Implementing the stubbed reactor operations tracked at `docs/TODO.md:292-294`
  (`weaken_area_boss`, `kill_all_monsters`, environment saga actions). Those keep their
  current behaviour; if `move_environment`/`reset_environment` move to the library, the
  TODO lines must be re-pointed at the new location rather than deleted.
- Changing anything in `atlas-saga-orchestrator`. The payloads on the wire are
  unchanged except where §4.3 explicitly converges a previously-omitted field.

## 3. User Stories

- As a backend engineer adding a parameter to `spawn_monster`, I want to change one
  function and have the compiler point me at every service that must adapt, so that I
  cannot ship a partial three-service edit.
- As a content author writing a reactor rule, I want `drop_message` to accept the same
  parameters it accepts in a map rule, so that I do not have to remember which service
  the rule will run in.
- As a reviewer, I want a shared operation's behaviour asserted once in a library unit
  test, so that a service-level test regression cannot silently diverge one copy.
- As an on-call engineer, I want a malformed script parameter to fail loudly with the
  operation name and the offending value, so that a broken rule is diagnosable from
  logs instead of silently producing a monster at (0,0).

## 4. Functional Requirements

### 4.1 Shared operations package

- **FR-1.** A new package `libs/atlas-script-core/ops` holds one exported function per
  shared operation. All four services import it.
- **FR-2.** Each shared operation is a **pure step-builder**: it takes the operation's
  parameters, a value resolver, and a target context, and returns a saga step
  descriptor plus an error. It performs no network, Redis, Kafka, or REST I/O, and it
  does not call `sagaP.Create`.
- **FR-3.** The returned value carries everything a caller needs to either create a
  single-step saga or append to a batch: step id, status, saga action, and payload.
  Step-id composition is caller-controlled (see FR-8).
- **FR-4.** The package exposes a `Resolver` abstraction for parameter values.
  `atlas-npc-conversations` supplies a resolver backed by its Redis conversation
  context (today's `evaluateContextValue` / `evaluateContextValueAsInt`,
  `operation_executor.go:99,179`). The other three services supply a resolver that
  performs the existing direct parse via
  `libs/atlas-script-core/context.EvaluateValueAsInt` semantics. Shared operations
  never call `strconv` on a raw param directly.
- **FR-5.** The package exposes a `Target` abstraction carrying the field coordinates
  a shared operation needs: world, channel, map, instance, and an optional
  position (`x`, `y`) used as the default when the operation's params omit it.
  `atlas-reactor-actions` populates the position from `ReactorContext.X/Y`;
  `atlas-map-actions` and `atlas-portal-actions` leave it unset;
  `atlas-npc-conversations` leaves it unset.
- **FR-6.** Every shared operation's parameter contract — required params, optional
  params, defaults, accepted aliases — is documented as a doc comment on its exported
  function. The doc comment is the authoritative contract; the existing inline comments
  in `operation_executor.go` are moved there, not duplicated.

### 4.2 Executor delegation

- **FR-7.** Each service's executor keeps ownership of: dispatch (its `switch` or
  `opTable`), operation classification, saga creation/batching, and its
  service-local operations. Its handler for a shared operation reduces to building the
  `Target`, calling the shared function, and handling the returned step.
- **FR-8.** Each service keeps its current `initiatedBy` string and step-id format.
  These are caller-supplied, not baked into the shared function, because they are used
  for saga attribution: `map-action-*`, `reactor-action-*`, and the npc conversation's
  existing `stepId`.
- **FR-9.** `atlas-portal-actions` keeps `opTable` as its dispatch and classification
  authority, including `opClassMoving` and the `movedCharacter` contract documented at
  `executor.go:73-86` (task-184). Delegating an op's body to the library must not change
  which ops are classified `opClassMoving`.
- **FR-10.** `atlas-npc-conversations` keeps its `(stepId, status, action, payload,
  error)` return shape and its batching of many operations into one saga. Its handler
  for a shared operation returns the library's step fields directly.
- **FR-11.** `atlas-npc-conversations`'s local `saga` package
  (`services/atlas-npc-conversations/atlas.com/npc/saga/`), currently a re-export shim
  over `libs/atlas-saga` (`model.go:9`, `builder.go:7`), is replaced by direct imports of
  `libs/atlas-saga`. Its genuine local members (`processor.go`, `producer.go`) stay.
  Rationale: CLAUDE.md — "Prefer straightforward moves over re-exported type aliases."
- **FR-12.** After delegation, no shared operation retains a second implementation
  anywhere under `services/`. This is verifiable: no `saga.<Action>` payload literal for
  any of the sixteen actions in §1 may be constructed outside `libs/atlas-script-core/ops`.

### 4.3 Semantic convergence

Divergences are resolved onto a **strict superset**: the union of accepted inputs, the
strictest error handling, and the fullest payload.

- **FR-13. `drop_message` / `send_message`.** One shared implementation. Accepts both
  `messageType` (map, portal, npc) and `type` (reactor) as the message-type key;
  `messageType` wins if both are present. Retains reactor's numeric mapping
  (`"5"→PINK_TEXT`, `"6"→BLUE_TEXT`) for both keys. Defaults to `PINK_TEXT` when absent
  — except that npc's `send_message` currently *requires* `messageType`; the converged
  op defaults instead of erroring, which is a relaxation for npc only.
  `message` remains required in all services.
- **FR-14.** The two operation names `drop_message` and `send_message` both remain
  valid dispatch keys and both route to the same shared implementation. No on-disk
  script is renamed.
- **FR-15. `spawn_monster`.** One shared implementation. `monsterId` required.
  `x`/`y` optional, defaulting to `Target`'s position when set, otherwise `0` — this
  relaxes npc, which currently requires them. `mapId` optional, defaulting to the
  target's map. `count` optional, default `1`. `team` optional, default `0`.
  All parse failures are **hard errors** naming the operation, the param, and the
  offending value; reactor's swallow-and-default behaviour is removed.
- **FR-16.** The `SpawnMonsterPayload` produced by the shared implementation always
  populates `Instance` (from the target) and `Team`. map-actions previously hard-coded
  `uuid.Nil` for `Instance` and omitted `Team`; after this change a map-action spawn in
  an instanced field spawns into that instance rather than the base map. This is a
  deliberate behaviour fix, not a regression.
- **FR-17. `stage_clear_attempt` vs `stage_clear_attempt_pq`.** Both names remain valid
  dispatch keys routing to one shared `StageClearAttemptPq` implementation, with the
  union of the two current param contracts.
- **FR-18. `warp` vs `warp_to_map`.** Both names remain valid dispatch keys routing to
  one shared `WarpToPortal` implementation. `warp` stays `opClassMoving` in portal's
  `opTable`.
- **FR-19.** Every other shared operation converges on the union of accepted params and
  the strictest error handling, following the same rule. Where a convergence changes
  observable behaviour, the change is listed in the design doc's convergence table with
  the before/after for each service.
- **FR-20. Data sweep.** Before the convergence lands, the seeded script data under the
  repo's rule/conversation JSON must be swept for values that currently rely on a
  silently-swallowed parse failure or on a now-stricter contract. Any script the sweep
  finds is fixed in the same change. A spot-check does not satisfy this requirement.

## 5. API Surface

No REST endpoint, Kafka topic, or message schema changes.

The affected "API" is the Go package surface of `libs/atlas-script-core/ops`, consumed
only in-repo. Its exact signatures are the design phase's decision; the PRD fixes only
the contract shape:

```
// Illustrative shape only — exact signatures are decided in design.
package ops

type Resolver interface {
    String(characterId uint32, paramName string, raw string) (string, error)
    Int(characterId uint32, paramName string, raw string) (int, error)
}

type Target struct { /* world, channel, map, instance, optional x/y */ }

type Step struct { /* status, action, payload */ }

func SpawnMonster(params map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
```

`libs/atlas-script-core` gains a dependency on `libs/atlas-saga` (for the payload types
and action constants). All four services already require both modules, so no service
`go.mod` gains a new module edge — but the library's own `go.mod` and the repo's
`replace` wiring must be updated, and the design phase must confirm this introduces no
import cycle (`libs/atlas-saga` must not depend on `libs/atlas-script-core`).

## 6. Data Model

No database entities, columns, or migrations change. No on-disk script schema changes.

The in-memory model changes are confined to the new `ops` package types (`Resolver`,
`Target`, `Step`) and the removal of the npc-conversations saga re-export shim
(FR-11), which is a type-identity change only — the underlying `libs/atlas-saga` types
are already the same types.

## 7. Service Impact

**`libs/atlas-script-core`** — new `ops` package with the sixteen shared operation
implementations and their unit tests; `go.mod` gains `libs/atlas-saga`.

**`libs/atlas-saga`** — no change expected. If a payload field must be added to express
a converged contract, that is a design-phase finding and must be called out explicitly.

**`services/atlas-map-actions`** — `script/executor.go` (323 lines) delegates
`drop_message`, `spawn_monster`, `show_intro`, `move_environment`, `reset_environment`.
Keeps `field_effect`, `lock_ui`, `unlock_ui`. Behaviour change: `Instance`/`Team` now
populated on spawns (FR-16); `type` accepted as a `drop_message` alias (FR-13).

**`services/atlas-reactor-actions`** — `script/executor.go` (566 lines) delegates
`drop_message`, `spawn_monster`, `move_environment`, `reset_environment`,
`stage_clear_attempt`. Keeps `hit_reactor`, `drop_items`, `spray_items`,
`broadcast_pq_message`, `update_pq_state`, `kill_all_monsters`, `weaken_area_boss`.
Behaviour change: spawn parse failures now error instead of defaulting (FR-15).

**`services/atlas-portal-actions`** — `script/executor.go` (691 lines) and
`script/optable.go` delegate `play_portal_sound`, `warp`, `drop_message`, `show_hint`,
`create_skill`, `update_skill`, `start_instance_transport`, `apply_consumable_effect`,
`save_location`, `warp_to_saved_location`, `start_quest`. Keeps `block_portal` and
`cancel_consumable_effect`. `opTable` and `IsMovingOperation` are unchanged in
structure and classification.

**`services/atlas-npc-conversations`** — the largest change.
`conversation/operation_executor.go` (2,738 lines) delegates its sixteen shared cases
and drops the corresponding inline bodies; supplies a Redis-context-backed `Resolver`;
replaces the `saga` re-export shim (FR-11). All dialogue, cosmetic, quest-progress,
pet, storage, and award operations stay local and unchanged.

**`atlas-saga-orchestrator`** — no code change, but it is the consumer of every payload
touched here. The cross-service seam (CLAUDE.md: "trace the event into its consumers by
hand") must be traced for `SpawnMonster` specifically, since FR-16 changes a field that
was previously always `uuid.Nil` from map-actions.

**`atlas-ui`** — no change. Script editors write the JSON; op names are unchanged and
new param aliases are additive.

## 8. Non-Functional Requirements

- **Performance.** Step construction is in-process and allocation-bound; no new I/O is
  introduced on any operation path. The npc resolver's Redis reads are the same reads
  the inline code performs today, at the same call sites.
- **Multi-tenancy.** Shared operations must not read tenant state. Tenant resolution
  stays in the service layer (npc's `tenant.MustFromContext`); anything tenant-scoped
  reaches the shared op through the injected `Resolver`, never through a package-level
  or ambient lookup.
- **Observability.** Each service keeps its own debug logging at the dispatch site, with
  its own contextual detail (reactor logs the classification, map logs the map id).
  Shared operations do not log; they return errors. Error text includes the operation
  name, the parameter name, and the offending value.
- **Security.** No new external input path. Convergence is strictly more validating,
  not less, except for the two documented relaxations (FR-13 npc `messageType` default,
  FR-15 npc `x`/`y` default).
- **Testing.** Every shared operation gets table-driven unit tests in
  `libs/atlas-script-core/ops` covering: required-param-missing, each optional-param
  default, each parse failure, each accepted alias, and the resulting payload. Existing
  service-level executor tests (`executor_test.go` in each of the four) are kept and
  must still pass; where a test asserted a now-converged behaviour, it is updated with
  the change noted in the commit.
- **Verification.** Flagless `tools/verify.sh` must exit 0 before the branch is
  considered done (CLAUDE.md "Done means verified").

## 9. Open Questions

- **OQ-1.** Package name and placement: `libs/atlas-script-core/ops` vs. a sibling
  module `libs/atlas-script-ops`. A sibling avoids adding an `atlas-saga` dependency to
  `atlas-script-core`, which currently has a near-empty dependency set (`go.sum` is
  163 bytes). Design phase decides; the sibling option should be seriously weighed if
  the dependency would make `atlas-script-core` unusable anywhere it is used today.
- **OQ-2.** Whether `Step` should be `libs/atlas-saga`'s own step type directly rather
  than a new type in `ops`. Reusing it removes a conversion but hard-couples the
  signature to the saga library.
- **OQ-3.** FR-16 changes map-actions spawns to carry a real `Instance`. Is there a
  seeded map-action script that spawns in an instanced field and depends on the current
  base-map behaviour? The FR-20 sweep must answer this before the change lands.
- **OQ-4.** Does the `Resolver` need a per-operation escape hatch for npc's
  arithmetic-expression params, or does `context.EvaluateArithmeticExpression` cover
  every case the inline npc code handles? Design phase must read
  `operation_executor.go:99-230` in full before deciding.
- **OQ-5.** Whether reactor's `stage_clear_attempt` and npc's `stage_clear_attempt_pq`
  are genuinely the same operation or differ in payload. Confirm against both bodies
  before applying FR-17; if they differ materially, drop FR-17 and leave them separate.

## 10. Acceptance Criteria

- [ ] `libs/atlas-script-core/ops` (or the OQ-1 sibling) exists, with one exported
      step-builder per shared operation from the §1 table, each with a doc-comment
      parameter contract.
- [ ] No shared operation function performs network, Redis, Kafka, or REST I/O, and
      none calls a saga processor.
- [ ] All four executors delegate their shared operations; no `saga.<Action>` payload
      literal for any of the sixteen §1 actions is constructed under `services/`.
- [ ] `atlas-portal-actions` `opTable` retains every current entry and every current
      `opClassMoving` classification; `optable_test.go` passes unchanged.
- [ ] `atlas-npc-conversations` still batches multiple operations into one saga, and
      its `(stepId, status, action, payload, error)` handler shape is intact.
- [ ] `services/atlas-npc-conversations/atlas.com/npc/saga/model.go` and `builder.go`
      no longer re-export `libs/atlas-saga` types; call sites import the library
      directly.
- [ ] `drop_message` and `send_message` both dispatch to one implementation that accepts
      `messageType` and `type`, applies the `"5"`/`"6"` numeric mapping, and defaults to
      `PINK_TEXT`.
- [ ] `spawn_monster` hard-errors on any unparseable `x`, `y`, `count`, `mapId`, `team`,
      or `monsterId`, with the operation name and offending value in the message.
- [ ] Every `SpawnMonsterPayload` built by the shared implementation carries `Instance`
      and `Team`, including on the map-actions path.
- [ ] A convergence table exists in the design doc listing every behaviour change with
      the before/after per service.
- [ ] The FR-20 data sweep is complete and its result recorded: either "no seeded script
      affected" with the search that establishes it, or the list of scripts fixed.
- [ ] Table-driven unit tests in the ops package cover, per operation:
      missing-required-param, each optional default, each parse failure, each alias.
- [ ] The four services' existing executor tests pass; any updated assertion is
      accompanied by a note tying it to the FR that changed it.
- [ ] The `SpawnMonster` seam into `atlas-saga-orchestrator` is traced by hand and a
      test asserts the instance-carrying contract.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `docs/TODO.md:292-294` re-pointed at the new locations for any operation that moved.
