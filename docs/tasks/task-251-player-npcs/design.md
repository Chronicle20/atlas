# Player NPCs (Imitated NPCs) — Design

Task: task-251-player-npcs
Status: Approved
Created: 2026-08-21
Input: [prd.md](prd.md) (v1, approved)

---

## 0. How to read this document

The PRD is approved, but four of its factual premises are wrong against the repository and the WZ
data, and one of them changes the shape of the feature. §1 records those corrections and the
evidence for each; the rest of the design is written against the corrected premises, not the PRD's
original text. Where this design deviates from a numbered PRD requirement, the deviation is called
out inline with the requirement id.

---

## 1. Corrections to the PRD's premises

### C-1 — The imitate pool is 193 ids, and its per-branch distribution makes PRD §4.3 unusable

**This is the finding that reshapes the design.** PRD FR-3.1 estimates 269 usable entries from a
v117 reference WZ and treats the branch partitioning as workable. Measured against the two stock WZ
trees available on this machine — the `ms_1172` and `AtlasMS` checkouts' `wz/Npc.wz`, which are
byte-identical in this respect — the real figure is **193** entries carrying `info/imitate == 1`,
spanning 9901000–9901919, distributed as:

| Branch | Job category | Usable ids | Which |
|---|---|---|---|
| 10 | Warrior | 1 | 9901000 |
| 11 | Magician | 1 | 9901100 |
| 12 | Bowman | 1 | 9901200 |
| 13 | Thief | 1 | 9901300 |
| 14 | Pirate | 0 | — |
| 15 | Dawn Warrior | 52 | 9901500–9901551 |
| 16 | Blaze Wizard | 17 | 9901600–9901616 |
| 17 | Wind Archer | 50 | 9901700–9901749 |
| 18 | Night Walker | 50 | 9901800–9901849 |
| 19 | Thunder Breaker | 20 | 9901900–9901919 |
| 20–25 | Aran / Evan / Beginner / Noblesse / Legend / other | 0 | — |

The Cosmic reference checkout's `wz/Npc.wz` carries **5601** imitate entries — it is a patched WZ,
which is why the reference server's dense-branch scheme works there. Atlas serves whichever WZ the
deployment mounts, and PRD §2 forbids assuming the patched extension.

Reproduce the measurement against any WZ tree with:

```sh
grep -rl 'name="imitate" value="1"' <wz-root>/Npc.wz \
  | sed 's#.*/##; s/\.img\.xml//' | sort -u \
  | awk '{b=int($1/100); c[b]++} END {for (k in c) printf "%d %d\n", k, c[k]}' | sort -n
```

Under PRD §4.3 on a stock WZ, exactly one Warrior statue, one Magician, one Bowman and one Thief
could ever exist, and every Pirate, Aran, Evan and Beginner deployment would fail immediately with
`pool_exhausted`. The feature would be non-functional for the Explorer classes it is primarily aimed
at.

Two facts make the fix safe. First, a Player NPC's appearance is transmitted entirely by
`IMITATED_NPC_DATA` — the script id's own WZ content is never rendered, because its `stand`/`say`
canvases are 1×1 placeholders. Any imitate id renders any avatar identically. Second, the only hard
constraint on an id is FR-3.4: the entry must exist and must carry `info/imitate == 1`. Branch
partitioning is therefore bookkeeping, not a rendering or safety requirement.

**Resolution (D-1):** branch-preferred allocation with a global fallback across the validated pool.
See §4.

### C-2 — Every Hall of Fame map id already exists in `libs/atlas-constants`

FR-2.4 asserts the ten map ids are "absent today" and must be added. All ten are already declared in
`libs/atlas-constants/map/constants.go`:

| Map id | Existing constant | Line |
|---|---|---|
| 100000204 | `VictoriaRoadHallOfBowmen1Id` | 69 |
| 101000004 | `VictoriaRoadHallOfMagicians1Id` | 103 |
| 102000004 | `VictoriaRoadHallOfWarriors1Id` | 162 |
| 103000008 | `VictoriaRoadHallOfThieves1Id` | 191 |
| 120000105 | `TheNautilusTrainingRoom2Id` | 527 |
| 130000100 | `EmpressRoadKnightsChamber1Id` | 535 |
| 130000101 | `EmpressRoadKnightsChamber2Id` | 536 |
| 130000110 | `EmpressRoadKnightsChamber2ndFloorId` | 537 |
| 130000120 | `EmpressRoadKnightsChamber3rdFloorId` | 538 |
| 140010110 | `SnowIslandPalaceOfTheMaster1Id` | 566 |

FR-2.4 is satisfied by *referencing* these, not by adding new ones. Defining parallel constants would
violate CLAUDE.md's "check `libs/atlas-constants/` for an existing equivalent" rule. What genuinely
does not exist is the *grouping* — the job-category→map function and the podium-map set — which
belongs in the new service's routing package (§3.2), not in the shared constants library.

### C-3 — GM commands live in `atlas-messages`, not `atlas-channel`

PRD §7 places the FR-8.1/8.2 commands in `atlas-channel`. There is no command infrastructure in
`atlas-channel`. The GM chat-command registry is
`services/atlas-messages/atlas.com/messages/command/registry.go` — a `Producer`/`Executor` pair
(`command/types.go`) with per-domain subpackages (`buff/`, `character/`, `map/`, `monster/`,
`pet/`, …). The Player NPC commands go in a new `command/playernpc/` subpackage there.

Relatedly, PRD §7 places the tenant tuning values in `atlas-configurations`. Tenant configuration
resources are served by **`atlas-tenants`** at `/tenants/{id}/configurations/<name>` (see
`services/atlas-tenants/atlas.com/tenants/configuration/`, e.g. the `rankings` resource).
`atlas-configurations` owns the *seed templates* — which is where the `IMITATED_NPC_DATA` writer
registration goes. Both services are touched, for different reasons.

### C-4 — The PRD's opcode→version mapping is shifted three columns

PRD §7 states "0x04E on gms v83/84/87, 0x051 on gms v92, 0x053 on gms v95". Reading the
`IMITATED_NPC_DATA` row of `docs/packets/audits/STATUS.md:134` against the column order declared in
the file header (`gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95,
jms_v185`), the actual mapping is:

| Version | Opcode |
|---|---|
| gms_v48 | n/a (⬜ — no opcode) |
| gms_v61 | 0x04E |
| gms_v72 | 0x04E |
| gms_v79 | 0x04E |
| gms_v83 | 0x051 |
| gms_v84 | 0x053 |
| gms_v87 | 0x053 |
| gms_v92 | 0x056 |
| gms_v95 | 0x054 |
| jms_v185 | 0x055 |

Nine columns carry an opcode. `services/atlas-configurations/seed-data/templates/` holds eleven
templates (`gms_12_1` and `gms_48_1` in addition to the nine matrix columns); the writer is
registered in the nine that have an opcode, and *not* in `template_gms_12_1.json` or
`template_gms_48_1.json`. Registering a writer with no opcode is what produces a silent
mis-encode, so the omission is deliberate and must be stated in the plan.

`IMITATED_NPC_RESULT` (`STATUS.md:131`) stays ❌ and unclaimed, exactly as the PRD says.

---

## 2. Architecture

### 2.1 Shape

```
atlas-character ──LEVEL_CHANGED──┐
                                 │
atlas-messages (GM cmd) ──────┐  │
atlas-saga-orchestrator ──────┤  │
   (conversation deploy op)   │  │
                              ▼  ▼
                    COMMAND_TOPIC_PLAYER_NPC
                              │
                    ┌─────────▼──────────┐        ┌───────────────┐
                    │ atlas-player-npcs  │───────▶│  atlas-data   │ npc imitate flag
                    │                    │        │               │ map geometry + ground snap
                    │  eligibility       │───────▶├───────────────┤
                    │  allocation        │        │atlas-character│ appearance + equipment + gm
                    │  positioning       │───────▶├───────────────┤
                    │  snapshot          │        │atlas-rankings │ world / job rank
                    │  persistence       │        └───────────────┘
                    └─────────┬──────────┘
                              │
                  EVENT_TOPIC_PLAYER_NPC_STATUS
                     DEPLOYED / UPDATED / REMOVED / REPOSITIONED
                              │
                    ┌─────────▼──────────┐
                    │   atlas-channel    │  broadcast + map-enter spawn
                    └────────────────────┘
                              │
                              ▼  GET /api/player-npcs?filter[mapId]&filter[worldId]
```

`atlas-player-npcs` is a new REST + Kafka + Postgres service following
`docs/adding-a-new-service.md`. It is the sole authority on which Player NPCs exist, what they look
like, where they stand, and which script ids are in use. Nothing else writes that state.

### 2.2 Why a new service rather than folding into an existing one

Three candidates were considered and rejected.

**Fold into `atlas-maps`.** `atlas-maps` tracks *who is currently in a field* — ephemeral,
Redis-shaped, per-channel state. Player NPCs are the opposite: durable, per-world, survive
restarts, and need their own relational schema with two tables and four unique constraints. Mixing a
persistent relational aggregate into an ephemeral presence service would give `atlas-maps` a database
it does not currently need.

**Fold into `atlas-data`.** `atlas-data` is read-only WZ projection. Player NPCs are mutable
server-authored state. This would break the service's single defining invariant.

**Fold into `atlas-npc-conversations`.** That service owns conversation state machines, not map
objects, and has no map-geometry or ranking dependencies.

A dedicated service also gives the allocation-and-positioning transaction (§4.4, §5.4) a single
database it fully controls, which is what makes the concurrency story in §8 tractable.

### 2.3 Package layout inside `atlas-player-npcs`

```
atlas.com/player-npcs/
  main.go
  playernpc/            aggregate: model, entity, processor, producer, rest, resource, requests
  playernpc/equipment/  equipment child rows
  allocation/           script-id pool: branch mapping, usability check, allocator
  position/             grid + podium positioners, geometry types, reorganization
  routing/              job category -> hall-of-fame map, podium-map set
  snapshot/             appearance + rank capture, assembled from the read clients
  eligibility/          the FR-1.1 / FR-6.1 predicate
  configuration/        tenant tuning values (read client against atlas-tenants)
  character/            read client (appearance, equipment, gm, level, job)
  data/npc/             read client (imitate flag)
  data/map/             read client (VR bounds, foothold tree / ground snap)
  ranking/              read client
  kafka/consumer/character/   LEVEL_CHANGED
  kafka/consumer/playernpc/   DEPLOY / REDEPLOY / REMOVE
  kafka/message/playernpc/    topic + envelope definitions
  kafka/producer/             status events
```

`position/` and `allocation/` are pure packages: they take values in and return values out, touch no
database and no HTTP client. That is deliberate — they hold the only genuinely tricky arithmetic in
the feature (§5) and must be unit-testable without any infrastructure. `snapshot/` is the impure
assembler that fans out to the read clients; `playernpc/processor.go` orchestrates the transaction.

---

## 3. Domain model

### 3.1 Aggregate

The aggregate root is a Player NPC: identity (`characterId`, `name`), placement (`worldId`, `mapId`,
`scriptId`, `objectId`), frozen appearance (`gender`, `skin`, `face`, `hair`, `jobId` + equipment
rows), frozen ranks, and position (`x`, `cy`, `fh`, `rx0`, `rx1`, `dir`). Equipment is a child
collection, cascade-deleted with the root.

Follow the repo's immutable-model convention: unexported fields, accessor methods, a Builder for
construction, `Transform`/`Extract` between entity, domain model and REST model. No `*_testhelpers.go`
— test setup goes through the Builder (CLAUDE.md repository conventions).

The schema is PRD §6 as written, with one addition and one clarification:

- **Add `overall_job_rank uint32`.** PRD §6's table omits it while PRD §5's resource shape includes
  it. FR-3.7 derived it from the script id, which D-1 makes impossible (§4). It becomes a stored
  column populated from the counter in §6.3.
- `dir` defaults to 1 (FR-4.6). `rx0 = x + 50`, `rx1 = x - 50` are computed at deploy time and
  stored, not recomputed on read, so a later change to the constant cannot silently move existing
  NPCs.

### 3.2 Routing (`routing/`)

`HallOfFameMapFor(set constants.SkillJobSet, jobId job.Id) _map.Id` implements PRD FR-2.1's table by
job category, referencing the existing constants from C-2. `IsPodiumMap(mapId) bool` is FR-2.3's
five-element set. Neither carries a literal map id, satisfying FR-2.4's intent.

`HallOfFameMapFor` takes a resolved `constants.SkillJobSet` rather than being a pure lookup over
`jobId` alone: job wire id 500 is version-divergent (task-187 audit) — it means Pirate at GMS v61+
but Gm at GMS v48, and a raw `JobCategory`/`job.IsA` compare against the wire id cannot tell them
apart. The Pirate row resolves `jobId` through `set.Job.Resolve` to this tenant version's `Identity`
and branches on `job.IsAIdentity(jid, job.Pirate)` (which also covers the Brawler/Gunslinger
sub-jobs, wire 510/520), following the `atlas-character` `processor.go` house pattern
(`p.set().Job.Resolve` + `job.IsAIdentity`). Every other row — Warrior/Magician/Bowman/Rogue and the
Cygnus/Aran/Evan roots — is version-stable per that same audit and stays keyed on the raw
`JobCategory`/`job.GetType` category, as a pure lookup, with table-driven tests.

### 3.3 Max class level (FR-1.2)

`libs/atlas-constants/job` has no max-level helper today; `atlas-character` has a single flat
`MaxLevel = 200` (`services/atlas-character/atlas.com/character/character/experience_table.go:4`),
job-agnostic. FR-1.2 asserts Cygnus caps at 120, and PRD Open Question 5 correctly flags that as
unverified.

Add `job.MaxLevelFor(jobId job.Id) byte` to `libs/atlas-constants/job/` as a per-line table.
**Every line returns 200 until the Cygnus figure is verified.** This is not a stub: 200 is the
behaviour the server actually implements — `atlas-character` will never emit a `LEVEL_CHANGED` above
200 for any job — so a job-agnostic 200 is the correct, evidence-backed table today. The plan carries
an explicit verification task: check the Cygnus level cap against the target client versions'
`Character.wz`/job data, and set the Cygnus entries to 120 if and only if that check confirms it.
The helper's shape means that change is a one-line table edit, not a refactor.

---

## 4. Script-id allocation (`allocation/`)

### D-1 — Branch-preferred with global fallback

Given C-1, three strategies were weighed.

| Strategy | Stock WZ behaviour | Patched WZ behaviour | Cost |
|---|---|---|---|
| Strict branch (PRD §4.3) | 1 Explorer statue per class, 0 Pirate/Aran/Evan | Correct | Feature is unusable on stock WZ |
| Flat pool, no branches | 193 shared slots | Loses per-hall id grouping that the WZ supports | Simplest allocator |
| **Branch-preferred + global fallback** | 193 shared slots, Cygnus keeps its blocks | Identical to strict branch | Overall-job rank can no longer be derived from the id |

**Chosen: branch-preferred with global fallback.** It is byte-identical to the PRD's behaviour on a
patched WZ, degrades gracefully rather than failing on a stock one, and preserves the branch grouping
wherever the WZ actually supports it.

### 4.1 Algorithm

```
Allocate(tenant, world, jobId, mapId) -> scriptId | pool_exhausted
  branch := BranchFor(set, jobId, mapId)       // PRD §4.3 table, or the FR-3.3 GM formula
  inUse  := scriptIds already deployed for (tenant, world)
  1. for id in branch range, ascending:
         if id not in inUse and Usable(id): return id
  2. for id in the whole validated pool, ascending:      // fallback
         if id not in inUse and Usable(id): return id
  3. return pool_exhausted
```

`Usable(id)` is FR-3.4: `atlas-data` reports the NPC template exists **and** carries
`info/imitate == 1`. Both conditions must hold; either one false makes the id unusable. This is the
crash-prevention gate, so it is checked on the allocation path itself and not bypassed on any branch.
Result: a script id that fails validation can never be spawned, by construction, and a unit test
asserts rejection of both the missing-template and the `imitate == 0` case (PRD acceptance criteria).

The fallback scans the same validated pool, so a fallback-allocated id is exactly as safe as a
branch-allocated one.

### 4.2 The validated pool

The pool is the id range 9901000–9906599 filtered by `Usable`. Rather than probe `atlas-data`
per-candidate on every deploy — up to 5600 REST calls in the worst case — `allocation/` builds a
**per-tenant usable-id set once, lazily, and caches it for the process lifetime**. WZ data is
immutable for a running deployment (`atlas-data` projects it read-only), so the set cannot go stale
without a restart. Building it is one paginated `atlas-data` sweep of the range. This turns
allocation into an in-memory set difference against the `script_id`s already stored for
(tenant, world).

The FR-3.3 GM branch formula (`26 + 4 * (mapId / 100000000)`, ids from 9902600) is implemented as
written and will normally find nothing in-branch on a stock WZ, at which point the global fallback
picks up. That is the intended behaviour: a GM deploy onto an arbitrary map succeeds as long as
*any* usable id remains, and fails with `pool_exhausted` and a clear message rather than spawning an
unusable id (FR-3.3).

### 4.3 Consequence: overall-job rank becomes stored

FR-3.7 derived overall-job rank from the id's offset within its branch. Under D-1 an id can come from
another branch entirely, so the derivation is no longer meaningful. Overall-job rank joins world-job
rank as a stored counter (§6.3), which is also what makes the two fields consistent with each other.

### 4.4 Release

FR-3.6: removing a Player NPC deletes the row, which removes its `script_id` from the in-use set —
the id is reusable with no extra bookkeeping. The pool is derived (usable set minus in-use set), so
there is no free list to maintain and nothing to leak.

---

## 5. Positioning (`position/`)

Both positioners are pure functions over: the map's VR bounds, the foothold tree (or a ground-snap
callback), the tuning values, and the list of already-placed rectangles. No I/O.

### 5.1 Grid positioner (FR-4.2)

```
dx(step) = areaX / (step + 1)
dy(step) = areaY / 2 + areaY / 2^(step + 1)
```

Walk the lattice from the VR bounds inset by `initialX`/`initialY`. At each point, snap to ground;
if the snapped point's `dx × dy` rectangle intersects no placed rectangle, that point wins. `step`
starts at 0 and increases to `areaSteps` as the map fills.

### 5.2 Podium positioner (FR-4.3)

State is the `(step, count)` pair encoded as `count * 32 + step`. For rank `r` at step `s`:

```
platform = r / s
relative = (r % s) + 1
x        = platformX(platform) + 100 * relative / (s + 1)
y        = platformY(platform)

platformX: platform 0 -> -50, platform 1 -> -170, otherwise 70
platformY: platform 0 -> -47, otherwise 40
```

`count >= 3 * step` raises the step (bounded by `areaSteps`) and triggers re-organization.

Note the division by `s`: at `step == 0` this is a divide-by-zero. The positioner treats the encoded
pair as starting at step 1 for the podium path, and a unit test pins the `step == 0` input as an
explicit error rather than a panic. This is exactly the kind of arithmetic edge the pure-package
boundary exists to make testable.

### 5.3 Ground snap — D-2

PRD Open Question 3 asks whether to expose a snap endpoint from `atlas-data` or reimplement the walk
from the REST foothold tree.

`atlas-data` already owns the whole mechanism: `calcPointBelow`
(`services/atlas-data/atlas.com/data/map/processor.go:132`) computes the ground point below an
(x, y) including the slope interpolation, `FootholdTreeRestModel.findBelow`
(`services/atlas-data/atlas.com/data/map/model.go:35`) locates the foothold, and `findById`
(`model.go:78`) resolves a foothold by id. The tree is already serialized on the map resource as
`footholdTree` (`services/atlas-data/atlas.com/data/map/rest.go:29`).

**Chosen: add a snap endpoint to `atlas-data`.** `GET /api/maps/{mapId}/ground?x=<x>&y=<y>` returns
`{x, y, fh}`. Rationale: the alternative — shipping the entire foothold tree to
`atlas-player-npcs` on every lattice probe and reimplementing `calcPointBelow` there — duplicates
non-trivial geometry that already exists, and duplicated geometry drifts. It also violates
CLAUDE.md's "don't call another layer's internals across a service boundary": the snap computation
belongs to whoever owns the foothold tree.

The endpoint returns the foothold id alongside the point because FR-4.6 stores `fh`, and
`calcPointBelow` already has it in hand — returning it costs nothing and saves a second call.

The lattice walk issues one snap call per candidate point. To keep the walk bounded (PRD §8
performance), the endpoint accepts a repeated `point` parameter and returns a snapped point per
input, so a full step's lattice is one round trip.

### 5.4 Re-organization (FR-4.4)

When `organizeArea` is enabled and the current step has no free slot:

1. Load every Player NPC on (tenant, world, map), ordered by ascending `script_id` — deployment
   history order.
2. Recompute every position at `step + 1`.
3. **Persist all new positions in one transaction.**
4. Emit one `REPOSITIONED` event carrying the full list.

Persist-then-broadcast, never the reverse (PRD §8 correctness-under-failure). A crash between 3 and 4
leaves the database correct and clients stale; the next map enter re-reads from the database and
repairs itself. A crash between a partial 3 and 4 is impossible because 3 is a single transaction.

`atlas-channel` handles `REPOSITIONED` by despawning and respawning every listed object on every
channel of that world (FR-4.4's "never leave a client holding a stale position").

Failure at `areaSteps` with no free slot is `map_full` (FR-4.5) and nothing is spawned.

---

## 6. Appearance and rank snapshot (`snapshot/`)

### 6.1 Capture

At deploy, fetch from `atlas-character`: name, gender, skin, face, hair, job id, gm flag, level, and
the equipped inventory. Equipment capture keeps only signed slots in 1–11 and 101–111 (FR-5.2);
everything else is dropped at the boundary so no out-of-range slot ever reaches the codec.

The snapshot is frozen (FR-5.3). No consumer subscribes to appearance-change events; the only writes
are deploy and re-deploy.

`jobId` is stored as the job *category* — `(jobId / 100) * 100` — per PRD §6, because that is what
the rank grouping and the branch mapping key on.

### 6.2 Re-deploy (FR-5.4)

`PATCH /api/player-npcs/{id}` refreshes appearance and ranks in place. Script id, object id and
position are immutable through this path. It emits `UPDATED`, which `atlas-channel` turns into a
re-broadcast of `IMITATED_NPC_DATA` to everyone on the map — no despawn/respawn, because the object
itself has not moved and the client re-reads the avatar from the data packet.

### 6.3 Ranks — D-3

PRD Open Question 4 asks whether world-job rank means "deployment order" or "current standing".

`atlas-rankings` exposes, per character per world, `Rank` (overall within the world) and `JobRank`
(within the world and job) — `services/atlas-rankings/atlas.com/rankings/ranking/rest.go`. There is
no cross-world ranking in Atlas.

| Field | Source | Meaning |
|---|---|---|
| `world_rank` | `atlas-rankings` `Rank` for (character, world) | current standing, frozen at deploy |
| `overall_rank` | `atlas-rankings` `Rank` for (character, world) | same value — see below |
| `world_job_rank` | counter over (tenant, world, job category) | deployment ordinal |
| `overall_job_rank` | counter over (tenant, job category) | deployment ordinal |

`overall_rank` equals `world_rank` because Atlas has no cross-world ranking to draw from. This is
recorded explicitly rather than left implicit: the column exists, carries a defensible value, and
becomes distinct for free if cross-world ranking is ever added. Inventing a cross-world number now
would be fabricating data.

The two job ranks are **deployment ordinals**, not current standing: `MAX(rank) + 1` over the
matching rows at insert time, inside the same transaction as the insert. This mirrors the reference
server's running counter, is stable (a statue's ordinal never changes because someone else levels),
and is the only reading consistent with FR-3.7's original intent of "position in job history".
Deriving them from live ranking data would make a frozen snapshot contain a number that was already
stale the moment it was written.

Both counters are computed under the same per-map serialization as allocation (§8), so two
simultaneous deployments cannot collide on an ordinal.

---

## 7. Packet and channel work

### 7.1 `IMITATED_NPC_DATA` codec

New codec in `libs/atlas-packet/npc/clientbound/`, encode-side, two arms, following the immutable
struct + `Operation()` writer-name convention of the neighbouring `spawn_request_controller.go` /
`remove_controller.go` pair.

Avatar arm (`0x01`): `scriptId`, `name`, `gender`, `skin`, `face`, a literal `0`, `hair`, then the
my-equip list of `(slot byte, itemId int)` terminated by `0xFF`, then the masked-equip list likewise
terminated by `0xFF`, then the cash weapon id or 0, then three zero ints.

Remove arm (`0x00`): `objectId`.

Version gating uses the `MajorAtLeast` idiom, never a raw `> N` comparison
(`docs/packets/IMPLEMENTING_A_PACKET.md`). Byte-fixture tests cover each of the nine columns from
C-4. Writer registration lands in the nine seed templates that carry an opcode and in neither
`template_gms_12_1.json` nor `template_gms_48_1.json`.

The field order above comes from the PRD, which derived it from the client handler
`CNpcPool::OnNpcImitateData`. **The implementer re-derives it from the GMS v95.1 IDB before writing
the codec** and treats a disagreement as authoritative against this document — repo/IDA evidence
outranks a written summary (CLAUDE.md evidence rule).

### 7.2 Spawn path — D-4

PRD §7 says to reuse `NpcSpawnRequestControllerWriter` for the spawn arm; FR-7.4 says Player NPCs
must never participate in controller assignment. These contradict.

**Chosen: plain `SPAWN_NPC`, no controller grant.** On map enter, ordinary NPCs are materialized by
`NpcSpawnWriter` and only *then* optionally granted control
(`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:656`,
`spawnNPCForSession`). The object exists after the spawn packet; the grant is a separate,
optional step. Player NPCs take the first half and skip the second. This satisfies FR-7.4 directly,
and granting control of an object that never moves would be meaningless at best.

Ordering is `SPAWN_NPC` then `IMITATED_NPC_DATA`, always (FR-7.1) — the client needs the object in
its pool before the avatar data can attach to it.

The plan carries a verification task: confirm against the IDA export that `CNpcPool::OnNpcEnterField`
alone materializes the object without a controller packet, on the lowest supported version. If that
check fails, the fallback is grant-then-immediately-revoke, and FR-7.4 is reinterpreted as "never
enters the controller *registry*" — which is the requirement that actually matters.

### 7.3 Controller exclusion (FR-7.4)

Player NPC object ids are never passed to `npc/controller`'s `TryClaim`, `ElectFor`, or
`ReleaseFor`, because `atlas-channel` learns about Player NPCs from `atlas-player-npcs`, not from
`data/npc.ForEachInMap` — the two spawn paths are separate code. Player exit hands off control only
for ids the registry holds, and no Player NPC id is ever in the registry, so no special-casing is
required in the exit path. A test asserts this: deploying a Player NPC, then exiting the map, must
produce no controller hand-off for its object id.

### 7.4 Object ids — D-5

PRD Open Question 2 asks where object ids come from.

WZ NPC object ids are assigned per map as `i + 1`
(`services/atlas-data/atlas.com/data/map/reader.go:424`) — small integers, typically under 100 per
map. `libs/atlas-object-id` allocates the *shared* monster/reactor/drop namespace starting at
`MinId = 1000000`, Redis-backed and monotonic.

Neither fits. Using the shared allocator would make a decade-lived statue's id depend on a Redis
counter that resets on a flush, at which point a re-issued id could collide with a persisted Player
NPC. Using raw `i + 1` collides with WZ NPCs immediately (FR-7.5).

**Chosen: a reserved band, deterministically derived.**

```
objectId = PlayerNpcObjectIdBase + (scriptId - 9900000)     // base = 100000
```

The band 100000–999999 sits above every WZ NPC id and below `objectid.MinId`, so it collides with
neither. The derivation is deterministic, needs no counter, survives any restart or Redis flush, and
is unique per (tenant, world) for free because `script_id` already is — enforced by the
`(tenant_id, world_id, script_id)` unique constraint. The id offset stays inside the band for the
whole 9900000–9906599 range.

`PlayerNpcObjectIdBase` is declared in `libs/atlas-object-id` next to `MinId`, with a doc comment
recording the reservation, so the invariant is discoverable by anyone who later considers lowering
`MinId`. The `(tenant_id, world_id, map_id, object_id)` unique constraint from PRD §6 is retained as
the backstop.

### 7.5 Map enter and broadcast

`atlas-channel` gains a `playernpc/` package: a read client for
`GET /api/player-npcs?filter[mapId]&filter[worldId]`, a spawn helper, and a consumer for the four
status events.

- **Map enter (FR-7.1):** a new `routine.Go` block alongside the existing NPC/monster/summon blocks
  in `SpawnForSelf`, issuing spawn-then-data per Player NPC. Reuses the established shape exactly.
- **`DEPLOYED` (FR-7.2):** spawn-then-data to every character on that map, on every channel of the
  world.
- **`REMOVED` (FR-7.3):** the `SPAWN_NPC_REQUEST_CONTROLLER` remove arm
  (`npcpkt.NpcControllerRevokeBody`, which is the client's NPC-removal packet) followed by the
  `IMITATED_NPC_DATA` remove arm, to everyone on the map on every channel.
- **`UPDATED`:** `IMITATED_NPC_DATA` avatar arm only.
- **`REPOSITIONED`:** remove-then-respawn for each listed object.

---

## 8. Concurrency, transactions, failure

### 8.1 Serialization — D-6

PRD §8 requires that two simultaneous deployments to the same map allocate neither the same script id
nor the same position.

`libs/atlas-lock` is a *leader-election lease*, not a mutex — wrong tool. The right one is already
under the service's own control: a **Postgres advisory lock** taken at the top of the deploy
transaction, keyed by a hash of (tenant, world, map).

```
BEGIN
  pg_advisory_xact_lock(hash(tenant, world, map))
  read in-use script ids and placed rectangles for (tenant, world[, map])
  allocate script id                       -- §4
  find position                            -- §5
  compute rank ordinals                    -- §6.3
  INSERT player_npcs + player_npc_equipment
COMMIT
then emit DEPLOYED
```

The lock is transaction-scoped, so it releases on commit or rollback with no cleanup path and no
possibility of a leaked lock. The read-decide-write sequence is inside one transaction, so allocation
and insertion are atomic (PRD §8: "a deployment that fails after allocating an id must not leak that
id"). The four unique constraints from PRD §6 remain the backstop against any path that bypasses the
lock.

Emitting `DEPLOYED` *after* commit, not inside the transaction, means a client is never told about an
NPC that then rolls back. The reverse failure — commit succeeds, event emission fails — leaves a
persisted NPC that appears on the next map enter. That is the acceptable direction.

Re-organization (§5.4) takes the same lock, which is what keeps a concurrent deploy from placing an
NPC into a layout that is being rewritten underneath it.

### 8.2 Level-up trigger (FR-1.5)

The `LEVEL_CHANGED` consumer must never block or fail the level-up. `LevelChangedStatusEventBody`
carries only `{channelId, amount, current}`
(`services/atlas-parties/atlas.com/parties/kafka/consumer/character/kafka.go:206`) — no job, no gm
flag, no map. The consumer therefore fetches the character to evaluate eligibility. That is not a
gap: the deployment needs the full appearance snapshot anyway, so the fetch is on the path
regardless. **No change to `atlas-character` is required**, contrary to PRD §7's conditional.

Consumer behaviour: if `current != MaxLevelFor(job)`, drop silently — this is the overwhelmingly
common case and must stay cheap, so the level check happens before any fetch. Otherwise fetch,
evaluate the remaining eligibility conditions, and dispatch a `DEPLOY` command. Any failure is logged
at warn with character id, target map and reason, and swallowed (FR-1.5).

Idempotency (FR-1.6) rests on the `(tenant_id, world_id, map_id, name)` unique constraint, so a
duplicate `LEVEL_CHANGED` redelivery cannot produce a second statue even if two consumers race.

### 8.3 Failure codes

`pool_exhausted`, `map_full`, `duplicate`, `ineligible` — surfaced as `409` with a distinguishing
`code` on REST (PRD §5), and carried on the command-failure path so the conversation operation
(§9.1) and the GM command (§9.2) can branch on them (FR-6.3, FR-8.3). `422` when the character or
map cannot be resolved.

---

## 9. Integration surfaces

### 9.1 Conversation engine

**Condition (FR-6.1).** A new `canSpawnPlayerNpc` condition. Conditions are declared once in
`libs/atlas-saga/validation.go` and re-exported by
`services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go`, which is where
evaluation lives. The evaluator calls
`GET /api/player-npcs/eligibility?characterId&mapId` and returns its boolean. The predicate is:
auto-deploy disabled for the tenant **and** level ≥ max class level **and** not a GM **and** no
Player NPC on that map.

Putting the predicate behind a single endpoint rather than reassembling it from four separate
conditions keeps the definition in one place — the same code path serves FR-1.1's automatic check and
FR-6.1's conversation check, so the two can never disagree.

**Operation (FR-6.2).** A new `deploy_player_npc` saga action in `libs/atlas-saga/model.go`,
handled by `atlas-saga-orchestrator`, which emits the `DEPLOY` command. It is not a local operation
(`atlas-npc-conversations`' `isLocalOperationType`) because it mutates cross-service state.
Parameters: optional `mapId`, defaulting to the character's current map. The step's failure status
carries the §8.3 code so a script can branch (FR-6.3).

Schema documentation is updated in `docs/npc_conversation_conversion_spec.md` and the conversation
JSON schema. Authoring the instructor conversation JSON is out of scope (FR-6.4).

### 9.2 GM commands

New `services/atlas-messages/atlas.com/messages/command/playernpc/commands.go`, registered via
`command.Registry().Add(...)`, following the `Producer`/`Executor` shape in `command/types.go`.

- Deploy for a named online character at the GM's current position and map, bypassing level and
  auto-deploy checks (`enforceEligibility: false`) but honouring script-id availability and the
  per-map duplicate rule (FR-8.1).
- Remove every Player NPC belonging to a named character, optionally map-scoped (FR-8.2).

Both report success or the specific §8.3 reason back to the invoking GM (FR-8.3). GM gating uses the
existing GM-level check in the command registry — no new authorization mechanism.

### 9.3 `atlas-data` changes

Two additions, both small:

1. `npc.RestModel` gains `imitate bool`; `services/atlas-data/atlas.com/data/npc/reader.go` parses
   `info/imitate` in the same block that already reads `trunkPut`, `hideName`, etc. (`reader.go:66`
   onward). This is the FR-3.4 source.
2. The ground-snap endpoint from D-2, wrapping the existing `calcPointBelow`.

### 9.4 Tenant configuration

Seven values, per C-3 registered as a new tenant configuration resource in `atlas-tenants` served at
`/tenants/{id}/configurations/player-npcs`, following the `rankings` resource pattern:

| Key | Default |
|---|---|
| `initialX` | 262 |
| `initialY` | 262 |
| `areaX` | 320 |
| `areaY` | 160 |
| `areaSteps` | 4 |
| `organizeArea` | true |
| `autoDeployEnabled` | true |

---

## 10. Testing strategy

**Pure-unit, no infrastructure** — `allocation/`, `position/`, `routing/`, `job.MaxLevelFor`. These
carry the feature's real logic and must be exhaustively table-tested:

- Grid lattice at each step, including the `areaSteps` exhaustion → `map_full`.
- Podium `(step, count)` encoding, platform/relative arithmetic, the `step == 0` error, and the
  `count >= 3 * step` step-raise.
- Allocator: in-branch hit; branch exhausted → global fallback; whole pool exhausted →
  `pool_exhausted`; **rejection of a missing template and of `imitate == 0`** (an explicit PRD
  acceptance criterion).
- Routing: every job category → correct map; the podium-map set.

**Service-level with a database** — the deploy transaction: idempotency under duplicate deploy,
re-deploy in place, rank-ordinal assignment, cascade delete of equipment rows, and **cross-tenant
isolation on the map-enter read path** (two tenants sharing a world id must see disjoint sets — a PRD
acceptance criterion).

**Codec** — byte fixtures for both arms on each of the nine version columns from C-4, with the
`packet-audit:verify` marker so the coverage-matrix cells actually promote. A cell that does not
promote is a failure, not a prose claim.

**Channel** — spawn ordering (`SPAWN_NPC` strictly before `IMITATED_NPC_DATA`), and the FR-7.4
assertion that no controller hand-off occurs for a Player NPC object id on player exit.

**Consumer** — `LEVEL_CHANGED` at max level deploys; below max level does not fetch at all; GM does
not deploy; already-deployed does not duplicate.

---

## 11. Risks

| Risk | Mitigation |
|---|---|
| Spawning a script id the client lacks crashes the client | FR-3.4 validation on the allocation path itself; unit tests for both rejection cases; the fallback pool is the same validated set |
| 193 shared slots fill up on a busy world | `pool_exhausted` is a clean, logged failure that spawns nothing. Operators wanting more must patch `Npc.wz` — out of scope per PRD §2, and now a documented, quantified consequence rather than a surprise |
| Codec field order wrong → visual corruption or crash | Re-derive from the GMS v95.1 IDB before writing; byte fixtures on all nine columns |
| D-4's spawn-without-controller assumption is wrong | Explicit IDA verification task in the plan, with grant-then-revoke as the named fallback |
| Cygnus max level unverified | Table returns 200 for every line — the behaviour the server actually implements — with a plan-phase verification task to set 120 if confirmed |
| Re-organization storms on a crowded map | Bounded by `areaSteps`; `organizeArea` is tenant-configurable and can be disabled |

---

## 12. Decisions summary

| Id | Decision | Deviates from PRD |
|---|---|---|
| D-1 | Branch-preferred script-id allocation with global fallback over the validated pool | FR-3.5 (strict branch) |
| D-2 | Ground snap stays in `atlas-data` behind a batched endpoint | Resolves Open Question 3 |
| D-3 | World/overall rank from `atlas-rankings` (equal, no cross-world data); job ranks are stored deployment ordinals | FR-3.7 (derived), resolves Open Question 4 |
| D-4 | Plain `SPAWN_NPC`, no controller grant | §7 Service Impact (controller-grant spawn) |
| D-5 | Object id deterministically derived into a reserved 100000–999999 band | Resolves Open Question 2 |
| D-6 | Postgres transaction-scoped advisory lock per (tenant, world, map) | Implements §8 concurrency |
| C-1 | Imitate pool measured at 193 ids with a lopsided distribution | Corrects FR-3.1, resolves Open Question 1 |
| C-2 | Hall of Fame map constants already exist; reference, don't redefine | Corrects FR-2.4 |
| C-3 | GM commands in `atlas-messages`; tuning config in `atlas-tenants` | Corrects §7 |
| C-4 | Corrected opcode→version mapping; nine templates, not eleven | Corrects §7 |
