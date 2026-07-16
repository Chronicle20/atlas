# Task 131 — Random Reward Items: Implementation Context

Companion to `plan.md`. Captures the key files, decisions, and dependencies an
engineer needs before executing the plan. Read this first, then `design.md` for
the evidence behind each decision.

## What we are building

End-to-end "use a reward box" flow. A player double-clicks a Consume item that
carries a `reward` node; the client sends `LOTTERY_ITEM_USE_REQUEST`; the server
rolls one prob-weighted reward, grants it (with expiration for period items),
consumes exactly one box atomically, and plays the effect / world-announce the
reward entry defines. Inventory-full preserves the box.

## Scope decision made during planning (READ THIS)

**v92 is DROPPED from this task** (implemented versions: **v83, v84, v87, v95**;
**jms** was added during the post-merge scope expansion — see design §2.6, its
opcode `0x06B` IDA-verified and handler registered in `template_jms_185_1.json`).

Reason: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
is a skeleton template — it has **0 `operations` blocks**, no `CharacterEffect`
writer, no chat `WorldMessage` writer, no `CharacterStatusMessage` writer, and no
`CharacterItemUseHandle`-family handler. The design (§2.2/§5.8) assumed v92 could
host the feature, but the presentation packets (effect, world-message,
inventory-full status) have no writer or operations table to resolve against, and
v92 has **no IDB** (project memory) so the missing opcodes/modes cannot be
verified — populating them would mean inventing values, which the project rules
forbid. This is the same situation that excluded jms (§2.6).

The user was asked to confirm this scope reduction but was away; the recommended
option (drop v92, like jms) was taken. **If the owner wants v92, it is a separate,
larger task** (build out the v92 template's operations tables + writers once a v92
IDB exists). Revisit before PR if needed. jms was already out of scope (§2.6).

## Design premise corrected during planning

- Design §2.2 claims the `LOTTERY_USE` operations entry is present in "all seed
  templates". Verified true for **v83/v84/v87 (=14), v95 (=16), jms (=15)** but
  **false for v92 (absent; v92 has no operations tables at all)** and gms_12
  (absent, out of scope). The v83 (14) / v95 (16) values match the IDA switch
  labels (design §2.2). No template edits to `LOTTERY_USE` are needed for the
  in-scope versions.

## The CREATE_ASSET contract gap (central to the grant path)

atlas-consumables does **not** currently have a `CREATE_ASSET` command or a
`CREATED`/`CREATION_FAILED` compartment status contract — its
`kafka/message/compartment/kafka.go` only knows `RESERVED` /
`RESERVATION_CANCELLED`. The authoritative contract lives in atlas-inventory
(`services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go`)
and is mirrored by atlas-saga-orchestrator. The plan (Task 4) adds the mirror to
atlas-consumables, field-for-field with atlas-inventory:

- `CommandCreateAsset = "CREATE_ASSET"`, `CreateAssetCommandBody{TemplateId,
  Quantity, Expiration time.Time, OwnerId, Flag, Rechargeable, UseAverageStats}`
- `StatusEventTypeCreated = "CREATED"`, `StatusEventTypeCreationFailed =
  "CREATION_FAILED"`, error code `CreateAssetInventoryFull =
  "CREATE_ASSET_INVENTORY_FULL"` (+ `CreateAssetTemplateNotFound`,
  `CreateAssetUnknownError`).
- **The `CREATED`/`CREATION_FAILED` events carry `transactionId` at the STATUS-EVENT
  TOP LEVEL, not in the body** (`CreatedStatusEventBody{type,capacity}`,
  `CreationFailedStatusEventBody{errorCode,message}`). The consumables
  `StatusEvent[E]` struct currently lacks a top-level `TransactionId` field — Task 4
  adds it (additive; RESERVED correlation still reads `e.Body.TransactionId`, so
  nothing breaks). The creation once-validator correlates on `e.TransactionId`.

## Atomicity / correlation model (grant-before-commit)

Reuse ONE transactionId `T` for the whole flow (mirrors `ConsumeScroll`):

1. `RequestItemReward` registers a RESERVED once-handler on `T` and reserves the
   box (USE compartment). Validation (non-empty rewards, Σprob>0) happens BEFORE
   reserving; a failed pre-check emits ERROR (unstick) with nothing to cancel.
2. On `RESERVED`, `ConsumeReward` rolls, then emits `CREATE_ASSET` with `T` and
   registers a creation once-handler on `T`.
3. On `CREATED` → `ConsumeItem(box, USE, T, slot)` + presentation events. On
   `CREATION_FAILED` → `CancelItemReservation(box, USE, T, slot)` + ERROR
   (INVENTORY_FULL if code is `CREATE_ASSET_INVENTORY_FULL`). Box preserved either
   way. The residual "CREATED then commit-emit lost" window is the same exposure
   `ConsumeScroll` already accepts (design §4.1).

The creation once-handler is typed to a **combined** body
`CreateResultEventBody{Type byte; Capacity uint32; ErrorCode string; Message string}`
so a single once-handler deserializes both CREATED and CREATION_FAILED (each
populates its subset) and branches on `e.Type`. Registering a second handler
inside the running RESERVED handler is fine — `consumer.GetManager()` is global,
and once-handlers are one-shot, fired at distinct lifecycle points.

## Key reference files (source of truth for patterns)

| Concern | File |
|---|---|
| Serverbound codec pattern | `libs/atlas-packet/inventory/serverbound/item_use.go` (+ `_test.go`) |
| Packet test helpers (`test.Variants/CreateContext/RoundTrip`) | `libs/atlas-packet/test/context.go`, `roundtrip.go` |
| Lottery-effect codecs (already exist, unwired) | `libs/atlas-packet/character/clientbound/effect.go:617,662` |
| Lottery-effect body factories | `libs/atlas-packet/character/effect_body.go:233-243` (`WithResolvedCode("operations","LOTTERY_USE")`) |
| Inventory-full status message | `libs/atlas-packet/character/clientbound/status_message.go:49`; body `character/status_message_body.go:21` |
| World-message blue text codec | `libs/atlas-packet/chat/clientbound/world_message.go:122`; channel writer `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go:72` |
| Consumables consume arms + `ConsumeScroll` reserve/commit precedent | `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` |
| Consumables compartment producer/processor | `services/atlas-consumables/atlas.com/consumables/compartment/{processor,producer}.go` |
| Consumables once-validator | `services/atlas-consumables/atlas.com/consumables/kafka/once/compartment/once.go` |
| Consumables command message contract | `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go` |
| Consumables command consumer (dispatch) | `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go` |
| Consumables event providers | `services/atlas-consumables/atlas.com/consumables/consumable/producer.go` |
| Consumables data-client pattern (to mirror for item-strings) | `services/atlas-consumables/atlas.com/consumables/data/consumable/{requests,processor,rest}.go` |
| Weighted pick with crypto/rand | `services/atlas-gachapons/atlas.com/gachapons/reward/processor.go:121` (`selectTier`) |
| CREATE_ASSET contract (authoritative) | `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go:24,100,177-260` |
| CREATE_ASSET producer to mirror | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/compartment/{processor,producer}.go:56-66,15-33` |
| Channel serverbound handler pattern | `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go` |
| Channel consumable command processor/producer | `services/atlas-channel/atlas.com/channel/consumable/{processor,producer}.go` |
| Channel consumable event consumer (arms + unstick) | `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go` |
| Channel foreign-effect / map fan-out | `services/atlas-channel/atlas.com/channel/socket/handler/effects.go`; `kafka/consumer/monsterbook/consumer.go:44-53` |
| Channel channel-wide broadcast (world message) | `services/atlas-channel/atlas.com/channel/kafka/consumer/gachapon/consumer.go` (`AllInChannelProvider`) |
| Channel handler/validator registration | `services/atlas-channel/atlas.com/channel/main.go:879` (handlerMap), `:904-909` (validators) |
| Item-strings REST endpoint (name lookup) | `services/atlas-data/atlas.com/data/item/string_resource.go:27` → `GET /data/item-strings/{itemId}` `{name}` |
| Seed templates (add handler entry) | `services/atlas-configurations/seed-data/templates/template_gms_{83,84,87,95}_1.json` `socket.handlers` |

## Verified constants / values

- Serverbound opcodes (registry + IDA): v83 `0x070` (IDA fn 0xa1249f), v84 `0x070`
  (lineage), v87 `0x073` (registry/CSV), v95 `0x07C` (IDA fn 0x9d6c50). jms `0x06B`
  and v92 `0x07B` are out of scope.
- Serverbound body: `slot int16, itemId int32` — **no** leading updateTime (IDA
  §2.1). Invariant across versions.
- `period` unit = **MINUTES** (design §2.3): `expiration = now + period*minute` when
  `period > 0`; `period <= 0` (default −1) = no expiration (zero `time.Time`).
- Roll = clean single weighted pick, crypto/rand (design §2.4); 56 v83 boxes are
  pure weights, no "chance of nothing" tables.
- `LOTTERY_USE` operations value: v83/v84/v87 = 14, v95 = 16 (present in templates).

## Changed modules (verification targets)

`go test -race ./...`, `go vet ./...`, `go build ./...` clean, plus
`docker buildx bake` for each, plus `tools/redis-key-guard.sh`:

- `libs/atlas-packet` (codec) → rebuilds every service that imports it; bake the
  touched services below at minimum.
- `services/atlas-data/atlas.com/data` → `docker buildx bake atlas-data`
- `services/atlas-consumables/atlas.com/consumables` → `docker buildx bake atlas-consumables`
- `services/atlas-channel/atlas.com/channel` → `docker buildx bake atlas-channel`
- `services/atlas-configurations` (templates ship in its image) → `docker buildx bake atlas-configurations`

## Rollout (not code — documented + performed if canonical baseline in play)

atlas-data consumables are stored JSON documents; existing tenants lack the new
`Effect`/`worldMsg`/`period` reward fields until re-ingestion. Absent fields
degrade gracefully (`""`/`-1`). See Task 15 and design §5.7. Live tenants also need
the new serverbound handler entry PATCHed into their config + atlas-channel
restart (seed templates apply only at tenant creation; project memory:
new-opcodes-not-in-live-tenant-config).
