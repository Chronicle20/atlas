# Cash Name Change & World Transfer — Design

Version: v1
Status: Draft
Created: 2026-08-14
PRD: [`prd.md`](prd.md)

---

## 1. What is already true

Everything in this section was read from source in this worktree, or decompiled from
the GMS v83 IDB (`MapleStory_dump.exe.i64`, session `41f13e0d`) during design. Nothing
here is recalled.

### 1.1 The classifier already splits 540 into two arms, version-scoped

`GetCashSlotItemType` (`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1117-1140`)
already handles `item.ClassificationCharacterImprints`:

| item id prefix | GMS ≥ 95 | everything else |
|---|---|---|
| `5400xxx` | `CashSlotItemType(53)` | `CashSlotItemType(52)` |
| `5401xxx` | `CashSlotItemType(54)` | `CashSlotItemType(53)` |

So **FR-1.2 is already satisfied by existing code** — the split is by item-id prefix and
the enum value is already version-shifted the same way every other 5xx family is. What is
missing is only the *dispatch arm* in `CharacterCashItemUseHandleFunc`, not the
classification.

Two defects in that block, both in scope to fix here:

- **Lines 1132-1138 are dead code**: an exact duplicate of the `5401` branch at 1125-1131.
  A copy-paste that almost certainly should have been a third prefix. The whole
  `ClassificationCharacterImprints` arm must be re-derived from the client's own
  classifier before we build on it (§3.7).
- The version gate is a raw `t.MajorVersion() >= 95` rather than the `MajorAtLeast`
  idiom. It is consistent with its ~40 sibling branches in the same function, so this
  design does **not** rewrite them; new code added by this task uses `MajorAtLeast`.

### 1.2 The purchase serverbound codecs already exist and already decode

`cash_shop_operation.go:184-195` decodes `ShopOperationBuyNameChange`
(`OldName`, `NewName`, `SerialNumber`) and `ShopOperationBuyWorldTransfer`
(`TargetWorld`, `SerialNumber`) in full. Both arms then `l.Infof` and `return`. The
mode bytes 46 / 49 are already config-resolved through `isCashShopOperation` →
`options["operations"]`, so DOM-25 is already honoured on this path.

IDA corroborates the mode byte: v83 `CCashShop::SendBuyTransferWorldItemPacket`
(`@0x473601`) builds `COutPacket(0xE5)` then `Encode1(0x31)` — 0x31 = 49, exactly the
`BUY_WORLD_TRANSFER` constant Atlas already carries.

**Consequence:** no new serverbound codec is needed for the two *purchase* ops. The
work on that path is behavioural only.

### 1.3 Name validation and a name-availability endpoint already exist

`atlas-character` owns both:

- `GET /characters/name-validity?name=&worldId=`
  (`services/atlas-character/atlas.com/character/character/name_validity_resource.go`)
- `ProcessorImpl.CheckNameValidity`
  (`services/atlas-character/atlas.com/character/character/processor.go:268-289`):
  length 3-12, regex `^[A-Za-z0-9぀-ゟ゠-ヿ一-龯]{3,12}$`,
  then `GetForName` and a **per-world** duplicate check (`c.WorldId() == worldId`).
- `getForName` (`.../character/provider.go:30-39`) is already
  `LOWER(name) = LOWER(?)` and already tenant-wide — the per-world narrowing happens in
  the processor, not the query.

`atlas-character-factory` consumes it over REST
(`services/atlas-character-factory/atlas.com/character-factory/character/name_validity_requests.go`).

**Consequence:** FR-3.1's "shared validator" already exists and is already shared. This
task does not extract anything; it adds a *scope* parameter and a reservation check to
the one implementation that is already the single source of truth.

### 1.4 `NAME_CHANGED` is emitted but consumed by nobody

`nameChangedEventProvider` (`.../character/producer.go:261-272`) is wired into the
character PATCH path (`.../character/processor.go:1926`). A repo-wide grep for
`NAME_CHANGED` outside atlas-character returns **zero consumers**.

Persisted denormalised name copies, found by grepping `entity.go` across every service:

| service | column | file |
|---|---|---|
| atlas-guilds | `members.name` | `services/atlas-guilds/atlas.com/guilds/guild/member/entity.go:16` |
| atlas-buddies | `buddy.character_name` | `services/atlas-buddies/atlas.com/buddies/buddy/entity.go:28` |
| atlas-rankings | `ranking.name` | `services/atlas-rankings/atlas.com/rankings/ranking/entity.go:23` |
| atlas-mts | `listing.seller_name` | `services/atlas-mts/atlas.com/mts/listing/entity.go:47` |
| atlas-merchant | `blacklists.name`, `merchant_visits.name` | `.../blacklist/entity.go:17`, `.../visit/entity.go:17` |

atlas-parties, atlas-messengers, atlas-marriages and atlas-trades hold the name only in
**in-memory registries** rebuilt at login (`.../parties/character/registry.go:36`,
`.../messengers/character/registry.go:35`, `.../marriages/character/model.go:5`,
`.../trades/trade/builder.go:55`). Because FR-2.4 forbids applying a rename to a live
character, those caches are always rebuilt *after* the rename lands. **They need no
`NAME_CHANGED` consumer.** This is a large reduction against the PRD's §7 table.

atlas-merchant's two name columns are name-*keyed* by design (a per-shop blacklist and
visitor log). Renaming a character silently orphans its blacklist entry. Decision in
§3.8.

### 1.5 Inventory is character-scoped, not world-scoped

`services/atlas-inventory/atlas.com/inventory/compartment/entity.go:14-20` keys
compartments by `(TenantId, Id, CharacterId, InventoryType)`. `asset/entity.go:22-30`
keys assets by `CompartmentId`. **There is no world column anywhere in atlas-inventory.**

**Consequence (resolves OQ-4):** FR-4.5's "inventory moves with the character" is a
no-op. Changing `characters.world` moves the inventory by construction. The world-transfer
saga loses its largest and riskiest step.

### 1.6 The client's own world-transfer gate — and it blocks on *family*

v83 `CCashShop::CheckTransferWorldPossible` (`@0x4734e5`) is the client-side pre-check.
Decompiled, it refuses with three distinct strings:

- `"Guild Master can not transfer worlds."`
- `"GM can not transfer worlds."`
- `SP_5017_YOU_HAVE_TO_QUIT_FAMILY__R_NTO_MOVE_TO_ANOTHER_WORLD`

The guild-master gate confirms FR-4.3. The GM gate and the **family gate are not in the
PRD at all** — the PRD lists atlas-families only under "audited for denormalised names."
The client itself will refuse to even send the request while the character is in a family,
so a server that permits it produces a state the client considers impossible. §3.6 adds
both as blocking conditions.

### 1.7 The check packets carry the account's second password

v83 `CCashShop::OnBuyNameChange` (`@0x470480`) calls `ask_SPW()` and passes its result
into `SendCheckNameChangePossiblePacket(characterId, spw)`. The availability-check
serverbound bodies therefore carry a **credential**, not just a candidate name.

**Consequence:** the SPW field must never reach a log line, and must be validated against
the account's stored PIC/SPW before the check is answered. See §3.5 and §8.

### 1.8 The IDB symbols in this neighbourhood are not trustworthy

`@0x47359c` is symboled `CCashShop::SendCheckNameChangePossiblePacket` but builds
`COutPacket(18)` — 18 = `0x012`, which `docs/packets/audits/STATUS.md:532` records as
**`WORLD_TRANSFER`** on v83, not `NAME_TRANSFER` (`0x010`, row 528). And `@0x470480`,
symboled `OnBuyNameChange`, calls `CheckTransferWorldPossible` and formats
transfer-world-specific error strings.

Either the two symbols are transposed in this IDB, or the matrix's v83 opcodes are. This
is a **derivation blocker for the whole check-packet family** and is resolved by the
normal `packet-verifier` procedure, not by picking one. See §3.7 and §9-R1. This is the
project's standing "don't infer feature from IDB names" rule biting on live evidence.

### 1.9 Version is a tenant property, not a world property

`libs/atlas-tenant/tenant.go:21-25` exposes `Region()` / `MajorVersion()` /
`MinorVersion()` on the tenant model; worlds are configured beneath a tenant. Two worlds
in one tenant cannot run different client versions.

**Consequence (resolves OQ-6):** no version-divergence exclusion is needed in the
eligible-world list. The non-goal in PRD §2 is vacuous rather than deferred.

---

## 2. Open questions, resolved

| OQ | Resolution | Evidence |
|---|---|---|
| OQ-1 concrete 540 item ids | **Partly resolved, one input still open.** The *split* is `5400xxx` vs `5401xxx` and is already in code. Which prefix is name-change vs world-transfer is NOT resolvable from this checkout (no WZ tree) and is entangled with §1.8's symbol transposition. Resolved by a plan task that queries atlas-data per version for classification-540 templates and reads their String.wz names, cross-checked against the client's own classifier arm. | §1.1 |
| OQ-2 per-version `CashSlotItemType` | **Resolved.** Already implemented and already version-scoped: 52/53 pre-v95, 53/54 on GMS ≥ 95. | `character_cash_item_use.go:1117-1140` |
| OQ-3 which safe point fires first | **Resolved: logout, always.** Both entry points (item use, cash-shop purchase) require a live channel session, so the character is online when the request is created; the next transition is necessarily their logout. Login-time drain is a *catch-up*, not the primary path. | §3.3 |
| OQ-4 inventory world-scoped? | **Resolved: no.** Purely character-scoped. FR-4.5's inventory step collapses to a no-op. | §1.5 |
| OQ-5 tenant-wide duplicate names in live data | **Cannot be surveyed from this checkout** and must not be assumed away. Design chooses a **query-level** tenant-wide check plus a partial unique index on *pending reservations only* — no new constraint on `characters`. This is correct whether or not duplicates exist. | §3.4 |
| OQ-6 per-world version divergence | **Resolved: impossible.** Version lives on the tenant. | §1.9 |
| OQ-7 reason-code enumerations | Deferred to per-cell derivation — this is exactly what `packet-verifier` produces, and inventing them here would be the failure mode the playbook exists to prevent. Design fixes the *server-side* reason taxonomy (§6) and maps it to wire codes at codec time. | §3.7 |
| OQ-8 offline notification delivery | **Resolved: a column, not new machinery.** `notified_at` on the pending-change row; atlas-character consumes its own `LOGIN` status event and re-emits `PENDING_CHANGE_RESOLVED` for any resolved-but-unnotified row. | §3.9 |
| OQ-9 whether the client accepts `CANCEL_*` outside the cash shop | Deferred to codec derivation, with a **designed fallback**: every resolved-notification also carries a pink-text world message (`WorldMessagePinkTextBody`). If the client ignores the packet outside the cash shop, the player still learns why their coupon came back. | §3.9 |

---

## 3. Architecture

### 3.1 Shape of the whole thing

```
  item use (540)  ──┐
                    ├──> atlas-channel: request flow ──REST──> atlas-character
  BUY_NAME_CHANGE ──┤                                            (pending_changes)
  BUY_WORLD_XFER  ──┘                                                  │
                                                                       │ PENDING_CHANGE_CREATED
  NAME_TRANSFER  ────> atlas-channel ──REST──> atlas-character         │ PENDING_CHANGE_RESOLVED
  WORLD_TRANSFER ────>   (availability check, synchronous)             ▼
                                                              atlas-channel (client feedback)

  LOGOUT status event ──> atlas-character applier
                            ├── NAME_CHANGE  : local PATCH  → NAME_CHANGED
                            └── WORLD_XFER   : saga (atlas-saga-orchestrator) → WORLD_CHANGED

  operator ──> atlas-ui ──REST DELETE──> atlas-character (cancel) ──> refund + notify
```

**One request record, two request types.** The alternative — a `character_name_changes`
table and a `character_world_transfers` table — was rejected: the lifecycle (pending →
terminal), the consumption/refund contract, the expiry sweep, the operator cancel
endpoint, and the notification delivery are *identical* between the two. Two tables means
two of everything for one differing column. The discriminator column costs one `type`
check at apply time.

### 3.2 Where the record lives: atlas-character

Alternatives considered:

- **atlas-cashshop.** It owns the consumed asset and the purchase entitlement, so refund
  would be local. Rejected: the *apply* step mutates `characters.name` / `characters.world`,
  the availability check is a `characters` query, and the reservation must be visible to
  character creation — three cross-service round trips to save one.
- **A new atlas-imprints service.** Rejected outright: `docs/adding-a-new-service.md`
  enumerates ~8 hand-maintained registration lists, and the service would own one table
  whose every read joins `characters`.
- **atlas-character (chosen).** The reservation is enforced by a partial unique index in
  the same database as the uniqueness check it guards — the only place that can make
  FR-3.3 and FR-3.2 atomic with respect to each other. Refund becomes one Kafka command
  outward, which we need for idempotency anyway.

### 3.3 The safe point is `LOGOUT`, with a login-time catch-up

Both entry points require an established channel session, so a pending request is always
created while the character is online. `logoutEventProvider`
(`services/atlas-character/atlas.com/character/character/producer.go:66-79`) already
publishes `LOGOUT` on `EVENT_TOPIC_CHARACTER_STATUS` with world/channel/map — the same
event ten other services already consume.

atlas-character gains a consumer on its **own** status topic:

- on `LOGOUT`: look up `PENDING` rows for that character, run apply.
- on `LOGIN`: (a) re-emit any resolved-but-unnotified transition (§3.9); (b) re-attempt
  any `PENDING` row whose apply previously failed — a catch-up, so a crashed applier or a
  Kafka gap self-heals rather than stranding the coupon until expiry.

Rejected alternative: hooking `atlas-login`'s `CharacterListWorldHandleFunc`
(`services/atlas-login/atlas.com/login/socket/handler/character_list_world.go`). It fetches
the list over REST and would have to *block* on a multi-service saga before rendering the
character list — a synchronous dependency on a compensating saga at the most
latency-sensitive point in the login flow. LOGOUT is asynchronous and already exists.

Self-consumption of one's own status topic is a new pattern for atlas-character. It is
still preferable to an inbound command from another service, because the applier's trigger
(the character left every channel) is a fact only atlas-character publishes.

### 3.4 Name uniqueness, reservation, and validity

`CheckNameValidity` gains a scope, and a reservation check:

```go
type NameScope string
const (
    NameScopeWorld  NameScope = "WORLD"   // character creation — today's behaviour
    NameScopeTenant NameScope = "TENANT"  // name change (FR-3.2)
)
CheckNameValidity(name string, worldId world.Id, scope NameScope) (NameValidityResult, error)
```

- `NameScopeWorld` keeps the existing `c.WorldId() == worldId` filter byte-for-byte, so
  character creation behaviour is unchanged and atlas-character-factory needs no change
  beyond passing the scope explicitly.
- `NameScopeTenant` drops the world filter — `getForName` is already tenant-wide.
- **Both** scopes additionally reject a name held by a live reservation, so a rename in
  flight blocks a creation of the same name (FR-3.3). This is the one behaviour change
  character creation sees, and it is required.
- New `Reason` values: `"reserved"` alongside the existing `"length"`, `"regex"`,
  `"duplicate"`. The REST response gains `reserved` to satisfy the PRD §5
  `available|taken|reserved|invalid` contract without a second endpoint — the existing
  `/characters/name-validity` endpoint is extended, not duplicated. PRD §5's proposed
  `GET /characters/name-availability` is therefore **not** created; it would be a second
  name for a route that already exists.

**No new index on `characters`.** Adding a tenant-wide unique index would fail the
migration on any tenant that already has a cross-world duplicate (OQ-5, unsurveyable from
here), and the failure mode would be a service that will not start. The tenant-wide check
is a query over an already-lowered-name lookup; the *reservation* index lives on the new
table where we control every row we create.

### 3.5 The availability check is synchronous REST

`NAME_TRANSFER` / `WORLD_TRANSFER` are interactive — the client blocks its dialog on the
result. The handler in atlas-channel calls atlas-character's name-validity endpoint (name
change) or an eligibility endpoint (world transfer) over REST and writes the result
packet in the same handler invocation. No Kafka round trip; a saga here would add hundreds
of milliseconds to a keystroke-batch interaction for no atomicity benefit — nothing is
mutated.

The SPW field carried by these packets (§1.7) is validated against the account before the
check is answered and is **never** placed in a log field. A `//nolint`-free rule: the
decoded struct's `String()` must redact it, since every handler in this codebase logs
`p.String()` at debug.

### 3.6 World-transfer eligibility

Blocking (request refused, nothing consumed):

| condition | source of truth | rationale |
|---|---|---|
| destination is the source world | — | trivially invalid |
| destination world does not exist / is full | atlas-world | FR-4.1 |
| no free character slot in destination | atlas-account character slots | FR-4.1 |
| name taken in destination world | atlas-character | FR-4.2 safety net |
| character is banned | atlas-ban | FR-4.2 |
| character is a **guild master** | atlas-guilds (`members.title == 1`) | client refuses it (§1.6) |
| character is a **GM** (`characters.gm != 0`) | atlas-character | client refuses it (§1.6) |
| character is in a **family** | atlas-families | client refuses it (§1.6) — **added vs PRD** |
| character has an open trade | atlas-trades | assets in escrow |
| character has an open hired merchant | atlas-merchant | assets in escrow |
| character has live MTS listings or bids | atlas-mts | assets in escrow |

The PRD (FR-4.4) says escrow-holding systems should be *settled* by the saga. This design
makes them **blocking** instead, checked at request time and re-checked at apply time:

- Every one of them is a player-visible, player-fixable state ("close your shop first"),
  and every one of them already has a first-class close/cancel flow the player can drive.
- Auto-settling someone's hired merchant or auction listing as a side effect of a world
  transfer destroys value silently and is not reversible by compensation — you cannot
  un-cancel an auction someone else already bid on.
- It removes the three highest-risk compensating steps from the saga.

Severed by the transfer (not blocking): party membership, buddy entries in both
directions, and guild membership for a non-master. Guild membership severance is the one
that genuinely needs compensation, because re-joining a guild is not a client-driveable
recovery.

### 3.7 Packet work

Eight matrix rows, **59 cells** to promote (counted from
`docs/packets/audits/STATUS.md` rows 157, 161, 204, 409, 413, 476, 528, 532):

| row | ❌ cells | versions |
|---|---|---|
| `CANCEL_NAME_CHANGE_RESULT` | 8 | v61…v95 |
| `CANCEL_TRANSFER_WORLD_RESULT` | 8 | v61…v95 |
| `CANCEL_NAME_CHANGE_BY_OTHER` | 7 | v72…v95 |
| `CASHSHOP_CHECK_NAME_CHANGE` | 9 | v48…v95 |
| `CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT` | 10 | v48…v95, jms |
| `CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE_RESULT` | 6 | v79…v95 |
| `NAME_TRANSFER` (serverbound) | 6 | v83…v95, jms |
| `WORLD_TRANSFER` (serverbound) | 5 | v83…v95 |

This follows the standing playbooks verbatim rather than inventing a procedure:
[`docs/packets/IMPLEMENTING_A_PACKET.md`](../../packets/IMPLEMENTING_A_PACKET.md) per op
via the `packet-implementer` agent, and
[`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md)
per cell via `packet-verifier`. A `coverage-manifest.yaml` declaring exactly these 59
cells is written first so `packet-completeness-critic` has something to diff against.

**Ordering constraint, not negotiable:** §1.8's symbol transposition is resolved *before*
any codec is written. The first plan task is a derivation pass that settles, from the IDB
read order rather than the symbol name, which opcode is `NAME_TRANSFER` and which is
`WORLD_TRANSFER` on v83, and reconciles the result with the matrix — amending
`docs/packets/audits/` if the matrix is what is wrong. Writing codecs against a
transposed pair produces two structurally-plausible decoders that are silently swapped,
and byte-fixture tests would pass on both.

The same pass re-derives the whole `ClassificationCharacterImprints` classifier arm (§1.1),
which fixes the dead duplicate at lines 1132-1138 and settles OQ-1's name-change /
world-transfer prefix assignment from the client rather than from item names.

`CASHSHOP_CHECK_NAME_CHANGE` (row 409) lists three distinct receivers including
`OnCheckDuplicatedIDResult` — treat it as a shared-opcode family and check
[`docs/packets/DISPATCHER_FAMILY.md`](../../packets/DISPATCHER_FAMILY.md) applicability
during derivation rather than assuming a single body.

All new handlers and writers are registered in every applicable template under
`services/atlas-configurations/seed-data/templates/` at their sorted `opCode` position,
passing `tools/template-opcode-order-guard.sh` and
`tools/template-duplicate-binding-guard.sh`. Mode bytes go through the `operations` table.

### 3.8 What consumes `NAME_CHANGED`

New consumers, one per persisted copy (§1.4): **atlas-guilds**, **atlas-buddies**,
**atlas-rankings**, **atlas-mts**. Each is a narrow `UPDATE … SET name = ? WHERE
character_id = ?`, idempotent by construction.

**atlas-merchant is explicitly excluded**, and this is a decision rather than an omission:
its `blacklists.name` and `merchant_visits.name` are name-keyed rows, not name-carrying
rows. Rewriting a blacklist entry on rename is a behaviour change to a moderation feature
(does a blocked player escape their block by renaming? — that is a product question), and
`merchant_visits` is a log. Both are out of scope; §10 records it so it is not mistaken
for something we missed.

In-memory registries (parties, messengers, marriages, trades) get **no** consumer, per
§1.4's reasoning. This is load-bearing on FR-2.4 — if the offline-only constraint is ever
relaxed, those four services become required consumers.

### 3.9 Notification delivery, including to an offline player

Two columns on the record carry it: `resolved_at` and `notified_at`.

- Transition to a non-`APPLIED` terminal state → emit `PENDING_CHANGE_RESOLVED`.
- atlas-channel consumes it. If the character has a live session, it writes the
  version-appropriate `CANCEL_*` packet **and** a `WorldMessagePinkTextBody`
  (`services/atlas-channel/atlas.com/channel/socket/writer/world_message.go:107`) —
  belt and braces against OQ-9, where the client may ignore the `CANCEL_*` packet outside
  the cash-shop UI. It then acks so `notified_at` is stamped.
- If there is no live session, nothing is stamped.
- On the next `LOGIN`, atlas-character re-emits `PENDING_CHANGE_RESOLVED` for every row
  with `resolved_at IS NOT NULL AND notified_at IS NULL`.

Rejected: a generic deferred-notification queue. Nothing else in the codebase needs one
today (a grep across atlas-channel and atlas-notes finds no such pattern), and building
one to serve a single caller is the wrong ratio.

### 3.10 Consumption and refund

Consumption is at request acceptance; refund on every non-`APPLIED` exit (FR-2.8).

- **Item path:** the 540 asset is destroyed via the existing `DestroyAsset` saga action;
  refund is `AwardAsset` of the same template into the CASH compartment. Both carry the
  record's `transaction_id`.
- **Purchase path:** the cash-shop entitlement is consumed by atlas-cashshop; refund
  credits it back.

Idempotency is the highest-risk surface in this design (a redelivered cancel that mints a
second coupon is precisely the known at-least-once failure mode in this codebase). It is
enforced at the record, not at the handler: the refund is emitted **only** by the state
transition that writes `status` away from `PENDING`, inside the same transaction, through
the outbox. A redelivered cancel command finds `status != 'PENDING'`, transitions nothing,
and emits nothing. This is the same shape the existing `message.Emit` + outbox pattern
already gives us in atlas-character.

### 3.11 The world-transfer saga

New saga type + actions in `libs/atlas-saga/model.go`, dispatched by
atlas-saga-orchestrator. With inventory a no-op (§1.5) and escrow blocking rather than
settling (§3.6), the saga is four steps:

| step | service | compensation |
|---|---|---|
| `validate_world_transfer` (re-check every §3.6 gate) | atlas-character | none needed (read-only) |
| `leave_guild_for_transfer` | atlas-guilds | re-add member at prior title |
| `leave_party_for_transfer` | atlas-parties | none — party membership is not restorable and is not durable state |
| `sever_buddies_for_transfer` | atlas-buddies | restore removed entries |
| `change_character_world` | atlas-character | set world back to `source_world_id` |

`change_character_world` is last, so a failure anywhere leaves the character in the source
world with only recoverable severances applied — and each of those has a compensation.

`character_pending_changes.source_world_id` exists precisely so compensation does not have
to reconstruct where the character came from.

### 3.12 atlas-ui

A `PendingChangesPanel` on `services/atlas-ui/src/pages/CharacterDetailPage.tsx`, backed by
a new `services/atlas-ui/src/services/api/pending-changes.service.ts` alongside the ~40
existing per-resource services (naming follows the directory's kebab-case convention, e.g.
`mts-listings.service.ts`).

Read + cancel only, per FR-2.10 — no create, no edit of a requested value. Cancel is behind
a shadcn/ui `AlertDialog` naming the character and the requested value, and invalidates the
React Query key on success. No `any` in the service layer; the JSON:API envelope is typed.

---

## 4. Data model

Table `character_pending_changes`, owned by atlas-character, migrated via the existing
`Migration` / `AutoMigrate` pattern in `services/atlas-character/atlas.com/character/character/entity.go`.

```go
type pendingChangeEntity struct {
    TenantId           uuid.UUID  `gorm:"not null;index:idx_pc_tenant_char_type_pending"`
    Id                 uuid.UUID  `gorm:"primaryKey;type:uuid"`
    CharacterId        uint32     `gorm:"not null;index:idx_pc_tenant_char_type_pending"`
    Type               string     `gorm:"not null;index:idx_pc_tenant_char_type_pending"` // NAME_CHANGE | WORLD_TRANSFER
    Status             string     `gorm:"not null"`  // PENDING|APPLIED|CANCELLED|REJECTED|EXPIRED
    RequestedName      *string
    RequestedNameLower *string    // generated at write time; the reservation key
    DestinationWorldId *world.Id
    SourceWorldId      world.Id   `gorm:"not null"`
    AssetId            *uint32
    Reason             string     `gorm:"not null;default:''"`
    TransactionId      uuid.UUID  `gorm:"not null"`
    CreatedAt          time.Time  `gorm:"not null"`
    ExpiresAt          time.Time  `gorm:"not null"`
    ResolvedAt         *time.Time
    NotifiedAt         *time.Time
}
```

Two partial unique indexes, created by raw DDL in the migration (GORM tags cannot express
a `WHERE` clause):

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_pc_one_pending_per_type
  ON character_pending_changes (tenant_id, character_id, type)
  WHERE status = 'PENDING';

CREATE UNIQUE INDEX IF NOT EXISTS idx_pc_name_reservation
  ON character_pending_changes (tenant_id, requested_name_lower)
  WHERE status = 'PENDING' AND type = 'NAME_CHANGE';
```

The first **is** FR-2.3; the second **is** FR-3.3's soft reservation. Neither is
duplicated in application code — the insert catches the unique-violation and maps it to
the distinct reason. `requested_name_lower` is a stored column rather than a functional
index on `lower(requested_name)` so the reservation lookup and the index agree exactly, and
so the "reserved" check in `CheckNameValidity` is a plain equality join.

Departures from PRD §6: `requested_name_lower`, `notified_at`, and `asset_id` nullable
(the purchase path has an entitlement reference, not an asset id — recorded in
`transaction_id`-correlated cash-shop state, not here).

---

## 5. Flows

### 5.1 Name change, happy path

1. Player uses a `5400xxx`/`5401xxx` coupon (whichever prefix derivation settles on) or
   completes `BUY_NAME_CHANGE`.
2. atlas-channel `POST /characters/{id}/pending-changes` with
   `{type: NAME_CHANGE, requestedName, assetId}`.
3. atlas-character validates (tenant scope + reservation), inserts `PENDING` in one
   transaction, emits `PENDING_CHANGE_CREATED` and the `DestroyAsset` command through the
   outbox. The insert either takes the reservation or violates the unique index — there is
   no window.
4. atlas-channel writes `NAME_CHANGE_BUY_DONE` and unlocks per the `ExclRequest` contract.
5. Player logs out → `LOGOUT`.
6. atlas-character's applier re-validates the name, PATCHes `characters.name` through the
   existing path (which already emits `NAME_CHANGED`), sets `APPLIED` + `resolved_at`,
   releasing the reservation.
7. atlas-guilds / atlas-buddies / atlas-rankings / atlas-mts update their copies.

### 5.2 Name change, lost race

At step 6 the name now belongs to a real character (another pending request cannot have
taken it — the index forbids that; a *creation* could, if it raced between reservation
release and re-check, or if the reservation check was bypassed).
→ `REJECTED`, `reason = "name_taken"`, refund, `CANCEL_NAME_CHANGE_BY_OTHER` (§3.9).

### 5.3 World transfer

Request-time §3.6 gates → `PENDING` → `LOGOUT` → §3.11 saga → `WORLD_CHANGED`.
Any saga step failure → compensations run, `REJECTED`, refund, notify. The character is
never in two worlds and never in none, because `change_character_world` is a single-row
update and is the last step.

### 5.4 Operator cancel

`DELETE /characters/{id}/pending-changes/{pcId}` → transition `PENDING` → `CANCELLED`
(409 if already terminal) → refund + notify, both driven by the transition (§3.10).

No serverbound handler is wired to this route. An explicit test asserts the route appears
in no socket-config template and in no `socket/handler` registration — the PRD lists this
as an acceptance criterion and it deserves a machine check, not an eyeball.

### 5.5 Expiry

A ticker in atlas-character (via `routine.Go`, per `tools/goroutine-guard.sh`) sweeps
`status = 'PENDING' AND expires_at < now()` → `EXPIRED` + refund + notify. Default 7 days,
tenant-configurable through the atlas-tenants configuration resource.

---

## 6. Reason taxonomy

Server-side reasons are a closed set, stored in `reason` and mapped to wire codes at codec
time (OQ-7). They are not free text.

`name_invalid_length` · `name_invalid_charset` · `name_taken` · `name_reserved` ·
`already_pending` · `world_same` · `world_unknown` · `world_full` · `no_character_slot` ·
`banned` · `is_guild_master` · `is_gm` · `in_family` · `trade_open` · `merchant_open` ·
`mts_listings_open` · `operator_cancelled` · `expired` · `saga_failed`

Every rejection path returns one of these (FR-5.1). A path that would return none is a bug,
and the test suite asserts exhaustiveness over the enum.

---

## 7. Testing

- **Reservation race:** two concurrent `POST`s for the same name — exactly one 201, one
  422 `name_reserved`. Against a real Postgres, since the index *is* the mechanism.
- **One-pending-per-type:** second request → 409 `already_pending`.
- **Refund idempotency:** deliver the cancel command twice; assert exactly one asset
  minted. This is the single most important test in the task.
- **Mid-saga failure:** inject a failure at `change_character_world`; assert the character
  is wholly in the source world, guild/buddy severances are compensated, and the coupon is
  back.
- **Offline notification:** resolve while offline, assert `notified_at` null, then emit
  `LOGIN` and assert re-emission.
- **Cancel unreachability:** assert no handler and no template binds the cancel route.
- **Name scope regression:** character creation still uses `NameScopeWorld` and still
  permits a cross-world duplicate; name change does not.
- **Packet:** byte fixtures per cell with `packet-audit:verify` markers, per the playbook.
- **UI:** Vitest over the panel and the confirm-dialog cancel path; `npm run build` (which
  type-checks tests) is part of verification, not `npm run test` alone.

---

## 8. Non-functional

- **Tenancy:** every query carries `tenant_id`; the destination world is resolved from the
  requesting session's tenant, so a cross-tenant transfer is not representable.
- **Security:** the SPW field (§1.7) is validated and never logged — the decoded struct's
  `String()` redacts it. Ownership of the character and of the asset is verified
  server-side; the cancel endpoint is operator-only.
- **Versioning:** new gates use `MajorAtLeast`; mode bytes come from `operations`.
- **Observability:** every transition logs tenant, character, type, from→to status, and
  reason. Saga compensations log what they undid.

---

## 9. Risks

- **R1 — symbol transposition (§1.8).** Highest. Mitigated by making derivation the first
  plan task and by refusing to write a codec until the matrix and the IDB agree.
- **R2 — OQ-1 needs a live atlas-data.** The 540 prefix→feature assignment cannot be
  settled in this checkout. It is a hard prerequisite for the item-use branch and is
  scheduled ahead of it. If atlas-data cannot answer, the client's own classifier arm
  (re-derived under R1) settles it instead — there are two independent routes, so this is
  a scheduling risk, not a blocker.
- **R3 — 59 packet cells** is the bulk of the task by volume and is fan-out-shaped. It
  parallelises across `packet-verifier` per cell, but the plan must not treat it as one
  task.
- **R4 — atlas-character self-consuming its own status topic** is a new pattern here.
  Consumer group naming must not collide with the outbound producer's, and the applier
  must tolerate redelivery (it does — the status transition is the idempotency key).
- **R5 — escrow gates are blocking, not settling (§3.6).** A deliberate departure from
  FR-4.4. If the intent was genuinely auto-settlement, this is the decision to revisit
  before planning.

---

## 10. Explicitly out of scope

- atlas-merchant blacklist / visit-log name rewriting on rename (§3.8) — a moderation
  policy question, not a name-propagation bug.
- Auto-settling trades, hired merchants, and MTS listings (§3.6) — blocked instead.
- `Cash/0543` Maple Life character creation, cross-tenant transfer, storage migration,
  a player-facing cancel surface, and commodity pricing — all per PRD §2.
- Rewriting the ~40 sibling `MajorVersion() >= 95` branches in `GetCashSlotItemType`
  (§1.1). New code uses `MajorAtLeast`; the existing block is not in this task's blast
  radius.
