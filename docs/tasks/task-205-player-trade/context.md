# task-205 Player Trade — Implementation Context

Companion to [`plan.md`](plan.md). Everything here is a verified repository fact
with a `file:line` citation, gathered during the planning pass. Read this before
starting any task; it is the "what already exists" half of the plan.

Inputs: [`prd.md`](prd.md) (approved), [`design.md`](design.md) (approved).
Worktree: `.worktrees/task-205-player-trade`, branch `task-205-player-trade`.

---

## 1. What already exists (do not rebuild)

### 1.1 Serverbound codecs — complete and fixtured

All in `libs/atlas-packet/interaction/serverbound/`:

| File | Type | Fields (getters) |
|---|---|---|
| `operation_create.go` | `OperationCreate` | `RoomType() byte`, `Title() string`, `Private() bool`, `Password() string`, `NGameSpec() byte`, `Slot() int16`, `ItemId() uint32` |
| `operation_invite.go` | `OperationInvite` | `TargetCharacterId() uint32` |
| `operation_invite_decline.go` | `OperationInviteDecline` | `SerialNumber()`, `ErrorCode()` |
| `operation_trade_put_item.go` | `OperationTradePutItem` | `InventoryType() byte`, `Slot() int16`, `Quantity() uint16`, `TargetSlot() byte` |
| `operation_trade_add_meso.go` | `OperationTradeAddMeso` | `Amount() int32` (**signed**) |
| `operation_trade_confirm.go` | `OperationTradeConfirm` | `Entries() []TradeConfirmEntry` (`Data() uint32`, `Crc() uint32`) |
| `operation_transaction.go` | `OperationTransaction` | `Entries() []TransactionEntry` |
| `operation_cash_trade_open.go` | `OperationCashTradeOpen` | `NProc()`, `RoomType()`, `TargetCharacterId()`, `Spw()`, `DwSN()`, `ShopId()`, `Unk2()`, `Position()`, `SerialNumber()`, `Birthday()` |

`operation_create.go:37-60` branches on raw numeric room types; `roomType == 3`
(trade) writes only `private`; `roomType == 6` (cash trade) likewise.

### 1.2 Version gate — already carries the trade CRC boundary

`libs/atlas-packet/interaction/serverbound/version_gate.go` (23 lines, whole
file):

```go
func tradeCrcPresent(t tenant.Model) bool {
	return (t.Region() == "GMS" && t.MajorVersion() >= 83) || t.Region() != "GMS"
}
```

It uses the raw `MajorVersion() >= 83` form. The idiomatic helper is
`tenant.Model.MajorAtLeast(v uint16) bool` (`libs/atlas-tenant/tenant.go:92-95`);
idiomatic call site: `libs/atlas-packet/login/clientbound/server_list_entry.go:33`.
**New gates written for this task use `MajorAtLeast`; the existing
`tradeCrcPresent` is reused as-is and not rewritten** (rewriting it would touch
verified cells for no wire change).

### 1.3 `interaction.Room` already encodes the trade frame

`libs/atlas-packet/interaction/room.go:148-227`. `Encode` writes
`roomType, capacity, position, visitors…, 0xFF` for **every** room type, then
`switch rm.roomType` with arms only for `PersonalShopRoomType` and
`MerchantShopRoomType`. A `TradeRoomType`/`CashTradeRoomType` room therefore
already encodes exactly the base frame and nothing more — which design §1.3
confirms is correct (v83 `CTradingRoomDlg` vtable+72 is `nullsub_94`
@`0x48314D`).

`decodeVisitorForRoom` (`visitor.go:89-120`) already handles trade visitors in
its `default` arm. `interaction.NewBaseVisitor(slot byte, avatar model.Avatar,
name string) Visitor` (`visitor.go:43`) is the constructor to use.

Room type constants — `room.go:15-28`, values from
`libs/atlas-constants/miniroom/room_type.go:14-19`: `Trade = 3`, `CashTrade = 6`.

**Missing:** a `NewTradeRoom` constructor. That is Task 1.

### 1.4 Clientbound body idiom

`libs/atlas-packet/interaction/clientbound/interaction_body.go`. Bodies are free
functions returning `func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte`.
Canonical one-code form (`interaction_body.go:107-111`):

```go
func CharacterInteractionInviteBody(roomType byte, name string, dwSN uint32) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return atlas_packet.WithResolvedCode("operations", CharacterInteractionModeInvite, func(mode byte) packet.Encoder {
		return NewInteractionInvite(mode, roomType, name, dwSN)
	})
}
```

Two-code form, when a second value also needs table resolution
(`interaction_body.go:129-137` and `:186-193`) — used by
`CharacterInteractionEnterResultErrorBody` and
`CharacterInteractionLeaveReasonBody`.

`atlas_packet.WithResolvedCode` is `libs/atlas-packet/resolve.go:15-22`;
`ResolveCode` (`resolve.go:29`) returns **99** on lookup miss, documented as
"will likely cause a client crash" — so a missing template key is loud in
practice but not a compile error. This is why Task 23 (templates) is mandatory.

Existing mode keys — `interaction_body.go:33-51`. Existing `leaveReason` keys —
`interaction_body.go:167-179` (`SHOP_CLOSED` 3, `USER_BANNED` 5,
`OUT_OF_STOCK` 14, `MINIGAME_CLOSED` 3, `MINIGAME_LEFT` 4,
`MINIGAME_EXPELLED` 5). **No trade mode keys and no trade leaveReason keys
exist.**

Wire structs live in `clientbound/interaction.go`: `InteractionEnter`
(`:89-118`), `InteractionEnterResultSuccess` (`:120-149`), `InteractionLeave`
(`:223-256`, fields `mode, slot, status byte`).

### 1.5 Asset encoding

`model.Asset` from `github.com/Chronicle20/atlas/libs/atlas-packet/model`
(`libs/atlas-packet/model/asset.go`). Constructor
`NewAsset(zeroPosition bool, slot int16, templateId uint32, expiration time.Time) Asset`
(`asset.go:63`). **`Encode` and `Decode` are pointer-receiver**
(`asset.go:206`, `asset.go:510`) — the value must be addressable at the call
site. Decode requires a pre-seed:

```go
rm.items[i].Asset = model.NewAsset(true, 0, 0, time.Time{})
rm.items[i].Asset.Decode(l, ctx)(r, options)
```

(`room.go:254-255`; `zeroPosition = true` in room contexts.)

### 1.6 Test harness

`libs/atlas-packet/test`: `pt.Variants` (`test/context.go:18-46`) = GMS v28,
v83, v87, v95, JMS v185, GMS v84. `pt.CreateContext(region, major, minor)`.
`pt.RoundTrip(t, ctx, encode, decode, nil)` (`test/roundtrip.go:22-34`) asserts
`reader.Available() == 0` after decode.

Canonical fixture-test shape to copy verbatim:
`libs/atlas-packet/interaction/serverbound/operation_trade_put_item_test.go`
(51 lines) — a `pt.Variants` round-trip test carrying one
`// packet-audit:verify packet=… version=… ida=0x…` line per version, plus a
separate hex-pinning test for versions outside `pt.Variants` (v72 there). The
gated variant is `operation_trade_confirm_test.go:17-68`, which recomputes the
gate inline in the test.

### 1.7 atlas-channel trade arms — all stubs today

`services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go`
(676 lines). Mode constants at `:20-69` already name every trade mode.

| Arm | Lines | Current state |
|---|---|---|
| `CREATE` roomType 3 | `:87-90` | `l.Debugf` + `return` |
| `CREATE` roomType 6 | `:132-134` | `l.Debugf` only |
| `INVITE` | `:134-140` | `l.Debugf` + `return` |
| `INVITE_DECLINE` | `:141-146` | `l.Debugf` + `return` |
| `VISIT` | `:147-183` | fans out to minigame + merchant |
| `CHAT` | `:184-200` | fans out to minigame + merchant |
| `EXIT` | `:201-236` | fans out to minigame + merchant |
| `CASH_TRADE_OPEN` nProc 0/4 roomType 6 | `:255-262` | `l.Debugf` + `return` |
| `TRADE_PUT_ITEM` | `:292-297` | `l.Debugf` + `return` |
| `TRADE_ADD_MESO` | `:298-303` | `l.Debugf` + `return` |
| `TRADE_CONFIRM` | `:304-310` | `l.Debugf` + `return` |
| `TRANSACTION` | `:311-317` | `l.Debugf` + `return` |

Session write idiom (direct, from a handler holding `s session.Model`):

```go
_ = session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(body)(s)
```

From a Kafka handler (no session in hand) —
`services/atlas-channel/atlas.com/channel/kafka/consumer/minigame/consumer.go:115-119`:

```go
func announceTo(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, body packet.Encode) {
	if characterId == 0 {
		return
	}
	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(body))
}
```

Channel-side fire-and-forget processor precedent —
`services/atlas-channel/atlas.com/channel/minigame/processor.go:16-68`.

### 1.8 atlas-mini-games — the template service

`services/atlas-mini-games/atlas.com/mini-games/`. Layout to mirror:

```
main.go
game/{builder,model,processor,producer,registry,resource,rest}.go  + game/mock/processor.go
record/{administrator,builder,entity,model,processor,provider,resource,rest}.go
data/<x>/{processor,requests,rest}.go + data/<x>/mock/processor.go
kafka/consumer/consumer.go
kafka/consumer/<topic>/consumer.go
kafka/message/<topic>/kafka.go
rest/handler.go
```

- Registry: `registry.go:72-210` — `sync.Once` singleton, one `sync.RWMutex`
  guarding `rooms map[tenant.Model]map[K]Room` **and**
  `members map[tenant.Model]map[uint32]K`, index maintained only inside
  Create/Update/Remove under the write lock. `ErrOwnerHasRoom`, `ErrRoomNotFound`.
- Processor: `processor.go:216-278` — `Processor` interface + `ProcessorImpl` +
  `NewProcessor(l, ctx, db) Processor` + `var _ Processor = (*ProcessorImpl)(nil)`.
- Outbox emit wrapper: `processor.go:307-322` (`emit` + `withTx`) — one DB
  transaction, `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))`.
- Producer providers: `producer.go:60-101`, generic `statusEventProvider[E]`.
  **DOM-25 note (`producer.go:17-47`): internal byte enums are mapped to
  semantic KEY strings at the emission boundary**; the channel resolves them to
  per-version numbers.
- `main.go:1-114` — `service.Bootstrap`, `database.Connect(l,
  database.SetMigrations(record.Migration, outboxlib.Migration))`, outbox
  drainer under `routine.Go`, one `InitConsumers`/`InitHandlers` pair per
  consumer, `server.New(l)…AddRouteInitializer(...)…Run()`.
- Character-status teardown consumer:
  `kafka/consumer/character/consumer.go:1-94` — three handlers, LOGOUT /
  MAP_CHANGED / CHANNEL_CHANGED, each calling `TeardownCharacter(characterId)`.
  A fourth arm lives in `kafka/consumer/session/consumer.go:41-56`
  (`SESSION_DESTROYED`, guarded by `if e.CharacterId == 0 { return }`).
- REST: `game/resource.go:36-78` (`InitResource(si)(db)`,
  `rest.RegisterHandler(l)(si)`), pagination via
  `paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)`
  + `paginate.Slice` + `server.MarshalPaginatedResponse`.

Kafka consumer config helper — `kafka/consumer/consumer.go:12-25`:
`NewConfig(l)(name)(topicToken)(groupId) consumer.Config`.

### 1.9 Invite contract

`services/atlas-invites/atlas.com/invites/kafka/message/invite/kafka.go`:
`EnvCommandTopic = "COMMAND_TOPIC_INVITE"`,
`EnvEventStatusTopic = "EVENT_TOPIC_INVITE_STATUS"`.

```go
type Command[E any] struct {
	TransactionId uuid.UUID          `json:"transactionId"`
	WorldId       world.Id           `json:"worldId"`
	InviteType    invite.Type        `json:"inviteType"`
	Type          invite.CommandType `json:"type"`
	Body          E                  `json:"body"`
}
type CreateCommandBody struct {
	OriginatorId character.Id `json:"originatorId"`
	TargetId     character.Id `json:"targetId"`
	ReferenceId  invite.Id    `json:"referenceId"`
}
type StatusEvent[E any] struct {
	TransactionId uuid.UUID         `json:"transactionId"`
	WorldId       world.Id          `json:"worldId"`
	InviteType    invite.Type       `json:"inviteType"`
	ReferenceId   invite.Id         `json:"referenceId"`
	Type          invite.StatusType `json:"type"`
	Body          E                 `json:"body"`
}
type AcceptedEventBody struct{ OriginatorId, TargetId character.Id }
type RejectedEventBody struct{ OriginatorId, TargetId character.Id }
```

`invite.Id` is `uint32` (`libs/atlas-constants/invite/constants.go:3`);
`invite.TypeTrade = "TRADE"` (`:12`). **This is why the room needs a `uint32`
handle in addition to its `uuid` id** (design §2.3) — a UUID does not fit
`ReferenceId`.

### 1.10 Saga primitives

`libs/atlas-saga/model.go`:

- `TradeTransaction Type = "trade_transaction"` **already exists** at
  `model.go:16` — the saga *type* is reserved; there is no trade Action.
- Storage/transfer actions (`model.go:122-131`): `TransferToStorage`,
  `WithdrawFromStorage`, `AcceptToStorage`, `ReleaseFromCharacter`,
  `AcceptToCharacter`, `ReleaseFromStorage`.
- `Saga` / `Step[T]` / `Status` — `model.go:198-284`.
- `AwardMesosPayload` — `libs/atlas-saga/payloads.go:65-74` (`Amount int32`,
  may be negative; `ActorType string`; `ShowEffect bool`).
- `ReleaseFromCharacterPayload` — `payloads.go:571-578`.
- `AcceptToCharacterPayload` is **orchestrator-local** (it references
  `asset2.AssetData`) —
  `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go:914-922`.

Orchestrator expansion precedent — `saga/processor.go:1266-1335`
(`expandTransferToStorage`) and `:1337-1400` (`expandWithdrawFromStorage`, the
closer analogue: `ReleaseFromStorage` + `AcceptToCharacter`). Helper
`assetDataFromCompartmentAsset(foundAsset)` builds the snapshot.

**A new expandable action must be registered in every one of these** or it fails
at runtime:

| Registry | Location |
|---|---|
| `isExpandableAction` gate | `saga/processor.go:1167-1184` |
| `expandAndProcessStep` switch | `saga/processor.go:1186-1264` |
| Event-acceptance map | `saga/event_acceptance.go:170` |
| Error mapper | `saga/error_mapper.go:25` |
| Character extractor | `saga/character_extractor.go:65,73` |
| Orchestrator unmarshal | `saga/model.go:1267,1309` |
| Shared-lib unmarshal | `libs/atlas-saga/unmarshal.go` |

`TestIsExpandableActionCoversExpansionSwitch`
(`saga/mts_expansion_test.go:262-280`) pins the first two in sync — it will fail
if you add the switch case without the gate.

### 1.11 atlas-inventory reservations

`services/atlas-inventory/atlas.com/inventory/compartment/processor.go:752-795`.
TTL is hard-coded at `:782`:

```go
_, err = GetReservationRegistry().AddReservation(p.t, transactionId, characterId, inventoryType, request.Slot, request.ItemId, uint32(request.Quantity), time.Second*time.Duration(30))
```

`ReservationRequest{Slot int16; ItemId uint32; Quantity int16}` —
`compartment/reservation_registry.go:17-21`.

Wire body — `kafka/message/compartment/kafka.go:20,72-81`:
`CommandRequestReserve = "REQUEST_RESERVE"`,
`RequestReserveCommandBody{TransactionId uuid.UUID; Items []ItemBody}`,
`ItemBody{Source int16; ItemId uint32; Quantity int16}`. Byte-identical copies
exist in atlas-channel, atlas-consumables and atlas-saga-orchestrator
(`kafka/message/compartment/kafka.go` in each).

Reservations already block the competing operations:
`GetReservedQuantity` is consulted by move, merge and drop at
`compartment/processor.go:507,589-590,698,856`.

> **Pre-existing bug, found during planning:** the
> `for _, request := range reservationRequests` loop at `:773-791`
> unconditionally `return`s the `mb.Put(...)` on its **first** iteration, so
> only the first reservation of a multi-item request is processed. Fixed in
> Task 7 (it is inside the function Task 7 already edits). atlas-trades issues
> one single-element reserve per staged item regardless, so trade does not
> depend on the fix.

### 1.12 atlas-tenants configuration system

All tenant configuration is one polymorphic JSONB table — `configuration/entity.go`:

```go
type Entity struct {
	gorm.Model
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey"`
	TenantId     uuid.UUID       `gorm:"type:uuid;not null"`
	ResourceName string          `gorm:"not null"`
	ResourceData json.RawMessage `gorm:"type:jsonb;not null"`
}
func (Entity) TableName() string { return "configurations" }
```

**Adding `trade-configs` requires no migration** — only the string
discriminator. The `mts-configs` precedent to mirror:

| Piece | Location |
|---|---|
| Six handlers (GET / GET-by-id / POST / PATCH / DELETE / seed) | `configuration/resource.go:817-1026` |
| Route registration | `configuration/resource.go:1192,1244-1249` |
| REST model + Transform + Extract | `configuration/rest.go:247-379` |
| Processor interface (12 methods) | `configuration/processor.go:112-155`, impl `:716-812` |
| Provider (`GetByTenantIdAndResourceNameProvider(tenantID, "mts-configs")(db)`) | `configuration/provider.go:157,191` |
| Kafka status events | `configuration/kafka.go:25-27,65-66` |
| Seed loader | `configuration/seed.go` (85 lines) |
| Path parser | `services/atlas-tenants/atlas.com/tenants/rest/handler.go:53-55` |

Consumer-side registry precedent (what atlas-trades mirrors):
`services/atlas-mts/atlas.com/mts/configuration/` — `model.go` (immutable Model
+ getters + `DefaultConfig()` at `:69-83`), `registry.go`, `requests.go`,
`rest.go`. `DefaultConfig()` is the fallback on any fetch miss or error: "the
service never hard-fails on a missing configuration resource."

> **Trap, verified:** `services/atlas-tenants/configurations/` contains **only**
> `rps-rewards/default.json`. There is no committed `configurations/mts-configs/`
> directory, so `SeedMtsConfigs` currently fails with "failed to read seed
> directory". `Dockerfile:138`/`:163` copy whatever exists under
> `configurations` → `/configurations`. Task 13 therefore ships
> `services/atlas-tenants/configurations/trade-configs/default.json` **and**
> tests the no-config fallback path, per design §8.

### 1.13 atlas-data `tradeBlock`

Present: `consumable/reader.go:49` (`m.TradeBlock = i.GetBool("tradeBlock", false)`,
field at `consumable/rest.go:46`) and `setup/reader.go:47` (field at
`setup/rest.go:13`).

Missing, must be added (Task 8):

| Domain | Struct | Reader | Style |
|---|---|---|---|
| equipment | `equipment/rest.go:17-49` | `equipment/reader.go:86-116` | single composite literal, `info.` receiver |
| etc | `etc/rest.go:7-15` | `etc/reader.go:40-53` | `m := RestModel{…}` then post-assignment |
| cash | `cash/rest.go:37-48` | `cash/reader.go:53+` | `i, err := cxml.ChildByName("info")` |

An unasserted fixture with `<int name="tradeBlock" value="1"/>` already exists at
`equipment/reader_test.go:40`.

### 1.14 Seed templates

`services/atlas-configurations/seed-data/templates/` — **eleven** files:
`template_gms_12_1.json` (24.6K, no interaction keys — out of scope, design
§11.4), then the ten in the PRD's version set: `gms_48`, `gms_61`, `gms_72`,
`gms_79`, `gms_83`, `gms_84`, `gms_87`, `gms_92`, `gms_95`, `jms_185`.

---

## 2. Decisions carried from design.md (do not re-litigate)

1. **Reserve at staging, move at settlement** (design §5.3, Option A). No escrow
   at put-item time. `release_from_character` is a soft-delete with no owner-side
   durable record, so escrow-at-staging is unrecoverable after a crash.
2. **The trade room reuses `interaction.Room`.** No third encoder, no enter-result
   tail (design §1.3 — v83 vtable+72 is `nullsub_94` @`0x48314D`).
3. **Trade completion is `LEAVE` (mode 10) + slot + status**, not a distinct mode
   (design §1.4). Statuses: 2 cancelled, 7 success, 8 failed, 9 cannot-carry,
   12 different-map, 13 CRC-failed. No new codec — only new `leaveReason` keys.
4. **Settlement refusal tears the room down** (design §6.1), contradicting the
   PRD's FR-4.9 "reverts nothing (the room is still live)". The client closes the
   dialog before showing the notice, so a surviving room is not a reachable state.
5. **Mode 17 is broadcast only after BOTH sides confirm** (design §6.2). Sending
   it on the first confirm makes the counterparty's client auto-reply `0x14`
   without its owner pressing Trade.
6. **`AWAITING_ATTESTATION` has a 5-second deadline**; on expiry the room settles
   anyway using the CRC lists from the two `TRADE_CONFIRM` payloads.
7. **Meso is logically reserved only** (design §5.4); no meso-reservation
   primitive is added.
8. **Cross-family room occupancy (PRD FR-1.2) is a best-effort check in
   atlas-channel**, not an enforced invariant (design §2.1). atlas-trades enforces
   its own single-room invariant authoritatively.
9. **atlas-trades runs `replicas: 1`, no HPA** (design §9) — the registry is
   process-local.
10. **Silent drop is correct for FR-4.1..4.4** (design §7): the reference client
    has no put-item-time error for "untradeable"; the empty slot is the feedback.
    Log server-side with item id and failing rule.
11. **gms_92's non-trade template gaps are recorded, not fixed** (design §10.4).
12. **`template_gms_12_1.json` is out of scope** (design §11.4).
13. **No ledger retention policy** (design §10.5).

---

## 3. Open items resolved during the per-version pass (Task 6)

These are derived per version by the procedure in design §10.1 — never assumed
from v83:

- Clientbound mode bytes for `TRADE_PUT_ITEM` / `TRADE_ADD_MESO` /
  `TRADE_CONFIRM`, and whether the mode-21 (`TRADE_MESO_LIMIT`) arm exists.
- `OnLeave` status-byte branches per version (drives the `leaveReason` table).
- Whether `vtable+72` is a nullsub (if not, `NewTradeRoom` needs a version gate).
- `TRANSACTION` presence below v83 — **hypothesis** (design §10.2): absent,
  because there is no CRC to attest. Confirmed or refuted per version.
  jms_v185 is the stated exception and must be checked directly.
- `CCashTradingRoomDlg` presence on gms_v48/61/72 and jms_v185 (design §10.3).

Absence is asserted **only** from a decompiled dispatcher lacking the case —
never from an unnamed symbol.

---

## 4. Findings to record (design §11)

| # | Finding | Handling |
|---|---|---|
| 11.1 | `operation_transaction.go` carries `// packet-audit:fname CCashTradingRoomDlg::Trade`, which on v83 sends `Encode1(0x11)` = `TRADE_CONFIRM`, not `TRANSACTION`. Only `0x14` sender is `CTradingRoomDlg::OnTrade` @`0x7c20bc`. | Fixed in Task 5, **together with matrix regeneration and re-pin** — an `fname` edit re-keys every evidence record referencing it. |
| 11.2 | Clientbound mode 21 (`TRADE_MESO_LIMIT`) was unmodelled. | Added in Task 4. |
| 11.3 | gms_92 template missing non-trade interaction keys. | Recorded only. |
| 11.4 | `template_gms_12_1.json` outside the version set. | Not touched. |
| 11.5 | `RequestReserve` TTL hard-coded 30 s. | Parameterised in Task 7. |
| new | `RequestReserve`'s loop returns on its first iteration (§1.11). | Fixed in Task 7. |
| new | `services/atlas-tenants/configurations/mts-configs/` does not exist, so `SeedMtsConfigs` fails at runtime (§1.12). | trade-configs ships its seed dir; mts is recorded only. |

---

## 5. Dependency order

```
Slice 1 (packet)         Tasks 1-6    ── independent of all service work
Slice 2 (prerequisites)  Tasks 7-8    ── independent; both block Slice 5
Slice 3 (skeleton)       Tasks 9-13   ── 9 blocks 10-13
Slice 4 (saga)           Tasks 14-15  ── 14 blocks 15
Slice 5 (behaviour)      Tasks 16-24  ── needs 1-15
```

Within Slice 5: 16 → 17 → 18 → 19 → 20 (atlas-trades), 21 → 22 (atlas-channel),
23 (templates), 24 (gates). 21/22/23 can run in parallel with 17-20 once 16 has
landed the shared contract.

---

## 6. Verification gates (project CLAUDE.md)

Every one of these must be clean before the PR (Task 24):

```
go test -race ./...            # every changed module
go vet ./...                   # every changed module
go build ./...                 # every changed service
docker buildx bake atlas-trades atlas-channel atlas-inventory \
                   atlas-data atlas-tenants atlas-saga-orchestrator
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/service-registration-guard.sh
tools/lint.sh --check
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
```

Note `tools/lint.sh --check` false-fails without nvm on PATH; run
`nvm use 22` first if the atlas-ui leg errors.

Changed Go modules: `libs/atlas-packet`, `libs/atlas-saga`,
`services/atlas-trades` (new), `services/atlas-channel`,
`services/atlas-inventory`, `services/atlas-data`, `services/atlas-tenants`,
`services/atlas-saga-orchestrator`.
