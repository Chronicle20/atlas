# Local Map Membership for Broadcasts (PS-2) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Every in-map broadcast in atlas-channel — movement, attack, chat, emote, and every other packet fanned out to players sharing a map — currently resolves its recipient set with a synchronous REST call to atlas-maps. `map.Processor` (`services/atlas-channel/atlas.com/channel/map/processor.go:31-70`) builds `CharacterIdsInMapModelProvider` on `requests.SliceProvider`, so `ForSessionsInMap`, `ForOtherSessionsInMap`, `ForSessionsInSessionsMap`, and `ForSessionsInMapAllInstances` each pay an HTTP round-trip through nginx per invocation. Movement is the highest-frequency packet type in the game loop; this is finding **PS-2 (Critical)** in `docs/architectural-improvements.md` (2026-07-02 review): an N+1 amplifier and a single point of latency/failure for all real-time gameplay.

The REST call is redundant with state atlas-channel already owns. Broadcast delivery is inherently pod-local: the resolved character ids are only ever used to look up sessions in the in-process session registry (`session.Processor.ForEachByCharacterId` → `Registry.GetByCharacterId`), and a character id with no local session is silently skipped. Every local `session.Model` already carries its current `field.Model` (world/channel/map/instance), maintained by the registry's `SetField`/`SetMapId` mutators, and the registry already exposes `AllInChannelProvider(worldId, channelId)`. Filtering local sessions by field therefore yields the same effective recipient set as today's REST-then-intersect — without the network hop.

Per the scope decision (owner, 2026-07-02): the **local session registry is the membership source**, chosen over event-sourcing a separate projection from the `MAP_STATUS` Kafka topic, precisely because the session's field must already be reliable for gameplay to work at all. This also dissolves the cold-start problem an event-sourced projection would have (the consumer starts at `LastOffset`): a restarted pod has no sessions until players reconnect, and each session (re)establishes its field on creation/warp — the registry is never stale relative to what the pod can deliver to. The reliability premise is not assumed: FR-1 requires an audit of every field-transition path, with fixes in-scope if gaps are found.

## 2. Goals

Primary goals:

- Zero REST calls to atlas-maps on the broadcast recipient-resolution path: all `map.Processor` recipient providers resolve from the local session registry.
- Recipient sets are equivalent to today's for every caller (same characters reached, modulo transition-window timing — see NFR-3).
- The session-field write path is audited and, where necessary, hardened so every map/instance transition updates the registry before dependent broadcasts fire.
- No wire-level behavior change: packets emitted to each recipient are byte-identical to today for identical inputs.

Non-goals:

- **PS-1** (attack-path REST fan-out), **PS-3/task-120** (monster-move mirror), **PS-4** (Redis optimistic-lock costs) — separate findings, out of scope.
- No changes to atlas-maps: it remains the authoritative membership service for every other consumer, keeps emitting `MAP_STATUS` events, and its REST API and Kafka contract are untouched.
- No changes to atlas-channel's existing `MAP_STATUS` consumer (`CHARACTER_ENTER`/`CHARACTER_EXIT` spawn/despawn broadcasts stay event-driven as-is).
- No new shared library; this is service-local to atlas-channel.

## 3. User Stories

- As a player, I want movement/chat/attack packets from others in my map to arrive without a cross-service HTTP round-trip in the fan-out path, so real-time gameplay stays smooth under load.
- As an operator, I want broadcast throughput decoupled from atlas-maps and nginx availability, so a maps-service blip no longer stalls or drops all in-map traffic.
- As an operator, I want nginx/atlas-maps request volume to stop scaling with packet rate × player density, so infrastructure load reflects actual state changes rather than broadcasts.
- As a developer, I want one clearly-owned recipient-resolution mechanism in atlas-channel, so future broadcast features don't each re-decide how to enumerate a map's players.

## 4. Functional Requirements

### 4.1 Session-field write-path audit (prerequisite)

- **FR-1.1** Enumerate every code path in atlas-channel that changes a character's world/channel/map/instance (map change, portal warp, instance enter/exit, channel change, login spawn-in, revive/forced return, GM warp commands, transport routes) and verify each updates the session registry via `session.Processor.SetField`/`SetMapId` (or equivalent) **before** any subsequent broadcast that depends on the new field.
- **FR-1.2** Any transition path found not updating the session field (or updating it after dependent broadcasts) is fixed in this task, not deferred.
- **FR-1.3** The audit result (path → update site, file:line) is recorded in the task folder so the design and review phases can verify coverage.

### 4.2 Local recipient resolution

- **FR-2.1** `map.Processor.CharacterIdsInMapModelProvider(f field.Model)` returns the character ids of local sessions whose field matches `f` exactly (world, channel, map, instance), sourced from the session registry with no REST call.
- **FR-2.2** `CharacterIdsInMapAllInstancesModelProvider(worldId, channelId, mapId)` returns character ids of local sessions matching world/channel/map across **all** instances, with no REST call.
- **FR-2.3** `ForSessionsInMap`, `ForOtherSessionsInMap`, `ForSessionsInSessionsMap`, `ForSessionsInMapAllInstances`, and `GetCharacterIdsInMap` all resolve through the local providers above. Filtering semantics (e.g. `NotCharacterIdFilter`) are preserved unchanged.
- **FR-2.4** Sessions without an assigned character (pre-login, character-select) are excluded from recipient sets.
- **FR-2.5** Resolution is safe under concurrent access from socket-handler and Kafka-consumer goroutines (the existing registry `RWMutex` discipline; any new index added for performance must follow it).

### 4.3 Caller audit and REST retirement

- **FR-3.1** Audit every caller of the `map.Processor` recipient providers (handlers, Kafka consumers, tasks). For each, confirm local-session semantics are sufficient — i.e. the result is used only to address local sessions or to reason about players on this channel's map, which is served exclusively by this pod. Any caller found to require atlas-maps' authoritative view (none expected) keeps an explicitly-named REST path, documented in the design.
- **FR-3.2** Once no callers remain, the now-unused REST plumbing in `atlas-channel/map` (`requests.go`, REST model, `requestCharactersInMap`, `requestCharactersInMapAllInstances`) is deleted — no dead code left behind.
- **FR-3.3** atlas-channel's other interactions with atlas-maps (e.g. emitting map ENTER/EXIT commands, consuming `MAP_STATUS` events) are unchanged.

### 4.4 Testing

- **FR-4.1** Unit tests cover: exact-field matching (including instance discrimination), all-instances matching, exclusion of character-less sessions, `NotCharacterIdFilter` composition, and empty-map results.
- **FR-4.2** A test demonstrates recipient-set correctness across a simulated map transition: a session warped from map A to map B stops receiving A-broadcasts and starts receiving B-broadcasts with no intermediate state in which it receives both or neither incorrectly.
- **FR-4.3** Tests follow the project Builder pattern for setup (no `*_testhelpers.go` constructors).

## 5. API Surface

No new or modified external endpoints. atlas-maps' `GET .../characters` endpoints remain for its other consumers; atlas-channel simply stops calling them on the broadcast path. `map.Processor`'s Go method signatures are preserved so call sites outside the package are untouched (internal implementation swap).

## 6. Data Model

No persistent data model changes. The session registry (in-memory, per-pod, tenant-scoped) is the existing store; at most this task adds an internal field-keyed index inside it (design-phase decision, see Open Questions) — never persisted, never shared cross-pod.

## 7. Service Impact

- **atlas-channel** — the only service with code changes:
  - `map/processor.go`: recipient providers re-implemented over the session registry.
  - `map/requests.go` + REST model: deleted once unused (FR-3.2).
  - `session/`: possible additions — a field-filtered enumeration provider and/or field index; any transition-path fixes surfaced by FR-1.
- **atlas-maps** — no changes. Load drops substantially (broadcast-driven reads disappear); it remains authoritative for membership queries from other services and for `MAP_STATUS` event emission.
- **nginx ingress** — request volume from the hottest internal path disappears; no config change.

## 8. Non-Functional Requirements

- **NFR-1 Performance.** Recipient resolution is in-process and lock-bounded: O(sessions in channel) scan or O(1) index lookup, no network I/O. Target: resolution cost is negligible relative to packet encode/write (no added REST latency, which today is the dominant term).
- **NFR-2 Availability.** Broadcast fan-out has no runtime dependency on atlas-maps or nginx. An atlas-maps outage no longer degrades in-map gameplay traffic.
- **NFR-3 Consistency.** Recipient sets are exactly as consistent as the session registry, which is updated synchronously in the warp path — this is *stronger* than today's chain (atlas-maps' view is itself an async Kafka-command projection, so the REST result can lag the local session state). The only semantic difference: a character whose Enter command atlas-maps hasn't processed yet is now included immediately. Document this in the design; it is an improvement, not a regression.
- **NFR-4 Multi-tenancy.** All resolution is tenant-scoped exactly as the session registry already is (`tenant.MustFromContext`); no cross-tenant leakage in enumeration.
- **NFR-5 Concurrency.** No new goroutines required; all shared state guarded per existing registry `RWMutex` discipline; `go test -race` clean.

## 9. Open Questions

1. **Field-keyed index vs. linear scan.** `AllInChannelProvider` + filter is O(N) per broadcast over channel session count (hundreds to low thousands); an index keyed by field would be O(recipients) but adds mutation complexity to every field update. Design phase should measure/estimate before choosing (linear scan is likely sufficient and simpler).
2. **Where the filter lives.** New provider on `session.Processor` (e.g. `AllInFieldProvider`) vs. filtering inside `map.Processor`. Design-phase decision; prefer whichever keeps session-registry internals encapsulated.
3. **Shadow verification.** Is a temporary sampled comparison (local result vs. REST result, logged on divergence) worth carrying through staging before deleting the REST plumbing, or are FR-4 tests plus staging playtest sufficient? Owner input welcome at design time; default is tests + staging without shadow mode.

## 10. Acceptance Criteria

- [ ] FR-1 audit document exists in the task folder listing every field-transition path with its registry-update site (file:line); any gaps found are fixed in this branch.
- [ ] No `requests.SliceProvider`/HTTP usage remains in `atlas-channel/map` recipient resolution; `grep` for `requestCharactersInMap` returns nothing in non-test code.
- [ ] All broadcast paths (movement, chat, attack, emote, and every other `ForSessionsInMap`/`ForOtherSessionsInMap` caller) compile against the unchanged method signatures and pass existing tests.
- [ ] FR-4 unit tests pass, including the map-transition correctness test; `go test -race ./...` clean in atlas-channel.
- [ ] `go vet ./...` clean; `docker buildx bake atlas-channel` succeeds from the worktree root; `tools/redis-key-guard.sh` clean.
- [ ] Playtest on a live tenant: two characters in one map see each other's movement/chat/emotes; a third in a different instance/map does not; warping between maps updates visibility immediately; atlas-maps access logs show no character-enumeration requests from atlas-channel during the session.
