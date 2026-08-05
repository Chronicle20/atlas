# Instance Route Effect Lifecycle — Implementation Context

Task: task-193 · Branch/worktree: `task-193-instance-route-effect-lifecycle`
PRD: [`prd.md`](prd.md) · Design: [`design.md`](design.md) · Plan: [`plan.md`](plan.md)

Everything below was read from this worktree at planning time. Line numbers are
from that state — re-check before relying on one.

---

## 1. The problem in one paragraph

Two subsystems each own half of one trip. `atlas-npc-conversations` and
`atlas-portal-actions` apply and cancel the dragon morph (item `2210016`) from
hand-written seed scripts; `atlas-transports` owns the instance lifecycle and
knows nothing about the effect. The portal cancels the morph only on one
specific exit portal, while `atlas-transports` has five terminal paths and none
of them cancel anything. Item `2210016` has `info/noCancelMouse = 1` and
`spec/time = 1800000` (30 min), so a player who idles out the 900 s travel
timer is delivered to the Temple of Time stuck as a Mini Draco for half an
hour. Separately, both flight transit maps declare `timeLimit = 900` and
`forcedReturn = 240000110` — the client's intent is that running out of flight
time sends you back to Leafre, not that it delivers you for free.

---

## 2. Key files

### `atlas-transports` — `services/atlas-transports/atlas.com/transports/`

| File | Why it matters |
|---|---|
| `instance/model.go:14` | `RouteModel` — immutable, private fields + getters. `TransitMapIds()` is the defensive-copy precedent for `EffectItemIds()`. |
| `instance/builder.go` | `RouteBuilder`. `Build()` holds all validation; note it stores `transitMapIds` as-is (copy-on-read is what enforces immutability). |
| `instance/model_json.go:12` | **The dangerous layer.** `routeModelJSON` + `MarshalJSON`/`UnmarshalJSON`. |
| `instance/route_registry.go:16` | `RouteRegistry` wraps `atlas.TenantRegistry[uuid.UUID, RouteModel]` — Redis-backed, so every processor route read goes through the JSON codec above. |
| `instance/processor.go` | The whole lifecycle. `StartTransport:102`, `HandleMapEnter:151`, `HandleLogout:247`, `TickArrival:323`, `TickStuckTimeout:367`, `GracefulShutdown:399`. `character2EnvCommandTopic` is defined at the bottom, `:434`. |
| `instance/producer.go` | Every Kafka provider for this package. New consumable providers belong here. |
| `instance/config/rest.go:19,50` | `InstanceRouteRestModel` + `ExtractRouteFor` — the atlas-tenants REST → domain hop. `resolveRouteId` derives the tenant-scoped uuid. |
| `instance/rest.go:41` | `TransformRoute` — debug REST. Already omits `transitMessage`, so it is not load-bearing; it is the only way an operator can see whether a live tenant re-seeded. |
| `kafka/message/instance_transport/kafka.go:35-37` | Cancel reasons live here. |
| `kafka/message/` | `channel`, `character`, `configuration`, `instance_transport`, `map`, `transport` — one local contract package per topic this service talks. The new `consumable` package follows the same shape. |
| `kafka/message/message.go` | `message.Buffer` — `map[string][]kafka.Message`, `Put` appends, `GetAll` returns a copy, `Emit` ranges the map (**randomised order across topics**; order within a topic is preserved). |
| `docs/domain.md:66`, `docs/kafka.md:147` | `RouteModel` field table; `## Topics Produced`. |

### `atlas-tenants` — `services/atlas-tenants/atlas.com/tenants/`

| File | Why it matters |
|---|---|
| `configuration/rest.go:379` | `InstanceRouteRestModel` — plain `uint32` for every id, unlike atlas-transports. |
| `configuration/rest.go:416` | `TransformInstanceRoute` — untyped JSONB → REST. Every attribute is projected explicitly; unknown ones vanish. The `transitMapIds` block at `:430-438` is the `[]interface{}` + `float64` pattern to copy. |
| `configuration/rest.go:477` | `ExtractInstanceRoute` — REST → JSONB. **Not in design.md's table.** Used by the POST/PATCH handlers at `configuration/resource.go:509,558`. |
| `configuration/seed/subdomain.go` | `Decode`/`Build` store the catalog file's `data` object **verbatim** into JSONB, so seeded attributes bypass `ExtractInstanceRoute` entirely. That is why Task 8's seed change works without Task 3 — but a later operator PATCH would still erase the fields without it. |
| `configuration/seed/groups.go:35,47,92-103` | The `instance-routes` seed group, its `POST …/seed` route, and the `AfterSeed` emit (with the created==0 && deleted>0 guard). |

### Consumer side (read-only for this task)

| File | Fact established |
|---|---|
| `atlas-consumables/kafka/message/consumable/kafka.go:27-36` | The canonical `Command[E]` envelope: `transactionId, worldId, channelId, mapId, instance, characterId, type, body`. |
| `atlas-consumables/kafka/consumer/consumable/consumer.go` (`handleApplyConsumableEffect`) | APPLY builds `channel.NewModel(worldId, channelId)` — the envelope's `mapId`/`instance` are ignored entirely. |
| same file (`handleCancelConsumableEffect`) | CANCEL builds `field.NewBuilder(worldId, channelId, mapId).SetInstance(instance).Build()` — but that field reaches `buff.Cancel`, and atlas-buffs' `Cancel` reads only `worldId`. So the map component is inert. |
| `atlas-saga-orchestrator/kafka/consumer/consumable/consumer.go:45-48` | `EFFECT_APPLIED` with `TransactionId == uuid.Nil` is explicitly skipped as a "non-saga effect application". This is why atlas-transports sends `uuid.Nil`. |
| `atlas-saga-orchestrator/kafka/message/consumable/kafka.go` | Precedent for a per-service local copy of the contract. Note it declares a *narrower* envelope (no `mapId`/`instance`) — the plan chooses the full-width version instead, per design D4. |

---

## 3. Decisions already settled (do not relitigate)

| # | Decision | Where argued |
|---|---|---|
| D1 | Two private helpers on `ProcessorImpl`, not inlined at six sites, not a sub-package | design §D1 |
| D2 | `effectItemIds` / `forcedReturnMapId`; `[]item.Id` + `_map.Id` per DOM-21 | design §D2 |
| D3 | Forced return emits `CANCELLED` with a new `TIMEOUT` reason, **not** `COMPLETED` — PRD FR-2.3 is overridden | design §D3 |
| D4 | Local `kafka/message/consumable` package mirroring atlas-consumables field-for-field | design §D4 |
| D5 | `MapId`/`Instance` left zero on both commands | design §D5 |
| D6 | Intra-buffer ordering kept as convention; **no test may assert cross-topic ordering** | design §D6 |
| D7 | Zero item id in a non-empty list is an error; zero forced-return map is "not set" | design §D7 |
| D8 | `HandleMapEnter`/`HandleLogout` log the cancelled-event put error and fall through to release | design §D8 |

Two additions the plan makes on top of the design, both justified in plan.md's
Self-Review: `ExtractInstanceRoute` as a fifth projection layer, and extracting
`forceCancelInstance` / `completeInstance` so the tick bodies are testable
without a clock injection this codebase does not have.

---

## 4. Verified facts worth not re-deriving

- **The event topic has one consumer, and it ignores both terminal event types.**
  `EVENT_TOPIC_INSTANCE_TRANSPORT` is consumed only by
  `atlas-channel/kafka/consumer/instance_transport/consumer.go`, which registers
  a handler for `TRANSIT_ENTERED` only. Nothing anywhere consumes `COMPLETED` or
  `CANCELLED`. D3's choice is behaviourally inert today and risk-free.
- **Ordering cannot regress.** Both consumable commands are keyed by
  `producer.CreateKey(int(characterId))` → same partition; `atlas-consumables`
  consumes serially (`libs/atlas-kafka/consumer/manager.go` takes the serial path
  when `maxInFlight <= 1`, the default is 1, and no service calls
  `SetMaxInFlight`). APPLY cannot overtake a later CANCEL for one character.
- **Seed uniformity is real.** All five relevant seed files are byte-identical
  across `gms/{12,48,61,72,79,83,84,87,92,95}_1` and `jms/185_1` — re-verified by
  md5 at planning time, one unique hash per file, no missing files. LF endings,
  no CRLF anywhere under `deploy/seed/gms` or `deploy/seed/jms`.
- **44 files carry an operation to remove** (4 × 11). `portal-dracoout.json` is
  present in all eleven and carries none — it is the leak that closes implicitly.
- **The only other user of these operations** is
  `npc-conversations/npc/npc-1101001.json` (item `2022458`, a stationary NPC
  blessing). Out of scope per PRD §2, and not an instance of the same bug shape:
  it has no transport and no terminal path to leak from.
- **Deployment is expected to be a no-op.** `COMMAND_TOPIC_CONSUMABLE` is in
  `deploy/k8s/base/env-configmap.yaml` and both overlays, and
  `deploy/k8s/base/atlas-transports.yaml` mounts the shared configmap via
  `envFrom`. Plan Task 10 Step 6 re-verifies rather than assuming.
- **`go.mod` is expected unchanged** in both services — `item` and `character`
  are packages inside `libs/atlas-constants`, already a dependency. If that turns
  out false, `docker buildx bake` becomes mandatory for the affected service.

---

## 5. Test harness notes

- `services/atlas-transports/atlas.com/transports/instance` has **no
  `processor_test.go` today**; the plan creates it. The package's existing test
  files already declare these helpers — do **not** redeclare them:
  `setupRouteTestRegistry`, `setupInstanceTestRegistry`,
  `setupCharacterTestRegistry`, `newTestTenantContext`, `newTestRoute`.
- The miniredis harness pattern is `miniredis.RunT(t)` →
  `goredis.NewClient(&goredis.Options{Addr: mr.Addr()})` → `InitXxxRegistry(rc)`.
  All three registries must be initialised for processor tests.
- Tenant context: `tenant.Register(uuid.New(), "GMS", 83, 1)` then
  `tenant.WithContext(context.Background(), tm)` (see `route_registry_test.go`).
  Note `config/rest_test.go` uses `tenant.Create` instead — both exist; match the
  file you are editing.
- `ProcessorImpl` can be constructed literally as
  `&ProcessorImpl{l: l, ctx: ctx, t: tm, p: nil}` from an in-package test: every
  `Xxx(mb)` method only buffers, and `p.p` is used exclusively by the
  `XxxAndEmit` wrappers. No mocks, no Kafka.
- Assertions read `mb.GetAll()[topic]` and `json.Unmarshal` the `kafka.Message`
  `Value`. Route fixtures use `NewRouteBuilder` per CLAUDE.md §Test Helper
  Pattern — no `*_testhelpers.go` files.
- `atlas-tenants`' `configuration/rest_test.go` already has a
  `toFloat64Attributes` helper at the bottom that mirrors a JSON round trip. Read
  it before use; extend it for slices rather than writing a second helper.

---

## 6. Traps

1. **Missing `model_json.go` is silent.** A field added to the model, builder,
   and config extractor but not the Redis codec passes every other test and is
   zero at every processor call site. Task 1 Step 1 is that guard; do not skip it.
2. **`TransformInstanceRoute` is an explicit allow-list.** An attribute it does
   not name never leaves atlas-tenants — no error, no log.
3. **`ExtractInstanceRoute` is the same trap on the write path.** Seeding
   bypasses it, so a seed-only test would not catch the omission.
4. **Do not assert cross-topic ordering.** `message.Emit` ranges a Go map.
5. **`TickStuckTimeout` / `TickArrival` call `time.Now()` internally.** They
   cannot be driven from a test; the plan extracts their loop bodies instead.
6. **Removing an operation can leave an empty state.** `npc-2082003.json`'s
   `applyBuff` state exists only to hold the apply — collapse the state and
   repoint `askTransform`'s first choice, don't leave `"operations": []`.
7. **Live tenants keep the bug until re-seeded.** The rollout step is a real
   deliverable (plan Task 10 Step 9), not a footnote.

---

## 7. Verification commands (worktree root unless noted)

```bash
# per-module, from services/<svc>/atlas.com/<svc>/
go test -race ./... && go vet ./... && go build ./...

# repo-root guards
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check          # needs nvm 22 on PATH; tools/lint.sh (no flags) fixes in place

# seed sweep — must print the PASS line
grep -rl 'consumable_effect' deploy/seed/ | xargs -r grep -l '2210016' || echo "PASS: no 2210016 effect ops remain"

# bake gate — only if a go.mod changed
git diff --name-only main -- '*/go.mod' '*/go.sum'
```

Not applicable: `tools/template-opcode-order-guard.sh` (no socket-config
template), `tools/service-registration-guard.sh` (no registration list),
`tools/skill-job-id-guard.sh` (no job/skill id comparison). Confirm with
`git diff --name-only main` rather than assuming.
