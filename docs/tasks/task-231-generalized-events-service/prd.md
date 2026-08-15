# Generalized Events Service (`atlas-events`) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-15

---

## 1. Overview

Atlas has no home for gameplay behavior whose lifecycle spans multiple domains. A Crimson Balrog
attack during a transport voyage needs the transport domain's schedule, the monster domain's
spawning, the channel domain's packet fan-out, and a probabilistic decision that belongs to none of
them. An anniversary 2x-EXP period needs a calendar, an occurrence record, and a buff applied at
character login. Today each such feature would either be embedded into an unrelated service or
would justify a microservice of its own.

This task introduces `atlas-events`, a service that owns **event policy and orchestration**: what
should happen, and when. It owns event definitions and their administrative enablement, durable
scheduled work, event occurrences and their state progression, occurrence history, and a queryable
REST surface for other services that must interrogate current event state. It does **not** own the
gameplay primitives an event uses — monsters remain the monster domain's, buffs the buff domain's,
vessels the transport domain's, packet delivery `atlas-channel`'s.

The architecture is proven with two deliberately dissimilar events. **Crimson Balrog** is
Kafka-triggered, delayed, probabilistic, scoped to a concrete voyage in one channel of one world,
owns spawned monsters and a map visual, and has two distinct completion paths. **Anniversary** is
calendar-scheduled, long-running, has no meaningful stage progression, and grants a character
modifier at login. Together they exercise every part of the generic infrastructure without
attempting to model every MapleStory event.

### 1.1 Grounding notes

The requirements below were written against the current code and against the reference
implementation's behavior, not against recollection. Six findings materially shaped this PRD; each
is cited where it applies.

- **F1 — `ARRIVED`/`DEPARTED` are observation-deck visuals, not voyage lifecycle.**
  `services/atlas-transports/atlas.com/transports/transport/processor.go:132-190` emits
  `EVENT_TOPIC_TRANSPORT_STATUS` `ARRIVED` on the `OpenEntry` transition and `DEPARTED` on
  `InTransit`, each carrying only `routeId` and `ObservationMapId`. The end of a voyage
  (`InTransit → AwaitingReturn`) emits **no Kafka event at all** — it only warps en-route characters
  to the destination.
- **F2 — There is no voyage or vessel identity.** `transport.Model`
  (`transport/model.go:12-26`) is a route plus a schedule; state is re-derived per tick from the
  clock. The route registry is tenant-scoped with **no world or channel**
  (`transport/route_registry.go:15-27`). On departure the processor fans out over
  `chanP.GetAll()` and warps staging→en-route in every channel of every world, on a shared map id
  with a nil instance uuid.
- **F3 — The Crimson Balrog boat packet already exists and is verified.** It is `CONTI_MOVE`
  (`CField_ContiMove::OnContiMove`), distinct from the `CONTI_STATE` docked-ship visual already
  used for `ARRIVED`/`DEPARTED`. `libs/atlas-packet/field/clientbound/conti_move.go` implements it
  with verify markers pinned for gms_v79/v83/v84/v87/v95 and jms_v185; its golden test uses
  `NewContiMove(10, 4)` — exactly the enemy-ship-appears body. `ContiMoveWriter` is registered at
  `services/atlas-channel/atlas.com/channel/main.go:778`, `ContiMoveBody(state, subState)` exists at
  `services/atlas-channel/atlas.com/channel/socket/writer/conti_move.go`, and all nine seed
  templates route it. **No packet implementation work is in scope.**
- **F4 — Monster spawn/despawn is reusable; correlation is not.** `COMMAND_TOPIC_MONSTER` already
  carries `SPAWN_FIELD` and `DESTROY_FIELD`, and `EVENT_TOPIC_MONSTER_STATUS` emits
  `CREATED`/`KILLED`/`DESTROYED` with field, `uniqueId` and `monsterId`
  (`services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go:11-42`,
  `.../monsters/monster/kafka.go`). Neither carries any notion of what caused the spawn.
- **F5 — `atlas-rates` has the composition seam but no drop-rate mapping.** Rate factors are keyed
  by source string (`"world"`, `"channel"`, `"buff:<id>"`) and buff-derived factors come from
  `buffToRateMappings`, which today covers only `HOLY_SYMBOL→exp`, `MESO_UP→meso`, `CURSE→exp`
  (`services/atlas-rates/atlas.com/rates/kafka/message/buff/kafka.go:72-76`). There is no
  `item_drop` mapping. The stat constants needed — `EVENT_RATE`, `EXP_BUFF_RATE`,
  `ITEM_UP_BY_ITEM` — do already exist in `libs/atlas-constants/character/temporary_stat.go`.
- **F6 — Reference behavior for the Balrog attack.** From `Cosmic/scripts/event/Boats.js` and
  `Cosmic/src/main/java/tools/PacketCreator.java:4639-4651`: two Crimson Balrogs (monster
  `8150000`) spawn in the **same en-route deck map** the passengers ride (`200090010` To Orbis,
  `200090000` To Ellinia) — not a separate attack map. The cabins (`200090011`, `200090001`) receive
  neither monsters nor the visual. The attack broadcasts `CONTI_MOVE(10, 4)` and changes BGM to
  `Bgm04/ArabPirate`; arrival broadcasts `CONTI_MOVE(10, 5)` and kills all monsters. The reference
  rolls once at takeoff and schedules the approach at 3 min + rand(0–60 s), then spawns 5 s later.
  The reference has **no** monsters-eliminated completion path; that is a deliberate addition here
  (§10.6).

---

## 2. Goals

Primary goals:

- Establish generic event infrastructure — definitions, occurrences, durable scheduled work,
  occurrence history — that new events extend without modifying.
- Prove it with two structurally dissimilar events, Crimson Balrog and Anniversary.
- Give transport voyages a durable identity so an occurrence can be scoped to a concrete voyage in a
  concrete channel (F1, F2).
- Give spawned monsters a generic provenance identifier so any spawning mechanism — not only
  events — can correlate and clean up what it created (F4).
- Expose event state over REST for gameplay consumers, and over the Web UI for administrators.
- Keep every gameplay primitive owned by its existing domain service.

Non-goals:

- A generic JSON event scripting language, an administrator-defined trigger/action builder, or a
  universal rules engine.
- A universal high-frequency game tick.
- Replacing Party Quest functionality. Party Quests remain a distinct domain even though they share
  concepts (occurrences, stages, scheduling, Kafka reactions). Shared infrastructure may be
  extracted later; this task does not attempt it.
- A dedicated microservice per event.
- Coverage of every MapleStory event.
- Packet implementation work — the boat codec already exists and is verified (F3).
- Editing event-specific configuration through the Web UI beyond enable/disable (§12.2).

---

## 3. User Stories

- As a **server administrator**, I want to see every configured event definition and whether it is
  enabled, so I can control what is permitted to occur.
- As a **server administrator**, I want to enable or disable a definition without restarting a
  service, so I can turn seasonal content on and off.
- As a **server administrator**, I want to distinguish "Crimson Balrog is enabled" from "a Crimson
  Balrog attack is happening right now," because the first is true indefinitely and the second is
  rare.
- As a **server operator**, I want a durable log of every occurrence with its scope, outcome and
  completion reason, so I can answer "did the Balrog fire on world 2 channel 8 last night, and how
  did it end?"
- As a **server operator**, I want a scheduled event to fire correctly even if `atlas-events`
  restarted across its scheduled instant, so a deploy does not silently cancel the anniversary.
- As a **player**, I want the Crimson Balrog attack to appear during a voyage — the enemy ship
  visual, the music, and the monsters — and to see it whether I was already on the deck or walked in
  from the cabin.
- As a **player**, I want the anniversary EXP and drop bonus to be active from the moment I log in
  during the event, and to stop when the event ends without needing to log out.
- As a **developer**, I want to add a third, substantially different event by writing only its
  configuration, trigger handling, occurrence behavior and domain commands (§16).

---

## 4. Domain Principles

Two concepts anchor the domain:

> An **EventDefinition** describes what may happen. An **EventOccurrence** represents what actually
> happened or is happening.

Every event that actually takes place is represented by an occurrence. An enabled definition is
never, by itself, evidence that the event is occurring. Failed trigger evaluations produce no
occurrence, which preserves the meaning of the occurrence table as a history of real events.

The ownership rule for everything else:

> If a decision exists specifically because the event exists, it belongs to `atlas-events`. If an
> operation has independent meaning outside the event, its implementation stays with its existing
> domain owner.

| Behavior | Owner |
|---|---|
| Spawn a monster | `atlas-monsters` |
| Decide that N Balrogs should spawn for this voyage | `atlas-events` |
| Manage monster HP, combat and death | `atlas-monsters` |
| Decide that Balrogs should disappear because the vessel arrived | `atlas-events` |
| Operate a transport route and its schedule | `atlas-transports` |
| Decide whether an attack can occur on this voyage | `atlas-events` |
| Apply or cancel a character buff | `atlas-buffs` |
| Decide that Anniversary grants that buff | `atlas-events` |
| Compose EXP/drop rate factors | `atlas-rates` |
| Send map object and field packets | `atlas-channel` |
| Decide that the enemy ship should be visible | `atlas-events` |

---

## 5. Functional Requirements — Event Definitions

**FR-D1.** An `EventDefinition` is persisted with: `id` (uuid), `tenantId`, `type`, `name`,
`enabled`, `configuration` (event-specific), `createdAt`, `updatedAt`.

**FR-D2.** `configuration` is opaque to the generic definition domain. Only the event-specific
implementation for a given `type` interprets it. The generic layer must not switch on `type` to
reach event behavior (§16).

**FR-D3.** `enabled = true` means the definition is permitted to produce new occurrences when its
trigger and eligibility conditions are met. It does not mean an occurrence is active.

**FR-D4.** Disabling a definition prevents new trigger evaluations and new occurrences.

**FR-D5.** Disabling a definition does **not** terminate occurrences that are already `ACTIVE`.
Administrative termination of a live occurrence is out of scope for this task.

**FR-D6.** Definitions are validated on write against the schema their `type` declares. An
invalid configuration is rejected with a JSON:API error rather than persisted and failing later at
trigger time.

**FR-D7.** Definitions for both event types are seeded so a fresh environment has a disabled
Crimson Balrog definition and a disabled Anniversary definition present and visible in the UI.

---

## 6. Functional Requirements — Event Occurrences

**FR-O1.** An `EventOccurrence` is persisted with: `id` (uuid), `eventDefinitionId`, `tenantId`,
`type`, `state`, `stage`, `context` (event-specific), `startedAt`, `nextTransitionAt` (nullable),
`completedAt` (nullable), `completionReason` (nullable).

**FR-O2.** Generic states are `ACTIVE`, `COMPLETED`, `CANCELLED`, `FAILED`.

**FR-O3.** `stage` is event-specific and optional. An event with no meaningful progression uses
`ACTIVE → COMPLETED` and leaves `stage` null.

**FR-O4.** Occurrences are durable and survive service restart with their state, stage and context
intact.

**FR-O5.** Completed occurrences are retained indefinitely as history. Nothing in this task deletes
or archives them.

**FR-O6.** Every state and stage change appends an `EventOccurrenceTransition` row (§7).

**FR-O7.** Occurrence `context` is event-specific and opaque to the generic layer, but the generic
layer must be able to filter on a set of promoted scalar scope columns (§11.3) so gameplay queries
do not require scanning JSON.

---

## 7. Functional Requirements — Occurrence Transition History

**FR-T1.** An `EventOccurrenceTransition` is persisted with: `id`, `occurrenceId`, `tenantId`,
`fromStage` (nullable), `toStage`, `occurredAt`, `triggerType`, `triggerReference` (nullable).

**FR-T2.** A transition row is written for occurrence creation, every stage change, and terminal
completion. Example for Crimson Balrog:

```
12:05:03  OCCURRENCE_CREATED   trigger=SCHEDULED_WORK   ref=<workId>
12:05:03  → ATTACKING          trigger=OCCURRENCE_START
12:07:41  → COMPLETED          trigger=MONSTER_KILLED   ref=<uniqueId>
          completionReason=MONSTERS_ELIMINATED
```

and for Anniversary:

```
Aug 20 00:00  OCCURRENCE_CREATED  trigger=SCHEDULED_WORK  ref=<workId>
Aug 20 00:00  → ACTIVE             trigger=OCCURRENCE_START
Sep 01 00:00  → COMPLETED          trigger=SCHEDULED_WORK  ref=<workId>
              completionReason=SCHEDULED_END
```

**FR-T3.** `triggerReference` carries the identifier of whatever caused the transition — the
scheduled work id, the monster `uniqueId`, the voyage id — so an operator can trace backwards.

**FR-T4.** Transition history is exposed on the occurrence detail REST resource and rendered in the
Web UI occurrence detail view.

---

## 8. Functional Requirements — Scheduled Event Work

**FR-S1.** `ScheduledEventWork` is persisted with: `id`, `tenantId`, `eventDefinitionId`,
`eventOccurrenceId` (nullable), `type`, `context`, `executeAt`, `state`, `claimedBy` (nullable),
`claimedAt` (nullable), `attempts`, `lastError` (nullable).

**FR-S2.** `type` distinguishes the two purposes:
- `TRIGGER_EVALUATION` — "should an occurrence begin now?" Has no `eventOccurrenceId`.
- `OCCURRENCE_TRANSITION` — "should this occurrence advance or terminate now?" Has one.

They share the scheduling infrastructure without being the same domain operation.

**FR-S3.** States are `PENDING`, `PROCESSING`, `COMPLETED`, `CANCELLED`, `FAILED`.

**FR-S4.** The scheduler is a poller, not a per-event tick. It periodically selects work where
`state = PENDING AND executeAt <= now` and executes it. No correctness-critical timer exists solely
as an in-memory goroutine.

**FR-S5.** Scheduled work is durable across restart. Work whose `executeAt` passed while the service
was down is picked up and executed on recovery, late rather than lost.

**FR-S6.** Multiple `atlas-events` replicas must never execute the same work concurrently. Claiming
uses a database-level lock (`SELECT ... FOR UPDATE SKIP LOCKED` or equivalent) that atomically moves
rows `PENDING → PROCESSING` and stamps `claimedBy`.

**FR-S7.** Work claimed as `PROCESSING` but never terminal — because the replica holding it died —
is reclaimed after a configurable lease timeout and retried.

**FR-S8.** Execution is idempotent. Re-executing work must not create a duplicate occurrence, spawn
duplicate resources, or complete an occurrence twice. Idempotency is enforced by the work row's own
state transition being the commit point, plus the guards in §14.

**FR-S9.** Repeated failure increments `attempts`, records `lastError`, and after a configurable
maximum moves the row to `FAILED` with the reason retained. A `FAILED` work row never blocks other
work.

**FR-S10.** Cancelling a definition's pending work is possible — e.g. an Anniversary definition
whose start time is edited before it fires — via `CANCELLED`.

---

## 9. Functional Requirements — Transport Voyage Identity

This section exists because F1 and F2 mean the events service has nothing to attach an occurrence
to today. These changes are owned by `atlas-transports`.

**FR-V1.** A voyage (one trip of one route) gains a durable identity. `atlas-transports` assigns a
`voyageId` (uuid) at the moment a route transitions to `InTransit`, and retains it until the route
transitions to `AwaitingReturn`.

**FR-V2.** Because a route trip is realized simultaneously in every channel of every world
(F2), the voyage identity is per (route, trip). The **per-channel** realization is identified by the
tuple `(voyageId, worldId, channelId)`. An occurrence is scoped to that tuple, not to the bare
`voyageId` and not to the bare map id.

**FR-V3.** `EVENT_TOPIC_TRANSPORT_STATUS` gains a voyage lifecycle distinct from the existing
observation-deck visuals. The existing `ARRIVED` and `DEPARTED` event types keep their current
meaning and payload so `atlas-channel`'s `CONTI_STATE` broadcast is unchanged. Two new event types
are added:

- `VOYAGE_DEPARTED`, emitted on `OpenEntry`/`LockedEntry → InTransit`.
- `VOYAGE_ARRIVED`, emitted on `InTransit → AwaitingReturn`. **This event does not exist today** and
  is the trigger §10.7 depends on.

**FR-V4.** Both new events carry enough context to establish scope without a follow-up query:
`routeId`, `voyageId`, `worldId`, `channelId`, `stagingMapId`, `enRouteMapIds`, `destinationMapId`,
`observationMapId`, and the departure instant. They are emitted once per channel, matching the
existing per-channel fan-out at `processor.go:150-186`.

**FR-V5.** Voyage identity survives an `atlas-transports` restart for the duration of a trip, so a
Balrog occurrence created mid-voyage still matches the arrival event. It is stored alongside the
route in the existing tenant-scoped Redis registry.

**FR-V6.** No existing consumer of `ARRIVED`/`DEPARTED` changes behavior. The `atlas-channel` route
consumer (`services/atlas-channel/atlas.com/channel/kafka/consumer/route/consumer.go`) must ignore
the two new types.

---

## 10. Functional Requirements — Event 1: Crimson Balrog Transport Attack

### 10.1 Purpose

A probabilistic Balrog attack during an applicable voyage. It proves Kafka-triggered evaluation,
delayed evaluation, probability, runtime eligibility, voyage/channel scoping, event-owned monster
and visual resources, externally-driven completion, idempotent cleanup, and map-entry event lookup.

### 10.2 Definition configuration

The Crimson Balrog definition configuration must support at least:

```
applicableRouteIds     []uuid
triggerDelay           duration       (after voyage departure)
triggerDelayJitter     duration       (optional; reference uses 3min + rand(0-60s))
attackProbability      float [0,1]    (reference: 0.42)
monsterId              uint32         (reference: 8150000)
monsterCount           uint32         (reference: 2)
spawnPosition          {x, y}         per attack map
attackMapIds           []map.Id       the deck maps that get monsters + visual
relatedMapIds          []map.Id       cabins etc — count toward "aboard", get no visual
backgroundMusic        string         (optional; reference: "Bgm04/ArabPirate")
```

**FR-B1.** No value above is hard-coded in the generic infrastructure or in the event package.
Monster count in particular is configuration, not the constant 2 or 3.

### 10.3 Departure trigger

**FR-B2.** `atlas-events` consumes `VOYAGE_DEPARTED` (FR-V3). On receipt it:
1. Loads Crimson Balrog definitions for the tenant.
2. Skips any that are disabled.
3. Skips any whose `applicableRouteIds` does not contain the event's `routeId`.
4. Creates one `TRIGGER_EVALUATION` scheduled work row per surviving definition, with
   `executeAt = departedAt + triggerDelay (+ jitter)` and a context carrying the full voyage scope
   from FR-V4.

**FR-B3.** No `EventOccurrence` is created at departure.

**FR-B4.** Redelivery of the same `VOYAGE_DEPARTED` must not create a second scheduled work row.
Deduplication key: `(tenantId, definitionId, voyageId, worldId, channelId)`.

### 10.4 Delayed evaluation

**FR-B5.** When the scheduled work becomes due, the evaluation, in order:
1. Verifies the voyage is still underway. If it has already arrived, the work completes with no
   occurrence.
2. Re-checks that the definition is still enabled. If disabled since departure, the work completes
   with no occurrence.
3. Performs the configured probability roll.
4. If the roll succeeds, determines whether any character is currently aboard.

**FR-B6.** "Aboard" means present in **any** map belonging to the voyage in this channel — the
attack maps *and* the related maps including the cabin.

**FR-B7.** A failed roll marks the work `COMPLETED` and creates no occurrence.

**FR-B8.** A successful roll with no characters aboard marks the work `COMPLETED` and creates no
occurrence.

**FR-B9.** A successful roll with at least one character aboard creates a Crimson Balrog
`EventOccurrence` in state `ACTIVE`, stage `ATTACKING`.

### 10.5 Occurrence start

**FR-B10.** Occurrence context records at minimum: `routeId`, `voyageId`, `worldId`, `channelId`,
`attackMapIds`, `relatedMapIds`, `monsterId`, `monsterCount`, and the resolved spawn positions.

**FR-B11.** Starting the occurrence orchestrates, in the attack maps of this channel only:
1. Activation of the enemy-ship visual.
2. Background music change, if configured.
3. Spawning of `monsterCount` monsters of `monsterId`, correlated to the occurrence (§10.8).

**FR-B12.** The visual is realized by `atlas-channel` broadcasting `CONTI_MOVE` with
`state = 10, subState = 4` to the attack map (F3, F6). `atlas-events` does not construct packets; it
emits an event that `atlas-channel` consumes.

**FR-B13.** Related maps — cabins — receive neither monsters nor the visual (F6).

### 10.6 Map entry

**FR-B14.** When a character enters a map, `atlas-channel` determines whether an active occurrence
exposes visual state for that map in that world and channel, and if so sends the enemy-ship visual
and applies the configured music. Entering a cabin produces neither.

**FR-B15.** The lookup is a REST query against `atlas-events` scoped to the concrete
`(type, state, worldId, channelId, mapId)`, satisfiable by an indexed query (§11.3).

**FR-B16.** The map-entry lookup must not block or fail character map entry. If `atlas-events` is
unreachable, the character enters the map without the visual and the failure is logged.

### 10.7 Completion

**FR-B17.** The occurrence has two normal completion paths:

```
                    ATTACKING
                    /       \
      all Balrogs killed     voyage arrives
               |                  |
    MONSTERS_ELIMINATED      VESSEL_ARRIVED
               \                  /
                   COMPLETED
```

**FR-B18.** *Monsters eliminated.* `atlas-events` consumes monster `KILLED` and `DESTROYED` status
events. When no monsters correlated to the occurrence remain, it removes the enemy-ship visual,
restores the map's normal music, and completes the occurrence with
`completionReason = MONSTERS_ELIMINATED`. (This path does not exist in the reference implementation
and is a deliberate addition — F6.)

**FR-B19.** *Voyage arrival.* `atlas-events` consumes `VOYAGE_ARRIVED` (FR-V3). If an active
occurrence belongs to that `(voyageId, worldId, channelId)`, it despawns all remaining correlated
monsters, removes the visual, and completes with `completionReason = VESSEL_ARRIVED`.

**FR-B20.** Cleanup is idempotent and tolerates resources that already vanished. Arrival cleanup
must succeed when every Balrog was killed a second earlier, and the two completion paths racing must
produce exactly one completion.

**FR-B21.** `completionReason` is retained permanently and is visible in the REST resource and the
Web UI.

### 10.8 Event-owned monsters

**FR-B22.** Monsters spawned for an occurrence are correlated via the generic spawn provenance
fields of §13 — `spawnSourceType = EVENT`, `spawnSourceId = <occurrenceId>`.

**FR-B23.** `atlas-monsters` remains authoritative for the monster entities. `atlas-events` does not
maintain a parallel authoritative representation of them; it tracks only the correlation needed to
answer "are any left?" and "despawn everything from occurrence X."

---

## 11. Functional Requirements — Event 2: Anniversary

### 11.1 Definition configuration

```
scheduledStart   timestamp
scheduledEnd     timestamp
expMultiplier    float   (initial: 2.0)
dropMultiplier   float   (initial: 2.0)
```

**FR-A1.** The multipliers are configuration. Neither the generic infrastructure nor the event
package hard-codes 2.0.

### 11.2 Scheduled start

**FR-A2.** An enabled Anniversary definition has a `TRIGGER_EVALUATION` scheduled for
`scheduledStart`. When due, and if the definition is still enabled, it creates an occurrence in
state `ACTIVE` with no stage.

**FR-A3.** On occurrence creation, an `OCCURRENCE_TRANSITION` work row is scheduled for
`scheduledEnd`.

**FR-A4.** Both scheduled instants survive service restart (FR-S5). A start whose instant passed
during downtime still creates the occurrence, provided `scheduledEnd` has not also passed.

**FR-A5.** The occurrence is the authoritative fact that Anniversary is occurring. No other service
compares the definition's dates against its own clock to infer event state.

**FR-A6.** Enabling a definition whose `scheduledStart` is already in the past but whose
`scheduledEnd` is in the future creates the occurrence immediately.

### 11.3 Character login

**FR-A7.** A character entering gameplay while an Anniversary occurrence is active receives the
occurrence's configured buff.

**FR-A8.** The mechanism is reaction to the character-login Kafka event, not a synchronous query in
the login critical path. The design phase confirms the login event's timing guarantees against the
requirement that the modifier be established before meaningful gameplay; if reaction alone cannot
meet that, the design may add a query at channel bootstrap, but must not make login block on
`atlas-events` availability.

### 11.4 The buff

**FR-A9.** The modifier is a real buff applied by `atlas-buffs`, not a bespoke rate injection.

**FR-A10.** `atlas-rates` gains `buffToRateMappings` entries so the buff's stat changes compose into
the `exp` and `item_drop` rate types (F5). This includes the **first** `item_drop` mapping in the
table — `rate.TypeItemDrop` is already defined and consumed by the calculator, but nothing feeds it
today.

**FR-A11.** The temporary stat types are selected from the existing constants in
`libs/atlas-constants/character/temporary_stat.go` — `EVENT_RATE`, `EXP_BUFF_RATE` and
`ITEM_UP_BY_ITEM` are the candidates. Selection is a design-phase decision and must be
version-correct: `libs/atlas-constants` is version-blind, so the design confirms the chosen stats
are meaningful on the v83 baseline client before committing.

**FR-A12.** The buff carries the originating `eventOccurrenceId` as correlation, so completion can
cancel exactly the buffs that occurrence granted without relying on buff identity alone.

**FR-A13.** Because the buff is client-visible, its wire representation must be correct on the
baseline client and must not regress any verified packet cell. If the chosen stat is not
representable on a supported version, the design records that version's behavior explicitly rather
than emitting a malformed buff.

### 11.5 Scheduled end

**FR-A14.** At `scheduledEnd` the occurrence transitions to `COMPLETED` with
`completionReason = SCHEDULED_END`.

**FR-A15.** Completion explicitly cancels the modifier for every affected online character. It does
not wait for characters to log out or for buff durations to expire.

**FR-A16.** After completion, newly logging-in characters do not receive the buff.

**FR-A17.** The completed occurrence remains queryable as history.

---

## 12. Functional Requirements — Generic Spawn Provenance

This generalizes what §10.8 needs. The events service is one caller; cyclic spawn points and future
GM-triggered spawns are others.

**FR-P1.** `COMMAND_TOPIC_MONSTER` `SPAWN_FIELD` gains two optional fields:
- `spawnSourceType` — an enumerated string identifying the spawning mechanism. Initial values:
  `CYCLIC` (the normal spawn-point path), `EVENT`, `SCRIPT`, `GM`. Absent or empty is treated as
  `CYCLIC` for backward compatibility.
- `spawnSourceId` — an opaque identifier scoped to that type. For `EVENT` it is the
  `eventOccurrenceId`.

**FR-P2.** `atlas-monsters` persists both fields on the monster instance for its lifetime.

**FR-P3.** `EVENT_TOPIC_MONSTER_STATUS` echoes both fields on at least `CREATED`, `KILLED` and
`DESTROYED`, so a consumer can correlate without a follow-up query.

**FR-P4.** A new command, `DESTROY_BY_SOURCE`, despawns every live monster matching
`(spawnSourceType, spawnSourceId)` within a field. It succeeds when zero monsters match — the
idempotent-cleanup case of FR-B20.

**FR-P5.** These fields are additive and optional on the wire. Existing producers that omit them
continue to work unchanged, and existing consumers that ignore them are unaffected.

**FR-P6.** The generic monster domain gains no knowledge of event semantics. It never interprets
`spawnSourceId`; it only stores, echoes and matches on it.

---

## 13. API Surface

All resources follow existing Atlas JSON:API conventions, keep REST representations separate from
domain models, and are tenant-scoped from the request context.

### 13.1 Definitions

```
GET   /events/definitions
GET   /events/definitions/{id}
PATCH /events/definitions/{id}
```

**FR-API1.** `GET /events/definitions` lists definitions with pagination per
`docs/rest-pagination.md`. Filterable by `type` and `enabled`.

**FR-API2.** `PATCH /events/definitions/{id}` supports changing `enabled`. Configuration editing via
the API is out of scope beyond what enable/disable requires (§2 non-goals).

**FR-API3.** A definition resource exposes `id`, `type`, `name`, `enabled`, `configuration`,
`createdAt`, `updatedAt`.

### 13.2 Occurrences

```
GET /events/occurrences
GET /events/occurrences/{id}
```

**FR-API4.** An occurrence resource exposes `id`, `type`, `state`, `stage`, `context`, `startedAt`,
`nextTransitionAt`, `completedAt`, `completionReason`, and a relationship to its definition.

**FR-API5.** `GET /events/occurrences/{id}` includes the transition history (FR-T4), either inline
or as an included relationship per JSON:API convention.

**FR-API6.** `GET /events/occurrences` supports filtering by, at minimum: `definitionId`, `type`,
`state`, `worldId`, `channelId`, `mapId`, `voyageId`, and a `startedAt` range. Exact filter syntax
follows existing Atlas JSON:API convention rather than the illustrative query strings below.

**FR-API7.** Two gameplay queries must be efficient and indexed, not full scans:

```
# atlas-channel map entry (FR-B15)
GET /events/occurrences?filter[type]=CRIMSON_BALROG&filter[state]=ACTIVE
                       &filter[worldId]=1&filter[channelId]=4&filter[mapId]=200090010

# is Anniversary happening? (FR-A5)
GET /events/occurrences?filter[type]=ANNIVERSARY&filter[state]=ACTIVE
```

**FR-API8.** To make FR-API7 indexable without JSON scans, `worldId`, `channelId` and `voyageId` are
promoted to nullable scalar columns on the occurrence, and applicable map ids are stored in a
child table `event_occurrence_map` (`occurrenceId`, `mapId`, `visual` boolean). The `visual` flag is
what distinguishes the deck from the cabin at query time (FR-B13, FR-B14).

**FR-API9.** No REST endpoint exposes another tenant's definitions, occurrences or scheduled work
(§17).

---

## 14. Data Model

New service database, migrated with the project's existing GORM entity + administrator pattern. All
tables carry `tenant_id`.

| Table | Purpose | Key columns |
|---|---|---|
| `event_definition` | §5 | `id`, `tenant_id`, `type`, `name`, `enabled`, `configuration` (jsonb), timestamps |
| `event_occurrence` | §6 | `id`, `tenant_id`, `event_definition_id`, `type`, `state`, `stage`, `context` (jsonb), `world_id`, `channel_id`, `voyage_id`, `started_at`, `next_transition_at`, `completed_at`, `completion_reason` |
| `event_occurrence_map` | FR-API8 | `occurrence_id`, `map_id`, `visual` |
| `event_occurrence_transition` | §7 | `id`, `tenant_id`, `occurrence_id`, `from_stage`, `to_stage`, `occurred_at`, `trigger_type`, `trigger_reference` |
| `scheduled_event_work` | §8 | `id`, `tenant_id`, `event_definition_id`, `event_occurrence_id`, `type`, `context` (jsonb), `execute_at`, `state`, `claimed_by`, `claimed_at`, `attempts`, `last_error` |

Indexes required:

- `scheduled_event_work (state, execute_at)` — the poller's hot path (FR-S4).
- `event_occurrence (tenant_id, type, state)` — the Anniversary query (FR-API7).
- `event_occurrence_map (map_id)` joined against active occurrences filtered by world/channel — the
  map-entry query (FR-B15). The design phase confirms the exact composite shape against the query
  plan rather than assuming.
- Unique constraint enforcing FR-B4's deduplication key on pending trigger work.

Migration notes: this is a greenfield service, so there is no data migration. `atlas-monsters` gains
two nullable columns for §12; existing rows default to `spawnSourceType = CYCLIC`,
`spawnSourceId = null`.

---

## 15. Service Impact

| Service | Change |
|---|---|
| **`atlas-events`** (new) | Everything in §5–§11. New service: see `docs/adding-a-new-service.md`, including the Dockerfile `COPY libs/...` entries the bake step checks. |
| **`atlas-transports`** | §9 — voyage identity, `VOYAGE_DEPARTED` / `VOYAGE_ARRIVED`, per-channel emission, identity persisted across restart. Existing `ARRIVED`/`DEPARTED` unchanged. |
| **`atlas-monsters`** | §12 — spawn provenance fields, echo on status events, `DESTROY_BY_SOURCE`. |
| **`atlas-channel`** | Consume the events service's visual events → broadcast `CONTI_MOVE(10,4)` / `(10,5)` and the music change to the attack map. Map-entry lookup (FR-B14–16). Ignore the two new transport event types (FR-V6). |
| **`atlas-buffs`** | Accept and retain the `eventOccurrenceId` correlation on apply; support cancel-by-correlation (FR-A12, FR-A15). |
| **`atlas-rates`** | `buffToRateMappings` entries for the Anniversary stats, including the first `item_drop` mapping (FR-A10). |
| **`atlas-ui`** | §16 — Definitions and Occurrences admin surface. |
| **`libs/atlas-packet`** | **No change.** The codec exists and is verified (F3). |
| **`atlas-configurations`** | **No change expected.** `ContiMove` is already routed in all nine seed templates (F3). Confirm during design; if any template lacks it, add it. |

---

## 16. Web UI — Events Administration

**FR-UI1.** A new Events section with two views: **Definitions** and **Occurrences**.

**FR-UI2.** The Definitions list shows event name, type, enabled state, a configuration summary, and
scheduled start/end where the type has them.

**FR-UI3.** An administrator can enable and disable a definition from the list.

**FR-UI4.** The UI visually distinguishes "definition is enabled" from "an occurrence is active."
Crimson Balrog enabled with no attack in progress must not read as an event in progress. Where a
type can have at most one concurrent occurrence, the row may show live occurrence state; where it
can have many, the row links to the filtered occurrence list instead of implying a single state.

**FR-UI5.** The Occurrences list shows occurrence id, event name/type, state, stage, scope summary,
start time, completion time, and completion reason, with active occurrences readily distinguishable
from historical ones.

**FR-UI6.** The Occurrences list is filterable by definition/type, state, active-vs-historical, a
date range, and — where context permits — world and channel. Not every context property needs to be
a filter in this task.

**FR-UI7.** The occurrence detail view shows definition, id, state, stage, started, completed,
completion reason, full context, and the transition history (FR-T4).

**FR-UI8.** Crimson Balrog detail additionally shows world, channel, route, voyage, attack maps and
event-owned monster status. Anniversary detail additionally shows scheduled end and the configured
EXP and drop multipliers.

**FR-UI9.** The UI follows the frontend guidelines — JSON:API typing, TanStack Query, tenant
context, shadcn/ui, react-hook-form + Zod where forms exist.

---

## 17. Event-Specific Behavior Boundary

**FR-X1.** Generic infrastructure owns definitions, occurrences, transitions, persistence,
scheduling, lifecycle mechanics, administrative operations and common querying.

**FR-X2.** An event-specific implementation owns: identifying its triggers, evaluating its
conditions, building its occurrence context, reacting to relevant domain events, progressing its
stages, scheduling its future work, issuing its domain commands, determining completion, and
orchestrating its cleanup.

**FR-X3.** The generic layer must not contain a switch statement carrying the behavior of every
supported event. Event implementations register themselves against the generic layer; adding an
event does not edit a central dispatch table containing event logic. A registry mapping `type` to a
handler is acceptable; a `switch type { case CRIMSON_BALROG: ...spawn monsters... }` is not.

**FR-X4.** Suggested package shape, subject to design:

```
event/
  definition/
  occurrence/
  transition/
  scheduling/
events/
  crimsonbalrog/
  anniversary/
```

---

## 18. Non-Functional Requirements

### 18.1 Reliability

**FR-N1.** No correctness-critical event timer exists solely as an in-memory goroutine (FR-S4).

**FR-N2.** Reprocessing a Kafka message must not schedule duplicate evaluations, create duplicate
occurrences, spawn duplicate resources, or complete an occurrence more than once.

**FR-N3.** Scheduled work execution is retry-safe (FR-S8).

**FR-N4.** Cleanup tolerates already-vanished resources (FR-B20, FR-P4).

**FR-N5.** After restart the service recovers active occurrences, pending trigger evaluations,
pending occurrence transitions, and overdue work.

**FR-N6.** A single event implementation failing — a bad configuration, an unreachable dependency —
must not stall the scheduler for other events. Work is isolated per row.

### 18.2 Correlation and observability

**FR-N7.** The identifier chain is traceable end to end:

```
triggering domain event → scheduled work → occurrence → command → created domain resource
```

**FR-N8.** Structured logging covers: definition enabled/disabled, trigger received, scheduled
evaluation created, scheduled evaluation executed, evaluation rejected and why, occurrence created,
occurrence transitioned, command issued, occurrence completed, occurrence failed, cleanup started
and finished.

**FR-N9.** Log entries carry the applicable correlation identifiers: `tenantId`, `definitionId`,
`occurrenceId`, `scheduledWorkId`, `transactionId`, `triggerReference`, `voyageId`, `worldId`,
`channelId`.

**FR-N10.** Logs are queryable in Loki. Note the project's standing gotcha: Loki has no `app`
label — select on `service_name`.

### 18.3 Multi-world / multi-channel

**FR-N11.** Nothing assumes a single world or channel. Concurrent independent voyages across worlds
and channels each evaluate and occur independently:

```
World 1  Ch 1  Voyage A  → no occurrence (roll failed)
World 1  Ch 2  Voyage B  → occurrence #101
World 2  Ch 1  Voyage C  → occurrence #102
World 5  Ch 15 Voyage D  → evaluation pending
```

**FR-N12.** Occurrence scope identifies the concrete gameplay instance (FR-V2), never a bare map id.

### 18.4 Tenancy

**FR-N13.** Every definition, occurrence, transition, scheduled work row and query is tenant-scoped
per existing Atlas conventions.

**FR-N14.** An occurrence in one tenant never affects or appears as gameplay state in another.

### 18.5 Performance

**FR-N15.** The map-entry lookup (FR-B15) is on the character map-entry path. It must be served from
an index (FR-API8) and must not block map entry on failure (FR-B16).

**FR-N16.** The scheduler's poll interval and batch size are configurable. The poll query is indexed
(§14) and must not degrade as completed work accumulates.

### 18.6 Implementation conventions

**FR-N17.** Follows the Atlas backend conventions: immutable domain models, builder construction,
providers for database access, processors for business logic, administrators where complex database
coordination is required, Kafka producers/consumers for cross-service operations, separate
REST/domain representations, curried functional composition where consistent with existing code.

**FR-N18.** Business logic lives in processors, not in REST handlers or Kafka consumers. Handlers and
consumers delegate.

**FR-N19.** Before defining any new domain type, alias or numeric constant, check
`libs/atlas-constants/` for an existing equivalent (DOM-21).

**FR-N20.** Test setup uses the project's Builder pattern. No `*_testhelpers.go` files.

---

## 19. Open Questions

1. **Anniversary stat selection (FR-A11).** Which of `EVENT_RATE`, `EXP_BUFF_RATE`,
   `ITEM_UP_BY_ITEM` correctly represents a 2x EXP and 2x drop buff on the v83 baseline client, and
   what is the correct conversion method for each in `buffToRateMappings` — additive, direct, or
   fixed? Must be resolved against WZ/IDA evidence in design, not assumed.

2. **Login timing (FR-A8).** Does the character-login Kafka event fire early enough that the buff is
   established before the player can meaningfully act? If not, what is the fallback that does not
   make login depend on `atlas-events` availability?

3. **Voyage identity storage (FR-V5).** Redis registry alongside the route, or a small persisted
   table in `atlas-transports`? Depends on how much identity loss on a Redis flush matters.

4. **Music restoration (FR-B18).** The reference never restores the BGM on monsters-eliminated — it
   only resets on arrival, when everyone is warped out anyway. Since FR-B18 is a new completion path,
   what should the music do? Options: restore the map default, or leave it until arrival.

5. **Concurrent-occurrence policy per type.** Should the generic layer enforce a per-type maximum of
   concurrent occurrences (Anniversary: one; Crimson Balrog: one per voyage/channel), or is that
   entirely the event implementation's business? Affects FR-UI4.

6. **`DESTROY_BY_SOURCE` scope (FR-P4).** Field-scoped, or global per tenant? Field-scoped is
   sufficient for Crimson Balrog and cheaper to index; global is more useful for future GM tooling.

7. **Scheduled work retention.** Completed and failed work rows accumulate forever under the current
   requirements. Is a retention policy needed in this task, or deferred?

---

## 20. Acceptance Criteria

### 20.1 Definitions

- [ ] Definitions are persisted and survive restart.
- [ ] `GET /events/definitions` returns them, paginated and filterable by type and enabled.
- [ ] `GET /events/definitions/{id}` returns one.
- [ ] `PATCH /events/definitions/{id}` toggles `enabled`.
- [ ] A disabled definition initiates no new trigger evaluations and no new occurrences.
- [ ] Disabling a definition leaves an already-active occurrence running.
- [ ] Enablement is representationally distinct from active-occurrence state.
- [ ] Seeded Crimson Balrog and Anniversary definitions exist, disabled, in a fresh environment.

### 20.2 Occurrences

- [ ] Every event that actually takes place has exactly one occurrence.
- [ ] Failed trigger evaluations create no occurrence.
- [ ] Occurrences are persisted and active ones survive restart with state, stage and context.
- [ ] `GET /events/occurrences` and `/{id}` work, with the filters of FR-API6.
- [ ] Completed occurrences remain queryable indefinitely.
- [ ] Transition history is written for creation, each stage change, and completion.
- [ ] Transition history is exposed on the detail resource.

### 20.3 Scheduling

- [ ] Scheduled work is durable across restart.
- [ ] Due work is processed without any per-event tick.
- [ ] Work whose `executeAt` passed during downtime executes on recovery.
- [ ] Two replicas running concurrently never both execute the same work row (verified under a
      concurrent test, not by inspection).
- [ ] Work orphaned in `PROCESSING` by a dead replica is reclaimed after the lease timeout.
- [ ] Re-executing work produces no duplicate occurrence or duplicate effect.
- [ ] Repeated failure lands the row in `FAILED` with `lastError` retained, without stalling other
      work.

### 20.4 Transport voyage identity

- [ ] A voyage has a durable `voyageId` for the duration of a trip.
- [ ] `VOYAGE_DEPARTED` is emitted per channel with the full context of FR-V4.
- [ ] `VOYAGE_ARRIVED` is emitted on `InTransit → AwaitingReturn`, per channel.
- [ ] Existing `ARRIVED`/`DEPARTED` payloads and semantics are unchanged, and the `atlas-channel`
      `CONTI_STATE` broadcast still fires exactly as before.
- [ ] `atlas-channel` ignores the two new event types.
- [ ] Voyage identity survives an `atlas-transports` restart mid-trip.

### 20.5 Crimson Balrog

- [ ] An administrator can enable and disable the definition.
- [ ] An applicable voyage departure creates durable delayed trigger work; a non-applicable route
      creates none.
- [ ] The configured delay survives restart.
- [ ] Redelivery of `VOYAGE_DEPARTED` creates no second work row.
- [ ] The probability roll happens after the delay, not at departure.
- [ ] A failed roll creates no occurrence.
- [ ] A successful roll with nobody aboard creates no occurrence.
- [ ] A character in the cabin counts as aboard.
- [ ] A definition disabled between departure and evaluation creates no occurrence.
- [ ] A voyage that already arrived creates no occurrence.
- [ ] A successful eligible evaluation creates an occurrence scoped to the correct
      world/channel/voyage/maps.
- [ ] Exactly `monsterCount` monsters of `monsterId` spawn, in the attack map only.
- [ ] Spawned monsters carry `spawnSourceType = EVENT` and `spawnSourceId = <occurrenceId>`.
- [ ] Characters already on the deck receive the enemy-ship visual.
- [ ] Characters entering the deck afterward receive it.
- [ ] Characters entering the cabin receive neither the visual nor the monsters.
- [ ] Killing all event-owned monsters removes the visual and completes with
      `MONSTERS_ELIMINATED`.
- [ ] Voyage arrival despawns remaining monsters, removes the visual, and completes with
      `VESSEL_ARRIVED`.
- [ ] Arrival cleanup succeeds when all monsters were already killed.
- [ ] The two completion paths racing produce exactly one completion and one reason.
- [ ] Simultaneous voyages in different worlds/channels evaluate and occur independently.
- [ ] The completed occurrence is visible in the Web UI with its reason.

### 20.6 Anniversary

- [ ] An administrator can enable and disable the definition.
- [ ] Start and end times are configurable.
- [ ] An enabled definition creates an occurrence at its scheduled start.
- [ ] Enabling a definition whose start has passed but whose end has not creates the occurrence
      immediately.
- [ ] Scheduled activation survives restart.
- [ ] The active occurrence is queryable by type and state.
- [ ] A character logging in during the occurrence receives the buff.
- [ ] The buff yields the configured EXP multiplier through `atlas-rates`.
- [ ] The buff yields the configured drop multiplier through `atlas-rates`.
- [ ] A character logging in outside the occurrence receives no event buff.
- [ ] The occurrence completes at its scheduled end with `SCHEDULED_END`.
- [ ] Scheduled completion survives restart.
- [ ] Online characters stop receiving the modifiers at completion, without logging out.
- [ ] Characters logging in after completion receive no buff.
- [ ] The completed occurrence remains available historically.

### 20.7 Spawn provenance

- [ ] `SPAWN_FIELD` accepts `spawnSourceType` and `spawnSourceId`.
- [ ] Omitting them behaves exactly as today (`CYCLIC`).
- [ ] `atlas-monsters` persists both for the monster's lifetime.
- [ ] `CREATED`, `KILLED` and `DESTROYED` status events echo both.
- [ ] `DESTROY_BY_SOURCE` despawns all matching monsters and succeeds when zero match.
- [ ] `atlas-monsters` contains no event-specific logic.
- [ ] Existing monster spawn behavior is unchanged — verified by existing tests still passing
      unmodified.

### 20.8 Web UI

- [ ] Administrators can view definitions and toggle enablement.
- [ ] Administrators can view active and historical occurrences.
- [ ] State, stage, context and completion reason are visible.
- [ ] Transition history is visible on the detail view.
- [ ] Crimson Balrog occurrences show world, channel, route, voyage, attack map and monster status.
- [ ] Anniversary occurrences show schedule and multipliers.
- [ ] Occurrences are filterable per FR-UI6.
- [ ] "Enabled" is never rendered in a way that reads as "occurring."

### 20.9 Architecture validation

- [ ] A third, substantially different event could be added by supplying only its configuration
      schema, trigger handling, occurrence behavior and domain commands/reactions — with no change
      to generic definition handling, occurrence persistence, the scheduler, the REST resource
      architecture, the UI structure, or any unrelated gameplay service. Demonstrated by a written
      walkthrough in the design document, not by implementing a third event.
- [ ] No generic-layer switch statement contains event behavior (FR-X3).

### 20.10 Gate

- [ ] `tools/verify.sh` (flagless) exits 0, including the docker bake for every touched service.
- [ ] Code review has run per the project's three-reviewer pattern before the PR is opened.

---

## 21. Architecture Summary

```
                 ┌───────────────────────┐
Kafka Events ───►│   Event Definitions   │
 VOYAGE_DEPARTED │   trigger evaluation  │
                 └───────────┬───────────┘
                             │ scheduled evaluation
                             ▼
                    ┌────────────────┐
                    │ Durable Event  │   claim-locked poller,
                    │   Scheduler    │   PENDING / executeAt <= now
                    └───────┬────────┘
                            │ criteria met
                            ▼
Kafka Events ──────►┌────────────────┐◄──── scheduled transitions
 VOYAGE_ARRIVED     │ EventOccurrence│
 MONSTER_KILLED     │ + event logic  │
 CHARACTER_LOGIN    └───────┬────────┘
                            │ commands
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
        atlas-monsters  atlas-buffs   atlas-channel
        (SPAWN_FIELD    (apply/cancel  (CONTI_MOVE
         + provenance)   + occurrence   10/4, 10/5)
                          correlation)
```

with read-oriented interrogation alongside:

```
atlas-channel map entry  ──query──►  GET /events/occurrences
NPC / gameplay service                 ?type=…&state=ACTIVE&mapId=…
                                            │
                                            ▼
                                  applicable active occurrences
```
