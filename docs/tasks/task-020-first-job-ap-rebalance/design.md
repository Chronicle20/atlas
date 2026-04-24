# First-Job AP Rebalance — Design

Version: v1
Status: Draft
Created: 2026-04-24
Companion PRD: `docs/tasks/task-020-first-job-ap-rebalance/prd.md`

---

## 1. Scope summary

This design covers the backend work needed to satisfy the PRD: a new `rebalance_ap` operation in the shared NPC/quest conversation operation dispatcher, a new `RebalanceAPAndEmit` processor method in `atlas-character`, saga-transport wiring between the two services, updates to five NPC JSONs and five quest JSONs, and conversion-spec documentation for both `rebalance_ap` (new) and `reset_stats` (existing, currently undocumented).

Aran first-job advancement is **not** modeled in this repository (no NPC or quest JSON for the Lilin / polearm interaction). It is removed from scope; the plan document will record this explicitly.

Cygnus first-job advancement is modeled as quest JSONs (`quest_20101.json` through `quest_20105.json`), not NPC JSONs as PRD §4.5 initially suggested. The `rebalance_ap` operation is dispatched from the shared conversation operation executor that both NPC and quest state machines call into, so a single handler covers both contexts.

## 2. Transport & service boundaries

### Decision: saga-orchestrated, matching `change_job`.

The existing `change_job` operation in `services/atlas-npc-conversations/atlas.com/npc/conversation/operation_executor.go:1227-1248` creates a `saga.ChangeJobPayload` as a saga step; `saga-orchestrator` drives the step across services; `atlas-character` registers a handler that invokes `ChangeJobAndEmit`. `rebalance_ap` follows the identical shape:

1. Conversation state machine (NPC or quest) hits a `rebalance_ap` action → operation dispatcher creates a new `saga.RebalanceAPPayload{CharacterId, WorldId, ChannelId, Targets}` step.
2. `saga-orchestrator` dispatches to `atlas-character`.
3. `atlas-character` handler invokes the new processor method `RebalanceAPAndEmit(transactionId, characterId, channel, targets)`.
4. Handler emits stat-change events and replies to the saga with success or failure.
5. On success, the saga progresses to the next step, which is `change_job` in every updated script.

### Rejected alternative: direct REST.

A synchronous REST call from `atlas-npc-conversations` to `atlas-character` would fit this operation individually, but mixing transports within a single advancement sequence is inconsistent with the surrounding `change_job` step and harder to reason about for saga-level compensation and retry. Saga-only keeps the flow coherent.

## 3. Algorithm placement (atlas-character)

### 3.1 Pure helper — `computeRebalance`

Package-private, zero external dependencies:

```go
type rebalanceTarget struct {
    Stat  stat.Type
    Floor int
}

type rebalanceResult struct {
    Str, Dex, Int, Luk int
    Unallocated        int
}

func computeRebalance(str, dex, intel, luk, unallocated int, targets []rebalanceTarget) (rebalanceResult, error)
```

Pure arithmetic implementing PRD §4.1 generalized to the multi-target shape decided in §4 of this design:

1. `reclaimed = max(0, str-4) + max(0, dex-4) + max(0, intel-4) + max(0, luk-4)`
2. Reset new stats to `Str=Dex=Int=Luk=4`.
3. For each target in `targets`, set the corresponding new stat to `target.Floor`. Reject duplicates at the caller's dispatch layer (see §4.2), so the helper trusts the slice has distinct stats.
4. `cost = sum(target.Floor − 4)` across all targets.
5. `newUnallocated = unallocated + reclaimed − cost`.
6. If `newUnallocated < 0`, return an error. This is unreachable at realistic Level-10 first-job state, but the helper guards against malformed input.

No DB, no context, no events. Fully unit-testable as a table (§7.1).

### 3.2 `RebalanceAP(mb *message.Buffer)` — transactional inner method

Following the existing pattern in `atlas-character/character/processor.go`:

- Read the character by ID inside the transaction.
- Call `computeRebalance` with the character's current `str, dex, intel, luk, ap` and the supplied targets.
- On error, surface it back to the saga (handler translates to a saga-step failure).
- On success, update the four primary stats and `ap` on the entity; persist.
- Buffer one `statChangedProvider` message on `mb` carrying all five updated stat types (`TypeStrength`, `TypeDexterity`, `TypeIntelligence`, `TypeLuck`, `TypeAvailableAP`) in a single event — the existing `Updates []stat.Type` + `Values map[string]interface{}` shape at `producer.go:247-262` supports multi-stat updates natively.
- Info-log the before/after stats and AP per PRD §8 Observability. One log line per rebalance, includes character ID, tenant ID (via context), and the targets slice.

### 3.3 `RebalanceAPAndEmit` — public entry point

Wraps `RebalanceAP` via the standard `message.Emit(p)` pattern used throughout the service. This is the method the saga handler invokes.

Signature:

```go
func (p *ProcessorImpl) RebalanceAPAndEmit(
    transactionId uuid.UUID,
    characterId uint32,
    channel channel.Model,
    targets []rebalanceTarget,
) error
```

### 3.4 HP / MP

Not touched. `computeRebalance` only accepts and returns STR/DEX/INT/LUK/AP. The entity update in §3.2 writes only those fields. This satisfies PRD §10.3 by construction — HP and MP are never read or written by this operation.

### 3.5 The speculative advancement-grant — out of scope for this operation

PRD §9 Open Question 1 speculates that v83 may grant an additional ~+5 AP on first-job advancement beyond the rebalance reclaim. This design commits to **no such grant inside `rebalance_ap`**. Evidence: the reference Pirate video matches the 4-step algorithm exactly (DEX 20 + 38 unallocated = 54 reclaimed − 16 cost).

If future research confirms the grant exists, it belongs on the post-`change_job` effect chain (an `atlas-character` event hook that runs after `ChangeJobAndEmit`), **not** coupled to `rebalance_ap`. `rebalance_ap` redistributes existing AP; an advancement grant is an additive side-effect of changing jobs. Keeping these separated means each operation has one responsibility, and a future "grant on advancement" change doesn't require modifying or re-testing rebalance.

## 4. atlas-npc-conversations side

### 4.1 Operation shape — multi-target

Deviation from PRD §4.3: the PRD specified single-stat parameters (`target_stat`, `floor`). Thunder Breaker's existing quest (`quest_20105.json`) double-gates on STR ≥ 20 **AND** DEX ≥ 20. Aligning the rebalance with the existing gate requires raising two stats; a second `rebalance_ap` call would reset the first's work. The operation therefore accepts an array of targets:

```json
{
  "operation": "rebalance_ap",
  "targets": [
    { "stat": "strength", "floor": 20 },
    { "stat": "dexterity", "floor": 20 }
  ]
}
```

Single-target (the common case) is a one-element array:

```json
{ "operation": "rebalance_ap", "targets": [ { "stat": "dexterity", "floor": 25 } ] }
```

Valid `stat` values are `"strength"`, `"dexterity"`, `"intelligence"`, `"luck"`. No HP/MP/SP. Duplicate stats in the array are rejected at dispatch.

### 4.2 Operation handler

New case in the `createSagaStepFromOperation` switch at `operation_executor.go:1227`:

```go
case "rebalance_ap":
    targetsRaw, ok := operation.Params()["targets"]
    if !ok { return "", "", "", nil, errors.New("missing targets") }
    targets, err := parseRebalanceTargets(characterId, targetsRaw, e.evaluateContextValueAsInt)
    if err != nil { return "", "", "", nil, err }
    if len(targets) == 0 { return "", "", "", nil, errors.New("rebalance_ap requires at least one target") }
    if hasDuplicateStats(targets) { return "", "", "", nil, errors.New("rebalance_ap targets must be distinct stats") }
    payload := saga.RebalanceAPPayload{
        CharacterId: characterId,
        WorldId:     worldId,
        ChannelId:   channelId,
        Targets:     targets,
    }
    return stepId, saga.Pending, saga.RebalanceAP, payload, nil
```

`parseRebalanceTargets` validates the inner shape (`stat` name in the allowed set; `floor` is an int or a context-value that evaluates to one), rejecting malformed entries at dispatch. Invalid scripts therefore fail loudly at the conversation layer, not at `atlas-character`.

`saga.RebalanceAP` is a new step-type constant. `saga.RebalanceAPPayload` is a new payload struct sitting alongside `saga.ChangeJobPayload`.

### 4.3 JSON script edits

For each affected script, the action sequence that previously did

```
[stat check] → [rejection on failure] → change_job → reset_stats → …
```

becomes

```
rebalance_ap → change_job → …
```

Per-file edits are:

1. **Remove** the stat-check condition and any unreachable rejection state it pointed to. In NPC JSONs this is a condition-and-branch pair; in quest JSONs it is a prerequisite entry (and the downstream rejection state, where one exists).
2. **Remove** `reset_stats` from the action sequence — `rebalance_ap` supersedes it for first-job advancement. The operation itself is retained in the operation dispatcher and documented (§5); only its usage in first-job scripts is removed.
3. **Insert** `rebalance_ap` immediately before `change_job`. Ordering is mandatory so the client receives the stat-change broadcast before the job-change broadcast, eliminating the intermediate frame where, e.g., a Magician displays STR 53.
4. **Retain** the Level 10 (Level 8 for some Magician paths) prerequisite unchanged.

### 4.4 Enumerated affected files

| File | Class | Target(s) |
|---|---|---|
| `services/atlas-npc-conversations/conversations/npc/npc_1012100.json` | Explorer Bowman (Athena Pierce) | `[DEX 25]` |
| `services/atlas-npc-conversations/conversations/npc/npc_1022000.json` | Explorer Magician (Grendel) | `[INT 20]` |
| `services/atlas-npc-conversations/conversations/npc/npc_1032001.json` | Explorer Warrior | `[STR 35]` |
| `services/atlas-npc-conversations/conversations/npc/npc_1052001.json` | Explorer Thief | `[DEX 25]` |
| `services/atlas-npc-conversations/conversations/npc/npc_1090000.json` | Explorer Pirate (Kyrin) | `[DEX 20]` |
| `services/atlas-npc-conversations/conversations/quests/quest_20101.json` | Dawn Warrior | `[STR 35]` |
| `services/atlas-npc-conversations/conversations/quests/quest_20102.json` | Blaze Wizard | `[INT 20]` |
| `services/atlas-npc-conversations/conversations/quests/quest_20103.json` | Wind Archer | `[DEX 25]` |
| `services/atlas-npc-conversations/conversations/quests/quest_20104.json` | Night Walker | `[LUK 25]` |
| `services/atlas-npc-conversations/conversations/quests/quest_20105.json` | Thunder Breaker | `[STR 20, DEX 20]` |

Aran: no corresponding script exists in the repository; confirmed by grep across both `conversations/npc/` and `conversations/quests/` for any `change_job` targeting jobId 2000 (none found). Removed from scope.

Thunder Breaker target set `[STR 20, DEX 20]` is chosen to align exactly with the existing double-gate in `quest_20105.json`, producing the same minimum stat distribution the current gate requires without inflating the floors.

## 5. Conversion spec updates

`docs/npc_conversation_conversion_spec.md` gains:

1. **`rebalance_ap`** entry in the operations list: parameters (`targets` — array of `{stat, floor}`), semantics (4-step algorithm of PRD §4.1 generalized to multi-target), ordering constraint (must precede `change_job` in the action sequence), validation rules (non-empty, distinct stats, allowed stat names), and one worked example drawn from the Explorer Pirate NPC (DEX 20) and one from Thunder Breaker (multi-target).
2. **`reset_stats`** entry — currently undocumented in the spec despite being used in the repo. Documented with semantics (reset STR/DEX/INT/LUK to 4, return surplus AP to unallocated pool), followed by a usage note: *"For first-job advancement scripts, use `rebalance_ap`. `reset_stats` is retained for GM-tool and non-advancement flows."*
3. **First-job advancement guidance section**: explicit instruction that first-job NPC and quest conversions must use `rebalance_ap` with the class floor and must not encode stat-minimum condition checks as advancement gates. Points readers at the affected-files table in §4.4 of this design as reference examples.

## 6. Resolved vs. deferred PRD open questions

| PRD §9 question | Resolution |
|---|---|
| Q1 — speculative +5 grant | **Resolved at design time.** Not part of `rebalance_ap`. If confirmed to exist in v83, belongs in a post-`change_job` effect chain (outside this task). See §3.5. |
| Q2 — Cygnus first-job flow | **Resolved at design time.** Five quest scripts exist (`quest_20101`..`quest_20105`); shared operation dispatcher means one `rebalance_ap` handler covers both NPC and quest call sites. Target set enumerated in §4.4. |
| Q3 — v83 client auto-opens stat window | **Deferred to testing** (PRD §9 acknowledges this). Not blocking the design. If manual testing reveals a gap, a follow-up task adds an explicit packet or yellow-chat notice. |
| Q4 — `reset_stats` deprecation vs. documentation | **Resolved at design time.** Documented in the conversion spec with a usage note steering first-job flows to `rebalance_ap`; retained in the code and dispatcher for non-advancement flows. See §5. |

## 7. Testing strategy

### 7.1 Unit tests — `computeRebalance`

Pure function, table-driven. Each row asserts input state → output state exactly.

| Input (S/D/I/L, unalloc) | Targets | Expected (S/D/I/L, unalloc) |
|---|---|---|
| 53/9/4/4, 0 | `[DEX 20]` | 4/20/4/4, 38 — Pirate reference video |
| 53/9/4/4, 0 | `[DEX 25]` | 4/25/4/4, 33 — Bowman / Thief / Wind Archer |
| 53/9/4/4, 0 | `[STR 35]` | 35/4/4/4, 23 — Warrior / Dawn Warrior |
| 53/9/4/4, 0 | `[INT 20]` | 4/4/20/4, 38 — Magician / Blaze Wizard |
| 53/9/4/4, 0 | `[LUK 25]` | 4/4/4/25, 33 — Night Walker |
| 53/9/4/4, 0 | `[STR 20, DEX 20]` | 20/20/4/4, 22 — Thunder Breaker |
| 53/9/4/4, 5 | `[DEX 20]` | 4/20/4/4, 43 — unallocated carries through |
| 4/4/4/4, 0 | `[DEX 20]` | error — insufficient AP |

The Pirate row is the acceptance-critical case per PRD §10.1 and §10.7; the Warrior row covers the surplus-return boundary per §10.2; the Thunder Breaker row covers the multi-target path.

### 7.2 Unit tests — `RebalanceAP(mb)`

Using a mocked `message.Buffer` and seeded in-memory character:

- Entity is written exactly once with the expected stat and AP values.
- Exactly one stat-change event is buffered, carrying all five expected `stat.Type` entries and the corresponding new values.
- HP and MP fields on the entity are bit-identical before and after (PRD §10.3).
- Error from `computeRebalance` (insufficient AP case) propagates to the caller; no entity write or event emission occurs.

### 7.3 Operation-handler tests — dispatcher

Unit tests on `createSagaStepFromOperation`'s `rebalance_ap` case:

- Missing `targets` param → dispatcher error, no saga step.
- Empty `targets` array → dispatcher error.
- Duplicate stat in `targets` → dispatcher error.
- Invalid stat name (`"hp"`, `"speed"`, `"banana"`) → dispatcher error.
- Non-integer `floor` (e.g., string that fails context-value evaluation) → dispatcher error.
- Valid single-target → `saga.RebalanceAPPayload` with one entry, correct stat and floor.
- Valid multi-target → payload with both entries preserved in order.

### 7.4 JSON-script validation test

Table-driven test in `atlas-npc-conversations` walking each of the ten files in §4.4 and asserting:

1. No stat-check condition gates the `change_job` transition (the removed gates do not creep back).
2. A `rebalance_ap` operation exists in the same action sequence that contains `change_job`.
3. `rebalance_ap` precedes `change_job` in that sequence.
4. `rebalance_ap` targets match the expected row in the table.

This converts PRD §10.4 ("all first-job NPC scripts updated") from a manual review item to an automated guard that prevents regressions from silent script edits.

### 7.5 End-to-end manual test

PRD §10.6 calls for scripted or documented manual testing. This design commits to a **documented manual test plan** (checklist written into the design and linked from the plan), exercised against a dev deployment post-implementation:

- Pirate (reference video match — PRD §10.7): Level 10 beginner → Kyrin → confirm DEX 20, unallocated 38.
- Warrior (surplus-return boundary — PRD §10.2): Level 10 beginner → Perion → confirm STR 35, unallocated 23.
- Thunder Breaker (multi-target): Level 10 beginner Noblesse → quest 20105 → confirm STR 20, DEX 20, unallocated 22.
- One of DEX-25 classes (Bowman, Thief, or Wind Archer): confirm DEX 25, unallocated 33.

Video match (PRD §10.7) is structurally satisfied by the unit test in §7.1 — the helper is deterministic, so matching on `(input → output)` is equivalent to matching the observed video. Manual Pirate walk-through confirms the end-to-end emission path as well as the arithmetic.

## 8. Non-functional notes

- **Performance:** `computeRebalance` is O(1) arithmetic on four ints plus the targets slice (length ≤ 4). Transactional write and event emission reuse existing code paths. No measurable impact.
- **Multi-tenancy:** The saga payload does not carry tenant information directly; tenant context flows through the saga infrastructure and the `atlas-character` handler as it does for `change_job` today.
- **Idempotency:** `rebalance_ap` is not idempotent. Single-fire semantics of the NPC/quest action state machines guarantee at-most-once execution per advancement.
- **Backwards compatibility:** Characters who already advanced under the old gated behavior are untouched. No migration needed.
- **Observability:** One info-level log per rebalance with character ID, before/after stats, before/after AP, and the targets slice.

## 9. Out-of-scope (explicit)

- Aran first-job advancement — no corresponding script in repo (§4.4).
- Resistance, Dual Blade, and post-v83 class advancement flows (PRD §2 non-goals).
- 2nd / 3rd / 4th job advancement stat gates (PRD §2 non-goals).
- Beginner auto-allocation logic (PRD §2 non-goals; `atlas-character/character/processor.go:1295-1310` stays).
- A tenant-configurable strict-gate mode (PRD §2 non-goals).
- The speculative advancement-grant — deferred until evidence exists, and will not live inside `rebalance_ap` even if added (§3.5).
- Stat-window auto-open client behavior — deferred to testing (§6, PRD §9 Q3).
