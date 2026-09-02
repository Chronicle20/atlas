# Cosmic Map-Action Parity — Design

Task: task-290-cosmic-map-action-parity
Input: [prd.md](prd.md) (approved)
Status: Draft for review
Created: 2026-09-02

---

## 0. How to read this document

The PRD is the requirements contract. This document does three things it does not:

1. **Corrects the PRD where repository evidence contradicts it.** §1 lists seven findings
   verified against source at file:line. Three of them change the shape of the work
   materially — G6 is not "schema only", `transportAvailable` does not express Cosmic's
   `docked`, and there is a fifth pre-existing defect the PRD did not find.
2. **Decides the architecture.** §2 is ten numbered decisions, each with the alternatives
   considered and the recommendation. §3–§6 are the resulting per-layer design.
3. **Says what planning must still resolve.** §9 is the residual unknown list, reduced
   from the PRD's six open questions to two genuine ones plus a per-script derivation
   step planning owns.

Nothing here is implementation. Anything stated as fact carries a file:line citation;
anything not verified is labelled as such.

---

## 1. Verified findings that amend the PRD

### F1 — The condition wire format truncates. G6 is not schema-only, and `in` is unreachable.

The map-action document's condition travels through four representations:

| Layer | Type | Fields carried |
|---|---|---|
| Seed JSON → REST | `script.RestConditionModel` (`services/atlas-map-actions/atlas.com/map-actions/script/rest.go:34-39`) | `type`, `operator`, `value`, `referenceId` |
| Domain | `condition.Model` (`libs/atlas-script-core/condition/model.go:8-17`) | + `step`, `worldId`, `channelId`, `includeEquipped` |
| Aggregator call | `validation.ConditionInput` (`services/atlas-map-actions/atlas.com/map-actions/validation/model.go:9-18`) | `type`, `operator`, `value`, `referenceId`, `step`, `worldId`, `channelId`, `includeEquipped` |
| Canonical wire | `saga.ValidationConditionInput` (`libs/atlas-saga/validation.go:64-74`) | all of the above **plus `Values []int`** |

Two independent gaps:

- **`step` is dropped twice.** `RestConditionModel` has no `step` field at all, so a seed
  cannot express one; and `evaluator.go:81-86` constructs `ConditionInput` with only
  `Type`/`Operator`/`Value`/`ReferenceId`, discarding `step` even if the domain model
  carried it. The aggregator **rejects** `questProgress` without a step —
  `rest.go:238-245` returns `"step is required for quest progress conditions"`, and
  `builder.go:324-330` sets the same error. **G6 therefore requires plumbing work at two
  layers, not a schema enum entry.** The PRD's table (§4.3, "G6 — schema enum only") is
  wrong.

- **`Values` does not exist below `libs/atlas-saga`.** The aggregator supports `in`
  (`validation/model.go:74`, `builder.go:243-244,290-291`) and *requires* a non-empty
  `values` array for it (`rest.go:221-223`). Neither `condition.Model` nor
  `RestConditionModel` nor map-actions' `ConditionInput` has such a field. Adding `in`
  to the schema operator enum per FR-1.2 without plumbing `values` produces a
  schema-valid document that fails at the aggregator with
  `"'in' operator requires 'values' array"` — the exact class of defect FR-1.1 exists to
  eliminate.

- Additionally, `evaluator.go:71-74` does `strconv.Atoi(cond.Value())` and errors on
  failure, so any non-integer condition value is unrepresentable today.

### F2 — `transportAvailable` cannot express Cosmic's `docked`. (PRD open question 2, resolved.)

`transportAvailable` evaluates `ctx.GetTransportState(referenceId) == "open_entry"`
(`services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go:525-534`).
The route state machine has **five** states
(`services/atlas-transports/atlas.com/transports/transport/state.go:8-20`):
`out_of_service`, `awaiting_return`, `open_entry`, `locked_entry`, `in_transit`.

The condition collapses those five to a boolean at `open_entry`. Cosmic's
`em.getProperty("docked") == "false"` gates *warping a lingering player into the
en-route map*. The correct predicate for that is `state == in_transit` (arguably
`locked_entry` or `in_transit`), **not** `!open_entry` — because `!open_entry` is also
true for `out_of_service` and `awaiting_return`, which would warp players aboard a
vessel that is not sailing. `transportAvailable` is genuinely insufficient. See D8.

### F3 — Fifth pre-existing defect: `spawn_monster` ignores the field instance.

`executor.go:174` hard-codes `Instance: uuid.Nil` in `SpawnMonsterPayload`, discarding
`f.Instance()`, which the field model exposes (`libs/atlas-constants/field/model.go:41`)
and which the orchestrator threads straight into the atlas-monsters URL
(`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/monster/requests.go:12,25`:
`worlds/%d/channels/%d/maps/%d/instances/%s/monsters`). On any instanced map the spawn
lands in the non-instanced field instead of the caller's instance.

This is invisible today because all nine seeded scripts run on non-instanced maps. It
becomes live the moment this task lands the G5 party-quest set (`922000000`,
`926000000`, `926000010`, `926120300`), which are instanced. **It is defect D5 and must
be fixed alongside FR-1.1–FR-1.4.** The PRD's defect list is four items; it is five.

### F4 — A version-agnostic seed root already exists, and map-actions is not using it.

`seeder.NewFilesystemCatalogSourceWithShared(envVar, fallbackRoot, sharedRel)`
(`libs/atlas-seeder/catalog.go:37-49`) returns roots as
`[<base>/<sharedRel>, <base>/<region>/<major>_<minor>]`, and `Seed`/`ReadStatus` merge
across roots with **later (more specific) roots winning on a relative-path collision**
(`catalog.go:41-44`, `catalog.go:73-80`). `deploy/seed/shared/all/` exists today with
`CATALOG_REVISION`, `events`, `instance-routes`, `routes`, `vessels`. Two services
already consume it: `services/atlas-events/.../definition/subdomain.go:106` and
`services/atlas-tenants/.../seed/resource.go:22`.

`atlas-map-actions` uses the single-root constructor
(`services/atlas-map-actions/atlas.com/map-actions/script/groups.go:16`). Switching it is
a one-line change that turns the PRD's 891-file outcome (81 × 11) into 81 files, and
turns FR-1.6's replication guard into a guard against something that can no longer
happen. See D1.

### F5 — `tools/catalog-lint` already models the map-action subdomains, and runs only in CI.

`tools/catalog-lint/subdomains.go:18-19` declares both
`map-actions/onUserEnter` and `map-actions/onFirstUserEnter` with `typ: "map-action"` and
the `^map-(.+)\.json$` filename pattern, and `main.go:50-80` already walks every seed
JSON and validates envelope type and filename-vs-id agreement. It is invoked from
`.github/workflows/catalog-lint.yml:35` and **is absent from `tools/verify.sh`** (no
`catalog-lint` match in that file). This is the natural home for the structural seed
checks, and wiring it into `verify.sh` closes a script/CI divergence that already exists.

### F6 — The `spawnIfAbsent` guard has an atomic home; the orchestrator is not it.

atlas-monsters exposes `GET .../maps/{mapId}/instances/{instanceId}/monsters`,
`DELETE` on the same collection, and the `POST` the orchestrator already calls
(`services/atlas-monsters/atlas.com/monsters/world/resource.go:35-38`). A guard
implemented in the orchestrator would be a cross-service read-then-write with a TOCTOU
window that two simultaneous map entries would both pass. atlas-monsters can decide
against its own registry inside the create path. See D5.

### F7 — Hook type is directory-derived; seed JSON is sorted and 2-space-indented.

Seed attributes carry `scriptName`, `description`, `rules` and **no `scriptType`**
(`deploy/seed/gms/83_1/map-actions/onUserEnter/map-goArcher.json`), although the domain
and REST models have one (`script/rest.go:22`). The hook comes from the subdomain path
(`script/subdomain_on_user_enter.go:24`, `script/subdomain_on_first_user_enter.go:23`).
Existing seeds are 2-space-indented with alphabetically ordered keys at every level.
Both facts belong in the corrected `/convert-map` contract (FR-1.3), which the PRD's
example envelope does not state.

---

## 2. Architecture decisions

### D1 — Seed layout: move map-actions to the shared root. **(Recommended: shared root.)**

| Option | Outcome |
|---|---|
| **A. `deploy/seed/shared/all/map-actions/` (recommend)** | 81 files. One edit per script. Replication cannot drift, so FR-1.6 becomes structurally unnecessary. Version-specific override still available by dropping a file at the version path — the merge rule already prefers it (F4). |
| B. 11× replication as the PRD specifies | 891 files. Every future edit is an 11-way fan-out. Needs a bespoke byte-identity guard (FR-1.6) whose only job is to catch a mistake option A cannot make. |
| C. Generate the 11 copies from one source at build time | Keeps the on-disk layout but adds a generator + `--check` step; strictly more machinery than A for the same result. |

Recommend **A**. It satisfies FR-6.1's *intent* (one behavior, every version) by
construction rather than by inspection, and it is a precedented pattern in this repo, not
a new one (F4). The migration moves the existing 99 files down to 9 and changes
`groups.go:16` to the shared constructor.

Two consequences to accept explicitly:

- The library's own doc comment warns that adding a shared root changes the service's
  composite `CATALOG_REVISION` (a `"+"`-join across roots, `libs/atlas-seeder/seed.go:143-155`)
  and triggers a **one-time spurious "seed catalog drift detected" warning** on first
  rollout. That is a log line, not a failure.
- FR-1.6 is not dropped, it is **restated**: catalog-lint gains a check that *no*
  map-action document exists under a version root unless a sibling comment records a
  deliberate version override. That turns "did you copy it 11 times" into "did you mean
  to fork this one" — the failure mode that survives option A.

**If the reviewer rejects D1**, everything else in this design stands unchanged; FR-1.6
reverts to a byte-identity check in catalog-lint over the 11 roots, and the conversion
step writes 11 copies. The decision is isolated to the seed layer.

### D2 — Condition plumbing: widen the condition contract once, for `step` and `values`. **(Recommended.)**

Per F1, `questProgress` (G6) and `in` (G6b, FR-1.2) are both blocked below the schema.
Three options:

- **A (recommend): add `step` and `values` to `RestConditionModel` and to map-actions'
  `ConditionInput`, and pass all four extra fields through `evaluator.go`.**
  `condition.Model` already carries `step`; it needs `values` added alongside. This makes
  the map-action condition contract a strict subset of `saga.ValidationConditionInput`
  and removes the truncation class of bug permanently. Cost: touches
  `libs/atlas-script-core`, which portal-actions, reactor-actions and npc-conversations
  also consume — additive fields only, so no consumer changes.
- B. Add only `step`; express `isCygnus()` as N sibling rules (one `jobId =` per Cygnus
  job) instead of one `in`. Cheaper, but multiplies aggregator round-trips per entry and
  leaves FR-1.2's `in` in the schema as a lie.
- C. Leave the contract alone and drop G6 and G6b from scope. Rejected: both are
  explicitly in the PRD, and the blocker is a field we can add ourselves.

Recommend **A**. Also fold in F1's third point: `evaluator.go` must stop assuming
integer values — parse per condition type rather than eagerly `Atoi`-ing.

`worldId`/`channelId` are already in `ConditionInput` and are needed by `mapCapacity`;
since we are opening this seam, plumb them from the field model rather than leaving two
more silently-zero fields behind.

### D3 — Unknown operations must fail. **(Recommended: error, with a parity guard.)** (PRD open question 4, resolved.)

`executor.go:47-49` logs and returns `nil` on an unknown operation; `evaluator.go:35-36`
routes *every* unknown condition to the aggregator, which at least rejects it. The
asymmetry is the problem: a typo'd operation is silently a no-op forever.

Recommend: **return an error from the `default:` arm.** The PRD's stated risk — that a
tenant seed carrying an unknown operation would start failing — is real but bounded, and
it is mitigated, not created, by the change: the operation was never doing anything.
Failing loudly surfaces it at the first map entry instead of never.

The change is only safe paired with a **schema/executor parity check** (D4) so a seed
cannot be authored with an operation the executor lacks. Ship them together; neither
alone is correct.

Note the blast radius honestly: `ExecuteOperations` (`executor.go:54-61`) aborts the
whole rule's remaining operations on the first error. An unknown operation mid-rule will
now suppress the operations after it rather than just itself. That is the intended
behavior (a partially-applied cutscene is worse than a loud failure), but it should be
stated in the executor's doc comment.

### D4 — Schema/Go drift: generate, don't mirror. **(Recommended: generator + `--check`.)**

FR-1.5 asks for a check that fails when the schema's condition enum and
`libs/atlas-saga/validation.go` diverge. Two established repo patterns:

- **Mirror guard** (`tools/npc-conversation-contract-mirror-guard.sh`) — diffs two
  hand-maintained copies. Correct, but leaves both copies hand-edited.
- **Generator + `--check`** (`tools/gen-topics.sh`, `gen-routes.sh`, `gen-tenant-tables.sh`,
  each wired into `verify.sh` as a `--check` step) — one source of truth, the artifact is
  regenerated and diffed.

Recommend the **generator** pattern: a `tools/gen-map-action-schema.sh` that emits the
condition-type enum from `libs/atlas-saga/validation.go`'s constants (plus the locally
evaluated `map_id`), the operator enum from the same file's operator constants (which
gives `in` for free, FR-1.2), and the operation enum from the executor's switch arms
(which gives D3's parity check for free). `--check` mode fails on drift. Three
hand-maintained lists collapse to zero.

The per-operation `allOf` param blocks stay hand-written — they are documentation of
intent, not derivable from Go — but the *set* of operations is generated, so a new
executor arm without a schema block is a lint failure rather than a silent hole.

### D5 — `spawnIfAbsent` belongs in atlas-monsters. **(Recommended.)**

Per F6. The flag rides `SpawnMonsterPayload` → orchestrator → the existing
`POST .../monsters` body, and atlas-monsters decides against its own field registry
before creating. Alternative (orchestrator does `GET` then `POST`) is a cross-service
TOCTOU: two players entering the map in the same tick both read "absent" and both spawn.

Per the project's cross-service seam rule, the acceptance test lives in **atlas-monsters**
and asserts the new contract (`spawnIfAbsent: true` + monster present ⇒ no create), with
a second test in map-actions asserting the param reaches the payload. A green build in
either service alone does not cover this seam.

The PRD's FR-2.2 (every converted spawn sets the flag) then becomes mechanically
checkable and should be a catalog-lint rule, not a review convention.

### D6 — The direction/cutscene subsystem (G8/G10/G11): discrete actions, not one composite. **(Recommended.)**

Seven-plus primitives (`set_direction`, `set_direction_status`, `set_direction_mode`,
`start_direction`, `lock_ui2`, `play_sound`, `send_direction_info`,
`set_standalone_mode`). Options:

- **A (recommend): one discrete saga `Action` + payload per primitive.** Matches the
  explicit precedent in `libs/atlas-saga/model.go:203-206`, which documents *why* two
  near-identical NPC-conversation actions were kept discrete rather than given a mode
  discriminator: "the orchestrator's handler dispatch is per-action, and a discriminator
  inside the payload would move branching somewhere the compensator and event-acceptance
  tables cannot see it." That reasoning applies verbatim here.
- B. One `play_direction` action with a `mode` discriminator. Fewer constants, but hides
  branching from `event_acceptance.go` and the compensator tables — the thing the repo
  has already decided against.
- C. A `direction` script sub-language inside one operation's params. Rejected: params
  are `map[string]string` by schema (`additionalProperties: {"type": "string"}`), and
  nesting a mini-language inside a string is the worst of both.

Recommend **A**. Each action gets an `event_acceptance.go` entry, a `model.go` unmarshal
case, a `handler.go` arm, and an atlas-channel packet writer. These are the largest
single block of work in the task and the strongest argument for D7's split.

Verification note: these are packet-emitting operations. Per `docs/packets/PROCESS.md`,
any new clientbound write needs its opcode and layout derived from the client, not
assumed. **Planning must treat each new atlas-channel packet as a packet task with its
own derivation step** — this is not "add a writer", and estimating it as such is how the
task overruns.

### D7 — Split into three plans on one branch. **(Recommended.)** (PRD open question 5, resolved.)

The PRD leaves the split to planning. The dependency structure decides it:

- **Plan A — Foundation.** Defects D1–D4 from the PRD plus F3's instance defect;
  D2's condition plumbing; D3's fail-loud executor; D4's schema generator; D1's shared-root
  migration; catalog-lint checks wired into `verify.sh`; `spawnIfAbsent` end to end
  (D5) and the `108010301` retrofit. **Everything downstream depends on this; nothing in
  it depends on anything downstream.**
- **Plan B — Category #1 (27 scripts).** Pure seed authoring on top of Plan A. No engine
  work. Mechanical, highly parallelizable, and the first real exercise of the corrected
  `/convert-map`.
- **Plan C — Engine gaps + category #2 (45 scripts).** G1–G14 grouped so each gap's
  engine work and its dependent scripts land together (a gap with no seeded script is
  unverified code).

One branch, three plan documents, `/execute-task` per plan. Rationale: Plan A is a
correctness gate the other two would silently violate; Plan B is throughput work that
should not wait behind Plan C's cross-service design; Plan C is where the real risk is
and deserves its own review surface. Splitting at the *branch* level instead would force
three PR review cycles over a shared foundation, which is worse.

### D8 — Add a `transportState` condition; keep `transportAvailable`. **(Recommended.)** (PRD open question 2, resolved.)

Per F2. Options:

- **A (recommend): new `transportState` condition** taking the route state as a value,
  compared with `=`/`!=`/`in`. Preserves `transportAvailable`'s existing meaning and
  callers; expresses `docked == false` precisely as `transportState in [locked_entry,
  in_transit]` (or `= in_transit`, pending the per-script derivation in §9). Requires
  D2's `values` plumbing to use `in`, which is another reason to take D2 option A.
  The state must be compared as a **named** value, not a raw ordinal — an integer
  encoding of a five-state string enum is a future renumbering bug. This is the one
  place a non-integer condition value is required, which is exactly what D2's
  "stop eagerly `Atoi`-ing" clause enables.
- B. Redefine `transportAvailable` to mean something else. Rejected: it has existing
  callers and a documented meaning.
- C. Add a boolean `transportDocked`. Rejected: bakes today's question into the contract
  and will need a third condition the next time a script cares about `awaiting_return`.

### D9 — G7 randomization happens in the executor, not the document. **(Recommended.)**

`pepeking_effect` picks one of three monster IDs. Options:

- **A (recommend): `spawn_monster` accepts `monsterIds` (comma-separated) in addition to
  `monsterId`, and the executor picks one uniformly.** No schema type changes (params
  stay `string`), no new operation, no new saga action. One executor branch.
- B. A `random` rule selector in the document model. A far larger change to the rule
  engine for one script.
- C. Three rules with a `random` condition. Requires a stateful/non-deterministic
  condition type, which the aggregator has no concept of.

Recommend **A**, with the constraint that `monsterId` and `monsterIds` are mutually
exclusive and the schema's `allOf` block enforces exactly one.

### D10 — G14 `explorer_quest` and G5's PQ verbs need their Cosmic semantics derived before their contracts are fixed.

These are the two gap groups where the PRD names an operation but not its semantics:

- `explorer_quest` — whether crediting an exploration region is expressible as the
  existing `SetQuestProgress`/`StartQuest` pair against quest 29005–29011, or genuinely
  needs a new action, depends on how Cosmic records the credit. **Design cannot decide
  this without the script body**; planning must read it from issue #1624's inventory and
  the WZ quest data before writing the task. Both outcomes are cheap; guessing is not.
- `reset_pq`, `reset_reactors` (all vs. state-filtered), `shuffle_reactors`,
  `clear_drops`, `count_monster` — each needs its owning service identified from the
  repo (`atlas-party-quests`, `atlas-reactors`, `atlas-drops`, `atlas-monsters`) and its
  existing REST surface checked the way F6 checked atlas-monsters', before a saga action
  is defined. `count_monster` in particular is a *query*, not a mutation, and probably
  belongs as an aggregator **condition**, not a saga operation — the PRD lists it under
  operations, which is likely a miscategorization worth confirming.

This is a deliberate, bounded deferral to planning's per-gap derivation step (§8), not a
blocker: the mechanism for both outcomes is already designed above.

---

## 3. Layer design

### 3.1 Seed layer

```
deploy/seed/shared/all/map-actions/
  onUserEnter/map-<scriptName>.json      (79)
  onFirstUserEnter/map-<scriptName>.json (2)
```

- Envelope: `{"data": {"type": "map-action", "id": "<scriptName>", "attributes": {...}}}`.
- Attributes: `scriptName`, `description`, `rules`. No `scriptType` — the hook is the
  directory (F7).
- Formatting: 2-space indent, keys sorted alphabetically at every level, trailing
  newline. Matching the existing nine exactly is what makes a diff reviewable.
- Migration: `git mv` the existing 9 documents from `gms/83_1` to the shared root, delete
  the other 90 copies, flip `script/groups.go:16` to
  `NewFilesystemCatalogSourceWithShared("SEED_CATALOG_ROOT", "./deploy/seed", "shared/all")`.

### 3.2 Document contract (`libs/atlas-script-core`, map-actions `rest.go`)

`RestConditionModel` gains `step string` and `values []int` (both `omitempty`).
`condition.Model`/`Builder` gain `values`. `script.transformRule` and its inverse carry
them. Nothing else consumes these fields, so the change is additive for portal-actions,
reactor-actions and npc-conversations.

### 3.3 Evaluator

- `map_id` gains `>`, `<`, `>=`, `<=` (G14a) in `evaluateMapId`'s operator switch
  (`evaluator.go:44-53`).
- `evaluateViaQueryAggregator` populates `Step`, `Values`, `WorldId`, `ChannelId` and
  `IncludeEquipped` from the condition and field models instead of dropping them
  (`evaluator.go:81-86`).
- Value parsing moves from an unconditional `Atoi` to per-type handling, so
  `transportState` (D8) can carry a string.

### 3.4 Executor

- `default:` returns an error (D3).
- New arms per gap, each following the existing shape: read params, validate presence,
  build a one-step saga, `sagaP.Create`.
- `executeSpawnMonster` uses `f.Instance()` (F3) and gains `spawnIfAbsent` (D5) and
  `monsterIds` (D9).
- Executor arms and the schema operation enum are kept in lockstep by the generator (D4).

### 3.5 Saga contract (`libs/atlas-saga`)

New `Action` constants + payloads + `unmarshal.go` cases, grouped by gap, discrete per
primitive (D6). Each also needs, in `atlas-saga-orchestrator`: a `model.go` type alias
and unmarshal case, an `event_acceptance.go` entry, and a `handler.go` dispatch arm — the
four-touchpoint pattern visible for `SpawnMonster` at `model.go:158,341,1251`,
`event_acceptance.go:346`, `handler.go:138,919`.

### 3.6 Aggregator

- New `transportState` condition (D8), following `TransportAvailableCondition`'s shape at
  `validation/model.go:525`, `builder.go:335`, `rest.go:271`, `model.go:50`.
- New `areaInfo` condition (G12c), same four touchpoints.
- Both must be added to `builder.go:210`'s accepted-type list, which is a single
  flat `case` — an easy omission that produces a confusing "unsupported condition" at
  runtime.

### 3.7 atlas-channel

Packet writers for the D6 direction primitives. **Per-packet derivation required**; see
D6's verification note.

---

## 4. Gap disposition summary

Amendments to the PRD's §4.3 table are in bold.

| Gap | Disposition |
|---|---|
| G3, G4, G9, G12a, G12b, G13 | Executor arm only; saga action exists. As the PRD states. |
| G6 `questProgress` | **Not schema-only (F1).** Needs `step` plumbed through `RestConditionModel`, `condition.Model` and `evaluator.go`. |
| G6b `isCygnus` | **Needs `values` plumbed (F1)**, then `jobId in [...]`. |
| G14a `map_id` ranges | Evaluator operator switch. As the PRD states. |
| G7 randomized spawn | `monsterIds` param on `spawn_monster` (D9). |
| G1a warp-to-map | New saga action + consumer. Confirmed absent — `WarpToPortal` requires a portal (`payloads.go:41-48`). |
| G1b docked state | **`transportAvailable` insufficient (F2); new `transportState` condition (D8).** |
| G1c music / boat effect | New saga actions + consumers. |
| G2 `spawn_npc` | New saga action + consumer. |
| G5 PQ verbs | New actions; **`count_monster` is probably a condition, not an operation (D10)**; owning services to be confirmed in planning. |
| G8/G10/G11 direction | Discrete actions per primitive (D6); each new packet needs derivation. |
| G12c `area_info` | New aggregator condition. |
| G14b `explorer_quest` | **Mechanism deferred to planning's derivation step (D10)**; may reduce to existing quest actions. |
| — | **New: D5-instance defect (F3), fixed in Plan A.** |

---

## 5. Error handling

- **Unknown operation** → error (D3), aborting the rule's remaining operations.
- **Unknown condition** → unchanged: forwarded to the aggregator, which rejects it. The
  schema generator (D4) makes an unknown condition unauthorable in the first place.
- **Missing required param** → error, as the existing arms already do
  (`executor.go:67-70`).
- **Aggregator rejection** → `EvaluateCondition` already propagates the error
  (`evaluator.go:88-91`); a failed rule does not silently pass.
- **Saga step failure** → the orchestrator's existing compensation path. New actions with
  no meaningful compensation (a cutscene cannot be un-played) declare that explicitly
  rather than registering a no-op compensator by omission.

---

## 6. Verification design

| Requirement | Mechanism |
|---|---|
| FR-1.5 schema↔Go drift | `tools/gen-map-action-schema.sh --check`, wired into `verify.sh` beside the other `--check` generator steps (D4). |
| FR-1.6 replication | Structurally impossible under D1. Replaced by a catalog-lint rule flagging any version-root map-action document as a deliberate-override-or-mistake. |
| FR-3.0 operation parity | Falls out of D4: the operation enum is generated from the executor switch. |
| FR-2.2 every spawn guarded | catalog-lint rule: `spawn_monster` without `spawnIfAbsent` fails (D5). |
| FR-1.4 quest-status shift | Not mechanically checkable (the +1 is invisible post-hoc). Enforced by the `/convert-map` contract and by review. Stated as such rather than pretended otherwise. |
| Cross-service seams | Per D5 and the project rule: a test in the **consumer** service asserting each new contract. |
| Existing gap | `catalog-lint` runs in CI but not `verify.sh` (F5). Add it, gated on `deploy/seed/` and `tools/catalog-lint/`. |

Every seeded document must validate against the generated schema. That check belongs in
catalog-lint too — it already opens and parses every seed file (`main.go:50-80`), so the
marginal cost is one schema load.

---

## 7. Testing strategy

- **Evaluator**: table tests for `map_id` operators; a test asserting `step`, `values`,
  `worldId`, `channelId` survive into `ConditionInput` (the F1 regression).
- **Executor**: per-arm tests asserting the built saga step's action and payload;
  a test asserting an unknown operation errors (D3); a test asserting `f.Instance()`
  reaches the payload (the F3 regression).
- **atlas-monsters**: `spawnIfAbsent` honored — present ⇒ no create, absent ⇒ create (D5).
- **Aggregator**: `transportState` per route state; `areaInfo`.
- **Seeds**: catalog-lint over the whole tree, per §6.
- **Gate**: flagless `tools/verify.sh` exits 0 before the branch is called done, per
  project policy; `--quick` does not count.

---

## 8. Execution order

1. **Plan A — Foundation.** §3.1–§3.4, D1–D5, F3, the `108010301` retrofit, all of §6.
   Nothing may be converted before this lands.
2. **Plan B — Category #1.** 27 documents. Seed-only.
3. **Plan C — Gaps + category #2.** Per gap: derive semantics (D10) → saga action +
   payload + orchestrator touchpoints → consumer + seam test → schema block → convert the
   gap's scripts. A gap is not done until at least one seeded script exercises it.

Within Plan C, order by cost: the executor-arm-only gaps (G3, G4, G9, G12a, G12b, G13)
first — they are the PRD's own material finding and deliver 8 scripts for little work —
then G6/G6b/G7/G14a (already unblocked by Plan A), then the new-action gaps, and G8/G10/G11
last, since D6 makes them packet work.

---

## 9. Residual unknowns

Reduced from the PRD's six. The four resolved above are noted with their resolution.

1. **`cannon_tuto_02` (PRD Q1, FR-3.1).** Referenced by `cannon_tuto_01`; not shipped by
   Cosmic. A genuine external unknown. Must be located in WZ data or the call dropped
   with written rationale before `cannon_tuto_01` converts. Blocks one script, not G8.
2. **`start_map_effect` vs. G1c's boat effect (PRD Q6).** Whether they are the same
   primitive. Resolvable from the Cosmic script bodies during Plan C's derivation step;
   if they are the same, it costs one action instead of two, and category #3's
   `dojang_Msg` gets a head start it is not entitled to.
3. **Per-gap semantic derivation (D10).** `explorer_quest`'s mechanism, the PQ verbs'
   owning services, `count_monster`'s correct category, and `transportState`'s exact
   predicate for the boarding scripts. Each is a read of the Cosmic script plus a repo
   check — producible work, assigned to planning, not deferred indefinitely.

Resolved here: PRD Q2 → F2/D8. PRD Q3 → F1/D2. PRD Q4 → D3. PRD Q5 → D7.

## 10. What this design does not cover

The PRD's non-goals stand unchanged: the 9 category-#3 scripts, the Pyramid PQ and dojo
subsystems, `resetEnteredScript()`, re-reviewing the Cosmic sources, and version-gating.

D1 is worth one explicit note against that last non-goal: moving to the shared root does
not introduce version-gating, it makes the *absence* of gating the default and leaves the
per-version override available for the case the PRD asks to be surfaced rather than
decided silently.
