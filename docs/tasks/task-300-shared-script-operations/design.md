# Shared Script Operation Implementations — Design

Task: task-300-shared-script-operations
Phase: 2 (design)
Input: `docs/tasks/task-300-shared-script-operations/prd.md` (v1, approved)
Status: Draft for review

---

## 1. Summary

Extract the operation *implementations* shared by `atlas-map-actions`,
`atlas-reactor-actions`, `atlas-portal-actions` and `atlas-npc-conversations` into a
new package `libs/atlas-script-core/ops`. Each shared operation becomes a **pure step
builder**: params in, `(Step, error)` out, no I/O. Every service keeps its own dispatch
table, saga assembly, step-id format, `initiatedBy` string, logging, and its
service-local operations.

The design is deliberately shaped around one observation from reading all four
executors: **the divergence between the four copies is entirely in the parameter
decoding and payload construction; everything around it is genuinely local.** Portal's
`warp` mints a saga id and registers a pending action before building the saga
(`executor.go:150-160`); npc batches N operations into one saga and post-processes the
built steps (`operation_executor.go:1007-1135`); reactor resolves a PQ instance over
REST (`executor.go:559`). None of that can or should move. The seam that *can* move is
exactly `params map[string]string → (saga.Action, payload)`.

---

## 2. Decisions on the PRD's open questions

### OQ-1 — Package placement: `libs/atlas-script-core/ops` (sub-package), not a sibling module

**Decided: sub-package.** Evidence:

- `libs/atlas-saga/go.mod` requires only `atlas-constants` and `google/uuid`. It does
  not depend on `atlas-script-core`, so there is **no import cycle** (§5.4).
- Exactly four `go.mod` files require `atlas-script-core` outside a `replace` line
  (`grep -rn "libs/atlas-script-core v" --include=go.mod services | wc -l` → `4`), and
  those four are precisely `map-actions`, `reactor-actions`, `portal-actions`,
  `npc-conversations` — all of which already require `atlas-saga` at the same version.
  So the new module edge is invisible to every current consumer.
- A sibling module would need its own `go.mod`, `go.work` entry, and a `replace` line
  in four services, to avoid a dependency that costs one lightweight module.

Cost accepted: a future consumer that wants only `condition`/`operation`/`context` also
pulls `atlas-saga` transitively. That is one small module (`uuid` + `atlas-constants`),
which is the cheaper of the two failure modes.

### OQ-2 — `Step` is a new small type in `ops`, not `libs/atlas-saga`'s step type

**Decided: new type.** `libs/atlas-saga` exposes step construction only through
`Builder.AddStep(id, status, action, payload)` — there is no free-standing constructor
for a `Step[any]`. Returning `saga.Step[any]` would require either a new exported
constructor in `atlas-saga` (a change the PRD §7 says must be called out and avoided if
possible) or building a throwaway saga per operation. A three-field value type in `ops`
avoids both and keeps the FR-8 "caller owns the step id" contract natural.

### OQ-3 — Instance on `SpawnMonster`: populate from the target **only when the spawn stays on the target's map**

The FR-20 sweep (§7) found `deploy/seed/*/npc-conversations/npc/npc-1063017.json`, which
spawns `9300346` into `mapId: 910510202` and then warps the character there. If the
shared builder copied `Target.Instance()` unconditionally, a spawn whose `mapId` param
points at a *different* map would carry an instance UUID that belongs to the current
map's field — addressing a field that does not exist.

**Rule:** `Instance` = `Target.Instance()` when the effective map id equals
`Target.MapId()`; otherwise `uuid.Nil`. This is stated on the `SpawnMonster` doc comment
and covered by a unit test. It preserves FR-16's intent (a map-action spawn inside an
instanced field lands in that instance) without inventing a cross-map instance.

### OQ-4 — The `Resolver` needs no per-operation escape hatch

`evaluateContextValueAsInt` (`operation_executor.go:179-208`) is exactly
`evaluateContextValue` followed by `EvaluateArithmeticExpression` when the result
contains `+-*/`, else `strconv.Atoi`. Every shared npc case reviewed
(`warp_to_map`, `create_skill`, `update_skill`, `spawn_monster`, `start_quest`,
`apply_consumable_effect`, `send_message`, `play_portal_sound`, `show_hint`,
`show_intro`, `start_instance_transport`, `save_location`, `warp_to_saved_location`,
`stage_clear_attempt_pq`) uses only those two helpers. The two-method `Resolver`
(`String`, `Int`) covers all of them.

The one thing that is *not* value resolution is npc's **defaulting from conversation
state**: `start_quest` falls back to `ctx.Context()["questId"]` and `ctx.NpcId()`
(`operation_executor.go:1976-2004`). That is a Redis read, so it stays in the service
and is passed in as an explicit default (§4.4).

### OQ-5 — `stage_clear_attempt` and `stage_clear_attempt_pq` are the same action, discriminated by payload field

`saga.StageClearAttemptPqPayload` (`libs/atlas-saga/payloads.go:1306-1309`) carries both
`InstanceId` (reactor) and `CharacterId` (npc), and the orchestrator branches on which
is set (`handler.go:3717-3734`). They are the same saga action with two entry points,
not two operations. **FR-17 is kept**, with the reactor's REST lookup
(`getPqInstanceByCharacter`) staying in the service and the resolved instance id passed
into the shared builder. The shared builder is thin but it is the one place that encodes
"exactly one of the two fields must be set", which today is implicit in two call sites.

---

## 3. Alternatives considered

| Approach | Why not |
|---|---|
| **A. Shared `Executor` base type** — a library type owning dispatch + saga creation, with services injecting a processor. | Kills the three local dispatch contracts that carry real behaviour: portal's `opClassMoving`/`movedCharacter` (task-184, `executor.go:73-86`), npc's batching + `builtStep` post-processing (`operation_executor.go:1007-1135`), reactor's `ReactorContext` positional defaults. Every one would come back as a hook. Rejected. |
| **B. Shared saga *builders*** — library returns a fully built `saga.Saga`. | Portal cannot use it: `warp` and `warp_to_saved_location` must mint the transaction id *before* building so the pending-action registry can key on it, and set a 5s timeout (`executor.go:26-33,150-170`). npc cannot use it either — it needs steps, not sagas. Rejected. |
| **C. Pure step builders (chosen).** | Matches the actual seam. Every one of the sixteen call sites reduces to "build a Target, call the function, use the returned status/action/payload". No I/O, so the library is unit-testable with a table and no mocks. |
| **D. Code generation from a param schema.** | The parameter contracts are irregular enough (aliases, sentinels like `expiration: "-1"`, positional defaults) that the generator would be larger than the code it emits. YAGNI. |

---

## 4. Architecture

### 4.1 Package layout

```
libs/atlas-script-core/ops/
    ops.go            // Step, Target (+ builder), Resolver, DirectResolver, param helpers
    ops_test.go       // helpers + Target/Resolver tests
    message.go        // SendMessage
    monster.go        // SpawnMonster
    environment.go    // MoveEnvironment, ResetEnvironment
    effect.go         // ShowIntro, ShowHint, PlayPortalSound, ApplyConsumableEffect
    skill.go          // CreateSkill, UpdateSkill
    movement.go       // WarpToPortal, WarpToSavedLocation, SaveLocation,
                      // StartInstanceTransport
    quest.go          // StartQuest, StageClearAttemptPq
    *_test.go         // one table-driven test file per implementation file
```

One file per family, not one per operation, and not one 1,000-line file. Each stays
small enough to hold in context whole.

### 4.2 `Step`

```go
// Step is one saga step described by a shared operation. It deliberately
// carries no step id: id composition is the caller's (FR-8).
type Step struct {
    status  saga.Status
    action  saga.Action
    payload any
}

func (s Step) Status() saga.Status  { return s.status }
func (s Step) Action() saga.Action  { return s.action }
func (s Step) Payload() any         { return s.payload }

// PayloadOf type-asserts a step's payload. Callers whose step-id format
// embeds a parsed field (map-actions' "spawn-%d-%d" uses the monster id)
// use this rather than re-parsing the param.
func PayloadOf[T any](s Step) (T, error)

// AppendTo adds the step to a saga builder under the caller's id.
func (s Step) AppendTo(b *saga.Builder, id string) *saga.Builder
```

`AppendTo` is what the three rule services use; npc uses the accessors directly because
its handler returns `(stepId, status, action, payload, error)` (FR-10).

### 4.3 `Target`

```go
type Target struct {
    field       field.Model // world, channel, map, instance
    x, y        int16
    hasPosition bool
    portalId    uint32
}

func NewTargetBuilder(f field.Model) *TargetBuilder
func (b *TargetBuilder) SetPosition(x, y int16) *TargetBuilder  // reactor only
func (b *TargetBuilder) SetPortalId(id uint32) *TargetBuilder   // portal only
func (b *TargetBuilder) Build() Target
```

Builder pattern per CLAUDE.md. `hasPosition` is what makes `spawn_monster`'s
"default to the reactor's position, else 0" a property of the *caller*, not a branch
inside the shared op. `portalId` backs `save_location`'s "default to the current portal"
(portal) vs "default to 0" (npc) with no per-service branch.

### 4.4 `Resolver`

```go
type Resolver interface {
    // String resolves a raw param value to its final string form.
    String(characterId uint32, param string, raw string) (string, error)
    // Int resolves a raw param value to an int, supporting arithmetic.
    Int(characterId uint32, param string, raw string) (int, error)
}

// DirectResolver resolves without conversation state: String is identity,
// Int is context.EvaluateValueAsInt. Used by map/reactor/portal actions.
type DirectResolver struct{}
```

- map / reactor / portal supply `ops.DirectResolver{}`.
- npc supplies an adapter over `evaluateContextValue` / `evaluateContextValueAsInt`,
  which keeps the Redis read at exactly the call sites that perform it today.

Shared operations never call `strconv` on a raw param (FR-4). They call the resolver and
then range-check through internal helpers:

```go
func requiredString(p map[string]string, r Resolver, cid uint32, op, name string) (string, error)
func optionalInt(p map[string]string, r Resolver, cid uint32, op, name string, def int) (int, error)
func rangedInt16(op, name string, v int) (int16, error)   // + uint16/uint32/byte/int8 siblings
```

All errors are built by one constructor so FR-19 / NFR-Observability hold uniformly:

```go
// ParamError: `spawn_monster: parameter "x" value "abc" is not a valid integer`
type ParamError struct{ Op, Param, Value string; Err error }
```

### 4.5 Signature shape

```go
func SpawnMonster(p map[string]string, r Resolver, t Target, characterId uint32) (Step, error)
```

Uniform for all sixteen except the three that need a caller-supplied default that comes
from service state:

```go
func StartQuest(p map[string]string, r Resolver, t Target, characterId uint32, d QuestDefaults) (Step, error)
    // QuestDefaults{QuestId, NpcId uint32}; portal passes the zero value,
    // npc passes the conversation-context values it already reads today.

func StageClearAttemptPq(t Target, characterId uint32, instanceId uuid.UUID) (Step, error)
    // reactor passes its REST-resolved instance id; npc passes uuid.Nil.
```

### 4.6 Per-service delegation

**map-actions** (`script/executor.go`): `switch` unchanged. Five handlers shrink to
build-target → call → `AppendTo`. `field_effect`, `lock_ui`, `unlock_ui` untouched.

**reactor-actions** (`script/executor.go`): `switch` unchanged. Five handlers delegate,
each building its Target with `SetPosition(rc.X, rc.Y)`. `stage_clear_attempt` keeps its
`getPqInstanceByCharacter` call and passes the result in. `drop_items`, `spray_items`,
`hit_reactor`, `broadcast_pq_message`, `update_pq_state`, `kill_all_monsters`,
`weaken_area_boss` untouched.

**portal-actions** (`script/optable.go`, `script/executor.go`): `opTable` keeps every
entry and every `opClass` verbatim — the delegation happens inside `executeX`, below the
table (FR-9). `warp`, `warp_to_saved_location` and `start_instance_transport` keep their
`uuid.New()` / registry / `SetTimeout(warpSagaTimeout)` wrapper and delegate only the
payload construction. `block_portal`, `cancel_consumable_effect` untouched.

**npc-conversations** (`conversation/operation_executor.go`): sixteen `case` arms in
`createStepForOperation` reduce to roughly

```go
case "send_message":
    st, err := ops.SendMessage(operation.Params(), e.resolver(), e.target(f), characterId)
    if err != nil { return "", "", "", nil, err }
    return stepId, st.Status(), st.Action(), st.Payload(), nil
```

`createSagaForOperation`, `createSagaForOperations`, the `builtStep` post-processing,
`appendCancelAllBuffsStep`, and every local/cosmetic/quest/pet/storage case are
untouched.

### 4.7 FR-11 — remove the npc `saga` re-export shim

`services/atlas-npc-conversations/atlas.com/npc/saga/model.go:9-171` and `builder.go`
are pure aliases over `libs/atlas-saga`. Delete both alias blocks; keep `processor.go`,
`producer.go`, and the one genuine local type `ValidateCharacterStatePayload`
(`model.go:174-199`, which wraps the service's own `validation.ConditionInput`).
`Processor.Create(s Saga)` becomes `Create(s sharedsaga.Saga)`. Five files import the
local package (`conversation/processor.go`, `conversation/operation_executor.go`, and
three test files); they switch to importing `libs/atlas-saga` directly. Because the
aliases are type identities, this is a mechanical rename with no behaviour change.

### 4.8 FR-12 — enforcing "one implementation"

Add `tools/script-ops-guard.sh`, wired into `tools/verify.sh` next to the existing
`*-guard.sh` steps and gated on `^services/atlas-(map|reactor|portal)-actions/|^services/atlas-npc-conversations/|^libs/atlas-script-core/`.
It fails if any of the sixteen payload literals
(`saga.SpawnMonsterPayload{`, `saga.SendMessagePayload{`, …) appears under `services/`
outside `atlas-saga-orchestrator` (which legitimately *consumes* them). Without this the
FR-12 acceptance criterion is a one-time manual grep that rots on the next feature.

---

## 5. Convergence table (FR-19)

Every behaviour change, per service. "—" = not implemented there today.

### 5.1 `SendMessage` (`drop_message` / `send_message`) — FR-13, FR-14

| | today | after |
|---|---|---|
| map | `messageType` key; default `PINK_TEXT` | also accepts `type`; numeric `5`/`6` mapped |
| reactor | `type` key only; `5`→`PINK_TEXT`, `6`→`BLUE_TEXT`; default `PINK_TEXT` | also accepts `messageType` (wins if both present) |
| portal | `messageType` key; default `PINK_TEXT` | also accepts `type`; numeric `5`/`6` mapped |
| npc | `messageType` **required** (`operation_executor.go:2132`) | optional, defaults to `PINK_TEXT`; also accepts `type` — **documented relaxation** |

`message` stays required everywhere. Both operation names stay valid dispatch keys.

### 5.2 `SpawnMonster` — FR-15, FR-16, OQ-3

| | today | after |
|---|---|---|
| map | `x`/`y` optional→0; hard error on bad parse; `Instance: uuid.Nil` (`executor.go:188`); `Team` omitted | `Instance` from target when map unchanged; `Team` from param (default 0) |
| reactor | `x`/`y` default to `rc.X`/`rc.Y`; **bad `x`/`y`/`count` silently keeps the default** (`executor.go:203,212,217`) | bad value is a hard `ParamError` — **behaviour change** |
| npc | `x`/`y` **required** (`operation_executor.go:1842,1852`); `Instance` never set | `x`/`y` optional, default 0 — **documented relaxation**; `Instance` set when the map is unchanged |

`monsterId` required; `mapId` optional (default = target map); `count` default 1; `team`
default 0; all parse failures hard-error. The PRD §1 says npc populates `Instance`; it
does not — only reactor does. FR-16's fix therefore applies to **both** map and npc.

### 5.3 `WarpToPortal` (`warp` / `warp_to_map`) — FR-18

| | today | after |
|---|---|---|
| portal | `mapId` required; `portalId` optional→0; `portalName` optional | unchanged |
| npc | `mapId` **optional**, defaults to 0 (`operation_executor.go:1417-1424`) | `mapId` **required** — **tightening**; the FR-20 sweep found 3,377 seeded `warp_to_map` ops and 594 `warp` ops, all with `mapId` (§7) |

`warp` keeps `opClassMoving` in portal's `opTable`; both names stay valid dispatch keys.

### 5.4 `CreateSkill` / `UpdateSkill`

| | today | after |
|---|---|---|
| portal | `skillId` required; `level`/`masterLevel` default 1; `expiration` optional, `-1` → +100y, else `UnixMilli` | unchanged |
| npc | `expiration` **ignored**, always +1y (`operation_executor.go:1561,1605`) | honours `expiration` with portal's semantics — **additive** |

### 5.5 `SaveLocation`

| | today | after |
|---|---|---|
| portal | `locationType` required; `mapId` default = current map; `portalId` default = **current portal** | unchanged (default comes from `Target.portalId`) |
| npc | `portalId` default **0** (`operation_executor.go:2566`) | unchanged (npc leaves `Target.portalId` unset → 0) |

### 5.6 `StageClearAttemptPq` — FR-17

| | today | after |
|---|---|---|
| reactor | REST-resolves the PQ instance, sends `InstanceId` | REST lookup stays local; instance id passed to the shared builder |
| npc | sends `CharacterId` only | unchanged |

Shared builder additionally guarantees "exactly one of instanceId/characterId is set",
which the orchestrator requires (`handler.go:3733`) but neither caller asserts today.

### 5.7 No observable change

`MoveEnvironment`, `ResetEnvironment` (map ≡ reactor, byte-for-byte apart from step id
and log line), `ShowIntro` (map ≡ npc), `ShowHint` (portal ≡ npc), `PlayPortalSound`,
`ApplyConsumableEffect`, `StartInstanceTransport`, `WarpToSavedLocation`,
`StartQuest` (npc's context defaults move to `QuestDefaults`, same result).

---

## 6. Cross-service seam: `SpawnMonster` → `atlas-saga-orchestrator`

Traced by hand per CLAUDE.md. `handleSpawnMonster`
(`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go:2087-2124`):

```go
f := field.NewBuilder(payload.WorldId, payload.ChannelId, payload.MapId).
    SetInstance(payload.Instance).
    Build()
...
h.monsterP.SpawnMonster(f, payload.MonsterId, payload.X, payload.Y, int16(fh), payload.Team)
```

`Instance` therefore selects the field the monster is spawned into. Today a map-action or
npc spawn always lands in the base field (`uuid.Nil`) even when the character is inside
an instanced field; after this change it lands in the character's instance. That is the
FR-16 fix, and it is why OQ-3's "same map only" rule matters. No orchestrator code
changes; a new orchestrator-side test asserts the instance-carrying contract
(`SetInstance(payload.Instance)` reaching `monsterP.SpawnMonster`).

`Team` is already read by the same call, so map-actions' previously-omitted `Team` was
being sent as the zero value — populating it from the param is additive.

---

## 7. FR-20 data sweep — result

Run over every seeded script in `deploy/seed/**/*.json` (all tenants and versions), by
parsing each document and walking every `{"type": …, "params": {…}}` node — not a
text grep, so it cannot miss a nested rule.

Findings:

| check | result |
|---|---|
| `spawn_monster` params (`monsterId` ×110, `x` ×110, `y` ×110, `count` ×33, `mapId` ×11) | **every value is a plain integer literal.** No script relies on reactor's swallowed parse failure. FR-15's tightening breaks nothing. |
| `spawn_monster` with `team` | none seeded. |
| `warp` / `warp_to_map` missing `mapId` | **none** (3,377 + 594 ops, all carry `mapId`). §5.3's tightening breaks nothing. |
| `drop_message` using the `type` key, or numeric `5`/`6` | **none seeded** — reactor's numeric mapping is currently dead in the seed data. It is still carried forward (FR-13) because tenant data outside the repo may use it. |
| `send_message` missing `messageType` | none — npc's relaxation is unused but harmless. |
| `spawn_monster` with a cross-map `mapId` | `deploy/seed/*/npc-conversations/npc/npc-1063017.json` (`mapId: 910510202`, then `warp_to_map` to the same map). This is the script that forces the OQ-3 rule. |

**Conclusion: no seeded script needs fixing.** The sweep script is committed as part of
the task folder and re-run during execution so the recorded result is not stale.

---

## 8. Testing

**Library (new).** Table-driven per operation in `libs/atlas-script-core/ops`, no mocks
— the ops are pure. Each table covers: every required param missing; every optional
param default; every parse failure (asserting `ParamError` names op, param and value);
every accepted alias; and the resulting payload field-by-field. Plus targeted cases:
`SpawnMonster` instance-carry vs. cross-map `uuid.Nil`; `SendMessage` `messageType`
winning over `type`; `CreateSkill` `expiration: "-1"`.

**Resolver.** A fake `Resolver` in the test package asserts that no shared op ever
parses a raw string itself (every value observed by the fake).

**Services (existing).** `executor_test.go` in map/reactor/portal and
`operation_executor_test.go` + `operation_executor_petevolution_test.go` +
`processor_rps_test.go` in npc must still pass. Assertions that encode a converged
behaviour are updated, each with a comment naming the FR that changed it
(reactor's swallowed-parse case; npc's required `messageType` / required `x`/`y`).
`optable_test.go` passes **unchanged** — that is the FR-9 regression detector.

**Guard.** `tools/script-ops-guard.sh` gets its own `_test.sh` alongside the other
guard scripts.

**Gate.** Flagless `tools/verify.sh` exits 0 before the branch is done.

---

## 9. Implementation order

1. `ops` package skeleton — `Step`, `Target`, `Resolver`, `DirectResolver`, param
   helpers, `ParamError` — plus their tests. `atlas-script-core/go.mod` gains
   `atlas-saga`.
2. The sixteen implementations + tests, family file by family file. No service touched
   yet, so the library lands green on its own.
3. map-actions delegation (smallest surface, exercises `Target` without position).
4. reactor-actions delegation (exercises `SetPosition` and the FR-15 tightening).
5. portal-actions delegation (exercises the saga-id/timeout wrapper staying local);
   `optable_test.go` must pass untouched.
6. npc-conversations: the context-backed `Resolver` adapter, then the sixteen cases.
7. npc-conversations: remove the `saga` re-export shim (FR-11) — separate commit,
   mechanical, easy to review or revert on its own.
8. `tools/script-ops-guard.sh` + verify.sh wiring; orchestrator seam test; re-run the
   FR-20 sweep; re-point `docs/TODO.md:292-294`. Note those three line references
   are already stale — `docs/TODO.md:292` points at `script/executor.go:250,260` for
   "environment object manipulation", but `move_environment`/`reset_environment` are
   implemented today at `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:284-350`.
   Re-point the two boss/monster entries at their current lines and retire the
   environment entry, since the work it tracks now exists (and moves to
   `libs/atlas-script-core/ops/environment.go`).

Steps 3–6 are independent of each other once step 2 lands.

---

## 10. Risks

| risk | mitigation |
|---|---|
| A tenant's out-of-repo script relies on reactor's swallowed parse failure and starts erroring. | Accepted per FR-15 (the PRD calls it a deliberate fix). The error names op, param and value, so it is diagnosable from one log line. |
| FR-16 changes where map-action spawns land, in production, silently. | OQ-3's same-map rule; an orchestrator seam test; and the change is called out in the PR body. |
| The npc file is 2,738 lines and the sixteen edits are interleaved with local cases. | One commit per group of cases, module-local `go test ./conversation/...` after each; the brief for each execution unit names the exact `case` line numbers. |
| Removing the shim (FR-11) touches five files including three test files. | Its own commit; type identities mean the compiler catches every miss. |
| `ops` gains a dependency others must carry. | Only four modules require `atlas-script-core`, all four already require `atlas-saga` (§2, OQ-1). |

---

## 11. Out of scope (restated)

Merging any service; moving npc's dialogue/cosmetic/quest-progress/pet/storage
operations; moving single-service operations (`field_effect`, `lock_ui`, `unlock_ui`,
`block_portal`, `cancel_consumable_effect`, `hit_reactor`, `broadcast_pq_message`,
`update_pq_state`, `drop_items`, `spray_items`); implementing the stubbed
`weaken_area_boss` / `kill_all_monsters`; any JSON schema, seeder, or `atlas-ui`
change; any change to `libs/atlas-saga` or `atlas-saga-orchestrator` beyond the new
seam test.
