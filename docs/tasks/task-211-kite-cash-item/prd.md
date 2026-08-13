# Kites (Cash Item Category 508) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-10

---

## 1. Overview

Cash item category **508** ("kite", internally a *message box* — the client class is
`CMessageBoxPool`, and the dialog field is `m_sHope`, i.e. a hope/graduation banner) lets a
player pin a short personal message to a map. The banner hangs at the spot where the player
used the item, is visible to everyone on the map including players who arrive later, and
disappears when the owner leaves.

Atlas already has the hardest half of this feature done and unused. All three clientbound
writers exist in `libs/atlas-packet/field/clientbound/` and are byte-verified against the
client:

| Op | Struct | Client fname |
|---|---|---|
| `SPAWN_KITE` | `FieldKiteSpawn` | `CMessageBoxPool::OnMessageBoxEnterField` |
| `REMOVE_KITE` | `FieldKiteDestroy` | `CMessageBoxPool::OnMessageBoxLeaveField` |
| `CANNOT_SPAWN_KITE` | `FieldKiteError` | `CMessageBoxPool::OnCreateFailed` |

What is missing is every path that would ever *reach* those writers. `GetCashSlotItemType`
maps classification 508 to `CashSlotItemType(18)`
(`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:809-811`)
but no handler arm consumes enum 18, so a player using a kite today falls through to the
terminal `l.Warnf` at `character_cash_item_use.go:611` and nothing happens. There is no
serverbound sub-body decoder, no owning service, no registry, no lifecycle, no map-entry
replay, and — critically — **`grep -i kite` across all eleven tenant socket-config templates
returns zero hits**, so even a correct emit would be silently dropped for lack of a writer
opcode binding.

A vestigial `kite.Model` exists at
`services/atlas-channel/atlas.com/channel/kite/model.go` with zero importers and a field set
(`x`, `y`, `ft`) that does not match the authoritative wire layout. It is dead scaffolding and
this task replaces it rather than building on it.

This PRD covers making kites work end to end: a new `atlas-kites` service that owns kite state,
a serverbound decode + handler arm in `atlas-channel`, map-entry replay, writer opcode
registration across all supported versions, and a correction to the `FieldKiteSpawn` struct.

## 2. Goals

Primary goals:

- A player who uses a category-508 cash item with a message sees a kite appear at their current
  position, and so does everyone else on the map.
- A player entering a map sees every kite already hanging there.
- A kite is removed for all viewers when its owner leaves the map or disconnects.
- Attempts that violate a placement rule produce `CANNOT_SPAWN_KITE` rather than silence.
- The three kite writers are bound to opcodes in every supported tenant template.
- `FieldKiteSpawn`'s sixth field is corrected from `kiteType` to `y` (see FR-2.1), with the
  coverage matrix re-verified for every affected cell.

Non-goals:

- Kite item acquisition (cash shop listing, pricing). The four item IDs already exist in WZ.
- Any UI beyond the client's built-in `CMessageBoxDlg` rendering.
- Persisting kites across a channel restart or migration. Kites are ephemeral and owner-bound;
  a lost owner session means a lost kite.
- Cross-channel or cross-instance kite visibility. A kite is scoped to a single `field.Model`.
- Profanity filtering or message moderation. Out of scope; noted as a risk in §9.
- Implementing the other unhandled cash slot types found adjacent in `GetCashSlotItemType`
  (510 song player, 520 currency sack, 525 wedding ticket). They remain unhandled.

## 3. User Stories

- As a player, I want to use a kite item so that a personal message hangs on the map where I am
  standing and other players can read it.
- As a player on a map, I want to see kites that were placed before I arrived so that I am not
  missing context everyone else has.
- As a player, I want my kite to come down when I leave so that I am not littering a map I am no
  longer in.
- As a player, I want a clear failure response when I cannot place a kite here so that I do not
  believe the item was consumed for nothing.
- As a player, I want my kite item back in my inventory after use so that I can place it again
  elsewhere (per §4 FR-4.1 — kites are not consumed).
- As a server operator, I want per-map and per-character kite limits so that a map cannot be
  papered over with banners.

## 4. Functional Requirements

### FR-1 — Serverbound decode

**FR-1.1** Add `libs/atlas-packet/cash/serverbound/item_use_kite.go` defining the sub-body
decoded after the common `ItemUse` prefix (`updateTime`, `source`, `itemId` —
`libs/atlas-packet/cash/serverbound/item_use.go:37-41`). The sub-body is the kite message: a
single length-prefixed ASCII string.

**FR-1.2** The struct follows the immutable-model convention (private fields, getters,
`NewX(...)` constructor) matching the sibling `item_use_*.go` files in that package. It must
provide both `Decode` and `Encode` so a byte fixture can round-trip it.

**FR-1.3** The decoder must be verified against the client's send path for the category-508 use
case in the GMS v95.1 IDB before the struct is considered final. If the client sends additional
trailing fields beyond the message, the struct must carry them. **This verification is a
required gate, not an assumption** — the field list above is derived from the clientbound
counterpart and the `CMessageBoxDlg` member set (`m_nItemID`, `m_sHope`, `m_sCharacterName`),
not yet from the serverbound encoder.

### FR-2 — Clientbound struct correction

**FR-2.1** `FieldKiteSpawn`'s sixth field is currently named `kiteType int16`
(`libs/atlas-packet/field/clientbound/kite_spawn.go:16-23`). It is **`y`**, a coordinate — not a
type discriminator. Evidence from `CMessageBoxPool::OnMessageBoxEnterField` (GMS v95.1 IDB
`0x6369c0`), decode order:

```
Decode4  -> id            (MESSAGEBOX +16)
Decode4  -> templateId    (MESSAGEBOX +56)
DecodeStr-> message       (MESSAGEBOX +8)
DecodeStr-> name          (MESSAGEBOX +12)
Decode2  -> x             (MESSAGEBOX +28)
Decode2  -> y             (MESSAGEBOX +32)
```

followed immediately by

```
*(v8 + 36) = *(v8 + 28) - 3;      // renderX = x - 3
*(v8 + 40) = *(v8 + 32) - 100;    // renderY = y - 100
IWzVector2D::RelMove(vec, *(v8+36), *(v8+40), ...)
```

Both int16s are fed as the X and Y arguments of a single `RelMove` call; the `-3` / `-100` are
sprite-anchor offsets, not semantic flags. The appearance is chosen by `templateId`, which is
the sole argument to `CItemInfo::GetItemProp` further down the same function. There is no
kite-type field on the wire.

**FR-2.2** Rename the field and the `NewKiteSpawn` parameter accordingly, and update the
stale comment `"nType/y (spawn y or kite type, +32)"` in
`docs/packets/audits/gms_v83/FieldKiteSpawn.json` and every sibling per-version audit JSON.

**FR-2.3** The rename is semantic only — **the encoded bytes must not change on any version**.
Existing fixtures (`kite_spawn_test.go`, `kite_v48_test.go`) must continue to pass unmodified
except for the identifier rename. Re-pin the evidence records under
`docs/packets/evidence/*/field.clientbound.FieldKiteSpawn.yaml` and regenerate the matrix so
every cell keeps its current status.

### FR-3 — `atlas-kites` service

A new service owning authoritative kite state, modelled on `atlas-chalkboards` (the closest
precedent: a player message pinned to a map that must be replayed to late joiners).
`atlas-chalkboards` — not `atlas-maps`/`mist` — is the template, because mists have no REST
resource and are therefore never replayed on map entry.

**FR-3.1** Domain model `kite.Model`, immutable, private fields + getters + Builder:

| Field | Type | Notes |
|---|---|---|
| `id` | `uint32` | Wire id. Tenant-scoped, unique per tenant. |
| `f` | `field.Model` | World/channel/map/instance scope. |
| `characterId` | `uint32` | Owner. |
| `name` | `string` | Owner's character name, denormalized for the spawn packet. |
| `templateId` | `uint32` | The 508 item id used (`05080000`–`05080003`). |
| `message` | `string` | Player-supplied. |
| `x` | `int16` | Placement, from the owner's position at use time. |
| `y` | `int16` | Placement, from the owner's position at use time. |
| `createdAt` | `time.Time` | Observability. |

**FR-3.2** Registry: Redis-backed and tenant-scoped via
`atlas.NewTenantRegistry` from `libs/atlas-redis`, exactly as
`services/atlas-chalkboards/atlas.com/chalkboards/chalkboard/registry.go:13-24` does. Redis
(rather than an in-process singleton) is required because kite state must be visible to any
`atlas-kites` replica serving the map-entry REST read. Per project guardrails, all keyed Redis
access goes through `libs/atlas-redis` so `tools/redis-key-guard.sh` stays clean.

**FR-3.3** `id` allocation must be unique per tenant and stable for the lifetime of the kite,
since `REMOVE_KITE` addresses the kite by `id` alone. Use a Redis `INCR`-backed counter through
`libs/atlas-redis` rather than a process-local counter, which would collide across replicas.

**FR-3.4** Processor with the project's standard split — pure `Method(mb)` variants composing
into a `message.Buffer`, plus `MethodAndEmit()` wrappers:
- `Create(body) (Model, error)` — validates placement rules (FR-5), registers, emits
  `KITE_CREATED`. On emit failure the registry insert must be rolled back, mirroring
  `services/atlas-maps/atlas.com/maps/mist/processor.go:94-106`.
- `Destroy(id, reason) (Model, error)` — removes from the registry, emits `KITE_DESTROYED`
  carrying the reason.
- `DestroyForCharacter(characterId, reason)` — bulk removal used by the owner-departure paths.
- `InMapModelProvider(f)` / `GetInMap(f)` — backing the REST read.

**FR-3.5** Validation failures in `Create` must not emit `KITE_CREATED`. They emit a
`KITE_CREATION_FAILED` status event (see §5) so `atlas-channel` can render
`CANNOT_SPAWN_KITE` to the requesting character only.

### FR-4 — Item handling

**FR-4.1** The kite item is **not consumed**. No `saga.DestroyAsset` step, no inventory
mutation. Placement is gated by the per-character limit (FR-5.2) instead of by item
consumption, so the flow is a direct Kafka command rather than a saga.

**FR-4.2** Ownership is still verified before the command is sent, using the existing
`cashItemInSlotFunc` check (`character_cash_item_use.go:654-661`), consistent with every other
arm in that handler.

### FR-5 — Placement rules

**FR-5.1 — Per-map cap.** At most `N` kites may exist in one `field.Model` at a time. `N` is
tenant-configurable (see FR-8.1) and defaults to **10**. A request exceeding it fails with
reason `MAP_FULL`.

**FR-5.2 — One per character.** A character may own at most one kite at a time, across all
maps. A request from a character who already owns a kite fails with reason
`ALREADY_PLACED`. (Alternative considered and rejected: silently replacing the existing kite —
rejected because it makes the outcome ambiguous when placement then fails a different rule.)

**FR-5.3 — Map eligibility.** A map may forbid kites. The gate is evaluated in `atlas-kites`
against the map's `fieldLimit` and `town` flags, read from `atlas-data` — the same shape as
`services/atlas-channel/atlas.com/channel/skill/handler/mysticdoor/mysticdoor.go:105-112`.
Failure reason `MAP_FORBIDDEN`.

> **Unresolved — see §9 Q1.** No `FieldLimit` enum exists in the GMS v95.1 IDB local types, and
> no client-side field-limit gate on kite placement was found. `libs/atlas-constants/map/field_limit.go`
> defines six bits, none of which is a message-box bit. The specific bit value must therefore be
> established during design; it must **not** be guessed. If no bit can be evidenced, FR-5.3
> falls back to a tenant-configured map allowlist/denylist (FR-8.1) and the `fieldLimit` check
> is dropped.

**FR-5.4** Rules are evaluated in `atlas-kites`, not `atlas-channel`, so the authoritative
registry is the thing enforcing its own invariants and two concurrent requests cannot both pass
a channel-side check.

### FR-6 — Lifecycle / teardown

**FR-6.1** A kite is destroyed when its owner leaves the map it is on. `atlas-kites` consumes
the existing character map-change / status events and calls `DestroyForCharacter` with reason
`OWNER_LEFT`.

**FR-6.2** A kite is destroyed when its owner logs out or its session dies, reason
`OWNER_LOGGED_OUT`.

**FR-6.3** There is no TTL and no tick task. WZ data for `Cash/0508.img.xml` carries no `time`,
`expire`, or duration property — the four items expose only `icon`, `iconRaw`, `slotMax=100`,
`cash=1`, and `iconReward` — so lifetime is owner-bound, not data-driven or clock-driven.

**FR-6.4** `KiteDestroyAnimationType` has two values (`kite_destroy.go:15-20`). The mapping from
destroy reason to animation type must be established from the client during design and stated
explicitly; until then no reason may be assigned an animation value by guess. Both known reasons
(`OWNER_LEFT`, `OWNER_LOGGED_OUT`) are the same class of event, so a single animation type for
both is the expected outcome.

### FR-7 — `atlas-channel` integration

**FR-7.1** Add the `CashSlotItemType(18)` arm to `CharacterCashItemUseHandleFunc`
(`character_cash_item_use.go`). It decodes the FR-1 sub-body, resolves the character's current
position, and issues the create command. Per the answered scope question, **the placement
coordinates come from the character's current server-side position, not from the packet.**

**FR-7.2** A `kite` client package in `atlas-channel` mirroring
`services/atlas-channel/atlas.com/channel/chalkboard/` — `processor.go` (REST drain +
`ForEachInMap` + `AttemptUse`), `producer.go` (command emit keyed on `characterId` for
per-character ordering), `rest.go`.

**FR-7.3** Delete `services/atlas-channel/atlas.com/channel/kite/model.go` and the
"Model-only domain" entry at `services/atlas-channel/docs/domain.md:793-803`, replacing both
with the FR-7.2 package. The existing model is unused and its field set is wrong.

**FR-7.4** Kafka consumers for `KITE_CREATED` → `session.Announce(...)(KiteSpawnWriter)` to the
map, `KITE_DESTROYED` → `KiteDestroyWriter` to the map, `KITE_CREATION_FAILED` →
`KiteErrorWriter` to the requesting character only.

**FR-7.5** Map-entry replay: add a kite pass to the map-enter block in
`services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`, alongside the
existing NPC (`:224`), monster (`:230`), summon (`:236`), drop (`:242`), reactor (`:248`), door
(`:254`), and chalkboard (`:265`) passes.

**FR-7.6** Register the three writers in `produceWriters()` — they are already listed at
`services/atlas-channel/atlas.com/channel/main.go:724-726`, so no change is expected here;
verify only.

### FR-8 — Configuration

**FR-8.1** Kite settings are tenant-configurable through the atlas-tenants configuration system
(`GET /tenants/{id}/configurations/{resource}`), following
`docs/` guidance for that system. Settings: per-map cap (FR-5.1, default 10) and the map
allow/deny policy for FR-5.3.

**FR-8.2** Per DOM-25, no client wire value may be hard-coded. Opcodes come from tenant config
via `opcodes.BuildWriterProducer` (`libs/atlas-opcodes/producer.go:19`).

### FR-9 — Template wiring

**FR-9.1** Bind `SpawnKite`, `DestroyKite`, and `SpawnKiteError` writer opcodes in **all eleven**
seed templates under `services/atlas-configurations/seed-data/templates/` — gms 12/48/61/72/79/83/84/87/92/95
and jms 185 — using the opcodes already recorded in the coverage matrix
(`docs/packets/audits/STATUS.md:330,332,336`):

| Version | CANNOT_SPAWN_KITE | SPAWN_KITE | REMOVE_KITE |
|---|---|---|---|
| gms v48 | 0x0C5 | 0x0C6 | 0x0C7 |
| gms v61 | 0x0CF | 0x0D0 | 0x0D1 |
| gms v72 | 0x0F0 | 0x0F1 | 0x0F2 |
| gms v79 | 0x0F8 | 0x0F9 | 0x0FA |
| gms v83 | 0x10E | 0x10F | 0x110 |
| gms v84 | 0x10E | 0x10F | 0x117 |
| gms v87 | 0x11F | 0x120 | 0x121 |
| gms v92 | 0x13D | 0x13E | 0x13F |
| gms v95 | 0x145 | 0x146 | 0x147 |
| jms 185 | 0x123 | 0x124 | 0x125 |

**FR-9.2** gms v12 is not in the matrix rows above. Establish whether v12 has the message-box
family at all; if it does not, it is `n-a` and gets no binding, and the `n-a` consistency gate
must agree.

**FR-9.3** Writers must be inserted at their sorted `opCode` position, and each needs an
`fname`, per `docs/packets/TEMPLATE_CONVENTIONS.md`. `tools/template-opcode-order-guard.sh` and
`tools/template-duplicate-binding-guard.sh` must both pass.

**FR-9.4** Bind the serverbound decode path only if FR-1.3 shows category 508 uses a distinct
opcode. It is expected to reuse the existing `CharacterCashItemUseHandle` binding
(`template_gms_83_1.json:660-662`), in which case no new handler entry is needed.

## 5. API Surface

### REST — `atlas-kites` (JSON:API, api2go)

Resource type `kites`. `GetName()` returns `kites`.

**`GET /kites`** — filtered by field, for map-entry replay.

Query parameters: `worldId`, `channelId`, `mapId`, `instanceId` — parsed exactly as
`services/atlas-chalkboards/atlas.com/chalkboards/chalkboard/resource.go:59-63` does.
Paginated, matching the `chalkboards_in_map` precedent.

```json
{
  "data": [
    {
      "type": "kites",
      "id": "1",
      "attributes": {
        "characterId": 12345,
        "name": "Player",
        "templateId": 5080000,
        "message": "congrats!",
        "x": 320,
        "y": -140,
        "worldId": 0,
        "channelId": 1,
        "mapId": 104040000,
        "instanceId": "00000000-0000-0000-0000-000000000000",
        "createdAt": "2026-08-10T00:00:00Z"
      }
    }
  ]
}
```

**`GET /kites/{id}`** — single kite. `404` if unknown to this tenant.

Errors follow the JSON:API error envelope. All requests are tenant-scoped via
`tenant.MustFromContext(ctx)`; a kite belonging to another tenant is indistinguishable from a
missing one.

Note: `GET` with no character filter is not exposed — kites are only ever read by field.

### Kafka

**Command topic** `COMMAND_TOPIC_KITE` — produced by `atlas-channel`, consumed by `atlas-kites`.
Keyed on `characterId` for per-character ordering (matching the chalkboard producer).

| Command | Body |
|---|---|
| `CREATE` | `worldId`, `channelId`, `mapId`, `instanceId`, `characterId`, `templateId`, `message`, `x`, `y` |
| `DESTROY` | `worldId`, `channelId`, `mapId`, `instanceId`, `kiteId` |

**Status topic** `EVENT_TOPIC_KITE_STATUS` — produced by `atlas-kites`, consumed by
`atlas-channel`. Keyed on `mapId` for per-map ordering (matching the mist producer,
`services/atlas-channel/atlas.com/channel/mist/producer.go:21-27`).

| Event | Body |
|---|---|
| `CREATED` | full kite projection (all `kite.Model` fields) |
| `DESTROYED` | `kiteId`, field scope, `reason` (`OWNER_LEFT` \| `OWNER_LOGGED_OUT`) |
| `CREATION_FAILED` | `characterId`, field scope, `reason` (`MAP_FULL` \| `ALREADY_PLACED` \| `MAP_FORBIDDEN`) |

`CREATION_FAILED` must carry `characterId` because `CANNOT_SPAWN_KITE` is a targeted response,
not a map broadcast. Note that `FieldKiteError` has an **empty body**
(`kite_error.go:15-19`) — the reason is server-side only, for logs; the client shows a generic
failure. Reasons are still modelled so the failure is diagnosable.

**Consumed by `atlas-kites`** — the existing character status topic, for FR-6.1/FR-6.2
(map change, logout). No new topic.

Per the project's Kafka conventions: `message.Buffer` batching, atomic `message.Emit(p)`,
curried `InitConsumers(l)(cmf)(groupId)`, and `producer.Provider = func(token)`. Topic env vars
must be suffixed per the new-service checklist so they do not fall back to an unsuffixed topic.

## 6. Data Model

No relational schema. Kite state is **Redis only**, tenant-scoped, via
`atlas.NewTenantRegistry[uint32, Model]` from `libs/atlas-redis` with key namespace `kite`.
There is no Postgres table and therefore no migration.

Justification: kites are ephemeral and owner-bound (FR-6). Nothing survives a logout, so nothing
needs durable storage. This matches `atlas-chalkboards`, which is likewise Redis-only.

Secondary indices needed to serve the access patterns without a full scan:

| Index | Purpose | Requirement |
|---|---|---|
| by `field` (world/channel/map/instance) | map-entry replay (FR-7.5), per-map cap (FR-5.1) | must not require scanning all kites for a tenant |
| by `characterId` | one-per-character (FR-5.2), owner teardown (FR-6.1/2) | must resolve without a scan |

An `id` allocation counter per tenant (FR-3.3), also in Redis.

Tenant scoping is structural: every key is namespaced by tenant, so cross-tenant reads are not
expressible rather than merely filtered.

## 7. Service Impact

**`atlas-kites` (new).** Full new service. Must be registered in every hand-maintained list
enumerated by `docs/adding-a-new-service.md` — `.github/config/services.json`, `docker-bake.hcl`,
`go.work`, the k8s base, **both** kustomize overlays, databases, and ingress — and
`tools/service-registration-guard.sh` must pass. Watch the documented silent-failure traps: an
unpinned `:latest` image, `behavior: replace` dropping configmap keys, and unsuffixed Kafka topic
fallback. Note it needs Redis but **no** database, so the database registration step may differ
from the doc's default path; confirm during design.

**`atlas-channel`.** New `kite/` client package (FR-7.2) replacing the dead model (FR-7.3); the
enum-18 arm in `character_cash_item_use.go` (FR-7.1); three Kafka consumers (FR-7.4); the
map-entry replay pass (FR-7.5).

**`libs/atlas-packet`.** New serverbound `item_use_kite.go` (FR-1); the `kiteType`→`y` rename in
`kite_spawn.go` (FR-2).

**`libs/atlas-constants`.** Possibly a new `FieldLimit` bit for FR-5.3 — **only** if §9 Q1
resolves with evidence. Per the project rule, check the existing package index before adding
anything new.

**`atlas-configurations`.** Writer opcode bindings across all eleven templates (FR-9).

**`docs/packets`.** Audit JSON comment fixes, evidence re-pin, and matrix regeneration for
`FieldKiteSpawn` (FR-2.2, FR-2.3).

**`atlas-data`.** Read-only consumer for map `fieldLimit`/`town` (FR-5.3). No change expected.

## 8. Non-Functional Requirements

**Multi-tenancy.** Every path is tenant-scoped via `tenant.MustFromContext(ctx)`. Redis keys are
tenant-namespaced. Opcodes and limits are resolved from tenant configuration, never hard-coded
(DOM-25).

**Performance.** Map-entry replay adds one REST call to an already multi-call map-enter path; it
must be a single indexed lookup by field, not a scan. Expected cardinality is bounded by FR-5.1
(default 10 per map), so the replay payload is small.

**Concurrency.** FR-5.1 and FR-5.2 are invariants of the registry and must hold under concurrent
requests. Two simultaneous placements by the same character must not both succeed. Note
`ForEachInMap` in `atlas-channel` is parallel — any shared state in the replay renderer is a
known hazard on this codebase and must be avoided.

**Observability.** Log kite create/destroy with tenant, character, field, and reason. Creation
failures log the specific `CREATION_FAILED` reason, since the client sees only a generic error.

**Security / validation.** Message length must be bounded server-side before it reaches the wire
encoder — the client field is a length-prefixed string and an unbounded message is both a packet
size risk and a griefing vector. The bound must be established from the client's own input
limit during design. Ownership is verified before the command is issued (FR-4.2), and the owner
name on the spawn packet is taken from server-side character state, never from the client.

**Goroutines.** Any concurrency must go through `routine.Go` per `tools/goroutine-guard.sh`.

**Verification gates.** Per `CLAUDE.md`: `go test -race ./...`, `go vet ./...`, `go build ./...`
in every changed module; `docker buildx bake atlas-kites` plus a bake for every service whose
`go.mod` was touched; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/service-registration-guard.sh`, `tools/lint.sh --check`,
`tools/template-opcode-order-guard.sh`, and `tools/template-duplicate-binding-guard.sh` all
clean.

## 9. Open Questions

**Q1 — Which `fieldLimit` bit forbids kites? (blocks FR-5.3)** No `FieldLimit` enum exists in the
GMS v95.1 IDB local types (`type_query` for `*FIELDLIMIT*` / `*FieldLimit*` returned zero
results), and no client-side gate on kite placement was found.
`libs/atlas-constants/map/field_limit.go` defines six bits — teleport `0x01`, mystic door `0x02`,
summoning bag `0x04`, migrate `0x08`, portal scroll `0x10`, teleport item `0x40`, exp loss
`0x80000` — none of which is a message-box bit. Resolution path: locate the server-side or
client-side check by xref during design. **If no bit can be evidenced, do not invent one** —
fall back to the tenant-configured allowlist/denylist (FR-8.1).

**Q2 — Destroy animation type mapping (blocks FR-6.4).** `KiteDestroyAnimationType1 = 0` /
`Type2 = 1`. Which reason maps to which? Resolve from
`CMessageBoxPool::OnMessageBoxLeaveField` (GMS v95.1 IDB `0x635d60`) during design.

**Q3 — Serverbound sub-body shape (blocks FR-1.1).** Message-only is the expectation, derived
from the clientbound counterpart and `CMessageBoxDlg`'s members; it is **not yet confirmed from
the client's encoder**. FR-1.3 makes this a gate.

**Q4 — Message length bound.** Needs the client's own input limit (see §8 Security).

**Q5 — gms v12 applicability (FR-9.2).** Does v12 have the message-box family? If not, `n-a`.

**Q6 — Does the per-map cap apply per-instance?** FR-5.1 says "one `field.Model`", which includes
`instanceId`, so instanced copies of a map each get their own cap. Confirm this is intended.

## 10. Acceptance Criteria

- [ ] A character using a category-508 cash item with a message sees the kite appear at their
      current position; other characters on the map see it too.
- [ ] A character entering a map with existing kites receives a `SPAWN_KITE` for each.
- [ ] A kite is removed for all viewers when its owner changes map (`OWNER_LEFT`).
- [ ] A kite is removed for all viewers when its owner logs out (`OWNER_LOGGED_OUT`).
- [ ] The kite item remains in the owner's cash inventory after use (FR-4.1).
- [ ] A second placement attempt by a character who already owns a kite yields
      `CANNOT_SPAWN_KITE` and no new kite (FR-5.2).
- [ ] Exceeding the per-map cap yields `CANNOT_SPAWN_KITE` and no new kite (FR-5.1).
- [ ] `CANNOT_SPAWN_KITE` reaches only the requesting character, not the map.
- [ ] `FieldKiteSpawn`'s sixth field is named `y`; the stale `"nType/y"` comment is gone from
      every per-version audit JSON; **no encoded bytes changed on any version**; every
      `FieldKiteSpawn` matrix cell retains its prior status after evidence re-pin and matrix
      regeneration.
- [ ] `grep -i kite` over `services/atlas-configurations/seed-data/templates/` returns bindings
      in every applicable template (all eleven, or ten plus a justified v12 `n-a`).
- [ ] `services/atlas-channel/atlas.com/channel/kite/model.go` is deleted and the
      `docs/domain.md` "Model-only domain" entry is replaced.
- [ ] `atlas-kites` appears in `.github/config/services.json`, `docker-bake.hcl`, `go.work`, the
      k8s base, both kustomize overlays, and ingress.
- [ ] `go test -race ./...`, `go vet ./...`, and `go build ./...` are clean in every changed
      module.
- [ ] `docker buildx bake atlas-kites` succeeds from the worktree root, as does a bake for every
      other service whose `go.mod` was touched.
- [ ] `tools/service-registration-guard.sh`, `tools/redis-key-guard.sh`,
      `tools/goroutine-guard.sh`, `tools/lint.sh --check`,
      `tools/template-opcode-order-guard.sh`, and `tools/template-duplicate-binding-guard.sh`
      all exit 0.
- [ ] Every open question in §9 is resolved with cited evidence, or explicitly closed with a
      stated fallback. None is resolved by guessing a value.
