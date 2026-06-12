# Move Character Change-Map Write to atlas-maps + Retire the Location Shim — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-12
---

## 1. Overview

The web UI's "Change Map" action has been a **silent no-op since commit `95744de34`** ("refactor(atlas-character): remove map ownership; query atlas-maps for shim", task-055). The UI issues `PATCH /api/characters/{id}` with `{ mapId }`; atlas-character's `Update` processor reads that `mapId`, logs a `Debug` line, and discards it (`character/processor.go:1752-1762`). The handler still returns `204 No Content`, so the UI believes the warp succeeded. No `MAP_CHANGED` event is emitted and the character never moves. The pre-task-055 code emitted `MAP_CHANGED` directly from this branch; task-055 deleted that emission and left a comment promising propagation "via atlas-maps' CHANGE_MAP / CHANGE_CHANNEL_REQUEST flows," which was never wired up.

The correct home for a location **write** is **atlas-maps**, which has owned authoritative location state since task-055. atlas-maps already serves `GET /characters/{id}/location`, and the nginx ingress already routes `/api/characters/{id}/location` to it (`deploy/shared/routes.conf:61-64`). This task adds the missing **write** side to that resource, reusing the existing `CHANGE_MAP` warp logic (`location.Set` → emit `MAP_CHANGED` → `TransitionMapAndEmit`) factored into a shared processor method so the Kafka consumer and the new REST handler share one authoritative path.

Beyond fixing the bug, this task **retires the backward-compat location shim** on atlas-character. Today, `GET /characters/{id}` echoes `mapId`/`instance` by calling atlas-maps' `location.GetField` on every request (`character/rest.go:78-92,122-123`), and a fan-out of consumers depends on that echoed value. This PRD migrates **every** consumer to read location from atlas-maps directly, then removes the shim from atlas-character's `Transform` and GET projection. The character resource's `MapId` field is retained **only** as a create-time input (spawn map), which is a legitimate, still-owned use.

## 2. Goals

Primary goals:
- Restore the UI "Change Map" feature: an admin sets a target map and the character actually warps (if online) and/or has its durable location updated (if offline).
- Make atlas-maps — the location owner — serve the location **write**, symmetric with the existing read.
- Eliminate duplicated warp logic: the new REST write and the existing `CHANGE_MAP` Kafka consumer share one authoritative processor method.
- Retire the atlas-character location shim: no service or UI view reads `mapId`/`instance` from `GET /characters/{id}` anymore; the shim is deleted.
- Validate the target map exists before warping; reject invalid maps with a clear 4xx.

Non-goals:
- Changing a character's **channel** (the separate `CHANGE_CHANNEL_REQUEST` flow) — the write endpoint accepts `mapId` only.
- Reworking the character-**creation** spawn-map contract. `RestModel.MapId` remains a create-time input on atlas-character's POST.
- Instanced-map / party-quest / portal-script warps. Admin warps go to the non-instanced map at the spawn portal.
- Any new UI surface beyond repointing the existing change-map dialog and the characters-table map column.
- "Can this character access this map" authorization/level-gating semantics from the legacy error string — validation is limited to *map existence* (see §4.3).

## 3. User Stories

- As an admin/GM using the web UI, I want to change a character's map and have the character actually move (or be relocated on next login), so that the button does what it claims.
- As an admin, I want an invalid map ID to be rejected with a clear error, so that I don't silently send a character nowhere.
- As a developer, I want location reads and writes to both go through atlas-maps (the owner), so that there is a single source of truth and no stale cross-service shim.
- As a maintainer, I want the dead `mapId` update branch and the GET-output location shim removed, so that deprecated backward-compat code is not carried indefinitely.

## 4. Functional Requirements

### 4.1 atlas-maps — location write endpoint

- **FR-1.1** Add a write route on the existing location resource: `PATCH /characters/{characterId}/location` (JSON:API). It is served by atlas-maps (the ingress route `^/api/characters/[^/]+/location(/.*)?$` → `atlas-maps:8080` already exists; no ingress change).
- **FR-1.2** Request body is a JSON:API `character-locations` resource whose `attributes` carry `mapId` (integer). `channelId` and `instance` in the body, if present, are ignored for this task (map-only warp); document this explicitly.
- **FR-1.3** On success the endpoint:
  1. Resolves the character's current field (world + channel) from atlas-maps' own location state.
  2. Builds the destination field: same world, same channel, target `mapId`, `instance = uuid.Nil` (non-instanced), spawn portal (`portalId = 0`).
  3. Persists the destination via the existing `location.Set` path.
  4. Emits the canonical `MAP_CHANGED` status event (`EVENT_TOPIC_CHARACTER_STATUS`) so atlas-channel warps an online client.
  5. Updates atlas-maps' per-map registries via `TransitionMapAndEmit`.
- **FR-1.4** The warp logic in **FR-1.3** steps 3–5 MUST be identical to the existing `CHANGE_MAP` Kafka consumer (`kafka/consumer/character/change_map.go`). Factor that body into a single shared processor method invoked by **both** the consumer and the new REST handler. No copy-paste divergence.
- **FR-1.5** If the character is **offline** (connected to no live channel), the durable location row is still updated (so the character spawns at the new map on next login). If **online**, the `MAP_CHANGED` event additionally warps the live client. The endpoint behaves the same in both cases from the caller's perspective (success = durable write succeeded).
- **FR-1.6** Response: `204 No Content` on success (no body), matching the existing character-update ergonomics the UI already handles. (Design phase may instead return the updated `character-locations` resource if that is more consistent with atlas-maps conventions — see §9.)

### 4.2 atlas-maps — target-map validation

- **FR-2.1** Before warping, validate that the target `mapId` **exists** (resolvable map data). atlas-maps determines existence via the mechanism it already uses for map data (e.g. atlas-data); design phase confirms the exact call.
- **FR-2.2** If the target map does not exist, return `400 Bad Request` (do not persist, do not emit). The error is distinguishable from infrastructure errors (5xx).
- **FR-2.3** If the character has **no existing location row** (HTTP-404 condition in the read path), the warp still proceeds using a best-effort current field — design phase decides whether this is an error (`404`/`409`) or a permitted "set location from scratch." Default: permit, defaulting unknown world/channel to the character's known world and channel 0 if not otherwise resolvable. Recorded as an open question (§9).

### 4.3 atlas-character — remove the dead write branch

- **FR-3.1** Delete the dead `mapId` branch in the `Update` processor (`character/processor.go:1752-1762`). atlas-character no longer participates in location writes in any form.
- **FR-3.2** `RestModel.MapId` is **retained as a create-time input only**: `handleCreateCharacter` continues to pass `input.MapId` as the spawn map to `CreateAndEmit` (`character/resource.go:161`). Document the field as input-only.

### 4.4 atlas-character — retire the GET-output location shim

- **FR-4.1** Remove the `location.GetField` call from `Transform` (`character/rest.go:78-92`) and stop populating `MapId` and `Instance` on the GET projection (`transformWithTemporal`, `rest.go:122-123`).
- **FR-4.2** Remove the `Instance` field from `RestModel` entirely (it is GET-output-only and has no create-input use; no other service mirrors it).
- **FR-4.3** `MapId` remains on `RestModel` for create input (FR-3.2) but is **absent from GET responses** after this change. Document the input-only asymmetry in the struct/Transform comments.
- **FR-4.4** atlas-character's outbound `mapId`/`instance` removal MUST NOT happen until **every** consumer in §4.5–§4.6 has been migrated. The shim deletion is the **last** step (see §7 ordering).
- **FR-4.5** The `location` client package in atlas-character (`location/requests.go`, `location.GetField`) may remain if still used elsewhere; if `Transform` was its only caller, remove it. Design phase verifies remaining callers.

### 4.5 Consumer migration — UI (atlas-ui)

- **FR-5.1** `ChangeMapDialog` **writes** the warp via a new/extended location service call (`PATCH /api/characters/{id}/location` with `{ mapId }`), replacing `charactersService.update(id, { mapId })`.
- **FR-5.2** `ChangeMapDialog` **reads** the character's current map from `GET /api/characters/{id}/location` (atlas-maps), not from `character.mapId`. All current-map uses — initial value, "differs from current" validation, cancel-reset, and the dialog description — switch to the location-sourced value.
- **FR-5.3** The characters table "Map" column (`characters-columns.tsx:151-157`) sources `mapId` from the character's location (via the location endpoint / a location-aware query) instead of `character.attributes.mapId`. The column must continue to render the map link, and must degrade gracefully (e.g. blank/"—") when location is unknown.
- **FR-5.4** Remove `mapId` (and any `instance`) from the UI `Character`/`CharacterAttributes` type (`types/models/character.ts:34`) once no UI code reads it off the character resource. A separate location type/model represents location.
- **FR-5.5** New/updated UI request bodies use the JSON:API envelope `{ data: { type, id, attributes } }` (per the project's input-handler contract). The `type` matches atlas-maps' location resource `GetName()` (`character-locations`).

### 4.6 Consumer migration — Go services

- **FR-6.1** **atlas-parties** (active consumer): `processor.go:268` builds a foreign-member field from `fm.MapId()` taken off the atlas-character response. Migrate this to fetch the member's location from atlas-maps (add/reuse a maps location client in atlas-parties), yielding a full field (world, channel, map, instance) rather than `mapId` with `channelId = 0`. Remove `MapId` from atlas-parties' `ForeignRestModel` (`character/rest.go:38,103`) once unused.
- **FR-6.2** **Passive consumers** — for each service whose character REST client mirror declares `MapId` (atlas-channel, atlas-login, atlas-consumables, atlas-npc-shops, atlas-cashshop, atlas-messengers, atlas-fame, atlas-query-aggregator): verify the extracted `MapId` is not used in business logic, then remove the field from that service's character `RestModel`/`Extract`. If a service **does** use it, migrate it to an atlas-maps location lookup (treat as active, like atlas-parties) — design phase confirms per service.
- **FR-6.3** **atlas-query-aggregator** requires explicit attention: if it re-exposes character `mapId` to downstream consumers (UI or others), removing it changes that aggregated contract. Design phase determines whether query-aggregator must itself source location from atlas-maps to preserve its output, or whether the field can be dropped.
- **FR-6.4** No service may retain a code path that reads `mapId`/`instance` from the atlas-character character resource after this task.

### 4.7 Regression coverage

- **FR-7.1** A test proves the new atlas-maps write endpoint emits `MAP_CHANGED` and persists the destination field (the exact assertion the old atlas-character path lost).
- **FR-7.2** A test proves the shared processor method is the single warp implementation used by both the Kafka consumer and the REST handler (e.g. consumer and handler call the same method; behavior parity test).
- **FR-7.3** A test proves invalid `mapId` yields `400` with no persistence and no emission.
- **FR-7.4** UI tests cover the dialog reading current map from the location endpoint and writing via the location endpoint.

## 5. API Surface

### New — atlas-maps

```
PATCH /characters/{characterId}/location        (ingress: /api/characters/{id}/location)
Content-Type: application/json

{
  "data": {
    "type": "character-locations",
    "id": "{characterId}",
    "attributes": { "mapId": 100000000 }
  }
}

200/204  success (see FR-1.6 / §9)
400      target map does not exist (FR-2.2); or malformed body
404      character has no location row (pending §9 decision in FR-2.3)
500      infrastructure failure (atlas-data/atlas-maps persistence)
```

`channelId`/`instance` attributes, if sent, are ignored for this task (map-only warp).

### Unchanged — atlas-maps

```
GET /characters/{characterId}/location          (already exists; now also the UI's current-map source)
```

### Modified — atlas-character

```
GET   /characters/{characterId}                 → response NO LONGER includes mapId / instance
PATCH /characters/{characterId}                 → mapId attribute ignored/removed (dead branch deleted)
POST  /characters                               → unchanged; mapId remains the spawn-map input
```

### Unchanged — ingress

No `routes.conf` changes; `/api/characters/{id}/location` already maps to atlas-maps.

## 6. Data Model

- No new tables or columns. atlas-maps already persists character location (it owns the location row since task-055; atlas-character's `MapId`/`Instance` entity columns were already dropped — see `character/entity.go:11-19`).
- The warp write reuses the existing `location.Set` persistence path.
- All location reads/writes remain `tenant_id`-scoped via the existing tenant-in-context mechanism.

## 7. Service Impact & Execution Ordering

Ordering matters: the atlas-character shim removal (§4.4) must be **last**, after all consumers are migrated, to avoid breaking live readers mid-rollout.

1. **atlas-maps** — factor shared warp processor method; add `PATCH .../location` handler + map-existence validation; tests (FR-7.1–7.3).
2. **atlas-ui** — repoint `ChangeMapDialog` (read + write) and the characters-table map column to the location endpoint; introduce a location service/type; UI tests (FR-7.4).
3. **atlas-parties** — migrate foreign-member field construction to an atlas-maps location lookup; drop `ForeignRestModel.MapId`.
4. **Passive Go services** (atlas-channel, atlas-login, atlas-consumables, atlas-npc-shops, atlas-cashshop, atlas-messengers, atlas-fame, atlas-query-aggregator) — verify-unused, then strip `MapId` from each character REST client; migrate any that turn out active.
5. **atlas-character** — delete the dead `Update` branch; remove the `Transform` shim, the GET projection `MapId`/`Instance`, and the `Instance` struct field; retain `MapId` as create-input; remove the now-unused `location` client if `Transform` was its only caller.

Each Go module changed must pass the full verification gate (`go test -race`, `go vet`, `go build`, `docker buildx bake atlas-<svc>`, `redis-key-guard.sh`). atlas-ui must pass `npm run build` (which type-checks tests).

## 8. Non-Functional Requirements

- **Multi-tenancy:** every new request/handler resolves tenant from context; location reads/writes stay tenant-scoped.
- **Observability:** the new write logs at appropriate levels (info on warp, warn/error on validation/infra failure) with `character_id`, `map_id`, `tenant`, and a transaction id; failures are not silently swallowed (the original bug was a silent no-op — avoid recreating that class).
- **Consistency:** exactly one authoritative warp implementation (FR-1.4); the REST and Kafka paths cannot diverge.
- **Backward-compat boundary:** because §4.4 changes the atlas-character GET contract, the consumer migration (§4.5–4.6) must be complete and verified first; a partial rollout that strips the shim early would break atlas-parties and the UI map column.
- **No new deprecated surface:** the write endpoint is the owner's endpoint; no new shim is introduced on a non-owner.

## 9. Open Questions

- **OQ-1 (FR-1.6):** Should the write return `204 No Content` (matches current UI ergonomics) or the updated `character-locations` resource (more RESTful/atlas-maps-idiomatic)? Default: `204`.
- **OQ-2 (FR-2.3):** When the character has no existing location row, should the warp `404`/`409`, or set location from scratch (defaulting world from the character, channel `0`)? Default: permit set-from-scratch.
- **OQ-3 (FR-2.1):** Exact map-existence check atlas-maps should use (direct atlas-data call vs. an existing internal resolver). Confirm in design.
- **OQ-4 (FR-6.2/6.3):** Per-service confirmation that each "passive" consumer truly doesn't use `mapId`; specifically whether atlas-query-aggregator re-exposes it and therefore must source location from atlas-maps rather than drop it.
- **OQ-5 (FR-1.3):** Source of the character's current **channel** for the destination field on an admin warp — atlas-maps' stored location channel vs. the live channel the character is connected to (the portal path reads the live socket session). Confirm the stored channel is correct for an online character, or whether a live-channel lookup is needed.
- **OQ-6 (FR-5.3):** Whether the characters-table map column should batch-fetch locations (N characters → N location calls) or whether a bulk/location-included query is needed to avoid a request fan-out. Performance consideration for the table view.

## 10. Acceptance Criteria

- [ ] `PATCH /api/characters/{id}/location` with `{ mapId }` warps an **online** character to the target map (client receives the map change) and updates the durable location row.
- [ ] The same endpoint updates the durable location of an **offline** character (verified by a subsequent `GET .../location`), with no error.
- [ ] An invalid/nonexistent `mapId` returns `400`, persists nothing, and emits no `MAP_CHANGED`.
- [ ] The atlas-maps REST handler and the `CHANGE_MAP` Kafka consumer invoke the **same** shared warp processor method (no duplicated logic).
- [ ] The UI "Change Map" dialog reads current map from `GET .../location` and writes via `PATCH .../location`; the button visibly moves the character in-game.
- [ ] The characters-table "Map" column renders from location data (not `character.mapId`) and degrades gracefully when location is unknown.
- [ ] atlas-parties constructs foreign-member fields from atlas-maps location; `ForeignRestModel.MapId` is removed.
- [ ] Every passive Go service's character REST client no longer declares `MapId` (or, where it was active, sources location from atlas-maps); no code path reads `mapId`/`instance` off the atlas-character character resource.
- [ ] `GET /characters/{id}` (atlas-character) response no longer contains `mapId` or `instance`; `RestModel.Instance` is removed; `RestModel.MapId` remains only as the POST create spawn-map input.
- [ ] The dead `mapId` branch in atlas-character's `Update` processor is deleted.
- [ ] All changed Go modules pass `go test -race`, `go vet`, `go build`, `docker buildx bake atlas-<svc>`, and `redis-key-guard.sh`; atlas-ui passes `npm run build`.
- [ ] New/updated automated tests cover FR-7.1–FR-7.4.
