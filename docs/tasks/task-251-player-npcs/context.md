# Player NPCs — Implementation Context

Task: task-251-player-npcs
Companion to [plan.md](plan.md). Read this before dispatching any implementer.

---

## 1. What changed between design and plan

The design (§7.1) required the implementer to re-derive `IMITATED_NPC_DATA` from the
GMS v95.1 IDB and to treat a disagreement as authoritative. That derivation was done at
plan time. It disagrees, and the disagreement is structural, not cosmetic. plan.md §0
records it in full; the summary:

| id | Correction | Evidence |
|---|---|---|
| P-1 | `IMITATED_NPC_DATA` is `count(byte)` + N × `{templateId int32, name string, AvatarLook}`. There are **no** `0x01`/`0x00` arms, no object id in the packet, and no remove arm. | `CNpcPool::OnNpcImitateData` — v61 `0x5efc2e`, v83 `0x6d97c6`, v95 `0x679500`; identical bodies |
| P-2 | `AvatarLook` is already implemented as `packetmodel.Avatar` (`libs/atlas-packet/model/avatar.go`), version gates included. The codec composes it. | `AvatarLook::Decode` — v83 `0x4e749a`, v95 `0x4f2c00`; compare `avatar.go:52,79,89` |
| P-3 | Removal needs a **new** `REMOVE_NPC` codec. `NpcControllerRevokeBody` only demotes control; it does not despawn. | `CNpcPool::OnNpcLeaveField` — v83 `0x6d9a25`, v95 `0x6792c0` (one `Decode4`); `remove_controller.go:14-21`; `STATUS.md:304` |
| P-4 | The ground endpoint mounts at `POST /data/maps/{mapId}/ground`, not `GET /api/maps/…`. A related foothold-only endpoint already exists. | `services/atlas-data/atlas.com/data/map/resource.go:32,44` |

Everything else in the design stands, including all six decisions D-1…D-6 and all four
corrections C-1…C-4. C-4's opcode table was independently re-confirmed against
`docs/packets/registry/*.yaml` during planning and is correct as written.

## 2. Key files by area

| Area | Anchor | Why it matters |
|---|---|---|
| Service scaffold | `services/atlas-notes/atlas.com/notes/` | The nearest DB-backed REST + Kafka + outbox service; `main.go`, `note/{entity,model,builder,administrator,provider,resource,rest,producer}.go` are the templates for Tasks 8, 12, 16, 17 |
| Registration | `docs/adding-a-new-service.md` | Four of its enumerations fail **silently**; `tools/service-registration-guard.sh` is the gate |
| Packet codecs | `libs/atlas-packet/npc/clientbound/spawn_request_controller.go`, `spawn_test.go` | Struct/`Operation()`/`Encode` shape and test scaffolding |
| Avatar wire | `libs/atlas-packet/model/avatar.go` | The whole `AvatarLook` payload, already version-gated |
| Equipment slots | `services/atlas-channel/atlas.com/channel/socket/model/avatar.go:10-31` | The repo's wire-slot convention: `Position * -1`; cash equip goes in the **visible** list, the real equip it masks goes in the **masked** list |
| Map geometry | `services/atlas-data/atlas.com/data/map/processor.go:132`, `model.go:35,78` | `calcPointBelow`, `findBelow`, `findById` — do not reimplement |
| Map enter | `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go:163-260` | `SpawnForSelf`; the NPC block at :229 is where the Player NPC block goes |
| Controller | `.../consumer.go:656-681` (`spawnNPCForSession`), `:620-640` (exit hand-off) | Shows spawn and grant are separate steps — D-4's basis |
| GM commands | `services/atlas-messages/atlas.com/messages/command/{types.go,registry.go,monster/commands.go}` | `Producer`/`Executor`; registered in `main.go:69-88` |
| Tenant config | `services/atlas-tenants/atlas.com/tenants/configuration/` — the `KiteConfig` set | The newest single-config-per-tenant resource; the exact shape to copy |
| Conditions | `libs/atlas-saga/validation.go`, `services/atlas-query-aggregator/.../validation/{model,rest,processor,context}.go` | `TransportAvailableCondition` is the nearest analogue — grep it to find every site a new condition must touch |
| Saga actions | `libs/atlas-saga/{model,payloads,unmarshal}.go`, `services/atlas-saga-orchestrator/.../saga/{handler,model,event_acceptance,character_extractor}.go` | `SpawnMonster` is the nearest analogue |

## 3. Decisions carried forward from the design

- **D-1** Branch-preferred allocation with a global fallback over the validated pool. The
  stock-WZ imitate pool is 193 ids, lopsidedly distributed (design §1 C-1); strict branch
  allocation would make the feature non-functional for Explorer classes.
- **D-2** Ground snap stays in `atlas-data`, batched (one round trip per lattice step).
- **D-3** `world_rank` and `overall_rank` both come from `atlas-rankings` `Rank` and are
  equal — Atlas has no cross-world ranking. The two job ranks are stored deployment
  ordinals, not live standings.
- **D-4** Plain `SPAWN_NPC`, no controller grant.
- **D-5** `objectId = 100000 + (scriptId - 9900000)` — deterministic, needs no counter,
  survives a Redis flush.
- **D-6** A transaction-scoped Postgres advisory lock keyed by `(tenant, world, map)`.
  `libs/atlas-lock` is a leader-election lease and is the wrong tool.

## 4. Cross-service contracts to trace by hand

`tools/verify.sh` cannot see a seam defect. Three seams need a manual trace before review:

1. **`atlas-player-npcs` → `atlas-channel`** — the four status events (`DEPLOYED`,
   `UPDATED`, `REMOVED`, `REPOSITIONED`). Task 17 defines them, Task 19 consumes them.
   Check that Task 19's tests assert the *new* payload shape, not a copied one.
2. **`atlas-tenants` → `atlas-player-npcs`** — the `player-npcs` configuration attribute
   names. Task 20 serves them, Task 13 decodes them. The rankings resource documents this
   coupling explicitly at `configuration/rest.go:817-820`; the same trap applies here.
3. **`libs/atlas-saga` → two services** — Task 22's condition and action must be declared
   once and consumed in `atlas-query-aggregator` and `atlas-saga-orchestrator`. A
   half-applied change compiles in the library and fails in a consumer.

## 5. Tasks deliberately left large

`tools/plan-lint.sh` F4 warns on tasks over ~6 files or crossing services. Five tasks
exceed it on purpose:

| Task | Size | Why it is not split |
|---|---|---|
| 7 — template routing | 10 templates + 1 test | One mechanical edit repeated per file, which batches fine. Splitting it would risk a partial routing set — the exact condition that produces a silent mis-encode |
| 8 — service scaffold | ~18 files | One registration checklist, not eighteen changes. `tools/service-registration-guard.sh` is a single all-or-nothing gate; a half-registered service fails it and is not independently reviewable |
| 13 — read clients | 5 clients × ~3 files | The same four-file template against five upstreams. Each is trivial; the value of splitting is lower than the cost of five dispatches |
| 19 — channel wiring | 7 files, one service | Spawn, broadcast, writer registration and controller exclusion are one behavioural contract; the ordering assertion (`SPAWN_NPC` strictly before `ImitatedNPCData`) spans them |
| 22 — condition + action | 3 modules | A shared-library constant plus its two consumers. Splitting leaves a consumer that does not compile |

Tasks 5 and 6 are single codecs but carry a per-version verification fan-out; if either
implementer reports `PARTIAL` at the tool-call cap, split it by version, not by file.

`plan-lint` also flags Tasks 14, 16 and 17 as "multi-service". Those are false positives:
each edits only `atlas-player-npcs`, and the second service in the Files block is a
**read-only anchor** — `atlas-channel`'s slot convention (Task 14), the tenant
route-ordering hazard (Task 16), the `LevelChangedStatusEventBody` definition and the
`atlas-notes` consumer template (Task 17). Nothing outside `atlas-player-npcs` is edited
in any of the three.

## 6. Known risks

| Risk | Mitigation |
|---|---|
| Spawning a script id the client lacks crashes the client | `Usable(id)` (exists **and** `imitate == 1`) is checked on the allocation path itself; Task 10 tests both rejection cases; the global fallback scans the same validated set |
| 193 shared slots fill up on a busy world | `pool_exhausted` is a clean, logged failure that spawns nothing. More slots require patching `Npc.wz` — out of scope per PRD §2 |
| `packetmodel.Avatar` ranges over Go maps | Encoding order for multi-equip avatars is non-deterministic. The Task 5 golden-byte fixture uses exactly one equip and no masked equip. If a deterministic multi-equip encoding is ever needed, that is a change to `avatar.go` affecting already-verified codecs — out of scope here |
| Cygnus max level unverified | Task 1's table returns 200 for every line, which is what `atlas-character` actually enforces; Task 2 is an explicit verification task with a "record the negative result" branch |
| Re-organization storms on a crowded map | Bounded by `areaSteps`; `organizeArea` is tenant-configurable |
| `REMOVE_NPC` version spread | Task 6 confirms one `Decode4` per version before pinning; three versions (v61/v83/v95) were already confirmed at plan time, and all three function bodies are `0x5e` bytes |

## 7. Operator steps this task cannot complete

Both are recorded in plan.md Task 8 and must be handed back:

- Create `atlas-player-npcs-main` on postgres.home before merge — pods crash-loop on
  SQLSTATE 3D000 until it exists (`docs/adding-a-new-service.md` §6.1).
- Flip the new GHCR package to public after the first image push, or the pod sits in
  `ImagePullBackOff` against a 401 while CI reports a clean build (§6b).
