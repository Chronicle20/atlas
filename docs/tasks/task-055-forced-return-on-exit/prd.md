# Forced Return on Exit — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-03
---

## 1. Overview

When a character disconnects from the channel server or changes channels, the map they were on may have a WZ-defined "forced return" target — a map the client expects to spawn on next time the character logs in. In v83 client-parity terms (Cosmic-derived), the relevant WZ fields are:

- `info/forcedReturn` — used unconditionally on save (sentinel `999999999` means "no forced return"). Implements eviction from PQ rooms, hidden streets, mini-dungeons, instruction halls, transit maps, etc. on disconnect / channel change.
- `info/returnMap` — used only when the character was at HP ≤ 0 at save. Implements "die, then disconnect — log back in at town instead of in front of the mob that killed you".

Atlas today has no general-purpose handling of either field on disconnect or channel change. Three services have built their own ad-hoc per-feature exit handlers:

- `atlas-maps/map/timer` (task-050 map time limits) — `ForceReturnIfTracked` emits `CHANGE_MAP` to the timer entry's recorded forced-return target on disconnect or channel-change for time-limited maps only.
- `atlas-transports/instance` — `HandleLogout` releases the instance, emits `CANCELLED`; `HandleLogin` warps off transit map to route start map on crash recovery.
- `atlas-party-quests/instance` — on logout, `LeaveAndEmit("disconnect")` warps to the per-PQ JSON `exit` field, emits `CHARACTER_LEFT`, releases the registry slot.

`atlas-character/Logout` and `atlas-character/ChangeChannel` themselves persist the character's *current* mapId, with no awareness of the WZ overrides. The result is that any feature wanting "evict on disconnect" semantics has to ship its own warp-on-logout consumer, and the WZ data that would already encode the correct target for ~78% of maps (especially every PQ room and transit map) is ignored.

This task introduces a single, shared map-resolution decision — **what map should this character end up on when they exit the current map?** — owned by `atlas-maps`, called by `atlas-character` on its logout and channel-change paths. The three existing per-feature implementations keep their domain bookkeeping (timer cleanup, transport CANCELLED, PQ CHARACTER_LEFT, registry releases) but stop computing exit maps themselves.

## 2. Goals

Primary goals:
- On disconnect, a character on a map with a non-sentinel WZ `forcedReturn` is persisted with the forced-return target as their stored mapId, so they spawn there on next login.
- On disconnect with HP ≤ 0 and a sentinel `forcedReturn`, the character is persisted with `returnMap` as their stored mapId (Cosmic parity for the disconnect-while-dead case).
- On channel change, a character on a map with a non-sentinel `forcedReturn` is moved to the forced-return target on the destination channel before the cross-channel handoff completes.
- The decision logic ("which map should the character end up on?") is owned in one place — `atlas-maps` — and exposed for `atlas-character` to consume.
- Existing per-feature exit handlers (timer, transports, PQ) continue to run their domain bookkeeping, but stop duplicating the map-resolution decision.
- Where redundant config is exposed (PQ JSON `exit` matching WZ `forcedReturn`), document the duplication and provide a path to retiring it in a follow-up.

Non-goals:
- Removing PQ definition `exit` field, transports `route.StartMapId()`, or any other per-feature target config in this task. Document overlap; defer cleanup.
- Changing portal-driven warps, NPC warp scripts, or the death/respawn flow (`atlas-channel/respawn` already honors `returnMap` on click-respawn — that path is unchanged).
- Touching cash-shop "preserve current map on cash-shop close" semantics.
- Adding new spawn-portal selection logic; the destination map's default spawn portal is used.
- Multi-character / party-aware return logic (each character is resolved independently).

## 3. User Stories

- As a player who disconnects in a KPQ room (`103000800`), I want to log back in at the KPQ entry lobby (`103000890`) instead of stuck inside the empty PQ room.
- As a player who changes channels while on a transit map (`200090000`, Ellinia ↔ Orbis boat), I want to land at the Orbis dock (`200000100`) on the new channel — not stuck on a now-defunct boat.
- As a player who dies and disconnects before clicking the death dialog, I want to log back in at the death return town instead of dead at the corpse spot.
- As a player on a regular hunting map (`100020000`, Henesys Hunting Ground 1) who disconnects, I want to log back in exactly where I was (no involuntary teleport).
- As a backend engineer adding a new "must-evict-on-exit" map type, I want to set `forcedReturn` in the WZ data (or its atlas-data equivalent) and have it work — I should not need to register a new logout consumer.
- As a backend engineer maintaining transports / party-quests / map-timer, I want the per-feature logout handler to own only domain bookkeeping (CANCELLED, CHARACTER_LEFT, timer cancel), not the warp target computation.

## 4. Functional Requirements

### 4.1 Resolution decision (owned by atlas-maps)

`atlas-maps` exposes a single map-resolution decision that answers: *"Given a character's current map, current HP, and current instance, what is their exit map id?"*

Resolution rules (in priority order):

1. **HP ≤ 0 AND map has a non-sentinel `returnMap`** → result = `returnMap`. Rationale: dying on a Forest of Poison Fog room and disconnecting should drop the player at Mouth of the Forest, not at the WZ `forcedReturn` for the kill room.
2. **Map has a non-sentinel `forcedReturn`** → result = `forcedReturn`.
3. **Otherwise** → result = current map id (no relocation; persist current map).

The sentinel value `999999999` is treated as "no override" at the **use site** — atlas-data is allowed to expose the raw value through atlas-channel's map DTO. (Q6 answer: do not normalize at the loader.)

The instance UUID is preserved when the resolved map is the current map; on relocation (rules 1 and 2), the instance UUID becomes `uuid.Nil` (matching today's PQ Leave / transports behavior).

### 4.2 Disconnect path (atlas-character → atlas-maps)

When `atlas-character.Logout` runs:

- It must consult the resolution decision before persisting.
- If the resolution returns a different map than the character's current mapId, the persisted mapId must reflect the resolved map (and the persisted instance must be `uuid.Nil`).
- The emitted `LOGOUT` status event continues to use the character's *actual* current field (so transports / map-timer / party-quests still see the original map and can run their bookkeeping). Domain bookkeeping fires in parallel, not sequentially.
- HP at the moment of logout is read from atlas-character's character model (the same source `atlas-channel/respawn` reads from).

### 4.3 Channel-change path (atlas-character → atlas-maps)

When `atlas-character.ChangeChannel` runs:

- It must consult the resolution decision before emitting the channel-changed event.
- If resolution returns a different map than the character's current mapId, the emitted `CHANGE_CHANNEL` event's `newField` carries the resolved map (and `uuid.Nil` instance). The `oldField` continues to carry the character's actual previous field, so per-feature consumers on the old channel can still react.
- HP-at-change-time is not relevant here: channel-change is already gated by `HP > 0` at `atlas-channel/socket/handler/channel_change.go:30`. Rule 1 (HP ≤ 0 → returnMap) cannot fire on this path.

### 4.4 Per-feature handler responsibilities

Each existing handler keeps its domain bookkeeping but stops computing the exit map itself:

- **`atlas-maps/map/timer.ForceReturnIfTracked`** — keep timer cancellation and `MapTimer.Disconnect` log/span. **Stop emitting `CHANGE_MAP`** — the unified resolver in atlas-character will already have persisted the timer's `forcedReturnMapId` (it equals WZ `forcedReturn` for time-limited maps). Verify this equivalence in design phase; if WZ and timer entry diverge for any time-limited map, we will not retire the timer's emit until the divergence is resolved.
- **`atlas-transports/instance.HandleLogout`** — keep instance release, `CANCELLED` emission, registry cleanup. **Keep** the `WarpToRouteStartMapOnLogoutAndEmit` call only if WZ `forcedReturn` for the transit map differs from `route.StartMapId()` in a way operators rely on. (WZ for `200090000` returns `200000100` Orbis dock; transports today returns `route.StartMapId()` Ellinia dock. Decision deferred to design phase.)
- **`atlas-party-quests/instance.LeaveAndEmit("disconnect")`** — keep `CHARACTER_LEFT` emission and registry cleanup. **Stop emitting** the warp-to-`def.Exit()` because the unified resolver covers it via WZ. The per-PQ JSON `exit` field stays in the model for non-disconnect leaves (NPC, expulsion, completion).

### 4.5 Sentinel handling

- The constant `999999999` lives in shared library (proposal: `libs/atlas-constants/map`). All consumers compare `mapId.IsSentinel()` or equivalent, never bare `== 999999999`.
- atlas-data continues to surface the raw value. atlas-channel's `data/map/rest.go` `ForcedReturnMapId` and `ReturnMapId` continue to be raw `_map.Id`.

## 5. API Surface

### 5.1 atlas-maps — new resolver

A new processor method on atlas-maps' map processor (or a dedicated `exit` sub-package — to be decided in design):

```go
// Conceptual signature; final form decided in design phase.
ResolveExitMap(currentField field.Model, hp uint16) (field.Model, ResolutionReason, error)
```

Where `ResolutionReason` is one of:
- `ReasonReturnMapDead` — rule 1 fired.
- `ReasonForcedReturn` — rule 2 fired.
- `ReasonStayPut` — rule 3 fired.

The resolver reads map data via the existing atlas-data pipeline. No new external dependencies.

Surface choice (in-process Go call vs Kafka command vs REST) is deferred to design phase. The current `atlas-maps` processor pattern suggests an in-process call from atlas-character via the existing map BFF, but atlas-character today does not depend on atlas-maps directly. Three candidate shapes:

1. **In-process via atlas-character's existing map data dependency** — atlas-character already consumes map info through atlas-data; pull the resolver into a shared helper that takes raw map info + HP and returns the resolution. Lowest-coupling; treat the "decision" as a pure function on already-fetched data.
2. **REST GET on atlas-maps** — `GET /maps/{mapId}/exit-resolution?hp={hp}`. Aligned with how atlas-character calls atlas-maps for other map lookups today (verify in design).
3. **No new surface; direct WZ field access** — atlas-character reads `forcedReturn` and `returnMap` from atlas-data through its existing map info DTO and applies the rules inline.

Strong lean toward option 1 (a shared `exitresolution` helper inside atlas-maps' Go module, importable by other services) because:
- The decision is pure given the inputs (mapId → forcedReturn, returnMap; HP).
- The same logic must run on disconnect (atlas-character) and on channel-change (atlas-character) and is referenced for verification in atlas-maps timer / transports / PQ.
- A REST round-trip on every logout is unnecessary overhead.

Final decision deferred to design phase.

### 5.2 atlas-character — modified emitters

No new endpoints. Internal modifications only:

- `character/processor.go::Logout` — reads HP and current field, calls resolver, persists resolved field if it differs.
- `character/processor.go::ChangeChannel` — reads current field, calls resolver, sets `newField` on the emitted event to the resolved field.

The Kafka envelope of the existing `LOGOUT` and `CHANNEL_CHANGED` status events is unchanged; only the field values inside change.

### 5.3 No new Kafka topics or message shapes

This task adds no new Kafka topics. Existing per-feature consumers continue to consume the existing events.

## 6. Data Model

No schema changes.

- `atlas-character.characters.map_id` (the persisted last-map column) is the only field whose value semantics change. Today it is "the map the character was on at logout"; after this task it is "the map the character should spawn on next login" — which equals the previous map for ~78% of disconnects (sentinel forcedReturn) and a different map for the rest.
- The `instance` (UUID) column similarly shifts: when relocation fires, the persisted instance is `uuid.Nil` (non-instanced destination), matching today's PQ Leave / transports semantics.

No migration required. The change applies to all logouts after deploy; no retroactive update of existing rows.

## 7. Service Impact

### atlas-maps
- New: shared exit-resolution helper (decision tree: returnMap-on-dead → forcedReturn → stay).
- Modified: `map/timer/processor.ForceReturnIfTracked` may stop emitting `CHANGE_MAP` (timer cancellation only) — pending design-phase verification that WZ `forcedReturn` always equals timer entry's recorded `forcedReturnMapId`.
- New constant or method for sentinel comparison (or imported from libs/atlas-constants).

### atlas-character
- Modified: `character/processor.Logout` consults resolver; persisted mapId + instance reflect resolution.
- Modified: `character/processor.ChangeChannel` consults resolver; emitted `newField` reflects resolution.
- New dependency on atlas-maps' shared resolver (or the chosen API shape).

### atlas-channel
- No code changes expected. `channel_change.go` already blocks dead channel-change. Map data DTO already exposes `ForcedReturnMapId` and `ReturnMapId`.

### atlas-transports
- Modified: `instance.HandleLogout` — keep instance release + `CANCELLED` emit; review whether to keep `WarpToRouteStartMapOnLogoutAndEmit` (WZ vs route.StartMapId divergence). Decision in design phase.

### atlas-party-quests
- Modified: `instance.Leave` (when `reason == "disconnect"`) — drop the warp-to-`def.Exit()` emit; keep `CHARACTER_LEFT` and registry cleanup. Other leave paths (NPC, expulsion, completion) keep the existing exit-map warp.

### atlas-data
- No changes. `forcedReturn` and `returnMap` are already exposed.

### libs/atlas-constants
- New: sentinel constant `MapIdNone = 999999999` (or method on `_map.Id`) for comparison ergonomics. Per `CLAUDE.md` DOM-21, before defining check the existing package; if `_map.Id` already has a sentinel concept, reuse it.

## 8. Non-Functional Requirements

**Performance**
- Disconnect and channel-change are already infrequent operations; one extra map-data lookup per call is acceptable. The resolver should not add a round-trip if atlas-character can compute the decision from cached/already-fetched map info.
- No additional Kafka traffic in the steady-state case (sentinel forcedReturn on regular hunting maps).

**Security / multi-tenancy**
- Resolver lookups must be tenant-scoped via the existing `tenant.MustFromContext(ctx)` pattern. No cross-tenant map-data leakage.
- All atlas-character emit paths already carry tenant context via the existing producer pattern.

**Observability**
- The resolver should emit a structured log line per non-trivial resolution: `{characterId, currentMap, resolvedMap, reason}`. Spans on the OTel pipeline (existing convention in atlas-maps timer) carry `forced.return.map.id`, `return.map.id`, and `resolution.reason` attributes.
- Existing log/span emission in `MapTimer.Disconnect`, transports `HandleLogout`, and PQ `Leave` is preserved.

**Backward compatibility / flags**
- Per the project's "no feature flags" guideline (CLAUDE.md), this lands behind no flag. Rollback path = revert.
- Pre-deploy: the only behavior change for the ~78% of "normal" maps (sentinel forcedReturn) is no behavior change at all. Risk surface is concentrated on PQ rooms, transit maps, instruction halls, hidden streets, and time-limited maps — all of which currently have feature-specific handlers that the unified resolver replaces.

## 9. Open Questions

1. **Resolver surface.** In-process Go helper imported by atlas-character vs REST endpoint vs inline-in-atlas-character. Lean: in-process helper. Decide in design.
2. **WZ vs timer entry equivalence.** For every time-limited map, does WZ `forcedReturn` equal the value `atlas-maps/map/timer` was registered with? If yes, retire the timer's `CHANGE_MAP` emit. If no, document and keep both.
3. **Transports terminus vs origin.** WZ `forcedReturn` for transit maps points to terminus dock; `route.StartMapId()` points to origin dock. Which is the desired DC behavior? (Cosmic-parity = terminus.) Decide in design with operator input.
4. **Sentinel constant placement.** `libs/atlas-constants/map` likely already has something close — confirm in design phase rather than defining a new one.
5. **PQ `exit` field retirement.** Once the disconnect path stops using it, do we keep the JSON field for non-disconnect leaves (NPC, expulsion) or migrate those to also read WZ? Out of scope for this task; flag for follow-up.
6. **Crashed-on-transit-map login recovery.** Today `atlas-transports/instance.HandleLogin` warps off transit maps detected at login. Does the new resolver's `forcedReturn` persistence make this redundant? Likely yes (the character would have been persisted at the WZ `forcedReturn` of the transit map, so login wouldn't even hit a transit map). Verify in design.

## 10. Acceptance Criteria

- [ ] A character on a map with WZ `forcedReturn = X` (non-sentinel) who disconnects logs back in at map X with `instance = Nil`.
- [ ] A character on a map with WZ `forcedReturn = 999999999` who disconnects logs back in at the same map and instance (no change vs. today).
- [ ] A character at HP = 0 on a map with WZ `forcedReturn = 999999999` and `returnMap = Y` who disconnects logs back in at map Y with `instance = Nil`.
- [ ] A character at HP = 0 on a map with both `forcedReturn = X` and `returnMap = Y` (both non-sentinel) who disconnects logs back in at map Y (rule 1 wins over rule 2).
- [ ] A character on a forcedReturn map who changes channels lands on the new channel at the forced-return map (not at the original map).
- [ ] A character in a KPQ room (`103000800`) who disconnects logs back in at `103000890` (KPQ lobby) — without requiring any party-quest-specific consumer to fire.
- [ ] A character on a transit map who disconnects logs back in at the WZ-defined post-transit dock (or at `route.StartMapId()` if the transports-specific override is preserved per Q3).
- [ ] `atlas-party-quests/instance.LeaveAndEmit("disconnect")` no longer emits a `CHANGE_MAP` warp; `CHARACTER_LEFT` and registry cleanup still fire.
- [ ] `atlas-maps/map/timer.ForceReturnIfTracked` either stops emitting `CHANGE_MAP` (if Q2 resolves to "always equivalent") or documents the divergence (if not).
- [ ] All affected services build cleanly and pass their existing test suites.
- [ ] New unit tests cover the three resolution rules, sentinel handling, and the resolver's tenant-scoped data fetch path.
- [ ] Integration test (or hand-verified scenario) confirms KPQ-room disconnect → log back in at KPQ lobby with no PQ membership.
