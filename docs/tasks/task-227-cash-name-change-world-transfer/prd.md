# Cash Name Change & World Transfer — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-14

---

## 1. Overview

Cash-shop item classification `540` (`item.ClassificationCharacterImprints`,
`libs/atlas-constants/item/constants.go:103`) covers the two "character imprint" cash
services a player can buy: a **name change** (rename an existing character) and a
**world transfer** (move an existing character to a different world in the same
tenant). Neither is implemented in Atlas today. The client-side scaffolding is
partly present — `BUY_NAME_CHANGE` (op 46) and `BUY_WORLD_TRANSFER` (op 49) are
decoded in `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_operation.go:184-195`,
and the `NAME_CHANGE_BUY_DONE` / `TRANSFER_WORLD_SUCCESS` / `TRANSFER_WORLD_FAILED` /
`TRANSFER_WORLD_NOTICE_REASON` clientbound arms exist in
`libs/atlas-packet/cash/clientbound/shop_operation_body.go` — but every one of those
handlers currently decodes the packet, writes an `l.Infof`, and returns. Nothing is
persisted, nothing is applied, and the *item-use* path (classification 540 in
`CharacterCashItemUseHandleFunc`) has no branch at all.

The whole surrounding packet family is also unimplemented: the availability-check
round trips (`NAME_TRANSFER` 0x010, `WORLD_TRANSFER` 0x012, and their
`CASHSHOP_CHECK_NAME_CHANGE`, `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT`,
`CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` responses) and the cancellation
family (`CANCEL_NAME_CHANGE_RESULT`, `CANCEL_TRANSFER_WORLD_RESULT`,
`CANCEL_NAME_CHANGE_BY_OTHER`) are ❌ on every applicable version in
`docs/packets/audits/STATUS.md` (rows 157, 161, 204, 409, 413, 476, 528, 532).

This task implements both flows end to end, at equal depth, with GMS-authentic
**deferred application**: a request is recorded as *pending*, is cancellable while
pending, and is applied when the character is next safely offline / at
character-select. That deferral is not a stylistic choice — it is the reason the
`CANCEL_*` packet family exists, and world transfer in particular cannot be applied
to a character that is live in a channel.

## 2. Goals

Primary goals:

- A player can change their character's name using a Cash/0540 name-change item or
  the cash-shop `BUY_NAME_CHANGE` purchase, and the rename takes effect.
- A player can move their character to another world in the same tenant using a
  Cash/0540 world-transfer item or the cash-shop `BUY_WORLD_TRANSFER` purchase.
- Both flows expose a pre-commit availability check so the client can tell the
  player "that name is taken" / "you cannot transfer to that world" before spending.
- Both flows are cancellable while pending — by an **operator**, from atlas-ui —
  and cancelling refunds the consumed item and notifies the player in-game.
- The client's name-change and world-transfer packet family is implemented and
  **verified** (matrix `✅`) on every version where it is not `⬜ n-a`.
- The transfer leaves no dangling cross-world state: guild/party/buddy memberships
  and in-flight asset commitments are resolved before the move lands.

Non-goals:

- Cross-**tenant** transfer. Worlds are tenant-scoped; a transfer never crosses
  tenant boundaries.
- Moving account storage between worlds (see FR-4.6 — storage never moves).
- Migrating a character to a world that belongs to a different client version /
  socket configuration than the source (out of scope; see §9).
- A **player-facing** cancel surface. The game client has no cancel packet to send
  (§4.2.1), and atlas-ui is an administrative SPA, not a player portal. A player
  web portal is well outside this task.
- An atlas-ui surface for *initiating* a rename or transfer. The operator surface in
  scope is read + cancel only (FR-2.10); operators do not grant renames or transfers
  from the console.
- Cash-shop pricing/commodity authoring for the 540 items (that is data, seeded
  separately).
- `Cash/0543` Maple Life character-creation items, which share the "character
  imprint" neighbourhood but are a separate backlog row.

## 3. User Stories

- As a player, I want to type a candidate new name and be told immediately whether
  it is available, so that I do not waste a cash item on a taken name.
- As a player, I want to use my Name Change coupon and have my character renamed,
  so that my new name shows in the character list, on the map, and to other players.
- As a player, I want to be told when my pending name change was invalidated because
  someone else took that name first, so that I know why nothing happened.
- As a player whose pending request was cancelled by staff, I want to be told in-game
  and get my coupon back, so that a cancellation is never a silent loss.
- As an operator, I want to see a character's pending name change or world transfer
  in atlas-ui, so that I can answer a support ticket without querying the database.
- As an operator, I want to cancel a player's pending request from atlas-ui and have
  the coupon refunded automatically, so that a mistyped name or a
  wrong-world request is recoverable — this is the only cancel path that exists,
  since the game client cannot send a cancel.
- As a player, I want to move my character to a world where my friends play, so that
  I can play with them without re-levelling.
- As a player transferring worlds, I want to be warned before I commit if I am the
  last character my account has in the source world, so that I do not strand my
  storage there.
- As a player, I want to be blocked (with a reason) from transferring while I am in
  a guild, in a party, running a hired merchant, or holding an open trade or auction
  listing, so that the move cannot corrupt those systems.

## 4. Functional Requirements

### 4.1 Entry points

- **FR-1.1** `CharacterCashItemUseHandleFunc`
  (`services/atlas-channel/.../socket/handler/character_cash_item_use.go`) MUST gain
  a branch for `item.ClassificationCharacterImprints` (540) that routes to the
  name-change or world-transfer request flow according to the item's cash slot item
  type, following the existing per-type branch pattern (cf. the Vicious Hammer,
  expiration-extender, and store-search branches).
- **FR-1.2** The version-scoped `CashSlotItemType` mapping for classification 540
  MUST be derived from the client, per version, the same way
  `viciousHammerCashSlotItemType` / `expirationExtenderCashSlotItemType` are — never
  hard-coded to a single value across versions. Where the two sub-flows (name change
  vs world transfer) map to distinct slot item types, each gets its own constant.
- **FR-1.3** The concrete Cash/0540 item IDs, and which of them are name-change vs
  world-transfer, MUST be resolved from WZ data via atlas-data — **not** from
  memory. They are currently unverified (§9, OQ-1).
- **FR-1.4** `CashShopOperationBuyNameChange` (op 46) and
  `CashShopOperationBuyWorldTransfer` (op 49) in `cash_shop_operation.go` MUST stop
  being log-only and MUST invoke the same request flow as the item-use path, so both
  entry points converge on one service-side request.
- **FR-1.5** The availability-check serverbound packets MUST be implemented and
  routed: `NAME_TRANSFER` (`CCashShop::SendCheckNameChangePossiblePacket`, 0x010) and
  `WORLD_TRANSFER` (`CCashShop::SendCheckTransferWorldPossiblePacket`, 0x012).
- **FR-1.6** The corresponding clientbound results MUST be implemented:
  `CASHSHOP_CHECK_NAME_CHANGE`, `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT`,
  `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT`.
- **FR-1.7** The cancellation family MUST be implemented:
  `CANCEL_NAME_CHANGE_RESULT`, `CANCEL_TRANSFER_WORLD_RESULT`,
  `CANCEL_NAME_CHANGE_BY_OTHER`.
- **FR-1.8** Every new/changed handler and writer MUST be registered in all
  applicable tenant socket-config templates under
  `services/atlas-configurations/seed-data/templates/`, in ascending `opCode` order,
  and MUST pass `tools/template-opcode-order-guard.sh` and
  `tools/template-duplicate-binding-guard.sh`. Any new dispatcher mode value MUST be
  config-resolved through the `operations` table (DOM-25), never a literal.

### 4.2 Pending-request lifecycle (shared by both flows)

- **FR-2.1** A request creates a **pending request** record with a discriminator
  (`NAME_CHANGE` | `WORLD_TRANSFER`), the character id, the requested value (new name
  or destination world id), the consumed asset reference, a status, and timestamps.
- **FR-2.2** Statuses: `PENDING`, `APPLIED`, `CANCELLED`, `REJECTED`, `EXPIRED`.
  Transitions are one-way out of `PENDING`; a terminal record is never reopened.
- **FR-2.3** A character MUST NOT hold more than one `PENDING` request of a given
  type at a time. A second request while one is pending is rejected with a distinct
  error reason, not silently queued.
- **FR-2.4** A `PENDING` request is applied at the **next safe point**: when the
  character is not present in any channel — i.e. at character-select/world-select
  time on the login path, or on the character's logout, whichever comes first.
  Application MUST NOT mutate a character that is live in a channel.

#### 4.2.1 Who can cancel — and why it is not the player

The three cancel packets in this family are **clientbound receivers only**:
`CWvsContext::OnCancelNameChangeResult`, `CWvsContext::OnCancelTransferWorldResult`,
`CWvsContext::OnCancelNameChangebyOther` (`docs/packets/audits/STATUS.md` 157, 161,
204). The client's entire cash-shop send surface, across every version's export index
(`docs/packets/audits/status.json`), is `SendBuyAvatarPacket`,
`SendBuyNameChangeItemPacket`, `SendBuyTransferWorldItemPacket`,
`SendChangeMaplePoint`, `SendCheckDuplicateIDPacket`,
`SendCheckNameChangePossiblePacket`, `SendCheckTransferWorldPossiblePacket`,
`SendGiftsPacket`, `SendTransferFieldPacket` — there is **no `SendCancel*` of any
kind**. The client can buy and can check availability; it can never ask the server to
cancel. The `CANCEL_*` packets are therefore one-way server→client *notifications*
that a pending request was cancelled or invalidated. In GMS the player-initiated
cancel lived on the account-management website, outside the game client.

Atlas has no player portal, so the cancel path in this task is **operator-initiated
from atlas-ui**, and the `CANCEL_*` packets deliver the outcome to the player.

- **FR-2.5** A `PENDING` request MAY be cancelled by an **operator** via the
  atlas-character REST endpoint (§5), surfaced in atlas-ui (FR-2.10). Cancellation
  moves the request to `CANCELLED` and triggers the refund (FR-2.8). There is no
  player-initiated cancel, in-game or otherwise.
- **FR-2.6** A pending request has a configurable expiry (default 7 days). On expiry
  it moves to `EXPIRED` and is refunded identically to a cancellation.
- **FR-2.7** If application-time re-validation fails (name taken in the interim,
  destination world gone, eligibility now violated), the request moves to `REJECTED`,
  the item is refunded, and the player is notified — for a name change specifically
  via `CANCEL_NAME_CHANGE_BY_OTHER` when the cause is another player taking the name.
- **FR-2.8** **Consumption is on request, refund is on non-application.** The
  Cash/0540 asset (or the cash-shop purchase entitlement) is consumed at the moment
  the request is accepted. Any exit from `PENDING` other than `APPLIED` restores the
  asset to the character's cash inventory. Refund MUST be idempotent — a redelivered
  cancel event must not mint a second item (see the known Kafka at-least-once
  redelivery failure mode).
- **FR-2.9** Every non-`APPLIED` terminal transition MUST notify the player through
  the version-appropriate clientbound packet: `CANCEL_NAME_CHANGE_RESULT` /
  `CANCEL_TRANSFER_WORLD_RESULT` for an operator cancellation or an expiry, and
  `CANCEL_NAME_CHANGE_BY_OTHER` when a name change was invalidated by another
  character taking the name. If the character is **offline** when the transition
  occurs, the notification MUST be deferred and delivered on their next login — an
  unread notification is not discarded, because otherwise the coupon reappears in the
  cash inventory with no explanation.
- **FR-2.10** atlas-ui MUST surface pending changes on the existing
  `CharacterDetailPage` (`services/atlas-ui/src/pages/CharacterDetailPage.tsx`):
  - a panel listing the character's pending-change records with type, requested value
    (new name / destination world), status, created and expiry timestamps;
  - a **Cancel** action on a `PENDING` record, behind a confirmation dialog that
    names the character and the requested value, calling the DELETE endpoint of §5;
  - the resolved history (`APPLIED` / `CANCELLED` / `REJECTED` / `EXPIRED`) with its
    reason, so an operator can answer "what happened to my coupon?" from the console;
  - React Query cache invalidation on success so the panel reflects the new state
    without a manual reload.
  The operator surface is **read + cancel only** — it MUST NOT be able to create a
  rename or transfer request, nor edit a requested value.

### 4.3 Name change

- **FR-3.1** The requested name MUST pass the **same validator that character
  creation uses** in atlas-character-factory — length, charset, and blocked-word
  rules are shared, not re-specified here. This SHOULD be factored into a shared
  validator rather than duplicated.
- **FR-3.2** The requested name MUST be **unique across every world in the tenant**,
  not merely the character's own world. This is deliberately stricter than character
  creation so that a later world transfer can never produce a collision.
- **FR-3.3** Accepting a name-change request **soft-reserves** the requested name for
  the lifetime of the pending request. While reserved, the name is unavailable to
  character creation, to other name-change requests, and to the availability check.
- **FR-3.4** Releasing the reservation happens on every terminal transition:
  `APPLIED` (name now really belongs to the character), `CANCELLED`, `EXPIRED`,
  `REJECTED`.
- **FR-3.5** The availability check (`NAME_TRANSFER` 0x010) MUST report unavailable
  for: an existing character name in any world of the tenant, an active soft
  reservation, or a name failing FR-3.1 validation.
- **FR-3.6** On application, atlas-character updates the name and emits the existing
  `NAME_CHANGED` status event (`StatusEventTypeNameChanged`,
  `services/atlas-character/.../kafka/message/character/kafka.go:235,358`). Every
  service that denormalises a character name — guild rosters, buddy lists, party
  members, messenger, marriage, ranking, merchant/auction listings — MUST consume
  that event and update its copy. Services that do not currently consume it MUST be
  identified and wired; a stale name copy is the primary failure mode of this
  feature.
- **FR-3.7** A rename applied while the character is offline requires no in-map
  broadcast. Because FR-2.4 forbids applying to a live character, no live re-spawn /
  re-broadcast path is required.

### 4.4 World transfer

- **FR-4.1** The destination world MUST be a different, existing, non-full world in
  the **same tenant**, and the account MUST have a free character slot in it.
- **FR-4.2** Eligibility pre-checks that **block** the request (each with a distinct
  reason code surfaced through `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` /
  `TRANSFER_WORLD_FAILED`):
  - character name already taken in the destination world (cannot occur if FR-3.2
    tenant-wide uniqueness holds for all names, but MUST still be checked at apply
    time as a safety net);
  - destination world full / no free character slot;
  - character is banned or otherwise restricted.
- **FR-4.3** Memberships that are **cancelled** as part of the transfer rather than
  blocking it: guild membership, party membership, and buddy-list entries (in both
  directions). Guild-master membership is the exception and MUST block the transfer
  with its own reason, since a guild cannot be left masterless.
- **FR-4.4** In-flight asset commitments MUST be cancelled/settled before the move:
  open player trades (atlas-trades), hired-merchant listings (atlas-merchant), MTS /
  auction listings (atlas-mts), and any pending duey-style deliveries. Where a system
  holds the player's assets, the assets MUST be returned to the character before the
  transfer proceeds; a transfer MUST NOT strand assets in a source-world service.
- **FR-4.5** What moves with the character: the **character record** (its
  `World` column, `services/atlas-character/.../character/entity.go`) and the
  character's **inventory** in full (all compartments, including cash and equipped).
- **FR-4.6** **Storage never moves.** Storage is keyed `(tenant, world, account)`
  (`services/atlas-storage/.../storage/entity.go`) and is shared by every character
  the account owns in that world, so moving it would rob siblings. A non-empty source
  storage does **not** block the transfer.
- **FR-4.7** When the transferring character is the account's **last** character in
  the source world, the player MUST be warned before committing — a pink-text world
  message via the existing `WorldMessagePinkTextBody`
  (`services/atlas-channel/.../socket/writer/world_message.go:107`) — stating that
  storage in the source world will become inaccessible. The warning is advisory: it
  does not block the transfer.
- **FR-4.8** The multi-service move MUST be executed as a compensating saga through
  atlas-saga-orchestrator, not as a fan-out of independent commands. Each step
  (membership severance, listing settlement, inventory re-home, character world
  update) MUST have a compensation, and a failure mid-saga MUST leave the character
  wholly in the source world with its item refunded — never half-moved.
- **FR-4.9** On successful application, the character appears in the destination
  world's character list at next login and is absent from the source world's.

### 4.5 Client feedback

- **FR-5.1** Every rejection path MUST reach the client as a specific reason, using
  the version-appropriate arm (`TRANSFER_WORLD_FAILED`,
  `TRANSFER_WORLD_NOTICE_REASON`, or the check-result packets) — never a silent drop
  or a generic failure.
- **FR-5.2** Success paths emit `NAME_CHANGE_BUY_DONE` / `TRANSFER_WORLD_SUCCESS`
  for the purchase, matching the existing item-blob body builders
  (`CashShopNameChangeBuyDoneBody`, `CashShopTransferWorldDoneBody`).
- **FR-5.3** The EnableActions/`ExclRequest` unlock contract MUST be respected: the
  request paths that do not warp unlock the client; paths that end in a
  world/character-select transition must not double-unlock.

## 5. API Surface

New REST resources follow JSON:API via api2go, tenant-scoped through
`tenant.MustFromContext(ctx)`.

### atlas-character

- `GET /characters/{characterId}/pending-changes` — list pending imprint requests
  for a character.
- `POST /characters/{characterId}/pending-changes` — create a request.
  Attributes: `type` (`NAME_CHANGE` | `WORLD_TRANSFER`), `requestedName` (name
  change), `destinationWorldId` (world transfer), `assetId` (the consumed 540 asset,
  or the cash-shop entitlement reference).
  Errors: `409` request already pending; `422` validation failure (invalid name,
  name taken, ineligible destination); `404` unknown character.
- `DELETE /characters/{characterId}/pending-changes/{id}` — cancel a pending
  request. **Operator-facing** — this is the only cancel path in the system (§4.2.1);
  no game-client packet reaches it. Errors: `409` if already terminal.
- `GET /characters/name-availability?name={name}` — tenant-wide availability check
  backing FR-3.5. Response distinguishes `available`, `taken`, `reserved`, `invalid`.

### Kafka

Commands (`COMMAND_TOPIC_CHARACTER_*`):

- `REQUEST_NAME_CHANGE` — `{ characterId, requestedName, assetId, transactionId }`
- `REQUEST_WORLD_TRANSFER` — `{ characterId, destinationWorldId, assetId, transactionId }`
- `CANCEL_PENDING_CHANGE` — `{ characterId, pendingChangeId, transactionId }`
- `APPLY_PENDING_CHANGE` — `{ characterId, pendingChangeId, transactionId }` (emitted
  at the safe point of FR-2.4)

Status events (`EVENT_TOPIC_CHARACTER_STATUS`):

- `NAME_CHANGED` — existing event, reused unchanged (`oldName`, `newName`).
- `WORLD_CHANGED` — new: `{ oldWorldId, newWorldId }`.
- `PENDING_CHANGE_CREATED` / `PENDING_CHANGE_RESOLVED` — `{ pendingChangeId, type,
  status, reason }`, consumed by atlas-channel to drive the client feedback of §4.5
  and by atlas-cashshop to drive refunds.

All command bodies carry a `transactionId` for saga correlation and MUST be
idempotent on redelivery (task-208 idempotency contract).

## 6. Data Model

New table owned by **atlas-character**: `character_pending_changes`.

| column | type | notes |
|---|---|---|
| `tenant_id` | uuid | not null; every query scoped by it |
| `id` | uuid | primary key |
| `character_id` | uint32 | not null; indexed with `tenant_id`, `status` |
| `type` | text | `NAME_CHANGE` \| `WORLD_TRANSFER` |
| `status` | text | `PENDING` \| `APPLIED` \| `CANCELLED` \| `REJECTED` \| `EXPIRED` |
| `requested_name` | text | null for world transfer |
| `destination_world_id` | byte | null for name change |
| `source_world_id` | byte | recorded at request time for compensation |
| `asset_id` | uint32 | consumed asset, for refund |
| `reason` | text | populated on `REJECTED` |
| `transaction_id` | uuid | saga correlation |
| `created_at` / `resolved_at` / `expires_at` | timestamptz | |

Constraints:

- Partial unique index on `(tenant_id, character_id, type)` where
  `status = 'PENDING'` — enforces FR-2.3 at the database level, not just in code.
- Partial unique index on `(tenant_id, lower(requested_name))` where
  `status = 'PENDING' AND type = 'NAME_CHANGE'` — this **is** the soft reservation of
  FR-3.3; no separate reservation table.
- Character-name uniqueness (FR-3.2) requires a tenant-wide (not per-world) uniqueness
  check against `characters`. Whether an index change on `characters` is warranted, or
  a query-level check suffices, is a design-phase decision — existing per-world
  duplicates in live data must be surveyed before adding a constraint.

Migration: GORM `AutoMigrate` in atlas-character, following the existing `Migration`
function pattern in `character/entity.go`. No destructive change to `characters`
beyond a possible index.

## 7. Service Impact

| Service | Change |
|---|---|
| **atlas-channel** | New 540 branch in `character_cash_item_use.go`; real bodies for `BUY_NAME_CHANGE` / `BUY_WORLD_TRANSFER` in `cash_shop_operation.go`; new handlers for `NAME_TRANSFER` / `WORLD_TRANSFER` check packets; new writers for the three `CASHSHOP_CHECK_*` and three `CANCEL_*` clientbound packets; consumer for `PENDING_CHANGE_*` events; pink-text warning of FR-4.7. |
| **atlas-character** | Owns `character_pending_changes`: entity, processor, REST, producers/consumers; applies the name change and the world column update; emits `NAME_CHANGED` and new `WORLD_CHANGED`; expiry sweep for FR-2.6. |
| **atlas-character-factory** | Name validator extracted/shared so FR-3.1 reuses creation rules; creation must additionally honour the soft reservation. |
| **atlas-saga-orchestrator** | New world-transfer saga with compensations (FR-4.8). |
| **atlas-cashshop** | Consumes the 540 asset on request; refunds it on non-application (FR-2.8); backs the purchase-entitlement variant of the same. |
| **atlas-inventory** | Re-homes the character's inventory to the destination world where inventory state is world-scoped; participates as a saga step. |
| **atlas-guilds** | Consumes `NAME_CHANGED` for roster names; severs membership on transfer; blocks transfer for a guild master. |
| **atlas-parties** | Consumes `NAME_CHANGED`; removes the member on transfer. |
| **atlas-buddies** | Consumes `NAME_CHANGED`; removes buddy entries in both directions on transfer. |
| **atlas-trades** | Cancels any open trade before the transfer proceeds; returns held assets. |
| **atlas-merchant** | Closes any hired-merchant shop and returns items/mesos. |
| **atlas-mts** | Cancels/returns auction listings. |
| **atlas-messengers, atlas-marriages, atlas-notes, atlas-rankings, atlas-families** | Audited for denormalised character names/world scoping; wired to `NAME_CHANGED` / `WORLD_CHANGED` where a stale copy would be visible. |
| **atlas-login** | Applies pending changes at the character-select safe point (FR-2.4); character list reflects the post-transfer world. |
| **atlas-configurations** | Handler/writer registration for the new opcodes in every applicable seed template. |
| **libs/atlas-packet** | New codecs for the check and cancel families; `cash/serverbound` purchase codecs unchanged. |
| **libs/atlas-constants** | Any new shared reason-code or slot-item-type constants; reuse existing `ClassificationCharacterImprints`. |
| **atlas-ui** | Pending-changes panel + cancel action on `CharacterDetailPage.tsx` (FR-2.10); new `pendingChanges.service.ts` alongside the existing `characters.service.ts`, typed against the JSON:API envelope; Vitest coverage for the panel and the confirm-dialog cancel path. |

## 8. Non-Functional Requirements

- **Multi-tenancy:** every table, query, Kafka message, and REST call is
  tenant-scoped via `tenant.MustFromContext(ctx)`. No cross-tenant transfer is
  representable.
- **Atomicity:** the world transfer is a saga with per-step compensation. A partial
  failure leaves the character wholly in the source world and refunds the item. A
  character MUST never exist in two worlds, nor in none.
- **Idempotency:** every command handler is idempotent on `transactionId`. Kafka is
  at-least-once and redelivery through a non-idempotent handler has previously
  duplicated items in this codebase — the refund path (FR-2.8) is the highest-risk
  spot and needs an explicit dedupe test.
- **Concurrency:** the name reservation is enforced by a database partial unique
  index, so two simultaneous requests for the same name cannot both succeed. The
  losing request receives a distinct "name taken" reason.
- **Version correctness:** all cash slot item types, opcodes, and dispatcher mode
  bytes are config-resolved per version (DOM-25); no raw `> N` version comparisons —
  use the `MajorAtLeast` idiom.
- **Observability:** each pending-change transition logs at info with tenant,
  character, type, and reason. Saga step failures log the compensating action taken.
- **Performance:** the availability check is a single indexed lookup and must return
  within the client's cash-shop interaction budget; it is issued interactively per
  keystroke-batch by the client.
- **Security:** the requested world and asset are validated server-side; a client may
  not transfer a character it does not own, nor spend an asset it does not hold. The
  cancel endpoint is operator-only and is never reachable from a game-client packet
  path — no handler may be wired to it.
- **Frontend:** the atlas-ui panel follows the FE-* guidelines — JSON:API-typed
  responses, tenant context from the existing provider (never a hard-coded tenant),
  TanStack React Query for fetch + invalidation, shadcn/ui dialog for the destructive
  confirm, and no `any` in the service layer.

## 9. Open Questions

- **OQ-1 (blocking design, not spec):** the concrete `Cash/0540` item IDs and their
  name-change vs world-transfer split are **unverified** — no WZ tree is mounted in
  this checkout. Resolve against atlas-data (per-version) before implementation; do
  not carry over item IDs from general MapleStory knowledge.
- **OQ-2:** the per-version `CashSlotItemType` values for classification 540 are
  unverified; derive from the client IDB per version, as was done for Vicious Hammer
  and the expiration extenders.
- **OQ-3:** which safe point actually fires first in Atlas's login flow — the
  character-select handler in atlas-login or a logout event from atlas-channel — and
  whether both need to trigger `APPLY_PENDING_CHANGE`. Trace the login path in design.
- **OQ-4:** whether inventory rows are genuinely world-scoped in atlas-inventory or
  purely character-scoped. If purely character-scoped, FR-4.5's inventory step
  collapses to a no-op and the saga shrinks considerably. Verify before planning.
- **OQ-5:** whether `characters` currently contains tenant-wide duplicate names in
  any live/baseline data. FR-3.2 tightening to tenant-wide uniqueness must not break
  existing data; survey before choosing constraint vs query-level check.
- **OQ-6:** whether a destination world running a different client version /
  socket configuration must be excluded from the eligible-world list. Assumed
  out of scope; confirm the tenant model actually permits per-world version
  divergence before relying on that assumption.
- **OQ-7:** the exact reason-code enumerations the client renders for
  `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT` and
  `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` — derive from the IDB during codec
  implementation, per version.
- **OQ-8:** FR-2.9's deferred notification needs a delivery mechanism for a player
  who was offline when their request resolved. A grep for an offline/deferred
  notification pattern across atlas-channel and atlas-notes found nothing, so this is
  likely new machinery — decide in design whether it is a flag on the pending-change
  record drained at login, or reuse of an existing queue.
- **OQ-9:** whether `CWvsContext::OnCancelNameChangeResult` and friends are accepted
  by the client at login/field-enter time or only in a specific UI context. If the
  client ignores them outside the cash shop, FR-2.9's delivery falls back to a
  pink-text world message and the packets serve only the in-cash-shop case. Verify
  against the IDB during codec implementation.

## 10. Acceptance Criteria

Feature behaviour:

- [ ] Using a Cash/0540 name-change item creates a `PENDING` name-change request,
      consumes the item, and reserves the name.
- [ ] Using a Cash/0540 world-transfer item creates a `PENDING` world-transfer
      request and consumes the item.
- [ ] `BUY_NAME_CHANGE` (46) and `BUY_WORLD_TRANSFER` (49) create the same requests
      as the item path and are no longer log-only.
- [ ] The availability check returns `taken` for an existing name in **any** world of
      the tenant and for a name held by an active pending reservation.
- [ ] A second pending request of the same type for one character is rejected with a
      distinct reason (enforced by the partial unique index, proven by a test).
- [ ] An operator can see a character's pending and resolved changes on
      `CharacterDetailPage` in atlas-ui, including the rejection reason.
- [ ] An operator can cancel a `PENDING` request from atlas-ui behind a confirmation
      dialog; the panel reflects `CANCELLED` without a manual reload.
- [ ] Cancelling a pending request refunds the item exactly once, including under a
      redelivered cancel event.
- [ ] An operator cancellation reaches the player as `CANCEL_NAME_CHANGE_RESULT` /
      `CANCEL_TRANSFER_WORLD_RESULT` while they are online, and is deferred to their
      next login when they are offline.
- [ ] No serverbound handler is wired to the cancel endpoint — cancel is reachable
      only from the operator REST path (§4.2.1).
- [ ] A pending name change applies at the next safe point; the character's name
      changes, `NAME_CHANGED` is emitted, and guild/party/buddy name copies update.
- [ ] A name change whose name was taken in the interim resolves to `REJECTED`,
      refunds, and sends `CANCEL_NAME_CHANGE_BY_OTHER`.
- [ ] A pending request past `expires_at` resolves to `EXPIRED` and refunds.
- [ ] A world transfer moves the character record and inventory to the destination
      world; the character appears in the destination world's character list and not
      the source's.
- [ ] Guild membership, party membership, and buddy entries are severed by the
      transfer; a guild **master** is blocked with a distinct reason.
- [ ] Open trades, merchant listings, and MTS listings are settled with assets
      returned before the transfer proceeds; a transfer with unreturnable state fails
      cleanly.
- [ ] Storage is untouched by the transfer, and the last-character-in-world case
      emits the pink-text warning.
- [ ] An injected mid-saga failure leaves the character wholly in the source world
      with the item refunded — verified by a test, not by inspection.

Packet coverage:

- [ ] `NAME_TRANSFER`, `WORLD_TRANSFER`, `CASHSHOP_CHECK_NAME_CHANGE`,
      `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT`,
      `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT`, `CANCEL_NAME_CHANGE_RESULT`,
      `CANCEL_TRANSFER_WORLD_RESULT`, and `CANCEL_NAME_CHANGE_BY_OTHER` are `✅` in
      `docs/packets/audits/STATUS.md` for **every** column not marked `⬜ n-a`, each
      with a pinned evidence record and a byte-fixture test.
- [ ] A `coverage-manifest.yaml` for task-227 declares exactly those op × version
      cells, and `packet-completeness-critic` reports no CHANGED-BUT-UNCLAIMED or
      CLAIMED-BUT-UNVERIFIED findings.
- [ ] All new handlers/writers are registered in every applicable seed template.

Build & verification gates (per CLAUDE.md):

- [ ] `go test -race ./...` and `go vet ./...` clean in every changed module.
- [ ] `docker buildx bake atlas-<svc>` clean for every service whose `go.mod` changed.
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
      `tools/template-opcode-order-guard.sh`,
      `tools/template-duplicate-binding-guard.sh`,
      `tools/template-movement-types-guard.sh`, and `tools/skill-job-id-guard.sh`
      all clean from the repo root.
- [ ] `npm run build` (which type-checks tests) and `npm run test` clean in
      `services/atlas-ui` — a passing Vitest run alone is not sufficient verification
      for this repo's UI changes.
- [ ] Code review (`superpowers:requesting-code-review`) run before the PR —
      including `frontend-guidelines-reviewer`, since atlas-ui TS files changed.
