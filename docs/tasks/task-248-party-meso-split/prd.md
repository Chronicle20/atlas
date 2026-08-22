# Party Split of Picked-Up Meso — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-21
---

## 1. Overview

When a character picks up a meso drop today, Atlas credits the **entire** amount to the picker.
The award path is `services/atlas-character/atlas.com/character/character/processor.go:928`
(`AttemptMesoPickUp`), driven by the `RESERVED` drop-status event consumed at
`services/atlas-character/atlas.com/character/kafka/consumer/drop/consumer.go:42-52`. That handler
credits `e.Body.Meso` to `e.Body.CharacterId` and nothing else. `atlas-channel` does resolve the
picker's `partyId` at pickup time (`services/atlas-channel/atlas.com/channel/socket/handler/drop_pick_up.go:22-27`),
but it is used only for drop **ownership** (`drop.Model.CanBeReservedBy`,
`services/atlas-drops/atlas.com/drops/drop/model.go:98`) — it never reaches the credit.

The reference behavior is an even split among party members co-located with the picker. In the
Cosmic reference server, `pickupItem` collects `mpcs = getPartyMembersOnSameMap()` for **any** map
item with `getMeso() > 0` (`<cosmic>/src/main/java/client/Character.java:1988-1991`), then divides:
`int mesosamm = mapitem.getMeso() / mpcs.size();` followed by `partymem.gainMeso(mesosamm, ...)`
per member (`:2013-2023` in the self-lootable-only branch, `:2058-2066` in the normal branch). If
`mpcs` is empty the picker receives the full amount. Eligibility is
`chrMap != null && chrMap.hashCode() == thisMapHash && chr.isLoggedinWorld()`
(`:5518-5538`) — object identity on the map, i.e. same channel **and** same map instance.

This feature moves the meso award from a single-recipient credit to an N-recipient split resolved
in `atlas-drops`, which is the service that already owns the drop's `field`, `ownerPartyId`, and
`playerDrop` provenance (`services/atlas-drops/atlas.com/drops/drop/model.go:23-58`). `atlas-drops`
publishes one `MESO_AWARDED` status event per recipient; `atlas-character` remains the sole owner
of the meso balance and credits each recipient independently.

## 2. Goals

Primary goals:

- When a character in a party picks up a meso drop, every eligible party member co-located with the
  drop receives an equal share of the meso.
- Every recipient — picker and non-picker alike — sees their meso balance update and the standard
  meso-gain notification.
- A character with no party, or whose party lookup cannot be resolved, receives the full amount
  exactly as today (no regression).
- The drop is consumed from the map exactly once, regardless of how many recipients are credited.

Non-goals:

- Party loot rules for **items** and **equipment** (ownership, explosive/free-for-all drops,
  quest-item routing). Only meso drops are in scope.
- Party EXP sharing.
- Meso rate multipliers, party meso bonuses, or any rate service integration
  (`services/atlas-rates` is untouched).
- Changing `CanBeReservedBy` / drop-ownership semantics.
- Level-range or job-based eligibility filters (Cosmic applies none for meso).
- Pet-pickup-specific behavior beyond what the existing reservation path already provides.

## 3. User Stories

- As a party member training on a map, I want mesos my partner loots to be shared with me, so that
  party play is economically equivalent to solo play instead of a race to the drop.
- As the character who picks up the meso, I want to receive my own equal share and the usual
  "+N mesos" notification, so that pickup feedback is unchanged from my perspective.
- As a party member standing on the same map, I want to see my meso total update and a gain
  notification when a partner loots, so that I know sharing is working without checking my inventory.
- As a party member who is offline, on another channel, or in a different map instance, I want to
  **not** receive a share, so that shares reflect actual co-presence.
- As a solo player, I want meso pickup to behave exactly as it does today, so that nothing regresses.

## 4. Functional Requirements

### 4.1 When the split applies

- **FR-1** — The split is evaluated when a drop with `meso > 0` is successfully reserved
  (`ProcessorImpl.Reserve`, `services/atlas-drops/atlas.com/drops/drop/processor.go:135-147`).
  Drops with `meso == 0` are unaffected.
- **FR-2** — The split applies regardless of drop provenance. A drop with `playerDrop == true` is
  split identically to a monster-dropped meso pile. This matches Cosmic
  (`Character.java:1988-1991` has no `isPlayerDrop()` test; the only `isPlayerDrop()` check,
  `:2011`, is a loot-eligibility gate on self-lootable-only maps, not a split gate).
- **FR-3** — A failed reservation (`RESERVATION_FAILURE`) produces no award events. Existing
  behavior is unchanged.

### 4.2 Recipient eligibility

- **FR-4** — If the reserving character has no party (`partyId == 0`), the recipient set is exactly
  the reserving character, receiving the full meso amount.
- **FR-5** — Otherwise, the recipient set is every member of the party whose recorded location
  matches the drop's field on **all four** dimensions — `worldId`, `channelId`, `mapId`, and
  `instance` — and whose `online` flag is `true`. `atlas-parties` exposes exactly these per member
  (`services/atlas-parties/atlas.com/parties/party/rest.go`, member shape mirrored at
  `services/atlas-channel/atlas.com/channel/party/rest.go:141-151`).
- **FR-6** — The reserving character is always in the recipient set, even if their own party-member
  location record is stale or reports offline. This guarantees the recipient set is never empty and
  that a pickup never silently awards nobody.
- **FR-7** — If the party lookup fails (transport error, unknown party, empty member list), the
  system degrades to FR-4: full amount to the reserving character. The failure is logged at error
  level; the pickup itself must not fail.
- **FR-8** — Duplicate character ids in the resolved recipient set are collapsed to one entry before
  the division.

### 4.3 Split arithmetic

- **FR-9** — Each recipient receives `floor(meso / N)` where `N` is the size of the deduplicated
  eligible recipient set. This mirrors Cosmic's integer division
  (`mesosamm = mapitem.getMeso() / mpcs.size()`).
- **FR-10** — The remainder `meso % N` is **discarded** — no recipient receives it. This is
  deliberate Cosmic parity, not an oversight; it must be covered by a test asserting the total
  credited is `N * floor(meso/N)`, not `meso`.
- **FR-11** — If `floor(meso / N) == 0` (the drop is worth less than one meso per member), each
  recipient is credited 0. No award event is emitted for a zero share; the drop is still consumed.

### 4.4 Award delivery

- **FR-12** — `atlas-drops` emits one `MESO_AWARDED` status event per recipient with a non-zero
  share, on `EVENT_TOPIC_DROP_STATUS`, in the same message buffer as the `RESERVED` event so that
  reservation and awards are emitted atomically.
- **FR-13** — Exactly one of the emitted `MESO_AWARDED` events is flagged as originating from the
  picker. Only that event's handler triggers the drop-pickup completion
  (`drop.NewProcessor(...).RequestPickUp(...)`, currently
  `services/atlas-character/.../character/processor.go:948`), so the drop is removed from the map
  exactly once.
- **FR-14** — If the picker's own share is zero (FR-11) but other recipients exist, or if all shares
  are zero, the drop-pickup completion must still fire exactly once. The completion is not
  conditional on a non-zero credit.
- **FR-15** — The existing `RESERVED` handler in `atlas-character`
  (`kafka/consumer/drop/consumer.go:42-52`) must **stop** crediting meso, so that no character is
  double-credited. Its non-meso responsibilities are unchanged.

### 4.5 Crediting and feedback

- **FR-16** — On `MESO_AWARDED`, `atlas-character` credits the named character's meso balance
  transactionally and emits, in the same outbox transaction:
  - a `MESO_CHANGED` status event with `ShowEffect: true` and `Amount` equal to the share, and
  - a `STAT_CHANGED` status event carrying `stat.TypeMeso`.
- **FR-17** — `ShowEffect: true` produces the standard meso-gain chat line for each recipient via
  the existing consumer at
  `services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:452-475`, which
  routes by character id to whichever channel session holds them. No new packet or writer is
  required.
- **FR-18** — Per-recipient failure is isolated. If one recipient's credit fails — most notably the
  `uint32` overflow guard at `processor.go:934-937` (`meso > math.MaxUint32 - c.Meso()`) — that
  recipient is skipped and the failure is logged; all other recipients are still credited and the
  drop is still consumed.
- **FR-19** — Meso credit remains the exclusive responsibility of `atlas-character`. `atlas-drops`
  never mutates a character's balance.

### 4.6 Multi-tenancy and ordering

- **FR-20** — Party resolution and all emitted events are tenant-scoped via the existing context
  tenant propagation; a party in one tenant can never receive a share of another tenant's drop.
- **FR-21** — All `MESO_AWARDED` events for one drop use the drop id as the Kafka partition key
  (matching `producer.CreateKey(int(d.Id()))` used by the existing drop status providers,
  `services/atlas-drops/atlas.com/drops/drop/producer.go:102`), so awards for a single drop are
  ordered relative to that drop's other status events.

## 5. API Surface

No new HTTP endpoints, and no changes to existing REST resources.

**New outbound REST dependency:** `atlas-drops` → `atlas-parties`.

- `GET /parties?filter[memberId]={characterId}` (or the equivalent by-member lookup already used by
  `services/atlas-channel/atlas.com/channel/party/processor.go:74-80` via
  `party.Processor.GetByMemberId`) — returns the party with its `members` JSON:API relationship
  included, each member carrying `worldId`, `channelId`, `mapId`, `instance`, `online`.
- `atlas-drops` currently has outbound REST clients only for `configuration` and `data/foothold`
  (`services/atlas-drops/atlas.com/drops/{configuration,data/foothold}/requests.go`). A new
  `party/` package (`requests.go`, `rest.go`, `processor.go`, `model.go`) is added following that
  established pattern and the shape of `services/atlas-channel/atlas.com/channel/party/`.
- Error cases: any non-2xx, transport error, or unparseable body from `atlas-parties` degrades to
  FR-7 (single-recipient award to the picker) — it never fails the reservation.

**New Kafka contract** — `EVENT_TOPIC_DROP_STATUS`, event type `MESO_AWARDED`:

```go
// services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go
const StatusEventTypeMesoAwarded = "MESO_AWARDED"

type StatusEventMesoAwardedBody struct {
    CharacterId uint32 `json:"characterId"`
    Amount      uint32 `json:"amount"`
    Picker      bool   `json:"picker"`
}
```

Carried in the existing `StatusEvent[E]` envelope, which already supplies `TransactionId`,
`WorldId`, `ChannelId`, `MapId`, `Instance`, and `DropId`
(`services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go`). The mirrored contract in
`services/atlas-character/atlas.com/character/kafka/message/drop/kafka.go` gains the same type
constant and body.

`RESERVED` (`StatusEventReservedBody`) is **unchanged** — no additive fields, no removals — so no
other consumer of that event is affected.

## 6. Data Model

No database schema changes. No new persisted entities, no migrations.

All state used by the split is already in memory or already exposed:

- `drop.Model` (`services/atlas-drops/atlas.com/drops/drop/model.go:23-58`) — supplies `meso`,
  `field` (world/channel/map/instance), `ownerId`, `ownerPartyId`, `playerDrop`. Held in the
  in-memory drop registry; unchanged by this feature.
- Party membership + per-member location and `online` flag — owned by `atlas-parties`
  (`services/atlas-parties/atlas.com/parties/character/model.go:13-23`), read over REST.
- Character meso balance — owned by `atlas-character`, mutated only through the existing
  `dynamicUpdate(tx)(SetMeso(...))` path inside `database.ExecuteTransaction`.

Tenant scoping is via the existing context-propagated tenant on both the REST call and the Kafka
headers; no `tenant_id` column is added anywhere.

## 7. Service Impact

**`atlas-drops`** (primary)

- New `party/` package: read-only client for `atlas-parties` (`GetByMemberId`), following the
  `configuration/` and `data/foothold/` client pattern.
- New pure split function: given the drop's `field`, the meso amount, the reserving character id,
  and the resolved party members, return the deduplicated recipient set with per-recipient amounts
  and the picker flag. Must be unit-testable without Kafka or REST.
- `ProcessorImpl.Reserve` (`drop/processor.go:135-147`): on `meso > 0`, resolve recipients and put
  the `MESO_AWARDED` providers into the same `message.Buffer` as the existing `RESERVED` provider.
- New `mesoAwardedEventStatusProvider` in `drop/producer.go`, keyed on the drop id.
- New service-discovery configuration for the `atlas-parties` base URL, matching how existing
  outbound clients are configured.

**`atlas-character`**

- `kafka/message/drop/kafka.go`: add `StatusEventTypeMesoAwarded` and `MesoAwardedStatusEventBody`
  to the mirrored contract.
- `kafka/consumer/drop/consumer.go`: add a `MESO_AWARDED` handler; **remove** the meso branch from
  `handleDropReservation` (FR-15).
- `character/processor.go`: `AttemptMesoPickUp` is reshaped — it credits the amount from the award
  event to the named character and emits `MESO_CHANGED` (with `ShowEffect: true`) alongside the
  existing `STAT_CHANGED`, and calls `RequestPickUp` only when the award is flagged as the picker's
  (FR-13). Overflow handling per FR-18.

**`atlas-parties`** — no code change; read-only consumer of its existing REST surface.

**`atlas-channel`** — no code change expected. The existing `MESO_CHANGED` consumer
(`kafka/consumer/character/consumer.go:452-475`) already routes the gain notification by character
id to the right channel session. `drop_pick_up.go` keeps resolving `partyId` for ownership.

**`atlas-saga-orchestrator`** — no change. This path is event-driven, not saga-driven.

**Deployment** — `atlas-drops` gains a service-discovery entry for `atlas-parties` in the compose
and k8s manifests under `deploy/`.

## 8. Non-Functional Requirements

- **Latency** — the split adds at most one REST round-trip to `atlas-parties` per meso-drop
  reservation, and only when `meso > 0`. Non-meso drops take no additional call. The lookup must not
  be performed for drops the character cannot reserve.
- **Resilience** — an `atlas-parties` outage must degrade to full-amount-to-picker (FR-7), never
  block or fail a pickup. The client uses the standard `atlas-rest/requests` timeout behavior.
- **Correctness under concurrency** — the reservation registry already serializes reservation of a
  given drop (`services/atlas-drops/atlas.com/drops/drop/registry.go:145-165`), so the split is
  computed at most once per drop. Award events must not be emitted on the
  already-reserved/failure paths.
- **Multi-tenancy** — every party lookup and every emitted event carries the tenant from context;
  cross-tenant leakage is a correctness failure, not a degradation.
- **Observability** — log at debug the recipient count and per-recipient share for each split; log
  at error a party-lookup failure (with the degrade taken) and any per-recipient credit failure
  (with the character id and reason).
- **Backward compatibility** — `RESERVED` is unchanged, so a mixed-version fleet does not
  double-credit: an old `atlas-character` ignores `MESO_AWARDED` and an old `atlas-drops` emits
  none. The one true incompatibility — new `atlas-drops` + old `atlas-character` — under-awards
  rather than over-awards, and must be called out in the rollout note.
- **Determinism in tests** — the split function is pure and table-testable; recipient ordering must
  be deterministic (sort by character id) so fixtures are stable.

## 9. Open Questions

1. **Self-lootable-only maps** — Cosmic has a distinct branch for `MapId.isSelfLootableOnly(mapId)`
   (Happyville trees, guild PQ) at `Character.java:2010-2050` which restricts *who may loot* a
   player drop but still splits the meso. Atlas has no equivalent `isSelfLootableOnly` concept that
   this spec found. Confirm during design whether that gate exists elsewhere in Atlas or is a
   separate absent feature (out of scope here either way, since it is a loot-eligibility rule, not a
   split rule).
2. **Party-member location freshness** — `atlas-parties` tracks member `field` and `online` from
   character lifecycle events. Design should confirm how stale that record can be (e.g. immediately
   after a map change) and whether a member mid-transfer can be wrongly included or excluded. FR-6
   protects the picker only.
3. **Pet pickup** — the reservation carries a `petSlot`, and Cosmic's `pickupItem` takes a
   `petIndex`. Confirm no additional per-recipient behavior is required when a pet performs the
   pickup (expected: none — the split is on the same reservation).
4. **Zero-share drops** — FR-11 credits nobody when `meso < N`. Confirm this is acceptable rather
   than awarding the full amount to the picker as a floor. Cosmic credits `gainMeso(0, ...)`, which
   is effectively the same outcome.

## 10. Acceptance Criteria

Behavior:

- [ ] A solo character picking up a meso drop receives the full amount, exactly as before.
- [ ] A character in a 3-member party, all three on the same map/channel/instance and online,
      picking up a 100-meso drop: each member's balance increases by 33; total credited is 99.
- [ ] Each of the three members sees a meso-gain notification with amount 33 on their own channel
      session.
- [ ] A party member who is offline receives no share and is excluded from `N`.
- [ ] A party member on a different channel receives no share and is excluded from `N`.
- [ ] A party member on a different map receives no share and is excluded from `N`.
- [ ] A party member on the same map but a different `instance` receives no share and is excluded
      from `N`.
- [ ] The picker receives a share even when their own party-member location record is stale.
- [ ] A player-dropped meso pile is split identically to a monster-dropped one.
- [ ] The drop is removed from the map exactly once; no duplicate `PICKED_UP`/pickup completion.
- [ ] With `atlas-parties` unreachable, the picker receives the full amount and the drop is
      consumed; an error is logged.
- [ ] A recipient at the `uint32` meso ceiling is skipped with an error log; the other recipients are
      still credited and the drop is still consumed.
- [ ] No character is double-credited: the `RESERVED` handler no longer awards meso.

Tests:

- [ ] Table-driven unit tests in `atlas-drops` for the pure split function covering: no party,
      party of 1, party of N with mixed eligibility, all four field dimensions as exclusion
      criteria, duplicate member ids, remainder discard, and `meso < N` (zero shares).
- [ ] A test in `atlas-drops` asserting `Reserve` puts exactly one `RESERVED` and the expected set of
      `MESO_AWARDED` providers into the buffer, with exactly one `Picker: true`.
- [ ] A test in `atlas-drops` asserting a failed reservation emits no `MESO_AWARDED`.
- [ ] A test in `atlas-character` asserting the `MESO_AWARDED` handler credits the balance and emits
      both `MESO_CHANGED` (`ShowEffect: true`) and `STAT_CHANGED`.
- [ ] A test in `atlas-character` asserting `RequestPickUp` fires only for the `Picker: true` award.
- [ ] A test in `atlas-character` asserting the overflow path skips one recipient without failing
      the others.
- [ ] All test setup uses the project's Builder pattern; no `*_testhelpers.go`.

Gate:

- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] `backend-guidelines-reviewer` audit clean for `atlas-drops` and `atlas-character`.
- [ ] No Cosmic citations in code comments; behavioral derivations cite IDA/WZ or repo source.
