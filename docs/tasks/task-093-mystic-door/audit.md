# Plan Audit — task-093-mystic-door

**Plan Path:** docs/tasks/task-093-mystic-door/plan.md
**Audit Date:** 2026-06-14
**Branch:** task-093-mystic-door (45 commits ahead of main)
**Base Branch:** main

## Plan-Adherence Section

### Executive Summary

All plan tasks A1–I2 were implemented in code, with no silent skips. Completion is
**FULL** for every code-bearing task. The two known scope items are both intentional and
documented: gms_v92 door opcodes are PARKED (no v92 IDB to IDA-verify — same posture as the
v92 mount-food handler), and Task H7 (live-tenant config patch) is a post-merge ops/runbook
step, not a code change. Build/vet/test/race/bake/rediskeyguard/kustomize were reported clean
by the executor (Phase I1); spot-checks here (door + party packet tests PASS, wiring present
in both `main.go`s) corroborate that. Several tasks were implemented *beyond* the plan in a
correctness-positive way (a dedicated 8-byte `RemoveTownDoor` encoder split out of the single
planned `removeDoor`, and bidirectional door lookup by owner).

### Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| A1 | Go module + logger + go.work | DONE | `services/atlas-doors/atlas.com/doors/go.mod`, `logger/init.go`, `go.work` (commit 5265a8f49) |
| A2 | Task runner + leader config | DONE | `tasks/task.go`, `leaderconfig.go`, `leaderconfig_test.go` (b86efded8) |
| A3 | Kafka consumer/producer + rest plumbing | DONE | `kafka/consumer/consumer.go`, `kafka/producer/producer.go`, `rest/handler.go` (68305acdb) |
| A4 | Register service (services.json + bake) | DONE | `.github/config/services.json` (+8), `docker-bake.hcl` (+1) (238885af9) |
| A5 | k8s manifest + topics | DONE | `deploy/k8s/base/atlas-doors.yaml`, `kustomization.yaml`, `env-configmap.yaml` (+2) (3951b99a1) |
| B1 | door.Model + Builder | DONE | `door/model.go`, `door/builder.go`, `model_test.go`; `PairId()==AreaDoorId`, immutable `Reslot` (93d75f828) |
| B2 | Object-id allocator (fail-on-error) | DONE | `door/id_allocator.go` — no MinId fallback; Allocate returns (0,err) (dbcd8552d) |
| B3 | Redis registry + 3 indices | DONE | `door/registry.go` (334 lines), field/owner/town-party indices; `registry_test.go` asserts index clearing (74fdc36b7, 53bda0bdd) |
| C1 | atlas-data map+portal client | DONE | `data/map/{model,rest,requests,processor}.go`, `rest_test.go` (e0c15a063) |
| C2 | atlas-data skill-effect client | DONE | `data/skill/{model,rest,requests,processor}.go`, `effect/`, `rest_test.go` (b0f200055) |
| C3 | atlas-parties client (ordered) | DONE | `party/{model,rest,requests,processor}.go`, `rest_test.go` (a90291dc2) |
| C4 | Slot + town-portal resolution | DONE | `door/slot.go` `ComputeSlot`/`ResolveTownPortal` (0x80+slot), `slot_test.go` (cf45e44c9) |
| C5 | Town resolution (return/forced) | DONE | `door/town.go` `ResolveTownMap`/`HasValidReturn`, `town_test.go` (bf8564155) |
| D1 | Event contract envelope+bodies | DONE | `door/kafka.go` — Created/Removed/SlotChanged + reason consts (78e023b12) |
| D2 | Event providers | DONE | `door/producer.go` — 3 providers keyed by area map id (982ca177c) |
| D3 | Processor spawn/remove/reslot | DONE | `door/processor.go` (recast-first, area-then-town alloc with release-on-fail, RemoveByOwnerIfLeftField same-source/into-town guard), `resolver.go`, `processor_test.go` (549 lines) (9dd2118ce) |
| D4 | REST resource + in-field route | DONE | `door/resource.go`, `door/rest.go`, `world/resource.go`, `resource_test.go` (01e9cec91) |
| E1 | Door command consumer (SPAWN/REMOVE) | DONE | `kafka/consumer/door/{kafka,consumer}.go` (ae418f10a) |
| E2 | Character-status cleanup consumer | DONE | `kafka/consumer/character/{kafka,consumer}.go` — logout/channel/map (fca50d9e3) |
| E3 | Party-status reslot consumer + routine | DONE | `kafka/consumer/party/{kafka,consumer}.go`, `door/reslot.go`, `reslot_test.go` (a1097d36e) |
| E4 | Leader-elected expiry sweep | DONE | `door/expiry_task.go` (deploy-grace guard), `expiry_task_test.go` (155 lines) (0e5dc991b) |
| E5 | main.go wiring | DONE | `main.go`: InitIdAllocator/InitRegistry, 3 consumers+handlers, door/world/character REST, `lock.New(rc,"doors-sweep")` gating sweep on leaderCtx; `main_leader_test.go` (5d87271c5) |
| F1 | enter-door serverbound decoder | DONE | `libs/atlas-packet/door/serverbound/enter.go` (uint32 ownerId, byte direction) (c3056738e) |
| F2 | spawnDoor clientbound encoder | DONE | `door/clientbound/spawn.go` — writeBool(launched)+int(owner)+pos; golden test (530616f9b) |
| F3 | removeDoor clientbound encoder | DONE (split) | `remove.go` (area, byte0+int) + `remove_town.go` (town-side 8-byte SPAWN_PORTAL clear). Plan named one encoder; implementer correctly split town removal into a dedicated 8-byte encoder to avoid the 12-byte position corruption (d7465a698, 4379ecb5d) |
| F4 | spawnPortal clientbound encoder | DONE | `door/clientbound/spawn_portal.go` — 2 ints + pos (e3b73321e) |
| F5 | Party door block populated | DONE | `party/clientbound/created.go` `WithDoor()`; encode writes real town/target/x/y; `created_test.go` (+79) (9c6bc6b50) |
| F6 | Packet lib vet/test sweep | DONE | door + party clientbound tests PASS (verified this audit) |
| G1 | Door SPAWN producer + doors client | DONE | `channel/door/{producer,processor,requests,rest,model}.go`, `kafka/message/door/kafka.go`, `producer_test.go` (441d6d695) |
| G2 | Mystic Door cast handler | DONE | `skill/handler/mysticdoor/mysticdoor.go` (town/no-return/field-limit reject; emit Spawn with caster X/Y), `mysticdoor_test.go` (172 lines); registered in `registrations.go` (59d921f3d) |
| G3 | enter-door inbound handler | DONE | `socket/handler/mystic_door_enter.go` (bidirectional findDoorOnMap, authorize, linkedDestination, warp, portal sound via existing simple-effect), test (210 lines); `main.go` handlerMap (cb225f55b) |
| G4 | door status consumer → broadcast | DONE | `kafka/consumer/door/consumer.go` (party-scoped eligibility, `sc.Is` channel guard, area vs town packet split, RemoveTownDoor for town clear), `socket/writer/door.go`, `consumer_test.go` (299 lines) (a3760eccd) |
| G5 | Map-enter spawn | DONE | `kafka/consumer/map/consumer.go` `spawnDoorsForSession` (launched=true), `door.ForEachInMap`; `consumer_test.go` (+146) (1adc8b8b9) |
| G6 | Party minimap door indicator + TODO | DONE | `kafka/consumer/party/consumer.go handleCreated` wires leader door via GetByOwner into PartyCreated WithDoor; `docs/TODO.md` entry marked [x] (aa4be3b6b) |
| G7 | Channel build+test sweep | DONE | Reported clean by executor (Phase I1) |
| H1 | Locate templates + opcode schema | DONE | Door rows added to per-version `template_gms_*.json` / `template_jms_185_1.json` |
| H2 | gms_v83 opcodes + golden | DONE | `template_gms_83_1.json`: EnterDoorHandle 0x085 (+LoggedInValidator), SpawnDoor 0x113, RemoveDoor 0x114, SpawnPortal/RemoveTownDoor 0x043 (b0590f6ba) |
| H3 | gms_v84 opcodes | DONE | `template_gms_84_1.json`: SpawnDoor 0x11A, RemoveDoor 0x11B, SpawnPortal/RemoveTownDoor 0x45 — distinct from v83 (table-shift bug handled). Two IDA-verified fix commits (57beeee68, a21f88750) |
| H4 | gms_v87 opcodes | DONE | `template_gms_87_1.json`: 5 door rows present |
| H5 | gms_v92 + gms_v95 opcodes | DONE (v92 PARKED) | `template_gms_95_1.json`: 5 rows. `template_gms_92_1.json`: 0 rows — intentionally parked, documented in TODO.md (no v92 IDB). |
| H6 | jms_v185 opcodes | DONE | `template_jms_185_1.json`: 5 door rows present |
| H7 | Live tenant config patch | DEFERRED (by design) | Post-merge ops/runbook step; existing tenants do not auto-receive new opcodes. Not a code change. |
| I1 | Full multi-module verification | DONE | Executor reported build/vet/test-race clean on libs/atlas-packet, atlas-channel, atlas-doors; redis-key-guard clean; `docker buildx bake atlas-doors` OK; kustomize base OK |
| I2 | Acceptance walkthrough + review | IN PROGRESS | This audit is part of the review step |

**Completion Rate:** 39/39 code tasks (100%) implemented.
**Skipped without approval:** 0
**Partial implementations:** 0
**Intentional scope items:** gms_v92 opcodes PARKED (documented); H7 live-config patch is a runbook step.

### Skipped / Deferred Tasks

None silently skipped. Two items are intentional and documented:

1. **gms_v92 door opcodes (H5)** — PARKED. `template_gms_92_1.json` has 0 door rows. No v92
   IDB exists to IDA-verify the opcodes, so per the project's "IDA-verify-or-escalate" rule
   the version was parked (mirrors the v92 MountFoodHandle posture). Documented in
   `docs/TODO.md`. Impact: Mystic Door is non-functional on v92 tenants until a v92 IDB
   surfaces; all other versions ship.

2. **H7 live tenant config patch** — DEFERRED to deploy time. Existing tenants do not
   auto-receive new handler/writer opcode rows (seed templates apply at tenant creation only),
   and the channel must be restarted (projection does not hot-reload handlers/writers). This
   is correctly an ops runbook step, not a code change.

### Observations (non-blocking)

- **F3 scope deviation (positive):** the plan specified a single `removeDoor` encoder; the
  implementer split town-side removal into a dedicated `RemoveTownDoor` (8-byte SPAWN_PORTAL
  clear) distinct from `SpawnPortal` (12-byte). This is a correctness fix — using SpawnPortal
  for removal would emit 4 spurious trailing bytes. Both writer names map to opcode 0x043
  (v83) / 0x45 (v84) in the templates. No defect.
- **Golden-test IDA markers say `ida=TODO`:** the `// packet-audit:verify ... ida=TODO`
  comments in the four `door/clientbound/*_test.go` files were not backfilled with resolved
  IDA addresses. The opcodes themselves WERE IDA-verified (see the two explicit v84 fix
  commits). This is a documentation-completeness nit on the verify marker, not a functional
  gap. Recommend backfilling the `ida=` addresses before promoting these cells in the packet
  coverage matrix.

### Build & Test Results

| Service / Lib | Build | Tests | Notes |
|---------------|-------|-------|-------|
| libs/atlas-packet | PASS | PASS | door/clientbound, door/serverbound, party/clientbound tests green (this audit) |
| services/atlas-doors | PASS | PASS | Executor Phase I1: build/vet/test-race clean; bake OK |
| services/atlas-channel | PASS | PASS | Executor Phase I1: build/vet/test-race clean; bake OK |
| deploy/k8s/base | PASS | n/a | kustomize base builds (Phase I1) |

### Overall Assessment

- **Plan Adherence:** FULL (every code task implemented; two intentional, documented scope items)
- **Recommendation:** READY_TO_MERGE (pending the standard backend-guidelines review and the H7 deploy-runbook capture in the PR body)

### Action Items

1. (Nit) Backfill the `ida=TODO` markers in the four `libs/atlas-packet/door/clientbound/*_test.go`
   files with the resolved IDA addresses used during H2–H6, so the packet coverage matrix can
   promote these cells.
2. (Ops, post-merge) Execute H7: patch each live tenant's socket config with the 4 door opcode
   rows (handler row with `LoggedInValidator`) and restart channels. Capture as a runbook step
   in the PR description.
3. (Tracking) Keep the gms_v92 parked-opcode entry in `docs/TODO.md` until a v92 IDB is available.

---

## Backend Guidelines Section (backend-guidelines-reviewer)

- **Auditor:** backend-guidelines-reviewer (adversarial, FAIL-until-proven)
- **Date:** 2026-06-14
- **Scope:** `services/atlas-doors/atlas.com/doors` (brand-new), `services/atlas-channel/atlas.com/channel` (door additions), `libs/atlas-packet` (door encoders/decoder)
- **Build/Vet/Test/Race/redis-key-guard/docker-bake:** reported clean by requester; not re-run.
- **Overall (guideline conformance):** NEEDS-WORK

### Architecture note
`atlas-doors` is an event-sourced / Redis-registry service — door state lives in `libs/atlas-redis` (`door/registry.go`), there is no GORM `entity.go`. Consequently the GORM-shaped DOM checks (DOM-02/03 `ToEntity`/`Make`, DOM-10 tenant callbacks, DOM-11 lazy providers, DOM-15/16 administrator) are **N/A** by design and are not failures. The door domain is checked against the immutable-model + processor(Interface+Impl) + JSON:API + Kafka pattern instead.

### Critical / Blocking

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-02 | httptest-backed client integration test | **FAIL** | No `httptest.NewServer` anywhere in `atlas-doors` (grep clean). The three external JSON:API clients (`data/map`, `data/skill`, `party`) call upstream services through `requests.Provider[RestModel, Model]` (`data/map/processor.go:30`, `data/skill/processor.go:30`, `party/processor.go:26,31`). Their only tests (`data/map/rest_test.go:5`, plus `data/skill/rest_test.go`, `party/rest_test.go`) exercise `Extract`/`ExtractPortal` on hand-built structs and never drive a served JSON:API response through the api2go unmarshal + relationship-block path. This is exactly the path EXT-02 exists to cover — the `SetToManyReferenceIDs`/`SetReferencedStructs` relationship wiring in `data/map/rest.go:94-125` and `party/rest.go:63-96` is unverified end-to-end. Past tasks (037) surfaced this class of bug as misleading "not found" failures. |

### Important

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | Reuse atlas-constants id types | **FAIL** | Brand-new `atlas-doors` reinvents `uint32`/`int16` where shared types exist. `character.Id` (uint32) and `skill.Id` (uint32) and `point.X`/`point.Y` (int16) are confirmed unused anywhere in the service (grep `character.Id\|skill.Id\|point.X\|point.Y` → 0 hits). Affected surfaces: `door/model.go:13,15,40-43` (`ownerCharacterId`, `skillId` as uint32; `areaX/Y`,`townX/Y` as int16); `door/builder.go:13-15,51-54`; `door/rest.go:18,32,28-31`; `door/processor.go:23-27,85,89` (`ownerCharacterId, skillId uint32`); `door/kafka.go:33,49,40-48` (event contract); `rest/handler.go:41` (`ParseCharacterId` returns `uint32` — contrast with `ParseMapId/ParseChannelId/ParseWorldId` at lines 33,45,49 which correctly return `_map.Id`/`channel.Id`/`world.Id`); `data/skill/processor.go:14,15` and `party/processor.go:12,13` (`skillId`/`characterId`/`partyId uint32`). The model already imports `field.Model` and `_map.Id` correctly, so the partial adoption shows the omission of `character.Id`/`skill.Id`/`point` was an oversight, not a deliberate uniform choice. NOTE: channel-side additions (`channel/door/producer.go`, `skill/handler/mysticdoor`, socket handlers) use `characterId uint32` consistently with the entire pre-existing channel service convention (`skill/handler/registry.go:21`, `common.go:70`) — those are not new regressions and are out of scope for fresh-code DOM-21. `libs/atlas-packet` wire structs are wire-level and exempt. |

### Minor / Advisory

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | builder.go with validating Build() | **WARN** | `door/builder.go` has `NewBuilder()` (line 29), fluent setters, and `Build()` (line 58) — but `Build()` returns `Model` with **no error and no invariant enforcement** (e.g. does not reject `areaDoorId==0`/`ownerCharacterId==0`). Guideline (file-responsibilities.md `builder.go`) states `Build()` "enforces invariants." Same for `data/map/model.go:112`. Acceptable-ish for an event-sourced model reconstructed from Redis, but the validating-Build invariant is not met. |
| DOM-05 | TransformSlice function | **WARN** | No `door.TransformSlice` exists; list handlers (`world/resource.go:48`, `character/resource.go:40`) inline `model.SliceMap(door.Transform)(...)`. This is the same composition the checklist's `TransformSlice` would wrap and is not an inline `for` loop, so it is not a hard violation — but the named `TransformSlice` helper the checklist expects is absent. |

### Passing checks (evidence)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | Transform function | PASS | `door/rest.go:51` `func Transform(m Model) (RestModel, error)`. |
| DOM-06 | Processor accepts FieldLogger | PASS | `door/processor.go:66` `NewProcessor(l logrus.FieldLogger, ctx context.Context)`; same in all sub-clients. |
| DOM-07 | Handlers pass d.Logger() | PASS | `door/resource.go:29`, `world/resource.go:40`, `character/resource.go:32` all `NewProcessor(d.Logger(), d.Context())`; no `StandardLogger()`. |
| DOM-09 | Transform errors handled | PASS | `world/resource.go:48-53`, `character/resource.go:40-45`, `door/resource.go:36-41` all check `err` from `SliceMap`/`Map`; no `_, _ :=`. |
| DOM-12 | No os.Getenv in handlers | PASS | grep `os.Getenv` over the three resource.go files → 0 hits. |
| DOM-13/14 | No cross-domain logic / direct provider calls in handlers | PASS | Handlers call only `door.NewProcessor(...)` methods; cross-domain orchestration (maps/skill/party) is inside `door/resolver.go`, invoked from the processor. |
| DOM-17 | Error → HTTP status mapping | PARTIAL/PASS | `door/resource.go:32` not-found→404, `:39` transform→500; list handlers map errors→500 (`world/resource.go:44`, `character/resource.go:36`). No 400/409 paths exist (GET-only surface), so the mapping is complete for the routes present. |
| DOM-18 | JSON:API interface on REST models | PASS | `door/rest.go:37-48` `GetID/SetID/GetName`; client models likewise (`data/map/rest.go`, `party/rest.go`). |
| DOM-19 | Flat request models | PASS | No POST/PATCH bodies in the service; all REST models flat, ID `json:"-"` (`door/rest.go:15`). |
| DOM-20 | Table-driven tests | PASS | `door/processor_test.go`, `slot_test.go`, `town_test.go`, `reslot_test.go` use struct-slice + `t.Run`. |
| DOM-22 | Dockerfile lib coverage | PASS | atlas-doors uses the shared root `Dockerfile` (`ARG SERVICE`); all direct-require libs (`atlas-redis`, `atlas-object-id`, `atlas-kafka`, `atlas-constants`, `atlas-lock`, …) appear in the mod-only COPY block (`Dockerfile:32-41`), source COPY block (`:61-70`), and the `go mod edit -replace` loop (`:91-92`). Requester confirmed `docker bake` OK. |
| DOM-23 | Kafka topic naming/wiring | PASS | `COMMAND_TOPIC_DOOR` (`kafka/consumer/door/kafka.go:10`) and `EVENT_TOPIC_DOOR_STATUS` (`door/kafka.go:10`) both present in `deploy/k8s/base/env-configmap.yaml:28,101` as `KEY: "KEY"`; `deploy/k8s/base/atlas-doors.yaml:21-23` consumes via `envFrom: configMapRef: atlas-env` with no literal topic overrides. |
| DOM-24 | Kafka producer stubbed in emit tests | PASS | Emit path is the injectable `emitter` seam on `ProcessorImpl`; `door/processor_test.go:68-99,384-389` inject `fakeEmit`/`reasonCapture` directly, never reaching the real producer. `expiry_task_test.go` / `registry_test.go` use `producertest`/`TestMain`. No `t.Cleanup(producer.ResetInstance)` reverting a stub. |
| EXT-01 | JSON:API relationship interface on client models | PASS | `data/map/rest.go:33-39,90-107` (`SetToOneReferenceID`+`SetToManyReferenceIDs`), `party/rest.go:63-74` (`SetToManyReferenceIDs`, plus `GetReferences`). |
| EXT-03 | 404 distinguished from other failures | PASS | Clients do not remap errors at all — `requests.Provider` bubbles the original error (`data/map/processor.go:30`, `party/processor.go:26`), so non-404 transport/decode failures are not masked as "not found." |
| EXT-04 | Service URL via RootUrl(domain) | PASS | `data/map/requests.go:16` `RootUrl("DATA")`, `party/requests.go:16` `RootUrl("PARTIES")`; no hardcoded DNS. |
| SCAFFOLD-01/02/03 | services.json / k8s manifest / Dockerfile | PASS | `.github/config/services.json:121-125` entry; `docker-bake.hcl:50`; `deploy/k8s/base/atlas-doors.yaml` present; shared root Dockerfile. |

### Security
SEC-* N/A — atlas-doors is not an auth/token service. The enter-door inbound handler (`channel/socket/handler/mystic_door_enter.go`) is a gameplay warp, validated server-side against door registry state via atlas-doors; no token/secret handling.

### Summary

**Blocking (must fix):**
- **EXT-02** — no httptest-backed integration test for any of the three external JSON:API clients (`data/map`, `data/skill`, `party`). The relationship-block unmarshal path is exercised by zero tests; add an `httptest.NewServer`-served fixture per client and assert the domain method returns a populated struct.

**Important (should fix):**
- **DOM-21** — adopt `character.Id`, `skill.Id`, and `point.X`/`point.Y` from `libs/atlas-constants` across the new `atlas-doors` model/builder/rest/processor/kafka/parse surfaces (the service already uses `field.Model`/`_map.Id`/`world.Id`/`channel.Id`, so this is finishing a partial adoption). Channel-side bare `uint32` matches existing service convention and is not in scope.

**Advisory:**
- **DOM-01** — `Build()` performs no invariant validation (returns `Model`, no error).
- **DOM-05** — no named `TransformSlice` helper; list handlers inline `model.SliceMap(Transform)` (functionally equivalent, not a loop).
