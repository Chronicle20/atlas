# Kites (Cash Item Category 508) — Implementation Context

Task: task-211-kite-cash-item
Companion to: [`prd.md`](prd.md), [`design.md`](design.md), [`plan.md`](plan.md)
Created: 2026-08-10

This document is the orientation packet for whoever executes `plan.md`. It
records the file inventory, the reference implementations to copy, the decisions
already made (and where they were made), and the things that will silently break
if ignored.

---

## 1. One-paragraph summary

Cash item category 508 ("kite" / message box) pins a short player message to a
map. All three **clientbound** codecs already exist in `libs/atlas-packet` and
are byte-verified. Nothing reaches them: no serverbound sub-body decoder, no
owning service, no handler arm, no map-entry replay, and **zero writer bindings
in any tenant socket template**. This task builds the missing 90% — a new
`atlas-kites` service, an `atlas-channel` client package + handler arm, a new
serverbound codec, a `kite-configs` tenant configuration resource, and the
template wiring — plus two byte-neutral packet renames the design proved from
the IDB.

---

## 2. Reference implementations (copy these, don't invent)

| Need | Copy from | Notes |
|---|---|---|
| Whole-service shape (Redis-only, no DB) | `services/atlas-chalkboards/atlas.com/chalkboards/` (~15 files) | Closest domain sibling: player message pinned to a map, owner-bound, replayed on map entry. |
| Registry (tenant-scoped Redis) | `services/atlas-chalkboards/.../chalkboard/registry.go:13-24` | `atlas.NewTenantRegistry[K,V]` from `libs/atlas-redis`. |
| Character-in-field index | `services/atlas-chalkboards/.../character/registry.go:14-26` | `atlas.NewTenantKeyedSet[field.Model]`. **Its field key is instance-blind — do not copy that bug** (design Q6). |
| Buffered emit + rollback on emit failure | `services/atlas-maps/atlas.com/maps/mist/processor.go:62-123` | `message.Emit(p)(func(buf *message.Buffer) error {...})`, `p.r.Remove(...)` on failure. |
| Paginated in-map REST list | `services/atlas-chalkboards/.../chalkboard/resource.go:58-108` | `paginate.ParseParams` / `paginate.Slice` / `MarshalPaginatedResponse`. |
| Channel-side client package | `services/atlas-channel/atlas.com/channel/chalkboard/` (5 files) | `requests.DrainProvider`, `ForEachInMap`, producer keyed on `characterId`. |
| Channel-side status consumer | `services/atlas-channel/atlas.com/channel/kafka/consumer/chalkboard/consumer.go` | `SetStartOffset(kafka.LastOffset)`, `sc.Is(tenant, worldId, channelId)` guard. |
| Tenant-config consumer (4 files) | `services/atlas-channel/atlas.com/channel/mts/configuration/` | `requests.go` + `rest.go` (`Extract` folds zeros to defaults) + `registry.go` (per-tenant cache) + `model.go`. |
| Tenant-config **producer** (singleton resource) | `services/atlas-tenants/.../configuration/` `*Rankings*` symbols | Smallest precedent: one config per tenant, GET/POST/PATCH/DELETE on a single path, **no** `/seed` and **no** `/{id}` routes. |
| Serverbound cash sub-body | `libs/atlas-packet/cash/serverbound/item_use_chalkboard.go` | Identical shape incl. the trailing-`updateTime` branch. |

---

## 3. File inventory

### Created

```
libs/atlas-packet/cash/serverbound/item_use_kite.go
libs/atlas-packet/cash/serverbound/item_use_kite_test.go

services/atlas-kites/atlas.com/kites/
  go.mod  main.go
  kite/          model.go builder.go registry.go processor.go producer.go
                 resource.go rest.go
  character/     model.go registry.go processor.go
  configuration/ model.go rest.go requests.go registry.go
  kafka/consumer/consumer.go
  kafka/consumer/kite/consumer.go
  kafka/consumer/character/consumer.go
  kafka/message/message.go
  kafka/message/kite/kafka.go
  kafka/message/character/kafka.go
  rest/handler.go
deploy/k8s/base/atlas-kites.yaml

services/atlas-channel/atlas.com/channel/kite/
  builder.go processor.go producer.go requests.go rest.go   (model.go rewritten)
services/atlas-channel/atlas.com/channel/kafka/message/kite/kafka.go
services/atlas-channel/atlas.com/channel/kafka/consumer/kite/consumer.go
```

### Modified

| File | Change |
|---|---|
| `libs/atlas-packet/field/clientbound/kite_spawn.go` | `kiteType` → `y` (byte-neutral) |
| `libs/atlas-packet/field/clientbound/kite_destroy.go` | `KiteDestroyAnimationType1/2` → `KiteDestroyAnimated/Silent` |
| `libs/atlas-packet/field/clientbound/kite_destroy_test.go`, `kite_v48_test.go` | identifier/comment follow-through |
| `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/FieldKiteSpawn.json` | row-5 `IDAComment` |
| `services/atlas-configurations/seed-data/templates/template_*.json` (10 of 11) | 3 writer bindings each |
| `services/atlas-channel/.../socket/handler/character_cash_item_use.go` | type-18 arm + `CashSlotItemTypeKite` const |
| `services/atlas-channel/.../kafka/consumer/map/consumer.go` | replay pass + `spawnKitesForSession` |
| `services/atlas-channel/atlas.com/channel/main.go` | consumer + handler registration |
| `services/atlas-channel/atlas.com/channel/go.mod` | (only if deps shift) |
| `services/atlas-channel/docs/domain.md:793-803` | rewrite the Kite section |
| `services/atlas-tenants/.../configuration/{rest,processor,provider,resource}.go` | `kite-configs` resource |
| `.github/config/services.json`, `docker-bake.hcl`, `go.work` | register `atlas-kites` |
| `deploy/k8s/base/{kustomization,env-configmap}.yaml` | resource + 2 topic keys |
| `deploy/k8s/overlays/main/kustomization.yaml`, `patches/atlas-env-env.yaml` | image pin, topic literals, `ATLAS_ENV` |
| `deploy/k8s/overlays/pr/kustomization.yaml` | image pin + regenerated topic literals |
| `deploy/shared/routes.conf`, `deploy/k8s/ingress.yaml` | 2 nginx locations (regenerated) |

### Deleted

Nothing. `services/atlas-channel/atlas.com/channel/kite/model.go` is **rewritten**, not
deleted — the package name stays, the field set changes (drops `ft`/`Type()`,
adds `characterId`/`createdAt`). It currently has zero importers
(`grep -rn "channel/kite"` → no hits), so the rewrite is free.

---

## 4. Decisions already made — do not re-litigate

Every one of these is settled in `design.md` §2/§3 with an IDB address or a repo
`file:line`. Re-deriving them wastes a cycle; contradicting them is a plan
violation.

| Question | Answer | Where |
|---|---|---|
| Which `fieldLimit` bit forbids kites? | **None exists.** The client's rule is `GetCurFieldID()/10000000 == 91` (Free Market) → refuse. Implemented as a tenant-configured prefix denylist, default `[91]`. No new `FieldLimit` bit. | design Q1 (`0x9ed017`–`0x9ed034`) |
| `FieldKiteSpawn` 6th field | It is **`y`**, not a kite type. Both int16s feed one `RelMove`; appearance comes from `templateId`. Confirmed on v95 (`0x6369c0`) *and* v83 (`0x65acdf`). | design Q/ADR-7b |
| Destroy animation byte | **Suppress-animation flag**, not a selector. `0` plays the despawn animation; non-zero removes silently. Both reasons use `0`. | design Q2 (`0x635d60`) |
| Serverbound sub-body | **One length-prefixed ASCII string** (`message`), plus the trailing `updateTime` on GMS ≤ 84. | design Q3 (`0x9ed271`) |
| Message length bound | **182 bytes** = 3 × 60-byte edit controls + 2 `'\n'` joiners. Enforced in `atlas-kites`, not the socket decoder. | design Q4 (`0x7824f0`, `0x4e2a10`) |
| gms v12 | **Out of scope, and not an `n-a` claim** — v12 has no matrix column, no audit dir, no IDB, and its template has no `CharacterCashItemUseHandle`. Scope is **ten** templates, not eleven (amends PRD FR-9.1). | design Q5 |
| Per-map cap scope | Per `field.Model`, i.e. **per instance**. | design Q6 |
| Item consumption | **Not consumed** (PRD FR-4.1). Direct Kafka command, no saga, no `DestroyAsset`. Deliberate deviation from retail. | PRD FR-4.1, design §9 |
| New handler template binding | **None.** All ten templates already bind `CharacterCashItemUseHandle` exactly once; type 18 rides it. Only 3 **writers** are added per template. | design ADR-8 |
| `produceWriters()` in channel main.go | **Already correct** at `main.go:724-726`. Verify only. | design ADR-8 |

---

## 5. Scope decision made during planning (flagged for the reviewer)

PRD FR-8.1 requires kite settings to be tenant-configurable through the
atlas-tenants configuration system. Design ADR-6 specifies the **consumer** side
(a four-file `configuration/` package in `atlas-kites`, mts-configs-shaped) but
does not enumerate the **producer** side, and PRD §7 Service Impact does not list
`atlas-tenants` at all.

Verified during planning: `atlas-tenants` has **no generic
`/configurations/{resource}` route**. Every resource (`routes`, `vessels`,
`instance-routes`, `rps-rewards`, `mts-configs`, `rankings`) is hand-registered
at `configuration/resource.go:1205-1252`. Without a `kite-configs` resource
there, the consumer's fetch always misses and the knobs are permanently pinned
to their compiled defaults — i.e. FR-8.1 would be unmet.

**Assumption taken:** plan Task 5 adds `kite-configs` to `atlas-tenants` on the
**`rankings` pattern** — a singleton per-tenant config with GET/POST/PATCH/DELETE
on `/tenants/{tenantId}/configurations/kite-configs`, **no `/seed` endpoint and
no `/{id}` sub-routes**. No `atlas-ui` admin page is built (no FR requires one,
and the resource is drivable over the API). If the reviewer wants the UI page or
a seed endpoint, that is an additive follow-up that does not invalidate anything
in this plan.

---

## 6. Traps that fail silently

1. **Templates.** `grep -i kite` over
   `services/atlas-configurations/seed-data/templates/` currently returns **zero
   hits**. Without a writer binding the emit is dropped with no error. Entries
   must go at their **sorted `opCode` position** (not next to a semantically
   related writer) and each needs an `fname` — `tools/template-opcode-order-guard.sh`
   and `tools/template-duplicate-binding-guard.sh` both gate this.
2. **Kafka topic env vars.** `libs/atlas-kafka/topic/provider.go` falls back to
   the bare token with only a warn log. A topic key present in
   `deploy/k8s/base/env-configmap.yaml` but missing from an overlay's
   `configMapGenerator` literals is **absent** in that env (the generator is
   `behavior: replace`), and the two sides then talk on different topics.
   `service-registration-guard.sh` checks key *parity*, not that the right *new*
   keys exist — verify by hand.
3. **`images:` pin.** The bump workflow only rewrites entries already present. A
   missing `images:` entry means `:latest` forever, silently.
4. **`ForEachInMap` is parallel** (`model.ParallelExecute()`). The replay
   operator must close over nothing mutable — construct a fresh `KiteSpawn` per
   model. This has bitten this codebase before.
5. **Instance threading.** The chalkboards character consumer builds its field key
   with `field.NewBuilder(w, c, m).Build()` (instance left at `uuid.Nil`) while
   the resource looks it up with `.SetInstance(instanceId)`. Instanced maps have
   therefore never replayed chalkboards. The status events *do* carry the
   instance. `atlas-kites` must thread it on every transition. **The chalkboards
   bug is out of scope** (different service, keeps the diff reviewable).
6. **Destroy ordering.** The character-index transition must capture the **old**
   field before the destroy emit, or `KITE_DESTROYED` fans out to the wrong map.
7. **No `EnableActions` on the kite arm.** The client dialog is modal
   (`CDialog::DoModal`) and unlocks itself; the sibling chalkboard *use* arm sends
   none either. Unlocking would widen the client's dup gate.
8. **No `packet-audit:verify` markers on the `ItemUseKite` fixtures.** There is no
   matrix row for a cash sub-body (its sibling `item_use_chalkboard_test.go` has
   none either). Adding markers for an op the registry does not know would break
   `packet-audit matrix --check`.

---

## 7. Key API surfaces (verified in-tree)

```go
// libs/atlas-redis
atlas.NewTenantRegistry[K comparable, V any](c *goredis.Client, ns string, keyFn func(K) string) *TenantRegistry[K, V]
  (r) Get/Put/Remove/Exists/GetAllValues/Update(ctx, t tenant.Model, ...)
atlas.NewTenantKeyedSet[K any](c *goredis.Client, ns string, keyFn func(K) string) *TenantKeyedSet[K]
  (s) Add/Remove/Members/IsMember/Clear(ctx, t tenant.Model, k K, members ...string)
atlas.NewIDGenerator(c *goredis.Client, ns string) *IDGenerator
  (g) NextID(ctx, t tenant.Model) (uint32, error)      // SETNX + INCR, starts at 1000000000
atlas.NewLock(c *goredis.Client, ns string) *Lock
  (l) Acquire(ctx, key string) (bool, error)           // SET NX EX 30s; key is NOT tenant-scoped — include the tenant in `key`
  (l) Release(ctx, key string) error
```

```go
// libs/atlas-constants
item.ClassificationMessageBanner = Classification(508)   // item/constants.go:78
field.NewBuilder(world.Id, channel.Id, _map.Id).SetInstance(uuid.UUID).Build()
```

```go
// atlas-channel
character.Model.Name() string   // character/model.go:104
character.Model.X() int16       // :244
character.Model.Y() int16       // :248
_map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(mapId, instance), func(session.Model) error) error
session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(session.Model) error) error
requests.DrainProvider[RestModel, Model](l, ctx)(url string, pageSize int, Extract, filters) model.Provider[[]Model]
```

---

## 8. Verification gates (all from the worktree root)

```bash
go test -race ./...            # in libs/atlas-packet, services/atlas-kites,
go vet ./...                   #    services/atlas-channel, services/atlas-tenants
go build ./...
docker buildx bake atlas-kites
docker buildx bake atlas-channel
docker buildx bake atlas-tenants
tools/service-registration-guard.sh
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check
git diff --exit-code docs/packets/audits/status.json docs/packets/audits/STATUS.md
```

`tools/lint.sh --check` needs nvm 22 on PATH or it false-fails on the atlas-ui
leg; run `tools/lint.sh` (no flags) first to fix formatting in place.

---

## 9. Current `FieldKiteSpawn` matrix row — must be unchanged at the end

`docs/packets/audits/STATUS.md:332`

| v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 | JMS185 |
|---|---|---|---|---|---|---|---|---|---|
| ✅ | 🟡ᶠ | 🟡ᶠ | 🟡ᶠ | ✅ | ✅ | ✅ | 🟡ᶠ | ✅ | ✅ |

The rename changes no bytes and no decompile hash, so the evidence records under
`docs/packets/evidence/*/field.clientbound.FieldKiteSpawn.yaml` (which record only
`function`/`address`/`decompile_sha256`) need **no** re-pin. Matrix regeneration
must be a literal no-op; a non-empty `git diff` on `status.json` means something
else changed and the task is not done.
