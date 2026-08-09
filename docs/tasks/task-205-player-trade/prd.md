# Player-to-Player Trade (1:1 Trade Window) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-09

---

## 1. Overview

Player-to-player trading is the single most-used economy feature in the game and
is entirely absent from Atlas. A character can right-click another character and
choose Trade; the target receives an invite; on accept, both sides open a
`CTradingRoomDlg` window where each may stage up to nine items and an amount of
meso. When both sides confirm, the staged contents swap atomically, minus a
meso tax deducted from each side's meso contribution.

Today the wire layer decodes every serverbound trade mode and then does nothing.
`services/atlas-channel/atlas.com/channel/socket/handler/character_interaction.go`
handles `CREATE` for `TradeMiniRoomType` (line 85-88), `INVITE` (135-140),
`INVITE_DECLINE` (141-146), `TRADE_PUT_ITEM` (276-281), `TRADE_ADD_MESO`
(282-287), `TRADE_CONFIRM` (288-295) and `TRANSACTION` (296-303) by emitting a
`l.Debugf(...)` and returning. No service owns trade state: a repo-wide search
for `TradeRegistry|trade_room|CompleteTrade|NewTradeMiniRoom` produces no
production call sites, and `invite.TypeTrade`
(`libs/atlas-constants/invite/constants.go:12`) is referenced only by
atlas-invites test fixtures — nothing ever produces a trade invite. The
serverbound codecs exist and are byte-fixtured
(`libs/atlas-packet/interaction/serverbound/operation_trade_*.go`, task-110), so
the missing work is server logic **plus the entire clientbound half**: there is
no `TRADE_PUT_ITEM` / `TRADE_ADD_MESO` / `TRADE_CONFIRM` / trade-complete body
in `libs/atlas-packet/interaction/clientbound/interaction_body.go`, no
`interaction.NewTradeRoom` constructor in `libs/atlas-packet/interaction/room.go`
(only `NewPersonalShopRoom` and `NewMerchantShopRoom` exist), and no clientbound
trade keys in any tenant writer table.

This task creates a new **atlas-trades** service that owns trade-room lifecycle
and settlement, implements the missing clientbound codecs across all ten client
versions, routes them in every seed template, and settles the swap through the
saga orchestrator so the exchange is atomic and compensatable. It also covers
the cash-item trade room (`CCashTradingRoomDlg`, mini-room type 6), the full
faithful set of trade restrictions, and a durable completed-trade ledger for GM
abuse investigation.

## 2. Goals

Primary goals:

- A character can invite another character in the same map to a 1:1 trade,
  the target can accept or decline, and both see a correctly populated
  `CTradingRoomDlg`.
- Each side can stage up to 9 items and an amount of meso, remove staged items
  before confirming, and see the counterparty's staged contents live.
- Both-side confirmation performs an atomic swap of items and meso, with a
  tenant-configurable meso tax deducted from each side's meso contribution.
- Either side can cancel (window close, `EXIT`, disconnect, map change,
  channel change) at any point before settlement, with everything returned to
  its owner and no partial state left behind.
- Trade is rejected faithfully — with the correct mini-room error code — when
  the item, the character, or the map disallows it.
- The cash-item trade room (mini-room type 6) works for cash-inventory items.
- Every completed trade is recorded in a queryable ledger (who gave what to
  whom, tax collected) for GM abuse investigation.
- All of the above works on all ten supported client versions: gms_v48,
  gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95 and
  jms_v185.

Non-goals:

- Cross-channel or cross-world trading. A trade is confined to two characters
  in the same map instance on the same channel.
- Trades with more than two participants.
- Any atlas-ui surface (admin or player-facing). The ledger is REST-only in
  this task.
- Changes to the hired-merchant / personal-shop room family (mini-room types 4
  and 5, owned by atlas-merchant) beyond what shared code requires.
- Changes to the mini-game room family (types 1 and 2, owned by
  atlas-mini-games).
- The Fredrick / item-retrieval NPC flow.
- Auto-trade, trade macros, or any trade-adjacent convenience feature not in
  the reference behavior.

## 3. User Stories

- As a player, I want to right-click another player and choose Trade so that I
  can exchange items without dropping them on the ground.
- As a player receiving a trade invite, I want to accept or decline it so that
  I am not forced into an unwanted window.
- As a player in a trade window, I want to stage items from any inventory
  compartment and remove them again before confirming, so that I can correct a
  mistake.
- As a player in a trade window, I want to stage meso and see the counterparty's
  meso so that I can price the exchange.
- As a player, I want the trade to be all-or-nothing so that I never lose items
  to a partial swap or a disconnect.
- As a player, I want to be told clearly and specifically why a trade was
  refused (item is untradeable, inventory is full, not enough meso, the other
  player is busy) so that I can fix it.
- As a player, I want to be prevented from trading an untradeable or
  quest-bound item so that I do not lose an item I cannot get back.
- As a GM, I want a queryable record of every completed trade so that I can
  investigate duplication and RMT abuse reports.
- As a tenant operator, I want to configure or disable the meso tax tiers so
  that I can tune my server's economy without a code change.

## 4. Functional Requirements

### FR-1 — Trade room lifecycle

- **FR-1.1** On serverbound `CREATE` with `roomType == TradeMiniRoomType` (3),
  atlas-channel calls atlas-trades to create a trade room owned by the
  requesting character. The room starts with exactly one occupant (the owner,
  position 0) and capacity 2.
- **FR-1.2** A character may be in at most one mini-room of any kind at a time.
  Creating or entering a trade room while the character already occupies a
  trade room, a mini-game room (atlas-mini-games) or a shop room
  (atlas-merchant) is rejected with `OTHER_REQUESTS` (mini-room enter error 3).
- **FR-1.3** Trade rooms are single-channel, single-map, and ephemeral. They are
  keyed by tenant and hold their state in an in-memory registry (the pattern
  established by `services/atlas-mini-games/atlas.com/mini-games/game/registry.go`
  — `sync.Once` singleton + `sync.RWMutex`, tenant-partitioned). Room state is
  NOT persisted; only the completed-trade ledger (FR-7) is durable.
- **FR-1.4** A trade room is destroyed when: both sides settle (FR-5), either
  side cancels (FR-6), either side disconnects, either side changes map, or
  either side changes channel.
- **FR-1.5** The room owner and the invited character occupy positions 0 and 1
  respectively. Position drives which side of the client dialog receives which
  update.

### FR-2 — Invite and accept

- **FR-2.1** On serverbound `INVITE`, atlas-trades issues an
  `invite.TypeTrade` invite through atlas-invites (`COMMAND_TOPIC_INVITE`,
  `CommandTypeCreate`, body `{originatorId, targetId, referenceId}`), where
  `referenceId` is the trade room's numeric handle. This is the first
  production producer of `invite.TypeTrade`.
- **FR-2.2** The target receives the clientbound `INVITE` body
  (`CharacterInteractionInviteBody(roomType, name, dwSN)`, already implemented at
  `libs/atlas-packet/interaction/clientbound/interaction_body.go:107`) with
  `roomType = 3` and the inviter's name.
- **FR-2.3** Invite rejections are surfaced through
  `CharacterInteractionInviteResultBody` (`interaction_body.go:113`) with the
  correct result code: target is already handling another request, target is
  in a trade-disallowed state, target does not exist, target is in a different
  map.
- **FR-2.4** On invite accept, the target enters the room: the target receives
  the clientbound `ENTER_RESULT` success body carrying the trade room blob
  (FR-8.2), and the owner receives the clientbound `ENTER` body
  (`CharacterInteractionEnterBody`, `interaction_body.go:145`) describing the
  arriving visitor.
- **FR-2.5** On serverbound `INVITE_DECLINE`, the inviter receives an
  invite-result notification and the pending room is destroyed (the owner is
  alone in it, so there is nothing to return).
- **FR-2.6** A pending, unanswered trade invite expires on the atlas-invites
  timeout; on expiry, the room is destroyed and the inviter notified.

### FR-3 — Staging items

- **FR-3.1** On serverbound `TRADE_PUT_ITEM`
  (`OperationTradePutItem`: `inventoryType`, `slot`, `quantity`, `targetSlot`),
  atlas-trades moves the specified quantity from the character's inventory into
  the trade room's escrow for that character at trade slot `targetSlot`
  (1..9).
- **FR-3.2** Staging uses the escrow primitives that already exist in the saga
  orchestrator: `ReleaseFromCharacter` removes the asset from the character's
  inventory into a holding state, `AcceptToCharacter` grants it into a
  character's inventory (`libs/atlas-saga/model.go:129-130`). An item staged in
  a trade is not present in the owner's inventory and cannot be double-staged,
  dropped, equipped, or sold.
- **FR-3.3** Staging is rejected — with the item left untouched — when the
  target trade slot is already occupied, when `targetSlot` is outside 1..9,
  when the quantity exceeds the stack, or when the item fails a restriction
  check (FR-4).
- **FR-3.4** Both sides receive the clientbound trade-put-item update carrying
  the staged asset, the trade slot, and which side staged it (FR-8.3).
- **FR-3.5** Staged items cannot be un-staged once placed — matching the
  reference client, which offers no remove action inside the trade window.
  Cancelling the trade (FR-6) is the only way to recover a staged item.
- **FR-3.6** Any staging or meso action by either side after either side has
  confirmed is rejected; the room is frozen from first confirm to settlement.

### FR-4 — Trade restrictions

Every rejection below must send the faithful mini-room error or
status-message, never a silent drop.

- **FR-4.1** An asset carrying `asset.FlagUntradeable` (0x08) or
  `asset.FlagMergeUntradeable` (0x200)
  (`libs/atlas-constants/asset/flag.go:14,16`) cannot be staged.
- **FR-4.2** An item whose WZ data sets `tradeBlock` cannot be staged.
  `tradeBlock` is currently exposed by atlas-data only for the consumable and
  setup readers (`services/atlas-data/atlas.com/data/consumable/reader.go:49`,
  `services/atlas-data/atlas.com/data/setup/reader.go:47`). Extending the
  equip, etc and cash readers to surface `tradeBlock` is in scope for this
  task; a missing flag today must not be silently treated as "tradeable".
- **FR-4.3** Quest items (the QUEST compartment) cannot be staged.
- **FR-4.4** An equipped item cannot be staged; only inventory-resident assets
  are eligible.
- **FR-4.5** A character below the tenant-configured minimum trade level
  cannot open or accept a trade. Default: no minimum (matching the reference
  behavior, which gates trade by item and map rather than level), configurable
  per tenant.
- **FR-4.6** Trade is refused in maps flagged trade-disallowed by WZ map data,
  with the mini-room enter error `TRADE_NOT_ALLOWED` (7) or
  `TRADE_NOT_ALLOWED_2` (20) as the version's client expects
  (`interaction_body.go:60,74`).
- **FR-4.7** A dead character (HP 0) cannot open, accept, or be invited to a
  trade — `NOT_WHEN_DEAD` (4).
- **FR-4.8** Meso staging is rejected when the amount exceeds the character's
  current meso, is negative, or would overflow the counterparty's meso cap.
- **FR-4.9** At settlement, the swap is refused when either side lacks free
  inventory slots in the required compartments for the incoming items, or when
  either side's meso cap would be exceeded. The refusal reverts nothing (the
  room is still live) and reports the reason to both sides.

### FR-5 — Confirm and settlement

- **FR-5.1** On serverbound `TRADE_CONFIRM`, the confirming side's slot is
  marked confirmed and the counterparty receives the clientbound trade-confirm
  notification (FR-8.4).
- **FR-5.2** When both sides have confirmed, atlas-trades runs the settlement
  pre-checks (FR-4.9) and, if they pass, submits a single saga to the
  saga-orchestrator that performs, as one compensatable unit:
  - `AcceptToCharacter` of each escrowed asset into the counterparty's
    inventory,
  - `AwardMesos` of each side's post-tax meso contribution to the counterparty,
  - the meso-tax deduction (FR-5.4).
- **FR-5.3** If any settlement step fails, the saga compensates: every escrowed
  asset returns to its original owner and no meso moves. Both sides are told
  the trade failed and the room is destroyed.
- **FR-5.4** The meso tax is applied to each side's staged meso independently at
  settlement. Given a staged amount `m`, the counterparty receives
  `m - floor(m * rate(m))`, and the difference is destroyed (removed from
  circulation), not credited to anyone.
- **FR-5.5** Tax tiers are tenant-configurable (FR-9). The shipped default
  reproduces the reference tiers: `m >= 100,000,000` → 6%; `>= 25,000,000` →
  5%; `>= 10,000,000` → 4%; `>= 5,000,000` → 3%; `>= 1,000,000` → 1.8%;
  `>= 100,000` → 0.8%; below 100,000 → 0%.
- **FR-5.6** On successful settlement, both sides receive the trade-complete
  notification (FR-8.5), the room is destroyed, and a ledger row is written
  (FR-7).
- **FR-5.7** Settlement is idempotent per room: a duplicate `TRANSACTION` or a
  retried saga must not double-award.

### FR-6 — Cancellation

- **FR-6.1** Serverbound `EXIT` (mode 10) from either participant cancels the
  trade before settlement.
- **FR-6.2** Cancellation returns every escrowed asset to its original owner
  via `AcceptToCharacter` and leaves both sides' meso untouched (meso is
  reserved logically, never moved, until settlement).
- **FR-6.3** Both sides receive the clientbound `LEAVE` body
  (`CharacterInteractionLeaveBody`, `interaction_body.go:157`) with the
  cancellation reason, and the room is destroyed.
- **FR-6.4** Disconnect, map change, and channel change of either participant
  are treated exactly as a cancel. Escrow recovery must not depend on the
  disconnecting client being reachable.
- **FR-6.5** If a settlement saga is already in flight when a cancel arrives,
  the cancel is ignored — settlement wins, and the client is reconciled by the
  settlement result.

### FR-7 — Completed-trade ledger

- **FR-7.1** Every settled trade writes one durable ledger row recording:
  tenant, world, channel, map, transaction id, both character ids and names,
  timestamp, the full list of assets each side gave (item id, quantity, and the
  asset's identity where it has one), each side's staged meso, each side's tax,
  and the settlement saga's transaction id.
- **FR-7.2** The ledger is queryable over REST (FR-API) filtered by character
  id and by time range, for GM investigation.
- **FR-7.3** Cancelled and failed trades are NOT written to the ledger; they are
  observable via structured logs and metrics only.
- **FR-7.4** Ledger rows are immutable — no update or delete endpoint.

### FR-8 — Packet layer

- **FR-8.1** The serverbound codecs already exist and are byte-fixtured
  (`libs/atlas-packet/interaction/serverbound/operation_trade_put_item.go`,
  `operation_trade_add_meso.go`, `operation_trade_confirm.go`,
  `operation_transaction.go`, `operation_create.go`, `operation_invite.go`,
  `operation_invite_decline.go`, `operation_cash_trade_open.go`). No new
  serverbound codec is expected; any that turns out to be a decode-order
  mismatch on a given version is fixed here.
- **FR-8.2** A new `interaction.NewTradeRoom(...)` constructor is added to
  `libs/atlas-packet/interaction/room.go`. The existing `Room` doc comment
  scopes it explicitly to the shop family; the trade-room enter blob layout
  must be derived per version from `CTradingRoomDlg::OnEnterResult` in the
  IDBs, not assumed to match the shop layout.
- **FR-8.3** A new clientbound trade-put-item body is added, keyed
  `TRADE_PUT_ITEM` in the writer `operations` table.
- **FR-8.4** New clientbound bodies are added for the meso update
  (`TRADE_ADD_MESO`) and the confirm notification (`TRADE_CONFIRM`).
- **FR-8.5** The trade-complete notification is added. Whether it is a distinct
  mode or the existing `LEAVE` (10) carrying a completion reason must be
  derived from `CTradingRoomDlg::OnPacket` per version — it is NOT assumed.
- **FR-8.6** Every mode byte for the above is **config-resolved** through the
  tenant writer `operations` table (DOM-25, the
  `atlas_packet.WithResolvedCode("operations", KEY, ...)` idiom used
  throughout `interaction_body.go`). No trade mode byte may be hard-coded in Go.
- **FR-8.7** Version divergence is expressed with the `MajorAtLeast` gate idiom
  in `libs/atlas-packet/interaction/.../version_gate.go`, never a raw
  `> N` comparison.
- **FR-8.8** Every new op × version cell is promoted in the coverage matrix
  (`docs/packets/audits/STATUS.md`) via the single-cell verify procedure
  (`docs/packets/audits/VERIFYING_A_PACKET.md`), with a byte-fixture test
  carrying a `packet-audit:verify` marker and a pinned evidence record. A cell
  that does not promote is a failure, not a prose claim.
- **FR-8.9** Serverbound `PLAYER_INTERACTION` (v83 0x07B), currently ❌ at
  `docs/packets/audits/STATUS.md:621`, is expected to promote as its arms
  become verified.

### FR-9 — Tenant configuration

- **FR-9.1** A new `trade` configuration resource is added to the atlas-tenants
  configuration system (the JSONB-backed
  `GET /tenants/{id}/configurations/{resource}` mechanism), holding:
  the meso tax tier table, a master enable/disable for the tax, the maximum
  staged item count (default 9), and the minimum trade level (default 0).
- **FR-9.2** A tenant with no `trade` configuration falls back to the shipped
  defaults; a missing configuration must not crash the service or silently
  disable trading.
- **FR-9.3** Tier tables are validated on load: descending thresholds, rates in
  [0, 1]. An invalid table is rejected with a loud error and the defaults are
  used.

### FR-10 — Cash trade room (mini-room type 6)

- **FR-10.1** The `CASH_TRADE_OPEN` arm at `character_interaction.go:239-275`
  currently decodes `nProc` 0, 4 and 11 and only logs. The `nProc == 0,
  roomType == CashTradeMiniRoomType` and `nProc == 4, roomType ==
  CashTradeMiniRoomType` branches must open and drive a cash trade room.
- **FR-10.2** A cash trade room follows the same lifecycle, invite, staging,
  confirm and settlement rules as FR-1..FR-7, sourcing items from the CASH
  compartment rather than the regular compartments.
- **FR-10.3** Cash items are staged and settled through the same escrow
  primitives; cash-item ownership and expiry semantics
  (atlas-asset-expiration) must be preserved across the transfer.
- **FR-10.4** Cash trade rooms are gated by version: `CASH_TRADE_OPEN` is
  absent from the gms_v48, gms_v61, gms_v72 and jms_v185 handler tables today.
  Whether that is a genuine client-side absence (matrix ⬜ n-a) or a template
  gap must be **verified against each version's IDB**, not assumed.

### FR-11 — Template routing

Verified gaps in the seed templates
(`services/atlas-configurations/seed-data/templates/`) as of this PRD:

- **FR-11.1** No template's `CharacterInteraction` **writer** `operations` table
  contains any trade key. All ten templates need the new clientbound keys from
  FR-8.3..FR-8.5 added at their version-correct byte.
- **FR-11.2** `template_gms_92_1.json` is the outlier: its
  `CharacterInteractionHandle` (0x8D) handler table contains only
  `CREATE`, `VISIT`, `CHAT`, `EXIT` and the memory-game keys — it is missing
  `INVITE`, `INVITE_DECLINE`, `CASH_TRADE_OPEN`, all four `TRADE_*`/
  `TRANSACTION` keys, and every merchant/personal-store key. Its writer table
  is likewise missing `CHAT`, `PERSONAL_STORE_ITEM_SOLD`,
  `MEMORY_GAME_PUT_STONE_ERROR` and both `MERCHANT_VIEW_*` keys. The trade keys
  are in scope for this task; the non-trade gaps are recorded as a finding in
  the design phase, not silently fixed here.
- **FR-11.3** `TRANSACTION` is present only in gms_v83, gms_v84, gms_v87 and
  gms_v95. `template_gms_48_1.json`, `template_gms_61_1.json`,
  `template_gms_72_1.json`, `template_gms_79_1.json` and
  `template_jms_185_1.json` lack it. Whether the mode exists in those clients
  must be verified per IDB before either adding the key or marking the cell
  n-a.
- **FR-11.4** `template_jms_185_1.json` also lacks `CASH_TRADE_OPEN` and the
  `TRADE_NOT_ALLOWED` enter-error key.
- **FR-11.5** Every template edit must keep both `handlers` and `writers`
  arrays in strictly ascending `opCode` order
  (`tools/template-opcode-order-guard.sh`) and must not introduce a duplicate
  `(implementation, opCode)` binding
  (`tools/template-duplicate-binding-guard.sh`).

## 5. API Surface

All endpoints are JSON:API (api2go), served by atlas-trades, tenant-scoped via
the standard tenant header decorator.

### `GET /trades/rooms`

List live trade rooms for the tenant. Query filters: `filter[characterId]`,
`filter[worldId]`, `filter[channelId]`, `filter[mapId]`.

Resource type `rooms`. Attributes:

| Field | Type | Notes |
|---|---|---|
| `roomType` | byte | 3 (trade) or 6 (cash trade) |
| `worldId` / `channelId` / `mapId` | uint | Field the room lives in |
| `state` | string | `PENDING_INVITE`, `OPEN`, `SETTLING` |
| `participants` | array | `{characterId, position, confirmed, mesoStaged, items[]}` |
| `createdAt` | timestamp | |

### `GET /trades/rooms/{roomId}`

Single live room. `404` when the room does not exist (a settled or cancelled
room is gone).

### `GET /trades/ledger`

Completed-trade ledger. Query filters: `filter[characterId]` (matches either
side), `filter[from]` / `filter[to]` (RFC3339). Paginated
(`page[number]`, `page[size]`, max 100).

Resource type `ledgerEntries`. Attributes:

| Field | Type | Notes |
|---|---|---|
| `transactionId` | uuid | The settlement saga transaction |
| `worldId` / `channelId` / `mapId` | uint | |
| `roomType` | byte | 3 or 6 |
| `settledAt` | timestamp | |
| `sides` | array (2) | `{characterId, characterName, mesoStaged, mesoTax, mesoDelivered, items[]}` |

`items[]` entries: `{itemId, quantity, assetId, referenceId}`.

### `GET /trades/ledger/{entryId}`

Single ledger entry.

### Error cases

| Condition | Status |
|---|---|
| Unknown room / entry | 404 |
| Missing or unresolvable tenant header | 400 |
| Malformed filter or page parameter | 400 |
| Page size above the cap | 400 |

No POST/PATCH/DELETE endpoints: trade rooms are driven exclusively by Kafka
commands from atlas-channel, and ledger rows are immutable.

### Kafka contracts

**`COMMAND_TOPIC_TRADE`** — atlas-channel → atlas-trades. One envelope with a
typed body per command: `CREATE_ROOM`, `INVITE`, `DECLINE_INVITE`,
`ENTER_ROOM`, `PUT_ITEM`, `ADD_MESO`, `CONFIRM`, `TRANSACTION`, `CANCEL`,
`CHAT`. Every command carries `transactionId`, `worldId`, `channelId`,
`characterId`.

**`EVENT_TOPIC_TRADE_STATUS`** — atlas-trades → atlas-channel. Status types:
`ROOM_CREATED`, `INVITE_SENT`, `INVITE_REJECTED`, `PARTICIPANT_ENTERED`,
`ITEM_STAGED`, `MESO_STAGED`, `PARTICIPANT_CONFIRMED`, `SETTLED`, `CANCELLED`,
`ERROR` (carrying the mini-room error key so atlas-channel writes the faithful
clientbound body). Every event carries the room id, both participants, and the
originating `transactionId`.

atlas-trades additionally produces `COMMAND_TOPIC_INVITE` commands
(`invite.TypeTrade`) and consumes `EVENT_TOPIC_INVITE_STATUS`, and produces
saga-orchestrator commands for escrow and settlement.

## 6. Data Model

atlas-trades is DB-backed (`atlas-trades` database) for the ledger only. Live
room state is in-memory.

### In-memory: `Registry`

Tenant-partitioned, `sync.Once` singleton + `sync.RWMutex`, following
`services/atlas-mini-games/atlas.com/mini-games/game/registry.go`. Holds
`map[tenant.Id]map[room.Id]Room` plus a `map[tenant.Id]map[character.Id]room.Id`
membership index so a character's room resolves in one lookup (the same
invariant atlas-mini-games maintains — a character is in at most one room).

Immutable domain models with private fields, getters, and a Builder
(no `*_testhelpers.go`): `Room`, `Participant`, `StagedItem`.

### Persistent: `trade_ledger_entries`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | NOT NULL, indexed |
| `transaction_id` | uuid | NOT NULL, unique with `tenant_id` (idempotency) |
| `world_id` | smallint | NOT NULL |
| `channel_id` | smallint | NOT NULL |
| `map_id` | integer | NOT NULL |
| `room_type` | smallint | NOT NULL (3 or 6) |
| `settled_at` | timestamptz | NOT NULL, indexed |

### Persistent: `trade_ledger_sides`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | NOT NULL, indexed |
| `entry_id` | uuid | FK → `trade_ledger_entries.id`, NOT NULL |
| `character_id` | integer | NOT NULL, indexed (GM lookup by character) |
| `character_name` | text | NOT NULL (denormalized — names change) |
| `meso_staged` | bigint | NOT NULL |
| `meso_tax` | bigint | NOT NULL |
| `meso_delivered` | bigint | NOT NULL |

Exactly two rows per entry.

### Persistent: `trade_ledger_items`

| Column | Type | Notes |
|---|---|---|
| `id` | uuid | PK |
| `tenant_id` | uuid | NOT NULL, indexed |
| `side_id` | uuid | FK → `trade_ledger_sides.id`, NOT NULL |
| `item_id` | integer | NOT NULL |
| `quantity` | integer | NOT NULL |
| `asset_id` | integer | Nullable — set for identity-bearing assets |
| `reference_id` | integer | Nullable — equip/pet/cash reference |

Migration notes: fresh tables, no backfill. GORM auto-migration on startup,
consistent with the rest of the fleet. Every query is `tenant_id`-scoped.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-trades** (new) | Owns trade-room registry, invite orchestration, escrow staging, settlement saga submission, the ledger, and the REST surface. Requires the full `docs/adding-a-new-service.md` registration pass: `.github/config/services.json`, `docker-bake.hcl` `go_services`, `go.work`, `deploy/k8s/base/atlas-trades.yaml` + kustomization + env-configmap topic vars, both the main and PR overlays (image pin, DB name suffix, `ATLAS_DB_NAMES`, topic literals, consumer-group patch), `deploy/shared/routes.conf` + regenerated ingress, the `atlas-trades-main` database, and `tools/db-bootstrap.sh`. `tools/service-registration-guard.sh` must pass. |
| **atlas-channel** | Replace the seven `l.Debugf`-only trade arms in `socket/handler/character_interaction.go` (lines 85-88, 135-146, 276-303) and the cash-trade arm (239-275) with calls into a new `trade` processor package. Consume `EVENT_TOPIC_TRADE_STATUS` and write the clientbound bodies. Extend the `EXIT`, `CHAT` and `VISIT` arms to notify the trade processor alongside the existing minigame and merchant processors. |
| **libs/atlas-packet** | New clientbound trade bodies and `interaction.NewTradeRoom`; version gates; byte-fixture tests per version. |
| **libs/atlas-saga** | Any new action needed for the atomic two-party swap. The existing `ReleaseFromCharacter` / `AcceptToCharacter` / `AwardMesos` primitives are expected to suffice; a `trade_settlement` composite that expands into them (the pattern of `expandTransferToStorage` at `services/atlas-saga-orchestrator/.../saga/processor.go:1266`) is the likely shape. |
| **atlas-saga-orchestrator** | Handle the settlement composite and its compensation. |
| **atlas-invites** | No code change expected — `invite.TypeTrade` already exists. First production producer of that type; verify the timeout and reject paths behave for a two-party ephemeral room. |
| **atlas-data** | Surface `tradeBlock` from the equip, etc and cash WZ readers (FR-4.2). Today only the consumable and setup readers expose it. |
| **atlas-tenants** | New `trade` configuration resource (FR-9). |
| **atlas-configurations** | Trade keys added to all ten seed templates, handler and writer tables (FR-11). |
| **docs/packets** | New evidence records and matrix cells; `STATUS.md` regenerated. |

## 8. Non-Functional Requirements

- **Multi-tenancy.** Every registry map, every DB query, and every Kafka
  message is tenant-scoped via `tenant.MustFromContext(ctx)`. A room in tenant
  A must be invisible and unreachable from tenant B.
- **Atomicity.** No code path may move an item or meso outside the settlement
  saga. A crash of atlas-trades mid-trade must leave every escrowed asset
  recoverable — escrow lives in the saga/inventory layer, not in the in-memory
  registry.
- **Crash recovery.** On startup, atlas-trades must reconcile any assets left
  in trade escrow by a previous process (returning them to their owners). The
  in-memory registry starts empty; a client whose room vanished is reconciled
  by the next interaction being rejected with a room-closed error.
- **Concurrency.** Two characters act on the same room simultaneously by
  construction. Room mutation must be serialized per room. Note the known
  hazard that `ForEachInMap` in atlas-channel is parallel — any shared state
  touched from a broadcast callback must be safe.
- **Latency.** A staging action must produce its clientbound update within
  200 ms p99 under normal load; settlement within 2 s p99. The client dialog
  is interactive and a visible lag reads as a lost item.
- **Security / abuse.** Every quantity, slot, and meso amount from the client is
  untrusted and re-validated server-side against the character's actual
  inventory and meso — the client's own view is never authoritative. Duplicate
  or replayed `TRANSACTION` commands must not double-settle (FR-5.7).
- **Observability.** Structured logs at every state transition carrying tenant,
  room id, both character ids, and transaction id. Metrics: trades opened,
  settled, cancelled, failed-at-settlement, escrow recoveries at startup, and
  total meso taxed.
- **Goroutines.** Every goroutine spawned via `routine.Go`
  (`tools/goroutine-guard.sh`). Any Redis use goes through `libs/atlas-redis`
  (`tools/redis-key-guard.sh`).
- **Wire values.** No hard-coded client-facing byte anywhere — mode bytes,
  error codes and room types are all config-resolved (DOM-25).

## 9. Open Questions

1. **Trade-room enter blob layout.** `interaction.Room` is explicitly scoped to
   the shop family; the mini-game rooms needed their own encoder. Whether the
   trade room can reuse `Room` or needs a third encoder must be settled by
   reading `CTradingRoomDlg::OnEnterResult` per version. Resolve in design.
2. **Trade-complete transport.** Whether completion is its own mode or a `LEAVE`
   (10) with a completion reason, and whether that differs across the ten
   versions. Resolve in design from the IDBs.
3. **`TRANSACTION` on legacy versions.** Absent from the gms_v48/61/72/79 and
   jms_v185 handler tables. Genuine client absence (matrix ⬜) or template gap?
   Determines whether those versions settle on `TRADE_CONFIRM` alone.
4. **`CASH_TRADE_OPEN` on gms_v48/61/72 and jms_v185.** Same question for the
   cash trade room — does `CCashTradingRoomDlg` exist in those clients at all?
5. **Meso reservation vs escrow.** Meso is proposed as logically reserved and
   moved only at settlement (FR-6.2). If a character can spend reserved meso
   through another channel of activity mid-trade, settlement can fail late.
   Whether meso needs true escrow is a design decision.
6. **Non-trade gms_92 template gaps.** The v92 template is missing many
   non-trade interaction keys (FR-11.2). Fixed here or split into a follow-up
   template task? Recommend recording as a finding and deciding in design.
7. **Ledger retention.** No retention policy is specified. Unbounded growth is
   acceptable at current scale but should be revisited.

## 10. Acceptance Criteria

**Feature behavior**

- [ ] Two characters in the same map can complete a trade of items and meso, on
      gms_v83, with both inventories and meso balances correct afterwards.
- [ ] The receiving character's window shows the counterparty's staged items and
      meso live, before either side confirms.
- [ ] Declining an invite closes the inviter's window and leaves no room behind.
- [ ] Cancelling a trade after staging returns every staged item to its owner.
- [ ] Disconnecting mid-trade returns every staged item to its owner; verified
      by killing the client and re-logging in.
- [ ] Attempting to stage an untradeable item, a quest item, or an equipped item
      is refused with the faithful error and the item is untouched.
- [ ] Settling into a full inventory is refused with the faithful error and
      nothing moves.
- [ ] A 10,000,000 meso contribution delivers 9,600,000 to the counterparty
      under the default tier table (4%), and the 400,000 difference exists
      nowhere afterwards.
- [ ] A tenant with a custom tier table gets its own rates; a tenant with the
      tax disabled gets a 0% deduction.
- [ ] A cash trade (mini-room type 6) completes on every version where the
      client supports it.

**Cross-version**

- [ ] The trade window works end-to-end on all ten versions: gms_v48, gms_v61,
      gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185
      — or the version is explicitly marked ⬜ n-a with IDB evidence for the
      arm that does not exist there.
- [ ] Every new op × version cell is ✅ in `docs/packets/audits/STATUS.md`, each
      backed by a byte-fixture test with a `packet-audit:verify` marker and a
      pinned evidence record.
- [ ] Serverbound `PLAYER_INTERACTION` (v83 0x07B) is no longer ❌.
- [ ] All ten seed templates carry the trade handler and writer keys at their
      version-correct bytes.

**GM / operations**

- [ ] `GET /trades/ledger?filter[characterId]=N` returns every settled trade
      that character participated in, both sides itemized with tax.
- [ ] `GET /trades/rooms` shows a live room mid-trade and 404s the room after
      settlement.
- [ ] Restarting atlas-trades mid-trade recovers every escrowed asset to its
      owner, logged and counted.

**Build & verification** (project CLAUDE.md gates)

- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean in every changed service.
- [ ] `docker buildx bake atlas-trades` and every other service whose `go.mod`
      changed builds from the worktree root.
- [ ] `tools/service-registration-guard.sh` clean.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
      `tools/skill-job-id-guard.sh`, `tools/buff-duration-guard.sh` clean.
- [ ] `tools/template-opcode-order-guard.sh`,
      `tools/template-duplicate-binding-guard.sh`,
      `tools/template-movement-types-guard.sh` clean.
- [ ] `tools/lint.sh --check` clean.
- [ ] Code review run (plan-adherence + backend-guidelines reviewers) before the
      PR is opened.
