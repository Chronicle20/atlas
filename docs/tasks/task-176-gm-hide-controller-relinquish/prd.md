# GM-Hide Relinquishes Monster & NPC Controller Eligibility — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-18
---

## 1. Overview

In the reference server (Cosmic) a GM who turns invisible relinquishes control of every
monster and NPC they were controlling, and is excluded from being (re)assigned as a
controller while hidden. On reveal, the now-visible GM becomes eligible again through the
normal least-loaded controller election. This keeps a hidden GM from silently owning mob
AI/aggro and NPC movement in a map where, from every other client's perspective, they are
not present — which manifests as mobs that stand still (their controller's client is
suppressing movement) and NPCs whose animation stalls.

Atlas today implements neither half of this behavior. Monster control is owned by
`atlas-monsters` (Redis-backed, multi-replica), whose controller election
(`getControllerCandidate`) picks the least-loaded character from the field's character
list with **no awareness of hide state** — a hidden GM is a valid candidate and will be
handed control. No event releases a GM's controlled mobs when they hide. Separately,
Atlas has **no NPC-controller election at all**: `atlas-channel` sends an
`NpcSpawnRequestController` packet to *every* session on map entry, so there is nothing to
relinquish and no single-controller concept to make hide-aware.

This task brings Atlas to Cosmic parity for both entities. It (a) makes the existing
monster-controller election hide-aware and adds relinquish/reassign on the hide/reveal
transitions, and (b) builds the missing NPC-controller election subsystem and makes it
hide-aware from the start. Hide state is sourced from the buff-status events that
`atlas-buffs` already emits; the character's current field is resolved live from
`atlas-maps` at the moment each event is handled, never carried on the buff event (see
§8, Design Constraints).

## 2. Goals

Primary goals:
- When a GM hides (SuperGmHide buff `APPLIED`), release every monster that GM controls in
  their current field and reassign each to a visible controller.
- Exclude hidden GMs from monster-controller candidacy, both at the hide transition and on
  any subsequent election trigger (e.g. a hidden GM entering a new map must not grab
  control).
- When a GM reveals (buff `EXPIRED`), re-run monster-controller election so the now-visible
  GM is eligible again via the existing least-loaded rule.
- Introduce a single-controller-per-NPC election in Atlas (assign on map enter, reassign on
  controller exit) and apply the same hide-aware relinquish/exclude/reveal semantics to it.
- Correctly handle multi-replica deployment: hide state consulted during election must be
  consistent across all `atlas-monsters` (and NPC-controller-owning) pods.

Non-goals:
- Changing the GM-hide toggle itself, the hide buff, or the self/foreign spawn suppression
  (owned by task-156 in `atlas-channel`).
- Making regular thief **Dark Sight** (`RogueDarkSightId`, `NightWalkerStage1DarkSightId`)
  relinquish control. Cosmic's `isHidden()` gates on GM-hide only; dark-sighted thieves
  retain control. This task filters on GM-hide exclusively.
- Adding `GmHideId` (9001004) handling. It is absent from the v83 GMS `Skill.wz`
  (job `900.img` contains only 9001000/1/2); only `SuperGmHideId` (9101004, in `910.img`)
  is real game data. task-156 registering only SuperGmHide was correct.
- Reworking monster aggro, skill picker, or NPC conversation/shop behavior.

## 3. User Stories

- As a GM, when I turn invisible, I want the monsters I was controlling to keep behaving
  normally for the players still in the map, so my presence doesn't freeze mob AI.
- As a player, I want monsters and NPCs to animate and aggro normally even when an unseen
  GM is standing in my map.
- As a GM, when I reveal myself, I want to resume participating in controller assignment
  like any other player.
- As a server operator, I want this to behave correctly regardless of how many
  `atlas-monsters` replicas are running.

## 4. Functional Requirements

### FR-1 — Hide-state signal (shared, both entities)
- FR-1.1: A consumer on `EVENT_TOPIC_CHARACTER_BUFF_STATUS` MUST treat a `Body.SourceId ==
  SuperGmHideId` (9101004) `APPLIED` event as "character entered hidden state" and the
  matching `EXPIRED` event as "character left hidden state".
- FR-1.2: Events whose `SourceId` is not `SuperGmHideId` MUST be ignored by this feature.
- FR-1.3: The current hidden-character set MUST be stored in shared state (Redis),
  tenant-scoped, so any replica performing a controller election observes the same set.
  An in-process/per-pod set is insufficient — `atlas-buffs` events are consumer-group
  partitioned to one pod, while elections run on the pod that owns the target field.
  (See §8 Design Constraints.)
- FR-1.4: `APPLIED` adds the character to the hidden set; `EXPIRED` removes it. Handling
  MUST be idempotent (duplicate `APPLIED`/`EXPIRED` for a character already in/out of the
  set is a no-op).

### FR-2 — Monster controller: relinquish on hide
- FR-2.1: On a GM-hide `APPLIED` event, resolve the character's **current** field via
  `atlas-maps` `GET /characters/{id}/location`.
- FR-2.2: Snapshot the set of monsters controlled by that character in that field, then
  `StopControl` each and reassign via `FindNextController`, mirroring the existing
  `handleStatusEventCharacterExit` two-step (snapshot-first to avoid the provider
  re-evaluation race documented there).
- FR-2.3: Reassignment MUST NOT select the now-hidden character (enforced by FR-4).

### FR-3 — Monster controller: reveal re-eligibility
- FR-3.1: On a GM-hide `EXPIRED` event, resolve the character's current field (FR-2.1) and
  re-run `FindNextController` for uncontrolled monsters in that field so the revealed GM
  can be picked by the normal least-loaded rule.
- FR-3.2: Reveal MUST NOT forcibly transfer control to the revealed GM — it only restores
  candidacy (parity with Cosmic's `aggroUpdateController` sweep).

### FR-4 — Monster controller: exclude hidden from candidacy
- FR-4.1: `getControllerCandidate` MUST exclude characters present in the hidden set from
  the candidate pool, both for initial assignment (monster create) and every reassignment
  path (`FindNextController`, character-enter, character-exit).
- FR-4.2: The puppet-vicinity owner bias MUST also skip a hidden owner.
- FR-4.3: If excluding hidden characters leaves no candidate (e.g. the only character in
  the field is a hidden GM), the monster is left uncontrolled — matching Cosmic, where a
  map with only a hidden GM has no controller.

### FR-5 — NPC controller: election subsystem (new)
- FR-5.1: Introduce a single-controller-per-NPC concept: exactly one non-hidden character
  in a field is the controller of a given NPC at a time (or none, if the field has only
  hidden characters / is empty).
- FR-5.2: On character map-enter, uncontrolled NPCs in that field MUST be assigned a
  controller via the same candidacy rule as monsters (least-loaded, hidden excluded).
- FR-5.3: On controller map-exit, that character's NPCs MUST be reassigned to another
  eligible character (or left uncontrolled if none).
- FR-5.4: The `NpcSpawnRequestController` packet MUST be sent only to the elected
  controller's session; non-controllers receive the plain `NpcSpawn` only.
- FR-5.5: NPC-controller state MUST be shared across replicas (same constraint as FR-1.3).

### FR-6 — NPC controller: hide-aware relinquish/reveal
- FR-6.1: On GM-hide `APPLIED`, NPCs controlled by that GM in their current field MUST be
  released and reassigned to a visible controller; the client-side controller grant MUST
  be revoked from the hidden GM.
- FR-6.2: A hidden GM MUST be excluded from NPC-controller candidacy (parity with FR-4).
- FR-6.3: On GM-hide `EXPIRED`, NPC-controller election MUST re-run so the revealed GM is
  eligible again (no forced transfer, parity with FR-3.2).

### FR-7 — Failure handling
- FR-7.1: If `GET /characters/{id}/location` fails or the character is not in a map
  (offline/in transition), the handler MUST skip the relinquish/reassign for that event
  and emit a debug log — no error propagation, no retry loop. (Parity with the fail-safe
  style in task-156's hide handler.)
- FR-7.2: The hidden-set mutation (FR-1.4) MUST still be applied even when the
  location-dependent action (FR-2/FR-6) is skipped, so candidacy exclusion (FR-4/FR-6.2)
  stays correct once the character is located by a later election trigger.

## 5. API Surface

No new externally-facing endpoints are strictly required for the monster half.

Consumed (existing):
- `atlas-buffs` → `EVENT_TOPIC_CHARACTER_BUFF_STATUS`, types `APPLIED` / `EXPIRED`, body
  carries `SourceId`, `CharacterId`, `WorldId` (no field — by design).
- `atlas-maps` → `GET /characters/{characterId}/location` returning the character's current
  `world / channel / map / instance` field.

Potentially new (NPC subsystem, to be settled in design):
- An NPC-controller state store and/or a query surface for "NPCs controlled by character X
  in field F", analogous to `atlas-monsters`' `ControlledByCharacterInFieldProvider`. Whether
  this lives behind REST, Kafka, or a shared Redis registry is an open architectural
  question (§9).

## 6. Data Model

- **Hidden-character set (monsters):** a tenant-scoped Redis set of `characterId`s currently
  GM-hidden, maintained by the buff-status consumer. Follow the existing
  `atlas-redis` `KeyedSet` pattern already used by `atlas-monsters` (`monster-map` index).
- **NPC-controller state (new):** a per-field NPC→controller mapping (and its inverse,
  controller→NPCs) in shared storage. NPCs are static map life sourced from `atlas-data`
  map data (`data/npc.ForEachInMap`), so a "live NPC" record may need to be materialized
  to hold controller assignment. Exact schema/home deferred to design (§9).
- All state MUST be tenant-scoped consistent with existing registries.

## 7. Service Impact

- **atlas-monsters** — new buff-status consumer; Redis hidden-set registry; hide-aware
  `getControllerCandidate`; relinquish/reassign on hide, re-election on reveal; new
  dependency on the `atlas-maps` character-location endpoint. `go.mod` likely unchanged
  (already depends on atlas-redis, atlas-kafka, atlas-rest) — confirm; if it changes,
  `docker buildx bake atlas-monsters` per CLAUDE.md.
- **atlas-buffs** — source of the hide signal. **No change** (event already emitted with
  `SourceId`). Explicitly do NOT add field to the event (§8).
- **atlas-maps** — provides `GET /characters/{id}/location` (already exists). Consumed; no
  change expected. If NPC-controller state is chosen to live here, that changes.
- **NPC-controller owner (TBD)** — a service must own the new NPC-controller election. The
  monster pattern lives in the dedicated `atlas-monsters`; there is no equivalent
  `atlas-npcs` live-state service today. Candidate homes (design decision, §9):
  `atlas-maps` (already tracks who is in a field), a new `atlas-npcs` service, or
  `atlas-channel` (where the spawn/control packets are emitted).
- **atlas-channel** — `spawnNPCForSession` (`kafka/consumer/map/consumer.go:593`) MUST stop
  unconditionally sending `NpcSpawnRequestController` to every session and instead send it
  only to the elected controller (FR-5.4).

## 8. Non-Functional Requirements & Design Constraints

- **Field is never carried on the buff event.** A character's field mutates over a buff's
  lifetime; GM-hide is effectively permanent (`HideBuffDuration = MaxInt32`) and the GM
  roams maps while hidden. The `channelId`/`mapId`/`instance` known at `APPLY` time is a
  stale snapshot by `EXPIRED`/`CANCEL` time (which also fire from stored buff state in
  `Cancel`/`CancelAll`/`ExpireBuffs`, where only `worldId` is retained). Acting on a stale
  field would relinquish controllers in the wrong map and miss the right one. The field is
  therefore resolved live from the location authority (`atlas-maps`) at event-handling
  time. `atlas-buffs` events stay a pure `{characterId, sourceId, world}` state-change
  signal.
- **Multi-replica correctness.** `atlas-monsters` runs multiple pods with Redis-backed
  state. Hide state consulted during election MUST be shared (Redis), not per-pod memory
  (FR-1.3). Buff-status consumption is consumer-group partitioned, so the pod that receives
  a hide event is generally not the pod running a given field's election — shared state is
  mandatory, not an optimization.
- **Goroutines / Redis / lint.** New async work MUST use `routine.Go`; new keyed Redis
  access MUST go through `libs/atlas-redis` (redis-key-guard); `tools/lint.sh --check` and
  the goroutine/redis guards MUST pass (CLAUDE.md Build & Verification).
- **Observability.** Log (debug) every relinquish/reassign/skip with character, field, and
  affected entity counts, sufficient to diagnose a "mobs frozen under hidden GM" report.
- **Game-data grounding.** GM-hide skill identity is v83-WZ-verified (`SuperGmHideId`
  9101004 present, `GmHideId` 9001004 absent). Re-verify against any other target version
  before extending.

## 9. Open Questions

- **OQ-1 (NPC-controller home).** Which service owns the new NPC-controller election and its
  shared state — `atlas-maps`, a new `atlas-npcs` service, or `atlas-channel`? This is the
  largest architectural decision and should be resolved in the design phase. Adding a new
  service triggers the full `docs/adding-a-new-service.md` checklist.
- **OQ-2 (NPC live-state materialization).** NPCs are static `atlas-data` life today. Does
  controller election require materializing per-field "live NPC" records, or can controller
  assignment key off the static `(field, npcObjectId)` alone?
- **OQ-3 (which NPCs need a controller).** Do all NPCs need single-controller election, or
  only movable ones? Confirm against client behavior whether static NPCs care about the
  controller grant. (Verify against v83 client / WZ life data, not assumption.)
- **OQ-4 (hidden-set lifecycle edges).** How is the hidden set reconciled on GM logout while
  hidden, channel change, or server restart (buff state persists in `atlas-buffs`; the
  hidden set is a derived cache)? Define a rebuild/repair path so the set can't leak stale
  entries that permanently bar a character from candidacy.
- **OQ-5 (reveal race).** task-156 documented an async-cancel vs sync-gated reveal race.
  Confirm ordering: the hidden-set removal on `EXPIRED` vs. any concurrent election must not
  leave a revealed GM permanently excluded or a mob permanently uncontrolled.

## 10. Acceptance Criteria

- [ ] A GM controlling monsters, on hiding, has those monsters reassigned to a visible
      player in the same map; the monsters continue to animate/aggro for that player.
- [ ] A hidden GM is never selected as a monster controller — on hide, on monster spawn, on
      entering a new map, or on another controller leaving.
- [ ] On reveal, the GM is eligible again and can be assigned control by the normal
      least-loaded rule (not force-assigned).
- [ ] A regular thief with Dark Sight active still controls monsters normally (not affected).
- [ ] `GmHideId` (9001004) is not handled; only `SuperGmHideId` (9101004) drives the feature.
- [ ] NPC-controller election exists: exactly one non-hidden controller per NPC per field;
      `NpcSpawnRequestController` is sent only to that controller.
- [ ] A hidden GM does not control NPCs; on hide their NPCs are reassigned; on reveal they
      are eligible again.
- [ ] Behavior is correct with ≥2 `atlas-monsters` replicas (hide state observed
      consistently regardless of which pod consumed the buff event).
- [ ] Location-lookup failure results in a skipped action + debug log, never an error crash
      or retry storm; the hidden-set mutation still applies.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...`, `tools/lint.sh --check`,
      redis-key-guard, and goroutine-guard all clean; `docker buildx bake` for every service
      whose `go.mod` changed.
- [ ] If a new service is introduced, `tools/service-registration-guard.sh` clean and
      `docs/adding-a-new-service.md` fully followed.
