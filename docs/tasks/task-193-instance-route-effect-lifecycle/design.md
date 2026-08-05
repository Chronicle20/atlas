# Instance Route Effect Lifecycle — Design

Task: task-193
PRD: [`prd.md`](prd.md)
Created: 2026-08-05
Status: Approved for planning

---

## 1. Premise and scope

The PRD is approved. This document settles *how* — architecture, alternatives,
and tradeoffs — and resolves the four open questions in PRD §9 against verified
source evidence. Every claim below cites `file:line` from this worktree; nothing
here is inferred.

The change has three independent halves that share one config surface:

1. **Route-owned effect lifecycle** — `atlas-transports` applies declared
   consumable effects on boarding and cancels them on all five terminal paths.
2. **Optional forced-return on travel-timer expiry** — honours the client's
   `forcedReturn` semantics for routes whose transit maps carry a `timeLimit`.
3. **Seed cleanup** — remove the now-duplicated effect operations from the four
   transport seed files across eleven version directories.

---

## 2. Architecture

### 2.1 Today

Two subsystems each own half of one trip:

```
NPC 2082003 / portal outTemple  ──saga──> APPLY_CONSUMABLE_EFFECT ──> atlas-consumables ──> atlas-buffs
        │
        └──> start_instance_transport ──> atlas-transports (instance lifecycle)
                                                │
                                                └── 5 terminal paths, none cancel anything

portal templeenter / undodraco  ──saga──> CANCEL_CONSUMABLE_EFFECT ──> atlas-consumables ──> atlas-buffs
portal dracoout                 ──(nothing — the known leak, PRD FR-3.6)
```

### 2.2 After

`atlas-transports` owns the whole trip. The route configuration is the single
declaration point:

```
NPC 2082003 / portal outTemple ──> start_instance_transport
                                        │
                        atlas-transports │ instance.ProcessorImpl
                                        │
   StartTransport ────────────> APPLY_CONSUMABLE_EFFECT  (per declared item)
                             └─> CHANGE_MAP (transit map 0)
                             └─> STARTED

   TickArrival ─────────────> CANCEL_CONSUMABLE_EFFECT  (per character × item)
                           └─> CHANGE_MAP (forcedReturn ?: destination)
                           └─> COMPLETED | CANCELLED(TIMEOUT)

   HandleMapEnter (!transit) ─> CANCEL … └─> CANCELLED(MAP_EXIT)
   HandleLogout ─────────────> CANCEL … └─> CANCELLED(LOGOUT)
   TickStuckTimeout ─────────> CANCEL … └─> CHANGE_MAP(start) └─> CANCELLED(STUCK)
   GracefulShutdown ─────────> CANCEL … └─> CHANGE_MAP(start)
```

The portal seeds keep only their warp/sound operations. `dracoout` needs no
edit at all: warping `200090510 → 240000100` (a non-transit map) drives
`HandleMapEnter`'s `!isTransit && inTransport` branch
(`instance/processor.go:161`), which now cancels. The leak closes as a
consequence of the lifecycle move, not as a seed fix.

### 2.3 The configuration path — four projections, not two

PRD FR-4 names two projection points. There are **four**, and each one silently
drops unknown fields:

| # | Layer | File | Failure if missed |
|---|---|---|---|
| 1 | JSONB → tenant REST | `services/atlas-tenants/atlas.com/tenants/configuration/rest.go:416` `TransformInstanceRoute` | Field never leaves atlas-tenants |
| 2 | REST → domain | `services/atlas-transports/atlas.com/transports/instance/config/rest.go:19,50` | Field never reaches the builder |
| 3 | **domain → Redis → domain** | `services/atlas-transports/atlas.com/transports/instance/model_json.go:12` | **Field survives the load, then vanishes on the first registry read** |
| 4 | domain → debug REST | `services/atlas-transports/atlas.com/transports/instance/rest.go:41` `TransformRoute` | Operators cannot see whether a live tenant picked up the config |

Layer 3 is the dangerous one and the PRD does not mention it. `RouteRegistry` is
a Redis-backed `atlas.TenantRegistry` (`instance/route_registry.go:16`), so
**every** route read in the processor round-trips through
`RouteModel.MarshalJSON`/`UnmarshalJSON`. A field added everywhere *except*
`model_json.go` would pass the config tests, pass the builder tests, and be zero
at every call site in `processor.go`. Layer 3 gets its own round-trip test.

Layer 4 is not load-bearing — `TransformRoute` already omits `transitMessage`
(`instance/rest.go:41-52`) — but adding both fields costs two lines and is the
only way an operator can confirm a live tenant re-seeded correctly (§8).
Include it.

---

## 3. Design decisions

### D1 — Emission lives in two private helpers, not inlined at six sites

**Chosen.** Add to `instance/processor.go`:

```go
func (p *ProcessorImpl) applyRouteEffects(mb *message.Buffer, route RouteModel,
    worldId world.Id, channelId channel.Id, characterId uint32)

func (p *ProcessorImpl) cancelRouteEffects(mb *message.Buffer, route RouteModel,
    worldId world.Id, channelId channel.Id, characterId uint32)
```

Both return nothing. Both loop over `route.EffectItemIds()`, `mb.Put` the
command, and log-and-continue on error (§6).

**Alternatives:**

- *(a) Inline the loop at each call site.* Rejected. Six copies of the same
  loop is precisely the shape the PRD is trying to eliminate — "a new terminal
  path cannot silently skip cleanup" (PRD §3). Six copies means a seventh path
  gets zero copies.
- *(c) A dedicated `effects` sub-package or injected processor.* Rejected as
  over-abstraction. There is one effect kind, one topic, and a typical declared
  count of one item. A sub-package would add an interface, a mock, and a
  constructor to save nothing. `instance/producer.go` already holds every other
  provider for this package (`producer.go:18-104`); the two new providers belong
  beside them.

Signature note: the helpers take `worldId`/`channelId` explicitly rather than a
`CharacterEntry`, because the two per-character paths
(`HandleMapEnter`, `HandleLogout`) receive world/channel as handler arguments
and do not have an entry in hand. The three fan-out paths (`TickArrival`,
`TickStuckTimeout`, `GracefulShutdown`) pass `entry.WorldId`/`entry.ChannelId`
from `inst.Characters()`. This satisfies FR-1.7 and matches exactly how the
existing warp and cancelled-event providers in each path already resolve the
field (`processor.go:175, 268, 345, 383-384, 419`).

### D2 — Attribute names: `effectItemIds` and `forcedReturnMapId`

**Chosen**, resolving PRD §9 Q1.

`forcedReturnMapId` mirrors the client's own WZ node. Verified in the PRD's
Appendix A against `Map.wz/Map/Map2/200090500.img.xml`: `forcedReturn = 240000110`.
Naming the field after the data it encodes makes the seed self-documenting and
makes a future audit against WZ trivial. `timeoutMapId` describes the trigger
instead of the concept and loses that link.

`effectItemIds` is a flat `[]item.Id`, matching the existing flat attribute
style (`transitMapIds`, `instance/config/rest.go:24`). The
`effects: [{itemId: …}]` alternative buys extensibility to non-consumable effect
kinds that do not exist and are not planned — YAGNI. The escape hatch is cheap
if it is ever needed: the store is schemaless JSONB, so a future object-array
form is a seed change plus one branch in `TransformInstanceRoute`, not a
migration.

Domain types use `libs/atlas-constants` per DOM-21: `[]item.Id`
(`libs/atlas-constants/item/constants.go:5`, `type Id uint32`) and `_map.Id`.

### D3 — Forced-return emits `CANCELLED` with a new `TIMEOUT` reason

**Chosen**, resolving PRD §9 Q2 **against** FR-2.3's provisional `COMPLETED`.

Verified: `EVENT_TOPIC_INSTANCE_TRANSPORT` has exactly one consumer,
`services/atlas-channel/atlas.com/channel/kafka/consumer/instance_transport/consumer.go`,
and it registers a handler for `TRANSIT_ENTERED` only (`consumer.go:44,60`).
A repo-wide grep for `EventTypeCompleted`/`EventTypeCancelled` on this topic
finds no consumer in any service. **Both options are behaviourally inert
today**, so the choice is purely about semantics for the next consumer.

`COMPLETED` on a forced return would tell a future listener — a quest-progress
or achievement consumer is the obvious candidate — that the character arrived at
the Temple of Time when they were in fact dumped back in Leafre. That is a
latent wrong-credit bug baked in for zero present benefit. Emit:

```go
it.CancelReasonTimeout = "TIMEOUT"   // new const in kafka/message/instance_transport/kafka.go
```

The branch is `if route.ForcedReturnMapId() != 0 { CANCELLED/TIMEOUT } else { COMPLETED }`,
which couples the event semantics to the config field that already *means*
"timer expiry is a failure mode here, not the delivery mechanism". Ferry routes
(no `timeLimit` in their transit-map WZ data — PRD Appendix A) keep `COMPLETED`
unchanged, satisfying the byte-identical-behaviour bar in PRD §8.

Cost: one constant, one `if`. Risk: zero consumers to break, verified.

### D4 — Local wire-contract package, mirroring existing practice

Add `services/atlas-transports/atlas.com/transports/kafka/message/consumable/kafka.go`
containing only what this service emits: `EnvCommandTopic`, the two command-type
constants, `Command[E]`, `ApplyConsumableEffectBody`, `CancelConsumableEffectBody`.

This is not duplication-by-accident; it is the established pattern. Every
service that talks this topic keeps its own copy —
`services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/consumable/`
already does, and `atlas-transports` does the same for `character`,
`instance_transport`, `map`, `channel`, and `configuration`
(`kafka/message/`). Cross-importing another service's Go package would break
the service boundary that CLAUDE.md §Code Patterns protects.

The struct must match `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go:28-36`
field-for-field, including `TransactionId`, `MapId`, and `Instance`, even though
this service leaves the latter two at zero (see D5).

### D5 — `MapId`/`Instance` in the emitted envelope are left zero deliberately

`APPLY` ignores the envelope's field entirely: `ApplyConsumableEffect`
(`atlas-consumables/consumable/processor.go:925`) discards the passed
`channel.Model` (`_ channel.Model`) and resolves the character's *live* map via
`character2.NewProcessor(...).GetMap(characterId)` (`processor.go:934`).

`CANCEL` builds a field from the envelope (`consumer.go:137`) and passes it to
`CancelConsumableEffect` → `buff.Cancel(f, …)`
(`consumables/consumable/processor.go:960-963`) → `character/buff/processor.go:43`
→ atlas-buffs. atlas-buffs' `Cancel` takes **only `worldId`**
(`atlas-buffs/character/processor.go:77`) and never reads the map. The map
component of that field is therefore inert on the cancel path.

Consequence: the logout path (FR-1.6) does not need to know the character's map,
which is fortunate because it does not have one. Set `WorldId`/`ChannelId` from
the resolved entry and leave `MapId`/`Instance` zero on both commands. The
existing `atlas-saga-orchestrator` producer does exactly this
(`saga-orchestrator/consumable/producer.go:18-48` — no `MapId` field set).

### D6 — Ordering: FR-1.2's "buffered before" is a convention, not a guarantee — and correctness does not need it

FR-1.2 asks that apply commands be buffered before the `CHANGE_MAP` command.
`message.Buffer` stores per-topic slices in a `map[string][]kafka.Message` and
`Emit` ranges over that map (`kafka/message/message.go:15,44-56`). **Go map
iteration order is randomised**, so there is no cross-topic ordering at emit
time. Ordering *within* `COMMAND_TOPIC_CONSUMABLE` is preserved (slice append
order).

This does not matter, and the analysis is worth recording because it is the one
place where moving apply out of the saga could plausibly regress:

- **Apply vs. the warp.** Irrelevant — `ApplyConsumableEffect` resolves the
  character's map at handling time (D5), so it produces a correct field whether
  it lands before or after the warp.
- **Apply vs. a fast cancel** (player boards and immediately exits to a
  non-transit map). This *would* be a real regression, because today the NPC
  path is saga-sequenced: the `applyBuff` state waits for `EFFECT_APPLIED`
  before transitioning to `startTransport`
  (`deploy/seed/gms/83_1/npc-conversations/npc/npc-2082003.json`, states
  `applyBuff` → `startTransport`). Removing that state removes the barrier.
  Verified that per-character FIFO still holds end-to-end:
  1. Both commands go to `COMMAND_TOPIC_CONSUMABLE` keyed by `characterId`
     (`producer.CreateKey(int(characterId))`) → same partition → broker order
     preserved; boarding strictly precedes any terminal path in wall-clock.
  2. `atlas-consumables` consumes that partition **serially**: the fetch loop
     takes the serial path when `maxInFlight <= 1`
     (`libs/atlas-kafka/consumer/manager.go:523-525`), the default is `1`
     (`consumer/config.go:112` — "Default is 1 (serial — today's behavior)"),
     and no service anywhere in the repo calls `SetMaxInFlight`.
  3. Both handlers produce onto the buff command topic, also keyed by
     `characterId`, so the same argument carries the second hop.

  APPLY therefore cannot overtake CANCEL for the same character.

**Action:** keep the intra-buffer ordering (it costs nothing and reads
correctly), but the plan must not write a test asserting cross-topic ordering,
and the implementation must not depend on it. Document the reasoning in a code
comment at `StartTransport`, since the next reader will ask.

### D7 — Builder validation

- `SetEffectItemIds([]item.Id)` — no requirement. `Build()` rejects a zero item
  id inside a non-empty list (FR-4.3); a caller that means "no effects" passes
  nil or omits the setter.
- `SetForcedReturnMapId(_map.Id)` — zero means "not set", never an error
  (FR-4.3). No validation.
- `EffectItemIds()` getter returns a defensive copy, matching
  `TransitMapIds()` (`instance/model.go:38-42`). `Build()` stores the slice
  as-is, also matching `TransitMapIds` precedent (`builder.go:88-98`) — the
  copy-on-read is what makes the model immutable in practice.

### D8 — Harden the terminal-path teardown against buffer failures

PRD §8 states: "a Kafka emission failure in a terminal path is logged and does
not prevent instance release … Leaking a buff is bad; leaking an instance is
worse."

The new helpers satisfy their half by never returning an error (D1). But
`HandleLogout` and `HandleMapEnter` today `return err` from the *cancelled-event*
`mb.Put` (`processor.go:175-178, 268-271`), and `ir.ReleaseInstance` sits
**after** that return (`processor.go:180-183, 273-276`). A failing event put
therefore skips instance release — the exact outcome the NFR forbids.

**In scope, three lines:** log the event-put error and fall through to the
release check instead of returning it. This is bounded, is named by the task's
own non-functional bar, and is on the same two methods this task already edits.
It is not a behaviour change in any success path.

---

## 4. Component-level changes

### 4.1 `atlas-transports`

| File | Change |
|---|---|
| `instance/model.go:14` | `RouteModel` gains `effectItemIds []item.Id`, `forcedReturnMapId _map.Id` + getters (`EffectItemIds()` copies). |
| `instance/builder.go:12` | Two setters, zero-item-id validation (D7). |
| `instance/model_json.go:12` | **Both fields in `routeModelJSON`, `MarshalJSON`, `UnmarshalJSON`** (§2.3 layer 3). |
| `instance/config/rest.go:19,50` | `InstanceRouteRestModel` gains `effectItemIds`/`forcedReturnMapId`; `ExtractRouteFor` threads them into the builder. |
| `instance/rest.go:41` | `TransformRoute` exposes both (operator verification, §8). |
| `kafka/message/consumable/kafka.go` | **New.** Local wire contract (D4). |
| `kafka/message/instance_transport/kafka.go:35-37` | New `CancelReasonTimeout = "TIMEOUT"` (D3). |
| `instance/producer.go` | Two new providers: `applyConsumableEffectProvider`, `cancelConsumableEffectProvider`. |
| `instance/processor.go` | Two helpers (D1); calls at six sites; forced-return branch + event-type branch in `TickArrival`; teardown hardening (D8). |
| `docs/domain.md:66`, `docs/kafka.md:147` | `RouteModel` fields; `COMMAND_TOPIC_CONSUMABLE` under "Topics Produced". |

No interface signatures change. `instance/mock/processor.go` and
`bootstrap.go:30-31` are untouched — all new behaviour is internal to existing
methods.

### 4.2 `atlas-tenants`

`configuration/rest.go:380` (`InstanceRouteRestModel`) and `:416`
(`TransformInstanceRoute`) gain both attributes. `TransformInstanceRoute` reads
untyped JSONB, so both need the existing defensive pattern — `float64`
assertions with an `[]interface{}` loop for the array, mirroring the
`transitMapIds` block at `rest.go:430-438`.

### 4.3 Seeds

**Route declarations** (2 files, `deploy/seed/shared/all/instance-routes/`):

| File | `effectItemIds` | `forcedReturnMapId` |
|---|---|---|
| `flight-leafre-temple-of-time.json` | `[2210016]` | `240000110` |
| `flight-temple-of-time-leafre.json` | `[2210016]` | `240000110` |

The second is behaviourally redundant — its `destinationMapId` is already
`240000110` — but it documents the client's `forcedReturn` and keeps the pair
symmetric (FR-3.2).

**Operation removals** (44 files = 4 × 11 version directories):

| File | Edit |
|---|---|
| `npc-conversations/npc/npc-2082003.json` | Delete the `applyBuff` genericAction state entirely; repoint `askTransform`'s first choice `nextState` from `applyBuff` to `startTransport`. Removing only the operation would leave a state with an empty `operations` array — dead weight with no purpose. |
| `portal-actions/portals/portal-outTemple.json` | Drop the first operation of `start_return_flight`, leaving `start_instance_transport` alone. |
| `portal-actions/portals/portal-templeenter.json` | Drop the `cancel_consumable_effect` operation; keep `play_portal_sound` + `warp`. |
| `portal-actions/portals/portal-undodraco.json` | Same. |
| `portal-actions/portals/portal-dracoout.json` | **No edit** (FR-3.6). Must be covered by an explicit test, not assumed. |

Verified prerequisite: all five files are **byte-identical across all eleven
version directories** (md5 confirmed for `gms/{12,48,61,72,79,83,84,87,92,95}_1`
and `jms/185_1`), LF line endings, no CRLF. That makes the safe mechanic:
hand-edit `gms/83_1`, verify the JSON, copy to the other ten, then re-assert
md5 uniformity. It is not a blind patch loop — the uniformity premise is
verified before and after.

Sweep confirmed exactly five distinct seed files repo-wide use these operations;
the fifth is `npc-conversations/npc/npc-1101001.json` (a `2022458` blessing,
explicitly out of scope per PRD §2). 44 files reference `2210016`, i.e. 4 × 11
— consistent with Appendix C and with zero strays.

### 4.4 Deployment

**No change, verified** (FR-4.4):
`COMMAND_TOPIC_CONSUMABLE` is present in `deploy/k8s/base/env-configmap.yaml:26`,
`deploy/k8s/overlays/main/kustomization.yaml:65`, and
`deploy/k8s/overlays/pr/kustomization.yaml:189`; `atlas-transports` mounts the
shared configmap via `envFrom` (`deploy/k8s/base/atlas-transports.yaml:20-22`).

---

## 5. Behaviour matrix

| Path | Method | Warp target | Effect | Event |
|---|---|---|---|---|
| Boarding | `StartTransport` | transit[0] | **APPLY** × n | `STARTED` |
| Boarding rejected (in transport / route missing) | `StartTransport` | — | **none** (FR-1.3) | — |
| Travel timer, no forced return | `TickArrival` | `destinationMapId` | **CANCEL** × n | `COMPLETED` |
| Travel timer, forced return set | `TickArrival` | `forcedReturnMapId` | **CANCEL** × n | `CANCELLED` / `TIMEOUT` |
| Non-transit map entered | `HandleMapEnter` | (already warped) | **CANCEL** × n | `CANCELLED` / `MAP_EXIT` |
| Logout | `HandleLogout` | — | **CANCEL** × n | `CANCELLED` / `LOGOUT` |
| Stuck timeout | `TickStuckTimeout` | `startMapId` | **CANCEL** × n | `CANCELLED` / `STUCK` |
| Graceful shutdown | `GracefulShutdown` | `startMapId` | **CANCEL** × n | (none today; unchanged) |
| Transit → transit | `HandleMapEnter` | — | **none** | `TRANSIT_ENTERED` |
| Any path, route declares no effects | all | unchanged | **zero commands** (FR-1.5) | unchanged |

Idempotency: a double cancel is harmless. `GetRegistry().Cancel` returns
`ErrNotFound` for an absent buff and the processor maps that to `nil`
(`atlas-buffs/character/processor.go:78-81`) — no event, no user-visible error.

---

## 6. Error handling

| Failure | Handling |
|---|---|
| `mb.Put` of an apply command fails during boarding | Log at error, continue. Boarding still proceeds — a missing morph is cosmetic; a rejected boarding is not. |
| `mb.Put` of a cancel command fails on a terminal path | Log at error, continue to the next item and the next character. Never abort teardown. |
| `mb.Put` of a cancelled-event fails on a terminal path | Log at error, fall through to instance release (D8). |
| `atlas-consumables` cannot resolve the character (logged out mid-flight) | Its own handler logs and returns; the cancel is a no-op. FR-1.6's "best effort" is satisfied by the producer never blocking, not by a delivery guarantee. |
| Route not found on a tick path | Unchanged — release the instance, skip everything (`processor.go:333-338`). |

Observability (PRD §8): every apply and cancel logs at info with character id,
route name, and item id, so a stuck-morph report resolves to a specific terminal
path from logs alone.

Tenancy: emitted commands inherit tenant headers from `producer.ProviderImpl(l)(ctx)`
(`processor.go:70`), and every terminal path already filters on
`inst.TenantId() != p.t.Id()` (`processor.go:308, 329, 376, 404`), so the ctx
tenant and the instance tenant always agree. No new header plumbing.

---

## 7. Testing strategy

No `instance/processor_test.go` exists today. Create it, in-package, using the
established miniredis harness (`instance_registry_test.go:16-21`). The `Xxx(mb)`
methods never touch `p.p`, so tests construct `&ProcessorImpl{l, ctx, t, nil}`
directly and assert on `mb.GetAll()[consumable.EnvCommandTopic]` — no mocks, no
Kafka. Route fixtures use `NewRouteBuilder` per CLAUDE.md §Test Helper Pattern;
no `*_testhelpers.go`.

| Test | Guards |
|---|---|
| `StartTransport` emits one APPLY per declared item | FR-1.2 |
| `StartTransport` rejection paths emit nothing | FR-1.3 |
| Each of the five terminal paths emits CANCEL per character × item | FR-1.4 |
| A route with no declared effects emits **zero** consumable messages on every path | FR-1.5, PRD §8 regression bar |
| `TickArrival` with `forcedReturnMapId` set warps there and emits `CANCELLED`/`TIMEOUT` | FR-2.2, D3 |
| `TickArrival` without it warps to destination and emits `COMPLETED` | FR-2.1, ferry regression |
| Exiting `200090510` → `240000100` (the `dracoout` shape) cancels | FR-3.6 — the leak this closes implicitly, tested explicitly |
| `RouteModel` JSON round-trip preserves both fields | §2.3 layer 3 — fails loudly if `model_json.go` is missed |
| `ExtractRouteFor` maps both attributes | §2.3 layer 2; extends `instance/config/rest_test.go` |
| `TransformInstanceRoute` projects both attributes | §2.3 layer 1, FR-4.1; extends `atlas-tenants/configuration/rest_test.go` |
| Builder rejects a zero item id; accepts a zero forced-return map | FR-4.3 |
| Seed sweep: no `apply_consumable_effect`/`cancel_consumable_effect` for `2210016` under `deploy/seed/` | PRD §10 — a grep assertion, run in verification |

---

## 8. Rollout for live tenants

Resolves PRD §9 Q3. The mechanism exists and needs no new code.

`atlas-tenants` registers an `instance-routes` seed group
(`configuration/seed/groups.go:35`) exposing `POST /tenants/configurations/instance-routes/seed`
(`groups.go:47`). A run replaces the tenant's rows and its `AfterSeed` hook
emits exactly **one** configuration-status event (`groups.go:101-103`).
`atlas-transports` consumes it and does a full `ClearTenant` + reload of the
instance-route registry
(`kafka/consumer/configuration/consumer.go:69-79`). No restart, no downtime.

Operator step, to go in the PR description:

1. `POST /tenants/configurations/instance-routes/seed` for each live tenant.
2. Confirm via `GET` on atlas-transports' route resource that
   `effectItemIds`/`forcedReturnMapId` are populated for the two flight routes —
   which is why §2.3 layer 4 is worth the two lines.

A tenant that is not re-seeded keeps today's behaviour for those routes,
including the bug. That is a safe default: the new fields are additive and
optional, so existing stored configurations continue to deserialize unchanged.

Guard worth knowing about: `AfterSeed` refuses to emit when a run deletes rows
but creates none (`groups.go:92-99`), which is the missing-catalog-mount
signature. If the status event never arrives, check that log line first — the
seed will have reported success.

---

## 9. Risks

| Risk | Severity | Mitigation |
|---|---|---|
| `model_json.go` missed → fields silently zero at every call site | **High** — passes all other tests | Dedicated round-trip test (§7); called out as its own plan task |
| 44-file seed edit drifts between version directories | Medium | Uniformity is md5-verified before and after; post-edit assertion is `md5 uniform` + `grep -c 2210016 == 0` |
| NPC state collapse breaks the conversation flow | Medium | The rewire is `askTransform` choice[0] → `startTransport`; verify the resulting JSON loads in `atlas-npc-conversations` |
| Live tenants not re-seeded → bug persists silently | Medium | §8 operator step in the PR description; layer-4 REST exposure makes it checkable |
| Apply/cancel reorder for a player who boards and instantly exits | Low | Analysed and ruled out in D6 (serial consumers + per-character keying) |
| `CANCELLED`/`TIMEOUT` breaks a consumer | None | Verified: no consumer of `COMPLETED`/`CANCELLED` exists anywhere (D3) |

---

## 10. Out of scope

Unchanged from PRD §2, plus one observation from the sweep (PRD §9 Q4): the only
other seed using these operations is `npc-conversations/npc/npc-1101001.json`
(item `2022458`), which is a stationary NPC blessing with no transport and no
transit map — it has no terminal path to leak from, so it is not an instance of
the same bug shape. No other apply-here/cancel-there pairing exists under
`deploy/seed/`.

---

## 11. Verification

Per CLAUDE.md §Build & Verification, on `atlas-transports` and `atlas-tenants`:

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in both modules.
- `docker buildx bake atlas-transports` / `atlas-tenants` from the worktree root
  if either `go.mod` changed.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check`
  clean from the repo root.
- `tools/template-opcode-order-guard.sh` and `tools/service-registration-guard.sh`
  are **not** applicable — no socket-config template and no service-registration
  list changes.
- Seed sweep: zero `2210016` effect operations remain under `deploy/seed/`.
- Code review before PR (`superpowers:requesting-code-review`).
