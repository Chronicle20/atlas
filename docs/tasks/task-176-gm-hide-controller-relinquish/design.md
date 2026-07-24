# GM-Hide Relinquishes Monster & NPC Controller Eligibility — Design

Task: task-176-gm-hide-controller-relinquish
Status: Proposed
PRD: [prd.md](prd.md)

---

## 1. Summary of Decisions

| # | Question | Decision |
|---|----------|----------|
| D1 | OQ-1: NPC-controller home | **atlas-channel** owns NPC-controller election, Redis-backed. No new service. |
| D2 | OQ-2: live-NPC materialization | **None.** Controller assignment keys off `(field, npcObjectId)` from static data; the Redis registry entry *is* the only live state. Uncontrolled = absent from the registry. |
| D3 | Hidden-state mechanism | **atlas-monsters:** Redis `TenantKeyedSet` projection of GM-hidden characters (PRD FR-1.3 mandate). **atlas-channel:** no second projection — the buff event itself is the trigger, and candidacy is winner-checked against atlas-buffs REST via the existing `buff.IsGmHidden` path. |
| D4 | OQ-4: hidden-set lifecycle | Derived cache; idempotent SADD/SREM; leader-gated periodic reconciliation task in atlas-monsters sweeps the set against atlas-buffs `GET /characters/{id}/buffs`. Fail-open: deleting the key restores pre-task behavior. |
| D5 | OQ-5: reveal race | Strict ordering: set mutation **before** any location-dependent action; hidden checks always read live Redis at election time; the reveal handler's own re-election sweep covers mobs orphaned by a concurrent stale-read election. No permanent-exclusion state is possible. |
| D6 | OQ-3: which NPCs | **All NPCs** get controller election (parity with today's behavior where every NPC gets the controller packet; no movable/static filtering). |

Two required companion changes surfaced during context review:

1. **NPC movement/animation relay.** `services/atlas-channel/atlas.com/channel/movement/processor.go` `ForNPC` echoes NPC movement **only to the moving character's own session** (`sp.IfPresentByCharacterId`). Today that is invisible because *every* client is granted control and animates NPCs locally. Under single-controller election, non-controller clients receive no NPC movement at all — NPCs would freeze for them. The controller's NPC action packets must be broadcast to the other sessions in the field.
2. **Controller-revoke packet.** `libs/atlas-packet/npc/clientbound/spawn_request_controller.go` hard-codes the leading byte to `1` (`CNpcPool::OnNpcChangeController`, grant arm). FR-6.1's revoke needs the remove-controller arm as a new codec. Its layout MUST be derived from the IDB per `docs/packets/IMPLEMENTING_A_PACKET.md` — not assumed.

## 2. D1 — NPC-Controller Election Lives in atlas-channel

### Alternatives considered

**(A) atlas-channel (chosen).** Election runs inline in the paths that already exist:

- Channel already consumes `MAP_STATUS` enter/exit (`kafka/consumer/map/consumer.go`) and does per-session NPC spawning there (`spawnNPCForSession`, line ~584).
- Channel already consumes `EVENT_TOPIC_CHARACTER_BUFF_STATUS` (`kafka/consumer/buff/consumer.go`) and routes events by **session presence** (`session.NewProcessor(...).IfPresentByCharacterId`), which also yields the GM's current field from `s.Field()` — no atlas-maps lookup needed for the NPC half.
- Channel already owns the NPC static-data processor (`data/npc`), the hide semantics from task-156 (`character/buff/hidden.go` `IsGmHidden` + the buff-REST fetch pattern used at `kafka/consumer/map/consumer.go:473`), and every NPC packet writer.
- The claim can be made **synchronously inside the spawn path**, so `NpcSpawn` → `NpcSpawnRequestController` are written to the same session in order. This eliminates by construction the cross-service spawn-vs-control ordering race class that atlas-monsters needed two documented workarounds for (`monster/processor.go` Create() in-place assignment comment; `spawnMonsterForSession` re-issue comment).
- NPC-controller state has **exactly one consumer**: atlas-channel's packet decisions. No other service reads it.

**(B) New atlas-npcs service.** Clean structural parity with atlas-monsters (registry + MAP_STATUS consumer + buff consumer + NPC_STATUS topic consumed by channel). Rejected: full `docs/adding-a-new-service.md` checklist and permanent deploy surface for a domain whose only live state is one small map; reintroduces the Kafka spawn-vs-control ordering race and would force the same mitigations monsters carries; channel still has to change anyway (spawn path, movement relay, revoke packet), so the new service saves nothing in channel.

**(C) atlas-maps.** It is the field-membership authority, but has no NPC knowledge, would still need a new Kafka topic + channel-side consumer (same race as B), and turns a location service into a controller-election engine. Rejected as scope creep.

The domain-purity objection to (A) — "channel is transport" — is acknowledged but outweighed: NPC control is intrinsically a *client presentation* concern (which client runs NPC AI/animation), it has no cross-service consumer, and the state is tiny. If a live-NPC domain ever emerges (scripted NPC movement, spawned NPCs), extraction into a service is a bounded refactor because the registry is isolated behind one package (§5.1).

### Multi-replica correctness (FR-5.5)

Controller state lives in Redis via `libs/atlas-redis` (`KeyedHash`), never pod memory. Channel pods route work by session presence, so the pod serving the GM's channel performs the mutation, and any pod observing the field reads the same registry. `services/atlas-channel/atlas.com/channel/go.mod` already carries the `atlas-redis` replace line; the `require` is added.

## 3. D3/D4/D5 — Hidden-State Signal

### 3.1 atlas-monsters: Redis hidden-set projection

New consumer package `kafka/consumer/buff` in atlas-monsters (registered in `main.go` with the service's shared `consumerGroupId`, like the existing `_map` consumer — shared-group is correct precisely because the effect is a shared-Redis mutation, per PRD §8):

- Handles `StatusEvent[AppliedStatusEventBody]` / `StatusEvent[ExpiredStatusEventBody]` from `EVENT_TOPIC_CHARACTER_BUFF_STATUS`.
- Filter: `Type == APPLIED|EXPIRED` **and** `Body.SourceId == int32(skill.SuperGmHideId)` (atlas-constants; same constant task-156 uses). Everything else ignored (FR-1.2). `atlas-buffs` emits only these two status types — cancel paths (`Cancel`/`CancelAll`/`ExpireBuffs`) all emit `EXPIRED`, so the toggle-off path (which arrives as a buff cancel, per task-156) is covered.
- New registry package `character/hidden` wrapping `atlasredis.TenantKeyedSet[string]`, namespace `hidden-characters`, one set per tenant, members = decimal `characterId`. `Add`/`Remove` (SADD/SREM, idempotent per FR-1.4), `MemberSet` (fetch-once per election).

**Handler flows (ordering is load-bearing, D5/FR-7.2):**

APPLIED:
1. `hidden.Add(characterId)` — always first.
2. Resolve current field: `GET /characters/{id}/location` on atlas-maps (new request in monsters' `map` package alongside `charactersInFieldUrl`; exact route confirmed against `services/atlas-maps/atlas.com/maps/character/location/resource.go` at plan time). On error or not-in-map: debug log, stop (FR-7.1) — the set mutation already happened (FR-7.2).
3. Snapshot `ControlledByCharacterInFieldProvider(f, characterId)` **once** into a `FixedProvider` — same snapshot-first pattern (and for the same provider-re-evaluation reason) as `handleStatusEventCharacterExit` in `kafka/consumer/map/consumer.go`.
4. `StopControl` each, then `FindNextController(CharacterIdsInFieldProvider(f))` each. The candidate filter (§4) guarantees the hiding GM cannot be re-picked because step 1 ran first.

EXPIRED:
1. `hidden.Remove(characterId)` — always first.
2. Resolve field as above; on error, debug log + stop.
3. `model.ForEachSlice(NotControlledInFieldProvider(f), FindNextController(CharacterIdsInFieldProvider(f)), model.ParallelExecute())` — restores candidacy only; no forced transfer (FR-3.2).

**Reveal race (D5, OQ-5):** an election on another pod that read the set between SREM and step 3 may skip the just-revealed GM and (if the GM was the only candidate) leave a mob uncontrolled — but step 3 of the same EXPIRED handler runs *after* SREM and sweeps exactly the uncontrolled mobs of that field. Conversely an election reading the set after SREM simply includes the GM. Exclusion is never cached per-election-cycle anywhere; every election reads Redis live, so no path can leave a character *permanently* excluded short of a lost EXPIRED event — which reconciliation (§3.3) repairs.

### 3.2 atlas-channel: no second projection

The NPC half needs hide state in two situations, and neither needs a set:

- **Transition triggers** (relinquish on hide, re-elect on reveal): the buff event itself says what changed; channel's existing buff consumer shape (`IfPresentByCharacterId` + `s.Field()`) locates the GM. If the GM's session is gone by consumption time, the no-op is correct — their map-exit already released their NPCs.
- **Candidacy exclusion** (FR-6.2): elections pick the least-loaded candidate, then **winner-check** them — fetch that one character's buffs (existing channel pattern: `buff.GetByCharacterId` + `IsGmHidden`, exactly as `spawnCharacterForSession` does) and step to the next candidate if hidden. NPC elections are rare (map enter/exit, hide/reveal) and hidden GMs are ~0–1 per field, so this is typically zero or one REST call; it avoids duplicating the projection *and* its reconciliation machinery in a second service.

### 3.3 D4 — hidden-set lifecycle & reconciliation (OQ-4)

- **Logout while hidden:** if atlas-buffs cancels buffs at logout, `EXPIRED` removes the entry. If the buff persists offline, the entry persists — harmless, because an offline character appears in no field's candidate pool, and still-correct on re-login (still hidden → still excluded).
- **Channel change / map change:** buffs survive; entry correctly persists.
- **Restarts:** Redis and atlas-buffs state both survive; the set needs no rebuild.
- **Drift repair (lost events):** a periodic task in atlas-monsters (the service already has leader election — `leaderconfig.go` — and a `tasks/` home), leader-gated, interval ~5 min: for each member of the hidden set, `GET /characters/{id}/buffs` from atlas-buffs; remove members with no active SuperGmHide buff. The set holds only currently-hidden GMs, so the sweep is a handful of calls at most. The inverse direction (hidden in atlas-buffs but missing from the set) is *not* swept: the failure is "hidden GM can be elected", i.e. pre-task behavior, and it self-heals on the next APPLIED/EXPIRED. Same fail-open property for ops: `DEL` on the Redis key degrades to current-main behavior, never breaks control.
- **Accepted limitation:** a *lost APPLIED event* also means the relinquish action never ran (GM keeps controlling mobs while hidden) until the GM changes maps or reveals. Repairing that from the sweep would need location resolution and relinquish logic in a task context; not worth it for a double-failure case.

## 4. Monster Half — atlas-monsters Changes

All election paths already funnel through one choke point, which is where exclusion goes:

**`getControllerCandidate` (`monster/processor.go:260`):**
1. Fetch `hidden.MemberSet()` once per call.
2. **Puppet bias (FR-4.2):** skip the vicinity owner if hidden.
3. **Pool (FR-4.1):** drop hidden ids when seeding `controlCounts` from `idp`.
4. **Empty pool (FR-4.3):** the current `index == 0` → `errors.New("should not get here")` becomes a typed sentinel `ErrNoControllerCandidate`. An empty pool is now a legitimate outcome (field contains only hidden GMs, or the hidden filter emptied it).
5. Redis read failure: log warn and proceed **unfiltered** (fail-open — degrade to pre-task behavior rather than leaving mobs uncontrolled).

**`FindNextController` (`monster/processor.go:305`):** treat `ErrNoControllerCandidate` as a debug-logged no-op success (mob stays uncontrolled; next enter/reveal trigger re-elects). Other errors keep current behavior. This automatically covers every existing trigger: `Create` initial assignment, character-enter, character-exit reassignment, and the new hide/reveal handlers — a hidden GM entering a new map cannot grab control, and a hidden GM's exit reassigns around them, with no per-call-site changes.

New surface in atlas-monsters:

| Piece | Location | Notes |
|---|---|---|
| Hidden registry | `character/hidden/` (new) | `TenantKeyedSet` wrapper; `sync.Once` singleton like `GetMonsterRegistry` |
| Buff-status consumer | `kafka/consumer/buff/` (new) + `kafka/message/buff/` message defs | Mirrors `_map` consumer registration in `main.go` |
| Location REST client | `map/requests.go` + location rest model | `GET /characters/{id}/location` from `MAPS` root URL |
| Reconciliation task | `tasks/` + atlas-buffs REST client | Leader-gated; interval configurable |

`go.mod` expected unchanged (atlas-redis, atlas-kafka, atlas-rest, atlas-constants already present) — confirmed against `services/atlas-monsters/atlas.com/monsters/go.mod`; if it changes anyway, `docker buildx bake atlas-monsters` per CLAUDE.md.

## 5. NPC Half — atlas-channel Changes

### 5.1 Controller registry (new package, e.g. `npc/controller`)

Redis `KeyedHash` (exists in `libs/atlas-redis/hash.go`), namespace `npc-controller`, key derived from tenant + field (same suffix discipline as monsters' `mapIdx`: `atlas:npc-controller:<tenantId>:<world>:<channel>:<map>:<instance>`), hash field = NPC objectId (`data/npc` `Model.Id()`), value = controller characterId.

- **Uncontrolled = absent** (D2). No live-NPC record is materialized; static data remains the NPC source of truth.
- **Claim** must be atomic across replicas/concurrent enters: `HSETNX`. `KeyedHash` has no `SetNX` today — add it to `libs/atlas-redis` (its legal home under redis-key-guard) with a test.
- **Release** = `HDEL`. Redis removes empty hashes automatically, so per-instance keys cannot leak after the field empties; no teardown sweep is needed.
- **Stale-entry self-repair:** an entry whose controller is no longer in the field (crashed pod, missed exit) is repaired lazily — any election/spawn that observes a controller absent from the field's current sessions re-claims (§5.2). No TTLs.
- Count-per-controller for least-loaded election comes from `GetAll` on the field's hash (NPC counts per map are small).

### 5.2 Election triggers

**Map enter** — inside the existing enter path next to `spawnNPCForSession` (`kafka/consumer/map/consumer.go:222`):
- `NpcSpawn` goes to the entering session for every NPC (unchanged).
- Then, per NPC: if the registry entry is empty **or stale** (recorded controller not in the field's sessions), attempt `HSETNX`-claim for the entering character *after* winner-checking them for hide (§3.2 — an entering hidden GM claims nothing). If this session is (or just became) the controller, send `NpcSpawnRequestController` — same session, strictly after its `NpcSpawn` (FR-5.4; no ordering race).

**Map exit** — in the existing exit handler (same consumer): release all entries held by the exiting character in that field; for each, elect the least-loaded non-hidden remaining session (winner-check), claim, and announce `NpcSpawnRequestController` to the new controller. No candidates → leave released (uncontrolled).

**Hide APPLIED** (new branch in `kafka/consumer/buff/consumer.go`, keyed on `SourceId == SuperGmHideId`, inside the existing `IfPresentByCharacterId` block): release the GM's entries in `s.Field()`, send the **revoke packet** (§5.3) to the GM for each, reassign to non-hidden others as in map-exit (FR-6.1). The relinquishing character is excluded from the candidate pool by construction.

**Hide EXPIRED** (same consumer): for each *uncontrolled* NPC in `s.Field()`, run the enter-style election with the revealed GM in the pool. Least-loaded still decides — no forced transfer (FR-6.3).

### 5.3 Packets

- **Grant:** existing `NpcSpawnRequestControllerWriter` — unchanged wire format, but now sent only to the elected controller (`spawnNPCForSession` loses its unconditional second Announce).
- **Revoke:** new clientbound codec for the remove-controller arm of `CNpcPool::OnNpcChangeController` (the existing struct hard-codes the grant byte `1`). Layout derived from the IDB and implemented per `docs/packets/IMPLEMENTING_A_PACKET.md` (packet-implementer flow, fixtures + matrix row); this is the only packet-lib change and it is additive.

### 5.4 Movement & animation relay (companion change 1)

`movement/processor.go` `ForNPC` and the non-movement branch of `socket/handler/npc_action.go`:

- **Guard:** drop NPC action packets from a session that is not the NPC's current controller (registry check) — prevents non-controllers (and spoofers) from animating NPCs.
- **Relay:** broadcast `NpcActionMove` / `NpcActionAnimation` to the *other* sessions in the field (`ForOtherSessionsInMap`, the same shape `ForPet` already uses) **in addition to** the existing echo to the controller. Without this, single-controller election would freeze NPCs for every non-controller client.

## 6. Error Handling (FR-7)

| Failure | Behavior |
|---|---|
| atlas-maps location 404/error (monster half) | Debug log, skip relinquish/re-elect; hidden-set mutation already applied (ordering §3.1) |
| Redis read fails during election | Warn log, proceed unfiltered (fail-open to pre-task behavior) |
| Redis write (claim/release) fails | Warn log, skip that NPC; lazily repaired by the next election trigger via stale-entry re-claim |
| atlas-buffs REST fails during winner-check | Warn log, treat candidate as not hidden (fail-open) |
| GM session absent when buff event consumed (channel) | No-op; map-exit already released their NPCs |
| No eligible candidate anywhere | Entity left uncontrolled (FR-4.3 / FR-5.3); debug log with field + count |

No retry loops anywhere; every skip is converged by the next election trigger (enter/exit/hide/reveal/reconciliation).

## 7. Observability

Debug logs on every relinquish/reassign/skip carrying characterId, field, and affected-entity count (PRD §8): monster hide-relinquish (N mobs), monster reveal-sweep (N uncontrolled), NPC claim/release/reassign/revoke, hidden-set add/remove, reconciliation removals (warn — it means an event was lost), and every fail-open path. Enough to diagnose "mobs frozen under hidden GM" from logs alone.

## 8. Testing

**atlas-monsters (unit, existing miniredis-style registry tests + Builder pattern):**
- `getControllerCandidate`: hidden excluded from pool; hidden puppet owner skipped; only-hidden field → `ErrNoControllerCandidate`; Redis failure → unfiltered fallback.
- `FindNextController`: sentinel → no-op success, no error spam.
- Buff handler: SourceId filter (Dark Sight ids ignored — acceptance criterion); idempotent add/remove; mutation-before-location ordering (location failure still mutates set); snapshot-then-reassign flow.
- Reconciliation: removes stale member, keeps active member.

**atlas-channel (unit):**
- Registry: HSETNX claim wins once under concurrent claim; release; stale-entry re-claim.
- Election: least-loaded pick; winner-check steps past hidden candidate; entering hidden GM claims nothing; exit reassignment; hide releases + revokes + reassigns; reveal elects only uncontrolled.
- `spawnNPCForSession`: controller session gets spawn+grant in order; non-controller gets spawn only.
- Movement guard: non-controller NPC action dropped; controller action relayed to others.

**libs/atlas-redis:** `KeyedHash.SetNX` semantics test.

**Packet lib:** byte-fixture tests for the revoke codec per the packet playbook (its own verify flow).

**Acceptance:** PRD §10 walked live on a ≥2-replica atlas-monsters deploy (hide on one pod's consumer, election on another).

## 9. Facts to Confirm at Plan Time

Verified in this design: buff event shape/types (`services/atlas-buffs/.../kafka/message/character/kafka.go` — APPLIED/EXPIRED only), channel buff-consumer session routing, `ForNPC` self-echo behavior, `KeyedHash` existence and missing `SetNX`, monsters' election choke point and snapshot pattern, grant packet's hard-coded `1`. Still to confirm during planning (marked, not assumed):

1. Exact atlas-maps location route shape (`character/location/resource.go`) for the monsters-side client.
2. atlas-buffs removes buff state *before* emitting `EXPIRED` (winner-check on a just-revealed GM must not still see the hide buff). If ordering is inverse, the reveal branch passes the revealed GM into the pool explicitly (the event already proves the state change), so this is a one-line hedge, not a redesign.
3. Channel consumer-group uniqueness per pod (implied by session-presence routing; confirm in channel `main.go`).
4. IDB read-order of the `OnNpcChangeController` remove arm (packet playbook step).
5. Whether atlas-buffs cancels buffs on logout (affects only which of two safe behaviors §3.3 exhibits).
