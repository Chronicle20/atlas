# Player NPCs (Imitated NPCs) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21

---

## 1. Overview

MapleStory's Hall of Fame maps are populated with "Player NPCs" — persistent, non-interactive map
objects that render as a *character avatar* rather than an NPC sprite. When a character reaches its
class's maximum level, the server deploys one of these into the job's Hall of Fame map, where it
stands permanently as a record of that achievement, wearing the equipment the character had at the
moment of deployment. Atlas has never implemented this: a repository-wide search for
`PlayerNpc|PlayerNPC|ImitatedNPC|imitated` across `services/` and `libs/` returns zero Go hits, and
the two packets that carry the feature — `IMITATED_NPC_RESULT` and `IMITATED_NPC_DATA`
(`docs/packets/audits/STATUS.md:131,134`) — are ❌ on every version column with no implementation.
Reaching max level in Atlas today produces nothing.

The client-side mechanism already exists in every targeted client and is well understood. A Player
NPC is spawned as an ordinary NPC over `SPAWN_NPC_REQUEST_CONTROLLER`, but with a *script id* whose
`Npc.wz` entry carries `info/imitate = 1` and 1×1 placeholder `stand`/`say` canvases. On seeing an
`imitate` NPC, the client renders a character avatar assembled from a follow-up `IMITATED_NPC_DATA`
packet (name, gender, skin, face, hair, equipped items, masked/cash equips, cash weapon) instead of
drawing an NPC sprite. The client-side handler is `CNpcPool::OnNpcImitateData`. Because the
placeholder canvases are 1×1, spawning an imitate script id *without* sending the follow-up data
yields an invisible object, and spawning a script id that does not exist in `Npc.wz` at all is a
known cause of client crashes.

This feature is a persistence-and-placement problem more than a packet problem. The server must
allocate a script id from a bounded pool of imitate-capable ids, choose a position on the target map
that does not collide with the Player NPCs already standing there, snapshot the character's
appearance, store all of it durably per tenant and world, and re-broadcast it to every character who
enters that map thereafter. This PRD specifies a dedicated service, `atlas-player-npcs`, to own that
state, with `atlas-channel` owning the wire and `atlas-character` supplying the trigger.

---

## 2. Goals

Primary goals:

- Deploy a Player NPC into the correct Hall of Fame map when a non-GM character reaches its class's
  maximum level, without operator intervention.
- Allow eligible characters to deploy manually by talking to a 1st-job instructor when automatic
  deployment is disabled for the tenant.
- Persist deployed Player NPCs per tenant and world so they survive service restarts and appear for
  every character who enters the map.
- Render deployed Player NPCs correctly on every supported client version — avatar appearance,
  equips, and position — via the `IMITATED_NPC_DATA` codec and the existing NPC spawn path.
- Place Player NPCs using the reference server's positioning algorithms (grid positioner and podium
  positioner), including gap-shrink steps and re-organization on overflow.
- Populate world rank, overall rank, world-job rank, and overall-job rank from real ranking data.
- Provide GM commands to deploy and remove a Player NPC by character name.

Non-goals:

- Producing or patching client `Npc.wz` assets. The pool of usable script ids is whatever the
  tenant's client already ships; the server never assumes ids beyond it.
- Making Player NPCs conversational, shoppable, or otherwise interactive. They are display-only.
  (The `rank_user` script referenced by the WZ entries is out of scope.)
- Live appearance tracking. A deployed NPC's look is frozen; it changes only on an explicit
  re-deploy.
- The 3rd-job instructor NPC conversations themselves (a separate documented gap in
  `docs/research/missing-features/npc-content.md`).
- Player NPCs on non-Hall-of-Fame maps by automatic deployment. GM deployment on an arbitrary map is
  in scope but is bounded by script-id availability (see §4.3).

---

## 3. User Stories

- As a player who reaches max level, I want a statue-like version of my character to appear in my
  job's Hall of Fame so that the achievement is visible to everyone on the world.
- As a player visiting a Hall of Fame map, I want to see every previously deployed Player NPC,
  correctly positioned and wearing the right equipment, every time I enter the map.
- As a player on a tenant where automatic deployment is disabled, I want to deploy my Player NPC by
  talking to my 1st-job instructor, and I want the instructor to tell me when I am not eligible.
- As a player who already has a Player NPC on a map, I want a second visit to the instructor to
  refresh its appearance to my current look rather than create a duplicate.
- As a GM, I want to deploy a Player NPC for a named character at my current position, and to remove
  a named character's Player NPCs, so that I can fix bad deployments and stage events.
- As an operator, I want to enable or disable automatic deployment and tune Player NPC spacing per
  tenant, so that a crowded Hall of Fame stays readable.

---

## 4. Functional Requirements

### 4.1 Eligibility and triggers

**FR-1.1** A character is *eligible* for automatic deployment when all of the following hold at the
moment its level increases: the new level equals the character's maximum class level; the character
is not a GM; and the character does not already have a deployed Player NPC on the target map.

**FR-1.2** Maximum class level is job-dependent. Explorer, Aran, and Evan lines cap at 200; the
Cygnus Knights line caps at 120 in pre-Big-Bang content. The implementation MUST derive this from a
job-aware helper rather than a single hard-coded constant, and MUST place that helper in
`libs/atlas-constants/job/` if no equivalent exists there.

**FR-1.3** When `autoDeployEnabled` is true for the tenant, an eligible level-up triggers deployment
into `hallOfFameMapFor(job)` (§4.2) with no player action.

**FR-1.4** When `autoDeployEnabled` is false, automatic deployment does not occur. Instead, the
conversation engine exposes an eligibility condition (`canSpawnPlayerNpc`, §4.6) that 1st-job
instructor conversations use to offer manual deployment.

**FR-1.5** Automatic deployment MUST be asynchronous with respect to the level-up itself — a failure
to deploy MUST NOT block or roll back the level-up, and MUST be logged at warn level with the
character id, target map, and failure reason.

**FR-1.6** Deployment is idempotent per (character name, map). A repeated trigger for a character
that already has a Player NPC on that map does not create a second one.

### 4.2 Hall of Fame map routing

**FR-2.1** The target map for automatic deployment is determined by job category:

| Job category | Map | Map id |
|---|---|---|
| Warrior (Explorer) | Hall of Warriors | 102000004 |
| Magician (Explorer) | Hall of Magicians | 101000004 |
| Bowman (Explorer) | Hall of Bowmen | 100000204 |
| Thief (Explorer) | Hall of Thieves | 103000008 |
| Pirate (Explorer) | Nautilus Training Room | 120000105 |
| Cygnus Knights (any branch) | Knights' Chamber | 130000100 |
| Aran | Palace of the Master | 140010110 |
| Beginner / unclassified | Knights' Chamber 2 | 130000110 |

**FR-2.2** The set of *Hall of Fame maps* also includes Knights' Chamber Large (130000101),
Knights' Chamber 2 (130000110), and Knights' Chamber 3 (130000120); these are valid deployment
targets but are not automatic-deployment destinations except as noted in FR-2.1.

**FR-2.3** The set of *podium* maps — those using the podium positioner rather than the grid
positioner (§4.4) — is exactly: 102000004, 101000004, 100000204, 103000008, 120000105.

**FR-2.4** These map ids MUST be declared in `libs/atlas-constants/map/` (they are absent today) and
referenced symbolically; no literal map id appears in service code.

### 4.3 Script-id allocation

**FR-3.1** Script ids are allocated from the *imitate pool*: `Npc.wz` entries in the
`9901000`–`9901919` range whose `info/imitate` value is 1. Verified against a non-Cosmic reference
WZ (`ms_1172`), stock clients ship 269 such entries in that range; the reference server's patched WZ
fills the range densely and extends it to 9906599. Atlas MUST NOT assume the extension.

**FR-3.2** The pool is partitioned into *branches* of 100 consecutive ids, keyed by job category, so
that a given Hall of Fame map draws from its own id block. Branch `b` covers
`9900000 + (b * 100)` through `9900000 + (b * 100) + 99`:

| Branch | Job category | Script id range |
|---|---|---|
| 10 | Warrior | 9901000–9901099 |
| 11 | Magician | 9901100–9901199 |
| 12 | Bowman | 9901200–9901299 |
| 13 | Thief | 9901300–9901399 |
| 14 | Pirate | 9901400–9901499 |
| 15 | Dawn Warrior | 9901500–9901599 |
| 16 | Blaze Wizard | 9901600–9901699 |
| 17 | Wind Archer | 9901700–9901799 |
| 18 | Night Walker | 9901800–9901899 |
| 19 | Thunder Breaker | 9901900–9901999 |
| 20 | Aran | 9902000–9902099 |
| 21 | Evan | 9902100–9902199 |
| 22 | Beginner | 9902200–9902299 |
| 23 | Noblesse | 9902300–9902399 |
| 24 | Legend | 9902400–9902499 |
| 25 | other | 9902500–9902599 |

Branches 20–25 fall outside the stock imitate range and will normally be empty on a stock client;
FR-3.4's existence check handles this without special-casing.

**FR-3.3** For a GM deployment onto a non-Hall-of-Fame map, the branch is
`26 + 4 * (mapId / 100000000)` — a per-continent block of 400 ids beginning at 9902600. These ids
are outside the stock imitate pool and will usually be unavailable; a GM deployment that cannot
allocate MUST fail with a clear message rather than spawning an unusable id.

**FR-3.4** Before an id is allocated it MUST be validated as *usable*: the `Npc.wz` entry exists AND
`info/imitate == 1`. Validation is a query against `atlas-data`'s NPC registry (§7). Spawning a
non-existent script id is a known client-crash cause and MUST be impossible by construction.

**FR-3.5** Allocation returns the lowest usable id in the branch not already in use by a deployed
Player NPC for that tenant and world. When the branch is exhausted, deployment fails with reason
`pool_exhausted`; the caller logs at warn level and does not retry.

**FR-3.6** A removed Player NPC's script id returns to the pool and MAY be reallocated.

**FR-3.7** Overall-job rank is derived from the script id, not stored independently:
`(scriptId % 100) + 1` for branches below 26, `((scriptId - 2600) % 400) + 1` otherwise. This makes
the id's position within its branch the NPC's job-history ordinal.

### 4.4 Positioning

**FR-4.1** Two positioners exist. Podium maps (FR-2.3) use the **podium positioner**; every other
map uses the **grid positioner**.

**FR-4.2** Grid positioner. Starting from the map's VR bounds inset by `initialX`/`initialY`, walk a
lattice with horizontal pitch `dx(step) = areaX / (step + 1)` and vertical pitch
`dy(step) = areaY / 2 + areaY / 2^(step + 1)`. At each lattice point, snap to the ground using the
map's foothold tree; if the snapped point's `dx × dy` rectangle does not intersect any existing
Player NPC's rectangle, that point is the next position. `step` begins at 0 and increases when the
map fills, up to `areaSteps`.

**FR-4.3** Podium positioner. Positions are computed from a `(step, count)` pair encoded as
`count * 32 + step`. For rank `r` and step `s`, the platform is `r / s` and the relative slot is
`(r % s) + 1`; position is `(platformX(platform) + 100 * relative / (s + 1), platformY(platform))`,
where `platformX` is −50 for platform 0, −170 for platform 1, and 70 otherwise, and `platformY` is
−47 for platform 0 and 40 otherwise. When `count >= 3 * step`, the step increases (bounded by
`areaSteps`) and the map is re-organized.

**FR-4.4** Re-organization. When `organizeArea` is enabled and a map has no free slot at the current
step, every Player NPC on that map is re-positioned at the next step, ordered by ascending script id
(script id is the deployment-history order). Re-organization MUST despawn every affected Player NPC
from every channel serving that map, persist the new positions, and re-spawn them — never leave a
client holding a stale position.

**FR-4.5** When no position can be found even at the maximum step, deployment fails with reason
`map_full`.

**FR-4.6** A stored position comprises `x`, `cy`, `fh` (foothold id below the point), `rx0` = x + 50,
`rx1` = x − 50, and `dir` (default 1).

**FR-4.7** Default tuning values, all tenant-configurable: `initialX` 262, `initialY` 262,
`areaX` 320, `areaY` 160, `areaSteps` 4, `organizeArea` true, `autoDeployEnabled` true.

### 4.5 Appearance snapshot

**FR-5.1** At deployment the service captures, from the character's current state: name, gender,
skin colour, face id, hair id, and every equipped item.

**FR-5.2** Equipped items are captured with their signed inventory slot. Only slots in the ranges
1–11 (normal equips) and 101–111 (cash/masked equips) are captured; all other slots are ignored.

**FR-5.3** The snapshot is frozen. Subsequent changes to the character's level, job, appearance, or
equipment do not alter a deployed Player NPC.

**FR-5.4** A re-deploy for a character that already has a Player NPC on the target map refreshes the
snapshot in place — same script id, same position, new appearance — and re-broadcasts
`IMITATED_NPC_DATA` to everyone on the map. Re-deploy is reachable from the manual instructor path
and from the GM command; it is never triggered automatically.

**FR-5.5** Rank fields are captured at deployment: world rank and overall rank come from
`atlas-rankings` for the character; world-job rank is the next ordinal within (world, job category);
overall-job rank is derived per FR-3.7. Ranks are frozen with the rest of the snapshot.

### 4.6 Conversation-engine integration

**FR-6.1** A new condition type makes eligibility queryable from NPC conversation JSON. It evaluates
true when: automatic deployment is disabled for the tenant, the character's level is at least its
maximum class level, the character is not a GM, and the character has no Player NPC on the given map.

**FR-6.2** A new conversation operation deploys (or re-deploys) the talking character's Player NPC
onto a specified map, defaulting to the character's current map.

**FR-6.3** The operation's outcome is reported back to the conversation so a script can branch on
success versus each failure reason (`pool_exhausted`, `map_full`, `ineligible`).

**FR-6.4** Authoring the instructor conversation JSON that uses these is out of scope for this task;
the engine capability and its schema documentation are in scope.

### 4.7 Map lifecycle and broadcast

**FR-7.1** When a character enters a map, every Player NPC deployed on that map (for the character's
tenant and world) is spawned to that character's session: first the NPC controller-grant spawn, then
`IMITATED_NPC_DATA`. The order matters — the client needs the object before the avatar data.

**FR-7.2** A newly deployed Player NPC is broadcast to every character currently on that map, on
every channel of that world.

**FR-7.3** Removing a Player NPC broadcasts the NPC controller removal followed by the
`IMITATED_NPC_DATA` remove arm, to every character on the map on every channel.

**FR-7.4** Player NPCs are not controlled by any client and never move. They MUST NOT participate in
NPC controller assignment, controller hand-off on player exit, or NPC movement handling.

> **Amended (bug report §5, `bug-player-npc-no-chat-balloon.md`).** Player NPCs DO participate in
> controller assignment: the grant is what reaches `CNpc::SetActive`, and without it the canned chat
> balloon never renders. The exit hand-off and re-election paths are object-id agnostic and cover
> them unchanged. What FR-7.4 still guarantees is that a Player NPC is never left dangling in the
> registry after its controller leaves.

**FR-7.5** Player NPC object ids MUST NOT collide with the object ids of WZ-spawned NPCs on the same
map.

### 4.8 GM commands

**FR-8.1** A GM command deploys a Player NPC for a named online character at the GM's current
position on the GM's current map, bypassing the level and auto-deploy checks but honouring
script-id availability (FR-3.3, FR-3.4) and the per-map duplicate rule (FR-1.6).

**FR-8.2** A GM command removes every Player NPC belonging to a named character, optionally scoped to
a single map.

**FR-8.3** Both commands report success or the specific failure reason to the invoking GM.

---

## 5. API Surface

All endpoints follow JSON:API conventions and are tenant-scoped by request context.

### `atlas-player-npcs`

**`GET /api/player-npcs?filter[mapId]=<id>&filter[worldId]=<id>`**
List deployed Player NPCs. `mapId` and `worldId` filters are the hot path used by `atlas-channel` on
map enter. Supports the repo's standard pagination.

Resource shape (`type: "player-npcs"`):

```
attributes:
  characterId    uint32
  name           string
  worldId        byte
  mapId          uint32
  scriptId       uint32
  objectId       uint32
  gender         byte
  skin           byte
  face           uint32
  hair           uint32
  jobId          uint16     # job category, i.e. (jobId / 100) * 100
  x              int16
  cy             int16
  fh             uint16
  rx0            int16
  rx1            int16
  dir            byte
  worldRank      uint32
  overallRank    uint32
  worldJobRank   uint32
  overallJobRank uint32
  equipment      [ { slot: int16, itemId: uint32 } ]
  deployedAt     timestamp
```

**`GET /api/player-npcs/{id}`** — fetch one.

**`POST /api/player-npcs`** — deploy. Request attributes: `characterId`, `worldId`, `mapId`, and an
optional `position` (`{x, y}`) for the GM path; when `position` is absent the service positions the
NPC itself (§4.4). Returns `201` with the created resource.

Error cases, all `409 Conflict` with a distinguishing `code`:

| `code` | Meaning |
|---|---|
| `pool_exhausted` | No usable script id remains in the branch |
| `map_full` | No free position at the maximum step |
| `duplicate` | Character already deployed on this map (deploy, not re-deploy) |
| `ineligible` | Level/GM checks failed on the eligibility-checked path |

`422` when the character or map cannot be resolved.

**`PATCH /api/player-npcs/{id}`** — re-deploy: refresh the appearance and rank snapshot in place
(FR-5.4). Position and script id are immutable through this endpoint.

**`DELETE /api/player-npcs/{id}`** — remove one.

**`DELETE /api/player-npcs?filter[characterId]=<id>[&filter[mapId]=<id>]`** — remove all of a
character's Player NPCs, optionally map-scoped (FR-8.2).

**`GET /api/player-npcs/eligibility?characterId=<id>&mapId=<id>`** — the FR-6.1 predicate, returning
`{ eligible: bool, reason: string }`. Consumed by the conversation-condition evaluator.

### Kafka

**Commands** (`COMMAND_TOPIC_PLAYER_NPC`): `DEPLOY` (characterId, worldId, mapId, optional position,
`enforceEligibility` flag), `REDEPLOY` (characterId, mapId), `REMOVE` (characterId, optional mapId).

**Events** (`EVENT_TOPIC_PLAYER_NPC_STATUS`): `DEPLOYED` (full resource payload), `UPDATED` (same,
after a re-deploy), `REMOVED` (id, objectId, mapId, worldId), `REPOSITIONED` (list of
{id, objectId, x, cy, fh, rx0, rx1} for a re-organization, plus mapId and worldId).
`atlas-channel` consumes all four to drive broadcast (§4.7).

### `atlas-data`

**`GET /api/npcs/{id}`** must expose whether the NPC template exists and whether it is an imitate
template. If the current NPC REST model does not carry the `info/imitate` flag, the reader
(`services/atlas-data/atlas.com/data/npc/reader.go`) and REST model gain an `imitate` boolean. This
is the FR-3.4 validation source.

---

## 6. Data Model

Two tables in the `atlas-player-npcs` database, both tenant-scoped through the standard GORM tenant
callbacks.

### `player_npcs`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | not null, indexed |
| `character_id` | uint32 | not null |
| `name` | string | not null — the duplicate check is by name, matching the reference server |
| `world_id` | byte | not null |
| `map_id` | uint32 | not null |
| `script_id` | uint32 | not null |
| `object_id` | uint32 | not null — stable map object id |
| `gender` | byte | not null |
| `skin` | byte | not null |
| `face` | uint32 | not null |
| `hair` | uint32 | not null |
| `job_id` | uint16 | job category, `(jobId / 100) * 100` |
| `x` | int16 | not null |
| `cy` | int16 | not null |
| `fh` | uint16 | not null |
| `rx0` | int16 | not null |
| `rx1` | int16 | not null |
| `dir` | byte | not null, default 1 |
| `world_rank` | uint32 | |
| `overall_rank` | uint32 | |
| `world_job_rank` | uint32 | |
| `created_at` / `updated_at` | timestamp | |

Constraints:

- Unique `(tenant_id, world_id, script_id)` — one NPC per allocated script id.
- Unique `(tenant_id, world_id, map_id, name)` — enforces FR-1.6 at the storage layer.
- Unique `(tenant_id, world_id, map_id, object_id)` — enforces FR-7.5 within the service.
- Index on `(tenant_id, world_id, map_id)` — the map-enter read path.
- Index on `(tenant_id, world_id, character_id)`.

### `player_npc_equipment`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | not null |
| `player_npc_id` | uuid | FK → `player_npcs.id`, cascade delete |
| `slot` | int16 | signed inventory slot as captured |
| `item_id` | uint32 | not null |

Unique `(tenant_id, player_npc_id, slot)`.

Migration notes: both tables are new; no backfill exists and none is possible (the feature has never
run). Removing a `player_npcs` row MUST cascade to its equipment rows.

---

## 7. Service Impact

**`atlas-player-npcs` (new).** Owns everything in §4.3, §4.4, §4.5, §6. Consumes the character
status topic to observe `LEVEL_CHANGED` (`services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:229`)
and evaluate FR-1.1. Reads character appearance and equipped inventory, map geometry from
`atlas-data`, NPC template validity from `atlas-data`, and ranks from `atlas-rankings`. Produces the
status events in §5. Follows the standard service scaffold (`docs/adding-a-new-service.md`).

**`atlas-channel`.** Consumes the four Player NPC status events and, on map enter, reads the map's
Player NPCs and announces them. Reuses the existing `NpcSpawnRequestControllerWriter`
(`services/atlas-channel/atlas.com/channel/npc/controller/announce.go`) for the spawn arm and adds
writers for the `IMITATED_NPC_DATA` spawn and remove arms. Must exclude Player NPC object ids from
NPC controller assignment (FR-7.4). Registers the GM commands in §4.8.

**`libs/atlas-packet`.** New `npc/clientbound` codec for `IMITATED_NPC_DATA` with two arms: the
avatar-data arm (`0x01`: scriptId, name, gender, skin, face, `0`, hair, then the my-equip list of
`(slot byte, itemId int)` terminated by `0xFF`, the masked-equip list terminated by `0xFF`, cash
weapon id or 0, then three zero ints) and the remove arm (`0x00`: objectId). Encoded across every
version column that carries an opcode for it (0x04E on gms v83/84/87, 0x051 on gms v92, 0x053 on
gms v95, and the remaining columns per `docs/packets/audits/STATUS.md:134`). Existing spawn
codec `npc/spawn_request_controller` is reused unchanged.

`IMITATED_NPC_RESULT` (`STATUS.md:131`) is not produced by the reference server and has no known
server-side use. It is documented here as intentionally not implemented; the coverage matrix row
stays ❌ and this task does not claim it.

**`libs/atlas-constants`.** Adds the Hall of Fame map ids (§4.2) to `map/`, the max-class-level
helper (FR-1.2) to `job/`, and the Player NPC script-id base and branch mapping (§4.3). Existing
constants MUST be checked before any new one is defined.

**`atlas-character`.** No change expected — the `LEVEL_CHANGED` status event already carries what
the trigger needs. If the event lacks the new level or job, extend it rather than polling.

**`atlas-data`.** NPC reader and REST model gain the `imitate` flag (§5). Map REST already exposes
the foothold tree (`services/atlas-data/atlas.com/data/map/rest.go:29`) and VR bounds, which the
positioner consumes; confirm the ground-snap helper is reachable, and add it if only the tree is
exposed.

**`atlas-rankings`.** Read-only consumer. `overall_rank` and `job_rank` already exist on the ranking
entity; no schema change expected.

**`atlas-npc-conversations`.** Adds the FR-6.2 deploy operation to the operation executor and
documents it in `docs/npc_conversation_conversion_spec.md` and the conversation JSON schema.

**`atlas-query-aggregator`.** Adds the FR-6.1 eligibility condition to the validation model.

**`atlas-configurations`.** Registers the new `IMITATED_NPC_DATA` writer and the GM command handlers
in every seed template that should carry the feature, and adds the §4.4/§4.7 tuning values
(`initialX`, `initialY`, `areaX`, `areaY`, `areaSteps`, `organizeArea`, `autoDeployEnabled`) to the
tenant configuration surface.

---

## 8. Non-Functional Requirements

**Multi-tenancy.** Every table carries `tenant_id`; every read and write goes through
`db.WithContext(ctx)` so the tenant callbacks apply. Script-id allocation, the duplicate check, and
the position search are all scoped to (tenant, world) — a Player NPC in one tenant never consumes an
id or a slot in another.

**Performance.** Map enter is the hot path. Listing a map's Player NPCs must be a single indexed
query on `(tenant_id, world_id, map_id)`; a Hall of Fame map holds at most a few hundred rows.
Deployment is rare and may be slower, but the position search MUST be bounded — the lattice walk is
`O(mapArea / (dx * dy))` per step with at most `areaSteps` steps.

**Concurrency.** Two simultaneous deployments to the same map must not allocate the same script id or
the same position. Allocation and insertion MUST be serialized per (tenant, world, map), and the
unique constraints in §6 are the backstop.

**Correctness under failure.** A deployment that fails after allocating an id must not leak that id;
allocation and row creation are one transaction. A re-organization that fails partway must not leave
persisted positions inconsistent with what clients were told — persist all new positions, then
broadcast.

**Client safety.** No script id is ever spawned without the FR-3.4 existence-and-imitate check.
This is a crash-prevention requirement, not a cosmetic one.

**Observability.** Log at info on deploy, re-deploy, remove, and re-organize, with tenant, world,
map, character, and script id. Log at warn with an explicit reason on every deployment failure.

**Security.** GM commands are gated by the existing GM-level check. The deploy REST endpoint is
internal-only, consistent with other service-to-service endpoints.

---

## 9. Open Questions

1. **Imitate-pool size on the actual target clients.** The 269-entry figure comes from a v117
   reference WZ; the density of imitate entries in the v83/v87/v92/v95 clients Atlas tenants use has
   not been measured. If a given branch has very few usable ids, that Hall of Fame fills quickly.
   Measuring this per client version is a design-phase task, not a blocker — the existence check
   makes a small pool safe, just small.

2. **Object id allocation across channels.** Player NPC object ids must be stable across channels and
   restarts (unlike ephemeral map objects) and must not collide with WZ NPC object ids. Whether the
   id is minted by `atlas-player-npcs` from a dedicated high range or reserved via the existing map
   object id authority is a design decision.

3. **Ground snap ownership.** The positioner needs "the ground point below (x, y)" and the foothold
   id below a point. `atlas-data` has the foothold tree and a `bSearchDropPos` used for drop
   positioning; whether to expose a general snap endpoint or reimplement the walk in
   `atlas-player-npcs` from the REST foothold tree needs deciding.

4. **World-job rank ordinal source.** The reference server keeps a running per-(world, job) counter
   seeded from the max stored value. Whether Atlas mirrors that or derives it from `atlas-rankings`
   at deploy time affects whether the number means "deployment order" or "current standing".

5. **Cygnus max level.** FR-1.2 asserts 120 for the Cygnus line pre-Big-Bang. This is general
   MapleStory knowledge and is **unverified** against the target client versions; confirm against
   the client or the repo's job data before implementing.

---

## 10. Acceptance Criteria

- [ ] A non-GM character reaching max class level on a tenant with `autoDeployEnabled` gets a Player
      NPC in the correct Hall of Fame map, visible to other characters on that map without a
      relog.
- [ ] The deployed NPC renders as that character's avatar — correct name, gender, skin, face, hair,
      and equipped items including cash/masked equips and cash weapon.
- [ ] Re-entering the map, and restarting `atlas-player-npcs` and `atlas-channel`, both leave every
      previously deployed Player NPC visible and correctly positioned.
- [ ] Reaching max level twice, or triggering deployment twice for the same character and map,
      produces exactly one Player NPC.
- [ ] With `autoDeployEnabled` false, no NPC is deployed on level-up, and the eligibility condition
      returns true for an eligible character and false for a GM, an under-levelled character, and a
      character already deployed on that map.
- [ ] The conversation deploy operation creates a Player NPC and reports the outcome distinctly for
      success, `pool_exhausted`, `map_full`, and `ineligible`.
- [ ] Filling a podium map past `3 * step` triggers re-organization: every Player NPC on the map moves
      to the next step's layout, ordered by script id, and every client on the map sees the new
      layout with no stale objects.
- [ ] Filling a grid map to `areaSteps` and attempting one more deployment fails with `map_full` and
      spawns nothing.
- [ ] Exhausting a branch's usable script ids fails with `pool_exhausted` and spawns nothing.
- [ ] No script id is ever spawned whose `atlas-data` NPC template is missing or lacks
      `info/imitate == 1`; a unit test asserts the allocator rejects both cases.
- [ ] The GM deploy command creates a Player NPC for a named character at the GM's position; the GM
      remove command removes that character's Player NPCs and every client on the map sees them
      disappear.
- [ ] Player NPCs never receive NPC controller assignment, and player exit from a map does not
      attempt to hand off control of one.
- [ ] `IMITATED_NPC_DATA` has an encode-side codec with byte-fixture tests on every version column
      the matrix lists an opcode for.
- [ ] Two tenants sharing a world id have independent Player NPC sets, script-id pools, and map
      slots; a test asserts cross-tenant isolation on the map-enter read path.
- [ ] `tools/verify.sh` exits 0.
