# Instance Route Effect Lifecycle — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-05
Issue: [Chronicle20/atlas#1181](https://github.com/Chronicle20/atlas/issues/1181)
---

## 1. Overview

Instance transport routes (`atlas-transports`, `instance` package) move a character from a start map, through one or more transit maps, to a destination map. Two of the twelve seeded routes — the Leafre ↔ Temple of Time dragon flights — additionally require the character to be morphed for the duration of the trip. Today that morph is applied and cancelled by an entirely different subsystem than the one that owns the trip: `atlas-npc-conversations` and `atlas-portal-actions` emit `apply_consumable_effect` / `cancel_consumable_effect` saga operations from hand-written seed scripts, while `atlas-transports` owns the instance lifecycle and knows nothing about the effect.

The two halves disagree on when a trip ends. The portal script cancels the morph only when the player walks through one specific exit portal. `atlas-transports` has five terminal paths — travel-timer arrival, map-exit cancellation, logout cancellation, stuck-timeout, and graceful shutdown — and none of them cancel anything. A player who idles in the flight map until the 900-second travel timer fires is warped to the destination still morphed, and item `2210016` carries `info/noCancelMouse = 1`, so the client will not let them right-click it off. The morph's `spec/time` is `1800000` ms, so they are stuck as a Mini Draco for up to thirty minutes.

The same split is the root of a second, independent defect. The seeded flight route delivers a timed-out player to their destination (Temple of Time) for free, which contradicts the client data: both flight maps declare `timeLimit = 900` and `forcedReturn = 240000110`, i.e. the client's intent is that running out of flight time returns you to Leafre. The ferry-style routes, by contrast, have no `timeLimit` at all in their transit maps — for those, arrival-by-timer genuinely *is* the delivery mechanism and must not change.

This task moves both concerns into the instance route configuration, making `atlas-transports` the single source of truth for the whole trip: it applies the route's declared effects on boarding and cancels them on every terminal path, and it honours an optional per-route forced-return map when the travel timer expires. Both are opt-in — a route that declares neither behaves exactly as it does today.

## 2. Goals

Primary goals:

- Make instance route consumable effects a property of the route, applied on boarding and cancelled on **every** terminal path, so no exit route can leak a buff.
- Remove the duplicated `apply_consumable_effect` / `cancel_consumable_effect` operations from the transport-related NPC-conversation and portal-action seeds, leaving one place where the effect is declared.
- Honour the client's `forcedReturn` semantics for routes whose transit maps carry a `timeLimit`, so a timed-out flight returns the player to the start region instead of delivering them.
- Keep both mechanisms optional and generic — any route may declare them; routes that do not are unaffected.
- Apply the configuration change to all seeded client versions (gms 12/48/61/72/79/83/84/87/92/95 and jms 185).

Non-goals:

- Scheduled (non-instance) transport routes — the `transport` package's boat/vessel model is out of scope.
- Any change to how `atlas-buffs` stores, expires, or cancels buffs, or to the client-side cancel protocol.
- Any change to the `2210016` item data itself (duration, `noCancelMouse`).
- NPC 1101001's unrelated `2022458` blessing buff, which is not a transport.
- A UI editor for instance routes (they are seed/JSONB-managed today; that stays true).

## 3. User Stories

- As a player flying to the Temple of Time, I want the dragon morph removed when my trip ends by any means, so that I am never left in a form I cannot cancel.
- As a player who idles out the flight timer, I want to be returned to Leafre as the client's map data intends, rather than delivered to a destination I did not fly to.
- As a player taking the Ellinia → Ereve ferry, I want the trip to continue delivering me on arrival exactly as it does today.
- As a server operator, I want a route's transit effects declared once in the route configuration, so adding a new morph-bearing route does not require editing NPC and portal scripts in eleven version directories.
- As a developer, I want the transport lifecycle to be the only thing that applies and cancels transit effects, so a new terminal path cannot silently skip cleanup.

## 4. Functional Requirements

### FR-1 — Route-declared effects

- **FR-1.1** `instance.RouteModel` gains an optional, ordered list of consumable effect item ids. Absent or empty means the route applies no effects; this is the default for the ten routes that do not need one.
- **FR-1.2** On successful boarding (`instance.ProcessorImpl.StartTransport`), for each declared item id, `atlas-transports` emits an `APPLY_CONSUMABLE_EFFECT` command on `COMMAND_TOPIC_CONSUMABLE`. The apply commands MUST be buffered **before** the `CHANGE_MAP` command that warps the character to the first transit map, preserving the current ordering (today the NPC applies the buff, then invokes `start_instance_transport`).
- **FR-1.3** Boarding rejection paths (character already in a transport, route not found) MUST NOT emit any apply command.
- **FR-1.4** On every terminal path, for each declared item id, `atlas-transports` emits a `CANCEL_CONSUMABLE_EFFECT` command on `COMMAND_TOPIC_CONSUMABLE` for each affected character. The terminal paths are:

  | Path | Method | Existing warp target |
  |---|---|---|
  | Travel timer arrival | `TickArrival` | destination (see FR-2) |
  | Entered a non-transit map | `HandleMapEnter` (`!isTransit && inTransport`) | none — the player already warped |
  | Logout during transit | `HandleLogout` | none |
  | Stuck timeout | `TickStuckTimeout` | route start map |
  | Graceful shutdown | `GracefulShutdown` | route start map |

- **FR-1.5** Cancel MUST be emitted for a route with no declared effects as a no-op (zero commands), not as an error.
- **FR-1.6** The logout path emits cancel on a best-effort basis. It is not an error if the character session is already gone; the command must not block or fail the instance teardown.
- **FR-1.7** Cancels are emitted per character, using that character's recorded `WorldId`/`ChannelId` from the instance's `CharacterEntry`, matching how the existing warp and event providers in each path already resolve the field.

### FR-2 — Optional forced-return on travel-timer expiry

- **FR-2.1** `instance.RouteModel` gains an optional forced-return map id. Absent (or zero) preserves today's behavior: `TickArrival` warps to `destinationMapId`.
- **FR-2.2** When set, `TickArrival` warps the character to the forced-return map id instead of `destinationMapId`.
- **FR-2.3** The emitted instance-transport event on this path remains `COMPLETED` (the instance did reach the end of its timer). No new event type is introduced. *See Open Questions §9 — a `CANCELLED` variant is the alternative.*
- **FR-2.4** Effect cancellation (FR-1.4) applies on this path regardless of which map the character is warped to.
- **FR-2.5** `TickStuckTimeout` and `GracefulShutdown` continue to warp to `startMapId` and are not affected by this field.

### FR-3 — Seed configuration changes

- **FR-3.1** `deploy/seed/shared/all/instance-routes/flight-leafre-temple-of-time.json` (`temple-of-time-flight`) declares effect item `2210016` and forced-return map `240000110`.
- **FR-3.2** `deploy/seed/shared/all/instance-routes/flight-temple-of-time-leafre.json` (`temple-of-time-return-flight`) declares effect item `2210016` and forced-return map `240000110`. Its `destinationMapId` already is `240000110`, so the forced-return field is behaviourally redundant there but is set for symmetry and to document the client's `forcedReturn` value.
- **FR-3.3** The remaining ten instance route seeds are unchanged — no effects, no forced-return.
- **FR-3.4** The `apply_consumable_effect` operation for item `2210016` is removed from the transport-boarding seeds in **every** version directory (`gms/{12,48,61,72,79,83,84,87,92,95}_1`, `jms/185_1`):
  - `npc-conversations/npc/npc-2082003.json` — the `applyBuff` generic-action state on the Leafre → Temple of Time path. If removing the operation leaves the state with no operations, the state is collapsed and its outcome rewired to `startTransport`.
  - `portal-actions/portals/portal-outTemple.json` — the first operation of the `start_return_flight` rule.
- **FR-3.5** The `cancel_consumable_effect` operation for item `2210016` is removed from the transit-exit portal seeds in every version directory:
  - `portal-actions/portals/portal-templeenter.json`
  - `portal-actions/portals/portal-undodraco.json`
- **FR-3.6** `portal-actions/portals/portal-dracoout.json` requires no new operation. It is today the one transit-map exit portal that lacks a `cancel_consumable_effect` op (its siblings have one), so it leaks the morph; under route-owned cleanup, exiting `200090510` to the non-transit map `240000100` triggers `HandleMapEnter`'s cancel path (FR-1.4). The seed is left as-is and the leak closes as a consequence of FR-1. This must be covered by an explicit test or verification step, not assumed.
- **FR-3.7** The affected seed inventory is uniform across all eleven version directories — see Appendix C. Every one of `gms/{12,48,61,72,79,83,84,87,92,95}_1` and `jms/185_1` contains all four files carrying the operation to remove, plus `portal-dracoout.json`. No directory is skipped and none is created.

### FR-4 — Configuration plumbing

- **FR-4.1** `atlas-tenants`' `TransformInstanceRoute` (`services/atlas-tenants/atlas.com/tenants/configuration/rest.go:416`) explicitly projects each known attribute from the stored JSONB. Both new fields MUST be added there, or they are silently dropped before reaching `atlas-transports`.
- **FR-4.2** `atlas-transports`' `config.InstanceRouteRestModel` and `config.ExtractRouteFor` gain the corresponding fields, threaded through `instance.RouteBuilder`.
- **FR-4.3** `instance.RouteBuilder.Build()` validation: neither field is required. A declared effect item id of zero is rejected as invalid. A forced-return map id of zero is treated as "not set", not as an error.
- **FR-4.4** No new Kafka topic is introduced. `COMMAND_TOPIC_CONSUMABLE` is already present in `deploy/k8s/base/env-configmap.yaml:26` and both overlays, and `atlas-transports` consumes the shared configmap via `envFrom` (`deploy/k8s/base/atlas-transports.yaml:20`), so no deployment manifest change is required. This must be re-verified rather than assumed at implementation time.

## 5. API Surface

No new or modified HTTP endpoints. Two surfaces change shape:

### 5.1 Instance route configuration resource (`instance-routes`)

Served by `atlas-tenants` at `GET /api/tenants/{id}/configurations/instance-routes`, consumed by `atlas-transports`. Two optional attributes are added:

```json
{
  "data": {
    "id": "temple-of-time-flight",
    "type": "instance-routes",
    "attributes": {
      "name": "temple-of-time-flight",
      "startMapId": 240000110,
      "transitMapIds": [200090500, 200090510],
      "destinationMapId": 270000100,
      "capacity": 1,
      "boardingWindowSeconds": 1,
      "travelDurationSeconds": 900,
      "transitMessage": "You are flying towards the Temple of Time. Navigate right to reach the entrance.",
      "effectItemIds": [2210016],
      "forcedReturnMapId": 240000110
    }
  }
}
```

- `effectItemIds` — `uint32[]`, optional. Consumable item ids whose effects the route applies on boarding and cancels on every terminal path. Absent or `[]` means none.
- `forcedReturnMapId` — `uint32`, optional. Map to warp to when the travel timer expires, instead of `destinationMapId`. Absent or `0` means deliver to `destinationMapId`.

Field names are provisional; see Open Questions §9.

### 5.2 Kafka — `COMMAND_TOPIC_CONSUMABLE` (new producer)

`atlas-transports` becomes a producer on this existing topic, emitting the two command shapes `atlas-consumables` already handles (`services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go`):

- `APPLY_CONSUMABLE_EFFECT` with `ApplyConsumableEffectBody{ itemId }`
- `CANCEL_CONSUMABLE_EFFECT` with `CancelConsumableEffectBody{ itemId }`

Both carry `transactionId`, `characterId`, and the field (`worldId`, `channelId`, `mapId`) in the envelope, matching the existing producer in `atlas-saga-orchestrator` (`consumable/processor.go:43,48`). `atlas-consumables` resolves cancel to `buff.Cancel(f, characterId, -int32(itemId))` (`consumable/processor.go:960`); no change is needed on the consumer side.

Error cases: emission failures are logged and do not abort the surrounding terminal-path teardown — an instance must always be released even if a cancel command cannot be buffered.

## 6. Data Model

No database schema changes. Instance routes are stored as JSONB in `atlas-tenants`' configuration table and held in-memory (Redis-backed registry) in `atlas-transports`.

`instance.RouteModel` (`services/atlas-transports/atlas.com/transports/instance/model.go:14`) gains:

| Field | Type | Notes |
|---|---|---|
| `effectItemIds` | `[]uint32` | Defensive copy on read, matching `TransitMapIds()` |
| `forcedReturnMapId` | `_map.Id` | Zero value means "not set" |

Migration notes: both fields are additive and optional, so existing stored tenant configurations remain valid and continue to deserialize. Live tenants seeded before this change will need the two flight routes re-seeded (or PATCHed) to pick up the new attributes; a tenant that is not re-seeded keeps today's behavior for those routes, including the bug. The re-seed path must be identified during planning — see [`docs/tasks/task-189-tenant-config-seed-provisioning/`](../task-189-tenant-config-seed-provisioning/) for the seed-provisioning mechanism.

## 7. Service Impact

| Service | Change |
|---|---|
| `atlas-transports` | Route model, builder, and config extraction gain two fields. New Kafka producer + message package for `COMMAND_TOPIC_CONSUMABLE`. `StartTransport` emits applies; `TickArrival`, `HandleMapEnter`, `HandleLogout`, `TickStuckTimeout`, `GracefulShutdown` emit cancels. `TickArrival` honours forced-return. Docs (`docs/domain.md`, `docs/kafka.md`) updated. |
| `atlas-tenants` | `TransformInstanceRoute` projects the two new attributes. |
| `deploy/seed` | 2 instance-route files (shared/all). Across 11 version directories: `npc-2082003.json`, `portal-outTemple.json`, `portal-templeenter.json`, `portal-undodraco.json` — remove now-redundant effect operations. |
| `atlas-consumables` | None — existing `APPLY`/`CANCEL_CONSUMABLE_EFFECT` handlers are reused as-is. |
| `atlas-portal-actions` | None to code. The `apply_consumable_effect` / `cancel_consumable_effect` operations remain supported for non-transport uses (e.g. NPC 1101001's blessing); only the transport seeds stop using them. |
| `atlas-npc-conversations` | None to code; seed-only. |
| `deploy/k8s` | Expected none (FR-4.4) — verify. |
| `atlas-ui` | None. |

## 8. Non-Functional Requirements

- **Multi-tenancy** — route configuration is already tenant-scoped via the derived-id mechanism in `config.resolveRouteId`. Emitted consumable commands must carry the tenant context of the instance's tenant (`inst.TenantId()`), which every terminal path already filters on.
- **Idempotency** — cancelling an effect the character does not have must be harmless. `buff.Cancel` on an absent source is already a no-op; a double-cancel (e.g. player takes the exit portal at the same moment the timer fires) must not produce a user-visible error.
- **Ordering** — apply commands are buffered before the boarding `CHANGE_MAP` command. Cancel commands are buffered before the terminal-path warp command where one exists.
- **Failure isolation** — a Kafka emission failure in a terminal path is logged and does not prevent instance release or character-registry cleanup. Leaking a buff is bad; leaking an instance is worse.
- **Observability** — each apply and cancel logs at info with character id, route name, and item id, so a stuck-morph report can be traced to a specific terminal path.
- **Performance** — instances are capacity-1 for the affected routes and the tick loops already iterate the same collections; the added work is O(characters × declared effects) with a typical declared-effect count of one.
- **Backwards compatibility** — a route with neither field set must produce byte-identical behavior to today. This is the regression bar for the ten unaffected routes.

## 9. Open Questions

1. **Attribute naming.** `effectItemIds` / `forcedReturnMapId` are provisional. Alternatives considered: `effects: [{itemId}]` (extensible to non-consumable effects later, more verbose now) and `timeoutMapId` (describes the trigger rather than the client concept). The client's own WZ node is `forcedReturn`, which argues for `forcedReturnMapId`. To settle during design.
2. **Event type on forced return.** FR-2.3 keeps `COMPLETED`. `CANCELLED` with a new reason (e.g. `TIMEOUT`) is arguably more accurate — the player did not complete the trip — but `atlas-channel` is the only consumer of this topic and would need checking for behavioral difference. To settle during design.
3. **Re-seeding live tenants.** Existing tenants need the two flight routes updated. Whether this is a seed re-run, a targeted PATCH, or a documented operator step is a planning question.
4. **Other morph-bearing flows.** Only the two ToT flights were found to combine a transport with a consumable effect. Whether any *non*-transport flow has the same leak shape (buff applied by one script, cancelled by another) is out of scope here but worth noting.

Resolved during specification:

- *Per-version seed inventory* — enumerated; see Appendix C and FR-3.7.
- *Logout behavior* — `atlas-buffs` does not drop buffs on logout. Its only logout handler untracks berserk (`kafka/consumer/characterstatus/consumer.go:59`); buffs carry an `expiresAt` and are restored with their remaining duration (`buff/model.go:52`, "remaining-duration contract"). FR-1.6's cancel is therefore a real fix, not defence-in-depth: without it, a player who logs out mid-flight logs back in still morphed.

## 10. Acceptance Criteria

Behavioral:

- [ ] A character who boards `temple-of-time-flight` is morphed (item `2210016`) exactly as today.
- [ ] A character who walks through the `templeenter` portal arrives at Temple of Time **without** the morph.
- [ ] A character who idles until the 900s travel timer expires is warped to **240000110 (Leafre)**, not Temple of Time, and arrives **without** the morph.
- [ ] A character who exits `200090510` via the `dracoout` portal to `240000100` loses the morph (the previously unfixed leak, FR-3.6).
- [ ] A character who logs out mid-flight and logs back in does not retain the morph.
- [ ] A character in transit when `atlas-transports` shuts down gracefully is returned to the start map without the morph.
- [ ] A character whose instance exceeds `MaxLifetime` is returned to the start map without the morph.
- [ ] Both directions (`temple-of-time-flight` and `temple-of-time-return-flight`) satisfy all of the above.
- [ ] The Ellinia → Ereve ferry (and the other nine effect-free routes) still deliver to `destinationMapId` on timer arrival, with no consumable commands emitted.

Configuration:

- [ ] `effectItemIds` and `forcedReturnMapId` survive the round trip from seed JSON → `atlas-tenants` JSONB → `TransformInstanceRoute` → `atlas-transports` `RouteModel`, proven by a test that would fail if the `TransformInstanceRoute` projection were omitted (FR-4.1).
- [ ] A route seed with neither field builds successfully and behaves identically to the pre-change build.
- [ ] No `apply_consumable_effect` or `cancel_consumable_effect` operation referencing item `2210016` remains anywhere under `deploy/seed/`, verified by grep across all version directories.

Verification (per [CLAUDE.md](../../../CLAUDE.md) §Build & Verification):

- [ ] `go test -race ./...` clean in `atlas-transports` and `atlas-tenants`.
- [ ] `go vet ./...` clean in both.
- [ ] `go build ./...` clean in both.
- [ ] `docker buildx bake atlas-transports` and `atlas-tenants` succeed from the worktree root if either `go.mod` changed.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and `tools/lint.sh --check` clean from the repo root.
- [ ] Unit tests cover each of the five terminal paths emitting cancel commands, and the boarding path emitting apply commands in the correct buffer order.

## Appendix A — Verified evidence

All game-data claims below were read from the local extracted WZ dump at `tmp/ec876921-c363-4cc6-9c51-5bb8d57f9553/GMS/83.1/` (tenant `ec876921-…`, GMS 83.1).

**Flight transit maps** — `Map.wz/Map/Map2/200090500.img.xml` and `200090510.img.xml`, `info` node:

| Node | 200090500 | 200090510 |
|---|---|---|
| `timeLimit` | 900 | 900 |
| `forcedReturn` | 240000110 | 240000110 |
| `returnMap` | 240000110 | 240000110 |
| `fly` | 1 | 1 |
| `fieldType` | 6 | 6 |

**Ferry/whale transit maps** — `200090030`, `200090010`, `200090070` `info` nodes contain `returnMap` and `forcedReturn` but **no `timeLimit` node at all**, and `fly` is `0` or absent. This is the data-level distinction between "player navigates, timer is a stranded-player fallback" and "timer is the delivery mechanism", and is why FR-2 must be opt-in per route.

**Morph item** — `Item.wz/Consume/0221.img.xml`, node `02210016`:

```xml
<imgdir name="info">
  <int name="price" value="1"/>
  <int name="slotMax" value="100"/>
  <int name="tradeBlock" value="1"/>
  <int name="noCancelMouse" value="1"/>
</imgdir>
<imgdir name="spec">
  <int name="hp" value="50"/>
  <int name="time" value="1800000"/>
  <int name="morph" value="16"/>
</imgdir>
```

`noCancelMouse = 1` is why the player cannot right-click the buff off. `spec/time = 1800000`: sampling every `spec/time` value across `Item.wz/Consume/` yields only multiples of 1000 (30000, 180000, 600000, 1200000, 1800000, 3600000, …), consistent with milliseconds — so the morph is **30 minutes**, not infinite as initially assumed in the issue triage.

## Appendix B — Current code references

| Concern | Location |
|---|---|
| Instance route model | `services/atlas-transports/atlas.com/transports/instance/model.go:14` |
| Route builder + validation | `services/atlas-transports/atlas.com/transports/instance/builder.go` |
| Config extraction (REST → model) | `services/atlas-transports/atlas.com/transports/instance/config/rest.go` |
| Boarding | `instance/processor.go` — `StartTransport` |
| Travel-timer arrival | `instance/processor.go` — `TickArrival` |
| Map-exit cancellation | `instance/processor.go` — `HandleMapEnter`, `!isTransit && inTransport` branch |
| Logout cancellation | `instance/processor.go` — `HandleLogout` |
| Stuck timeout | `instance/processor.go` — `TickStuckTimeout` |
| Graceful shutdown | `instance/processor.go` — `GracefulShutdown` |
| Tenant config projection | `services/atlas-tenants/atlas.com/tenants/configuration/rest.go:416` |
| Consumable cancel → buff cancel | `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:960` |
| Consumable command topic | `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go:15` |
| Existing consumable producer (reference) | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/consumable/processor.go:43,48` |
| Portal op executor | `services/atlas-portal-actions/atlas.com/portal/script/executor.go:67,70` |
| Buff logout handling (does *not* clear buffs) | `services/atlas-buffs/atlas.com/buffs/kafka/consumer/characterstatus/consumer.go:59` |
| Buff duration is milliseconds | `services/atlas-buffs/atlas.com/buffs/buff/model.go:142` |

## Appendix C — Per-version seed inventory

Enumerated across every seeded version directory. `Y` = file exists and carries the operation to remove; `P` = file present (no change required, see FR-3.6).

| Version dir | `npc-2082003.json`<br>(`apply`) | `portal-outTemple.json`<br>(`apply`) | `portal-templeenter.json`<br>(`cancel`) | `portal-undodraco.json`<br>(`cancel`) | `portal-dracoout.json` |
|---|---|---|---|---|---|
| `gms/12_1` | Y | Y | Y | Y | P |
| `gms/48_1` | Y | Y | Y | Y | P |
| `gms/61_1` | Y | Y | Y | Y | P |
| `gms/72_1` | Y | Y | Y | Y | P |
| `gms/79_1` | Y | Y | Y | Y | P |
| `gms/83_1` | Y | Y | Y | Y | P |
| `gms/84_1` | Y | Y | Y | Y | P |
| `gms/87_1` | Y | Y | Y | Y | P |
| `gms/92_1` | Y | Y | Y | Y | P |
| `gms/95_1` | Y | Y | Y | Y | P |
| `jms/185_1` | Y | Y | Y | Y | P |

Totals: 44 files carrying an operation to remove, plus the 2 shared instance-route files gaining attributes (`deploy/seed/shared/all/instance-routes/flight-leafre-temple-of-time.json`, `flight-temple-of-time-leafre.json`).
