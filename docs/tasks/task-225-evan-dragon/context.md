# Evan Dragon Entity — Implementation Context

Companion to [`plan.md`](./plan.md). Read this first: it is the map of "what
already exists that this work copies from", plus every fact the plan's code
depends on with a `file:line` citation.

PRD: [`prd.md`](./prd.md) · Design: [`design.md`](./design.md)

---

## 1. What is being built

Four packet codecs + one new service + a channel-side surface, so that an Evan
character's dragon renders on every client in the field.

| Piece | Location |
|---|---|
| Codecs | `libs/atlas-packet/dragon/{clientbound,serverbound}/` (new) |
| State owner | `services/atlas-dragons/atlas.com/dragons/` (new service) |
| Wire emitters / decoder | `services/atlas-channel/atlas.com/channel/` (new files in existing service) |
| Routing | 6 of 11 templates in `services/atlas-configurations/seed-data/templates/` |
| Matrix | `docs/packets/audits/status.json` — 24 cells |

---

## 2. The reference implementation: `atlas-summons`

`atlas-summons` is the closest existing analogue — a field-scoped, owner-keyed,
database-free entity service with a Redis registry and a channel-side broadcast
consumer. **Every file the plan creates has a named sibling there.** When in
doubt, open the summon file and match its shape.

| New dragon file | Copy the shape from |
|---|---|
| `dragons/dragon/model.go` | `services/atlas-summons/atlas.com/summons/summon/model.go` |
| `dragons/dragon/registry.go` | `.../summon/registry.go` (`storedSummon`, `toStored`/`fromStored`, `Registry`, `InitRegistry`) |
| `dragons/dragon/kafka.go` | `.../summon/kafka.go` (`StatusEvent[E]` envelope) |
| `dragons/dragon/producer.go` | `.../summon/producer.go` (`createdEventProvider` etc.) |
| `dragons/dragon/resource.go` | `.../summon/resource.go` (`RestModel`, `GetName()`, `Transform`) |
| `dragons/dragon/rest.go` | `.../summon/rest.go` (`InitResource`, `handleGetSummonById`) |
| `dragons/world/resource.go` | `.../world/resource.go` (in-field list endpoint) |
| `dragons/rest/handler.go` | `.../rest/handler.go` (`ParseWorldId`/`ParseChannelId`/…) |
| `dragons/kafka/consumer/consumer.go` | `.../kafka/consumer/consumer.go` (verbatim, s/summons/dragons/) |
| `dragons/kafka/consumer/character/` | `.../kafka/consumer/character/` (envelope + despawn cascade) |
| `dragons/kafka/consumer/dragon/` | `.../kafka/consumer/summon/` |
| `dragons/main.go` | `.../summons/main.go` (minus the leader-election sweep tasks) |
| channel `socket/writer/dragon.go` | `services/atlas-channel/.../socket/writer/summon.go` |
| channel `socket/handler/dragon_move.go` | `.../socket/handler/summon_move.go` |
| channel `dragon/{model,requests,processor,producer}.go` | `.../summon/*.go` |
| channel `kafka/consumer/dragon/consumer.go` | `.../kafka/consumer/summon/consumer.go` |
| channel `kafka/message/dragon/kafka.go` | `.../kafka/message/summon/kafka.go` |

`atlas-dragons` does **not** need: an id allocator (the owner character id is
the primary key), an owner index (same reason), expiry/sweep tasks, leader
election, or any of the effect/stats/inventory REST clients summons carries.

---

## 3. Facts the plan's code depends on (all verified)

### 3.1 Opcodes — registry-confirmed

Design §7.3's table was cross-checked against `docs/packets/registry/*.yaml`.
All 24 entries match:

| Op | dir | v83 | v84 | v87 | v92 | v95 | jms185 |
|---|---|---|---|---|---|---|---|
| `SPAWN_DRAGON` | cb | 181 | 185 | 194 | 209 | 206 | 187 |
| `MOVE_DRAGON` | cb | 182 | 186 | 195 | 210 | 207 | 188 |
| `REMOVE_DRAGON` | cb | 183 | 187 | 196 | 211 | 208 | 189 |
| `MOVE_DRAGON` | sb | 181 | 186 | 193 | 211 | 214 | 185 |

All four ops are `n-a` on `gms_v48`/`v61`/`v72`/`v79` and `incomplete` on the six
in-scope columns (`docs/packets/audits/status.json`). `gms_v12` is not a matrix
column at all, so `template_gms_12_1.json` is simply untouched.

Sanity check on v83: the summon writers already occupy `0xAF`–`0xB4`
(`template_gms_83_1.json`), so the dragon writers at `0xB5`–`0xB7` sit directly
after them, and serverbound `0xB5` is free in the handler array.

### 3.2 Evan is absent from the v83 job table

`libs/atlas-constants/job/version_gms_83_1_gen.go` contains zero `Evan` entries.
`version_gms_84_1_gen.go:73,78-87` is the first table binding `2001` and
`2200`–`2218`. This is why `HasDragon` (plan Task 3) is written against
`constants.For(...).Job.Resolve` — a v83 tenant fails to resolve and the
predicate returns false with no version special-case.

### 3.3 The Evan identity block is exclusively Evan

`libs/atlas-constants/job/identities_gen.go:83-92` — `EvanStage1 = 2200`,
`EvanStage2 = 2210` … `EvanStage10 = 2218`. Nothing else occupies `2200`–`2218`
(`AranStage4 = 2112` below, nothing above until the block ends). A closed range
comparison `id >= job.EvanStage1 && id <= job.EvanStage10` over **Identity**
values is therefore exact. Note this compares `job.Identity`, not `job.Id`, so
`tools/skill-job-id-guard.sh` is not implicated.

### 3.4 Character status events already carry everything

- `LOGIN` / `LOGOUT`: `channelId, mapId, instance`
  (`services/atlas-character/.../kafka/message/character/kafka.go:283-293`)
- `JOB_CHANGED`: `channelId, jobId` (`:295-298`)
- `MAP_CHANGED` / `CHANNEL_CHANGED`: mirrored already in
  `services/atlas-summons/.../kafka/consumer/character/kafka.go:38-56`

No producer change in `atlas-character`. `atlas-dragons` mirrors the envelope
locally exactly as `atlas-summons` does.

`JOB_CHANGED` carries no map id, and none of the five carry `x`/`y` — hence the
character REST fetch in `Create` (plan Task 5).

### 3.5 Wire primitives

- `w.WriteInt32(int32)` exists (`libs/atlas-socket/response/writer.go:31`) and
  writes 4 bytes — required for the dragon's 4-byte coordinates.
- `r.ReadInt32()` (`libs/atlas-socket/request/reader.go:72`), `r.Available()`
  (`:133`), `w.WriteByteArray` (`writer.go:65`).

### 3.6 Redis registry primitives

`libs/atlas-redis`:
- `NewRegistry[K,V](client, namespace, keyFn)` — `Get`/`Put`/`Remove`/`Exists`/
  `RemoveExisting`/`Update`/`GetAll` (`registry.go:25,39,53,62,73,91,205,339`).
- `NewKeyedSet[K](client, namespace, keyFn)` — `Add`/`Remove`/`Members`.

`Exists` and `RemoveExisting` are what make plan Task 5's idempotency (emit only
on the absent→present / present→absent transition) a two-line property rather
than a race.

Tests use `miniredis` exactly as
`services/atlas-summons/.../summon/registry_test.go:18-32` does.

### 3.7 Channel registration sites

Three hand-maintained lists in `services/atlas-channel/atlas.com/channel/main.go`:
- writer-name list — `:648-653` (summon writers)
- `handlerMap[...]` — `:856` (`summonsb.SummonMoveHandle`)
- consumer init/handlers — `:224` and `:459`

### 3.8 The in-field REST convention

Design §5.5 sketched `GET /dragons?filter[worldId]=…`. The established project
shape is a path-scoped resource:
`worlds/{w}/channels/{c}/maps/{m}/instances/{i}/summons`
(`services/atlas-summons/.../world/resource.go:30`, consumed via
`services/atlas-channel/.../summon/requests.go:11`). **The plan follows the
established path shape**, not the filter-param sketch — same capability, matching
convention, and it drops straight into `requests.DrainProvider`. Both nginx
location blocks are needed (`deploy/shared/routes.conf:476,501`).

### 3.9 Template entry shapes

Handler: `{"opCode","validator","handler","fname","services":["channel"]}`
(+ `"options"` for move handlers). Writer:
`{"opCode","writer","fname","services":["channel"]}`.

The `SummonMoveHandle` entry in `template_gms_83_1.json` (opCode `0xAF`) carries
a 23-element `options.types` array — that exact array is what
`DragonMoveHandle` must carry in the same template
(`tools/template-movement-types-guard.sh` requires byte-identity within a
template). `MOVE_HANDLERS` in that guard is at `tools/template-movement-types-guard.sh:37-43`
and must gain `"DragonMoveHandle"`.

---

## 4. Decisions carried from design.md (binding)

- **D-1** `atlas-dragons` is a real service; the registry is the authority for
  "which dragons are in this field".
- **D-2** `REMOVE_DRAGON` is implemented, verified, and sent — even though the
  client has no handler arm for it (design §2.4). It is not the mechanism that
  removes the dragon; leaving the field is.
- **D-3** v83 is routed and verified with no behaviour.

Two PRD statements the design corrected, which the plan follows:
- PRD FR-2.3 said "in-memory registry mirroring atlas-summons" — atlas-summons is
  Redis-backed. The plan is Redis-backed.
- PRD FR-5.3 mandated `MajorAtLeast` gating — nothing diverges across the six
  columns (design §2.6), so no gate is written. If per-cell verification surfaces
  a divergence, the gate goes in then, using `MajorAtLeast`.

---

## 5. Two wire traps that will misalign the packet if forgotten

1. **`SPAWN_DRAGON` coordinates are `int32`, not `int16`.** Every other entity
   in the protocol uses 2-byte coords. The model, the stored value, the event
   body, and the codec all carry `int32` end to end so the narrow type never
   enters the pipeline.
2. **`SPAWN_DRAGON` has a discarded 2-byte field** between `stance` and `jobId`
   (`CDragon::OnCreated` reads it and never assigns the result). Omitting it
   makes the client read `jobId` from the wrong offset.

Both are design §2.2 findings; both are covered by byte fixtures in plan Task 1.

---

## 6. Verification gates (from CLAUDE.md + design §12)

Run from the worktree root:

```
go test -race ./...          # in libs/atlas-packet, services/atlas-dragons, services/atlas-channel
go vet ./...                 # same three
go build ./...               # same three
docker buildx bake atlas-dragons
tools/service-registration-guard.sh
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check
```

Plus `packet-audit matrix --check` with 24 cells promoted and no previously-`✅`
cell regressed, and `superpowers:requesting-code-review` before the PR.

Known gotchas:
- `tools/lint.sh --check` false-fails without nvm on PATH, and contends on a
  golangci-lint lock across worktrees.
- The GHCR package for a brand-new service is created **private**; CI goes green
  while the pod sits in `ImagePullBackOff`. Flip it public after the first
  publish (`docs/adding-a-new-service.md` §6b).
- Updating a seed template does not update a provisioned tenant. Live behavioural
  verification requires reconciling the tenant socket config first, or the
  feature silently no-ops with a clean server log.
