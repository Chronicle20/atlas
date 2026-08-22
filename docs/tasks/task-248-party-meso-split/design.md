# Party Split of Picked-Up Meso — Design

Version: v1
Status: Draft
Created: 2026-08-21
Task: task-248-party-meso-split (renumbered from task-245 to resolve a task-number collision)
PRD: [prd.md](./prd.md)

---

## 1. Scope of this document

The PRD fixes the behavior. This document fixes **where the split is computed**, **how the award
reaches each character**, **how the code is shaped so the arithmetic is unit-testable**, and
**which PRD ambiguities resolve which way**. It does not enumerate implementation steps — that is
`plan.md`.

Three PRD requirements are internally inconsistent or under-specified as written. §7 resolves each
and states the resolution as binding. The rest of the PRD is taken as-is.

---

## 2. Architecture decision: where the split is resolved

The split needs four inputs: the drop's `field` (world/channel/map/instance), the drop's `meso`
amount, the reserving character id, and the party roster with each member's current location and
`online` flag. Three services could assemble them.

### Option A — resolve in `atlas-drops` (chosen)

`atlas-drops` owns the drop registry and is the only service that holds `field`, `meso`, and the
reservation outcome in one place (`drop/model.go:23-58`, `drop/processor.go:135-147`). It already
serializes reservation per drop through the registry (`drop/registry.go:148-167`), so the split is
computed at most once per drop by construction. It gains one new outbound REST dependency
(`atlas-parties`) and one new status event type.

- **For** — single evaluation point guaranteed by the existing reservation lock; the service that
  decides "this pickup succeeded" is the service that decides "and here is who gets paid"; the
  `RESERVED` contract is untouched, so no other consumer is disturbed.
- **Against** — a new service-to-service edge (`atlas-drops` → `atlas-parties`) on the pickup hot
  path, and drops must learn a domain concept (party membership) it has so far only passed through
  as an opaque `ownerPartyId`.

### Option B — resolve in `atlas-channel`

`atlas-channel` already resolves the picker's `partyId` at pickup time
(`socket/handler/drop_pick_up.go:22-27`) and already has a full party client with the exact
co-location filter this feature needs (`party/processor.go:99-104`, `MemberInMap`). It could compute
the recipient set and send it along with the reservation request.

- **For** — zero new service edges; `MemberInMap` is reusable verbatim; the eligibility snapshot is
  taken on the channel that actually holds the sessions, so it is the freshest possible view.
- **Against** — the recipient set would be computed **before** the reservation is known to succeed,
  so a lost race (`RESERVATION_FAILURE`) computes and discards a split, and a retried pickup
  recomputes it. Worse, it makes the channel authoritative over an economic outcome, and the
  recipient list would have to ride the `REQUEST_RESERVATION` command — widening a command contract
  that four services already produce. Rejected primarily on the "compute before knowing the
  reservation won" ordering hazard, which is exactly the double-award class of bug.

### Option C — resolve in `atlas-character`

Keep the `RESERVED` handler as the entry point and have `atlas-character` fan out the split itself.

- **For** — no new event type; the service that credits meso is the service that divides it.
- **Against** — `atlas-character` would need a party client *and* would have to reconstruct the
  drop's field from the event (which today it does incompletely — see §7.3). It also has no natural
  way to guarantee single evaluation: `RESERVED` is a broadcast event, and any redelivery
  re-executes the whole split rather than one idempotent per-recipient credit. Rejected.

**Decision: Option A.** The deciding factor is that the reservation registry already gives us
exactly-once evaluation for free; Options B and C both have to invent it.

The cost of Option A — the new `atlas-drops` → `atlas-parties` edge — is precedented:
`atlas-monster-death` already makes the same call for the same reason (attributing a kill to a
party, `monster/processor.go:59-63`). This design copies that client's shape.

---

## 3. Component design

### 3.1 `atlas-drops/party/` — read-only party client

New package modeled directly on `services/atlas-monster-death/atlas.com/monster/party/`, which is
the minimal read-only variant of the seven party clients already in the repo (channel, doors,
guilds, monster-death, party-quests, query-aggregator, saga-orchestrator).

Files: `requests.go`, `rest.go`, `model.go`, `processor.go`, `mock/processor.go`.

- `requests.go` — `parties?filter[members.id]=%d`, base URL from `requests.RootUrlFor(ctx, "PARTIES")`,
  matching `services/atlas-channel/atlas.com/channel/party/requests.go:13-19`.
- `rest.go` — `RestModel` + `MemberRestModel` with the JSON:API `members` relationship, copied in
  shape from `services/atlas-channel/atlas.com/channel/party/rest.go:17-170`. The member fields this
  feature needs are `worldId`, `channelId`, `mapId`, `instance`, `online`. **Do not** copy `name`,
  `level`, or `jobId` — drops has no use for them, and an unused field is one more thing to keep in
  sync.
- `model.go` — immutable `Model{id, members []MemberModel}` and
  `MemberModel{id, field field.Model, online bool}`. Unlike monster-death's `Model`, which carries
  only `id`, this one must carry the roster.
- `processor.go` — `Processor` interface with a single method `GetByMemberId(memberId uint32) (Model, error)`,
  implemented with `requests.SliceProvider` → `model.FirstProvider`, verbatim from
  `monster/party/processor.go:31-34`.

Only `GetByMemberId` is added. `GetById` is not needed: the reserving character's id is always in
hand, and going by member id avoids trusting the `partyId` the channel supplied on the command.

### 3.2 The pure split function

A free function in `atlas-drops/drop`, with no logger, no context, no REST, no Kafka:

```go
// Recipient is one character's share of a split meso drop.
type Recipient struct {
    CharacterId uint32
    Amount      uint32
    Picker      bool
}

// splitMeso divides meso evenly among the picker and every party member
// co-located with the drop. Returns recipients sorted by character id,
// with exactly one Picker: true.
func splitMeso(f field.Model, meso uint32, pickerId uint32, members []party.MemberModel) []Recipient
```

Contract:

1. Start the eligible set with `pickerId` unconditionally (PRD FR-6).
2. Add each member where `m.Online() && m.Field() == f` on all four dimensions — world, channel,
   map, instance (FR-5). The comparison is the same predicate as
   `services/atlas-channel/atlas.com/channel/party/processor.go:99-104` (`MemberInMap`); it is
   re-expressed here rather than imported, because reaching into another service's package across a
   boundary is disallowed by repo convention and this file is three lines.
3. Deduplicate by character id (FR-8).
4. `share := meso / uint32(len(set))`; the remainder is discarded (FR-9, FR-10).
5. Sort ascending by character id, mark `Picker` on `pickerId` (FR-13, determinism requirement in
   PRD §8).

`members == nil` (no party, or lookup failed) collapses to a one-element result `[{pickerId, meso, true}]`
— FR-4 and FR-7 are the same code path, which is the point: the degrade is not a special case, it is
the empty input.

This function is where the table-driven tests in the PRD's acceptance criteria live. It has no
dependency that needs a fake.

### 3.3 `ProcessorImpl.Reserve` changes

`Reserve` gains a party processor dependency. Rather than constructing one inline (which would make
`Reserve` untestable without a live `atlas-parties`), `drop.ProcessorImpl` adopts the
`ProcessorOption` pattern already used in `services/atlas-mounts/atlas.com/mounts/mount/processor.go:24-60`:

```go
type ProcessorOption func(*ProcessorImpl)
func WithPartyProcessor(pp party.Processor) ProcessorOption
func (p *ProcessorImpl) With(opts ...ProcessorOption) Processor
```

`NewProcessor` defaults the field to `party.NewProcessor(l, ctx)`. Tests inject
`party/mock.ProcessorMock`. Nothing else in `atlas-drops` changes construction.

Flow inside `Reserve`, after a successful `GetRegistry().ReserveDrop(...)`:

```
put RESERVED provider                            (unchanged)
if d.Meso() == 0 -> return                       (FR-1: no lookup for item drops)
members := resolveMembers(characterId)           (nil on any error, logged at error level)
for r := range splitMeso(field, d.Meso(), characterId, members):
    if r.Amount == 0 && !r.Picker: continue      (FR-11, as resolved in §7.1)
    put MESO_AWARDED provider for r
```

The failure branch (`RESERVATION_FAILURE`) is untouched and emits no awards (FR-3).

`resolveMembers` is a thin private method: call `GetByMemberId`, log-and-return-nil on error, return
`m.Members()` otherwise. It never returns an error to the caller — that is FR-7's degrade expressed
in the type signature, so no call site can accidentally fail a pickup on a party outage.

Both providers land in the same `message.Buffer`, so `message.Emit` flushes reservation and awards
as one batch (FR-12).

### 3.4 The `MESO_AWARDED` event

`atlas-drops/kafka/message/drop/kafka.go`:

```go
StatusEventTypeMesoAwarded = "MESO_AWARDED"

type StatusEventMesoAwardedBody struct {
    CharacterId uint32 `json:"characterId"`
    Amount      uint32 `json:"amount"`
    Picker      bool   `json:"picker"`
}
```

Emitted on `EVENT_TOPIC_DROP_STATUS` inside the existing `StatusEvent[E]` envelope, which already
supplies `TransactionId`, the full field, and `DropId`. The provider follows
`reservedEventStatusProvider` (`drop/producer.go:100-122`) exactly, keyed
`producer.CreateKey(int(d.Id()))` (FR-21).

Adding a type to this topic is safe for the four existing consumers (`atlas-channel`,
`atlas-reactors`, `atlas-character`, `atlas-inventory`): every handler on the topic opens with a
`if e.Type != <its type> { return }` guard, and handlers are registered per-type on the same topic
already (`services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:57-72`).

### 3.5 `atlas-character` changes

**Contract mirror** (`kafka/message/drop/kafka.go`) gains `StatusEventTypeMesoAwarded` and
`MesoAwardedStatusEventBody`. It must **also** gain the `TransactionId` and `Instance` fields the
mirror is currently missing relative to the producer — see §7.3.

**Consumer** (`kafka/consumer/drop/consumer.go`) registers a second handler on the same topic for
`MESO_AWARDED`, and `handleDropReservation` loses its `e.Body.Meso > 0` branch entirely (FR-15).
After the change `handleDropReservation` has no remaining body; it is deleted along with its
registration rather than left as an empty guard. `RESERVED` retains no `atlas-character`
responsibility.

**Processor.** `AttemptMesoPickUp` is replaced by a differently-shaped method — the current
signature bundles "credit meso" and "complete the pickup" into one call whose second half is
unreachable when the first half errors, which is precisely what FR-14 and FR-18 forbid. The
replacement:

```go
AwardPickedUpMeso(transactionId uuid.UUID, f field.Model, characterId uint32,
                  dropId uint32, meso uint32, picker bool) error
```

with this ordering:

1. If `meso > 0`, run the credit transaction — overflow guard, `SetMeso`, then in the same outbox
   transaction emit `MESO_CHANGED` (`ShowEffect: true`, `Amount` = the share) and `STAT_CHANGED`
   (`stat.TypeMeso`) (FR-16). A `meso == 0` award skips the transaction entirely.
2. Whatever step 1 returned — success, overflow, character-not-found — if `picker` is true, call
   `drop.NewProcessor(...).RequestPickUp(f, dropId, characterId)` (FR-13, FR-14, FR-18).
3. Return the credit error (if any) for logging. The consumer logs it and does not retry.

The credit failure and the pickup completion are deliberately decoupled; a comment at the call site
records why, in the style of the existing asymmetry note at `character/processor.go:915-920`.

`MESO_CHANGED` needs an `actorId`/`actorType`. The existing `RequestChangeMeso` path takes them from
its caller (`character/processor.go:881`). For a drop pickup the actor is the drop, not a character
or the system; the design uses `actorId = dropId` with `actorType = "DROP"`, a new value in a field
that is already a free-form string (existing values in-repo: `"SYSTEM"`, `"CHARACTER"`, `"ITEM"`).
`atlas-channel`'s `MESO_CHANGED` consumer ignores actor entirely
(`kafka/consumer/character/consumer.go:452-475`), so this is additive.

### 3.6 Notification path — no change

`ShowEffect: true` on `MESO_CHANGED` already produces the "+N mesos" line, routed by character id to
whichever channel session holds that character
(`services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go:468-475`). Since
each recipient gets their own `MESO_CHANGED`, each sees their own amount. No new packet, writer, or
opcode. `atlas-channel` and `atlas-parties` are unmodified by this task.

---

## 4. Data flow

```
channel: drop_pick_up  ──REQUEST_RESERVATION──▶  atlas-drops
                                                    │ ReserveDrop (registry lock)
                                                    │ GET /parties?filter[members.id]=picker  ──▶ atlas-parties
                                                    │ splitMeso(field, meso, picker, members)
                                                    ▼
                              EVENT_TOPIC_DROP_STATUS, key = dropId
                              ├── RESERVED                       (unchanged; no longer credits)
                              ├── MESO_AWARDED {A, share, true}   ──▶ atlas-character
                              ├── MESO_AWARDED {B, share, false}  ──▶ atlas-character
                              └── MESO_AWARDED {C, share, false}  ──▶ atlas-character
                                                                        │ credit + outbox
                                                                        ▼
                                            EVENT_TOPIC_CHARACTER_STATUS
                                            ├── MESO_CHANGED (ShowEffect) ──▶ channel: "+N mesos"
                                            └── STAT_CHANGED

atlas-character (picker's award only) ──REQUEST_PICK_UP──▶ atlas-drops ──PICKED_UP──▶ channel: despawn
```

The drop leaves the map through the existing `Gather` path, driven by exactly one `REQUEST_PICK_UP`
(the picker-flagged award), so the map-side despawn is unchanged and fires once.

---

## 5. Error handling

| Failure | Behavior | Requirement |
|---|---|---|
| `atlas-parties` unreachable / non-2xx / unparseable | `resolveMembers` returns nil → single-recipient full award; error logged | FR-7 |
| Party exists, no co-located members | Recipient set is `{picker}` → full award | FR-4/FR-6 |
| Reservation fails | `RESERVATION_FAILURE` only; no awards | FR-3 |
| One recipient's credit overflows `uint32` | That recipient skipped, error logged; other recipients credited; drop still consumed | FR-18 |
| Picker's own credit fails | Pickup completion still fires; picker gets no meso | FR-14/FR-18 |
| `MESO_AWARDED` redelivered | Recipient credited twice — see §8 | — |

---

## 6. Testing strategy

**`atlas-drops`**

- `splitMeso` table tests: no party; party of 1; party of N all eligible; each of the four field
  dimensions as the sole exclusion; offline member excluded; duplicate member ids collapsed;
  remainder discarded (`100 / 3 → 3 × 33`, total 99); `meso < N` (all shares 0); picker included
  despite a stale/offline own record; result sorted by character id with exactly one `Picker: true`.
- `Reserve` buffer tests using the existing `processor_test.go` harness (miniredis registry +
  tenant context, `processor_test.go:21-40`) plus an injected `party/mock` processor: asserts exactly
  one `RESERVED` and the expected set of `MESO_AWARDED` providers, exactly one `Picker: true`.
- A failed reservation emits `RESERVATION_FAILURE` and zero awards.
- A `meso == 0` drop makes no party call at all (mock asserts zero invocations) — guards the
  hot-path requirement in PRD §8.
- A party-lookup error yields exactly one award, full amount, `Picker: true`.

**`atlas-character`**

- `MESO_AWARDED` handler credits the balance and emits both `MESO_CHANGED` (`ShowEffect: true`,
  correct amount) and `STAT_CHANGED`.
- `RequestPickUp` fires for `Picker: true` and not for `Picker: false`.
- `RequestPickUp` still fires when the picker's credit overflows.
- A zero-amount picker award runs no transaction but still completes the pickup.
- `RESERVED` no longer credits meso (regression guard for FR-15).

All setup uses the project's Builder pattern; no `*_testhelpers.go` (repo convention).

---

## 7. Resolved PRD ambiguities

These are binding for `plan.md`.

### 7.1 FR-11 vs FR-14 — the zero-share picker

FR-11 says no award event is emitted for a zero share. FR-13/FR-14 say the picker's award is what
completes the pickup and that completion must fire even when the share is zero. Taken literally,
a drop with `meso < N` would emit no picker award and the drop would never be removed from the map.

**Resolution:** the picker's `MESO_AWARDED` is emitted **unconditionally** whenever the drop is
reserved and `meso > 0`, including at `Amount: 0`. Zero-amount suppression applies only to
non-picker recipients. `atlas-character` skips the credit transaction on `Amount == 0` but still
completes the pickup. This satisfies FR-11's intent (no pointless credit, no phantom "+0 mesos"
chat line) without breaking FR-14.

### 7.2 FR-18 — credit failure must not block completion

The current `AttemptMesoPickUp` returns `txErr` before reaching `RequestPickUp`
(`character/processor.go:944-949`), so today an overflowing picker leaves the drop on the map
forever. The replacement method inverts this: completion is unconditional on `picker`, and the
credit error is returned only for logging. This is a real behavior fix that FR-18 requires, not a
refactor.

### 7.3 The `atlas-character` drop-event mirror is lossy

`services/atlas-character/atlas.com/character/kafka/message/drop/kafka.go:57-65` declares
`StatusEvent[E]` **without** `TransactionId`, and the consumer therefore fabricates one
(`kafka/consumer/drop/consumer.go:48`, `uuid.New()`), breaking the transaction correlation the
producer establishes. The same handler also builds the field with
`field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).Build()` — dropping `Instance`, which this
feature makes load-bearing (FR-5 compares instance).

**Resolution:** the mirror gains `TransactionId`, and the `MESO_AWARDED` handler builds the field
with `.SetInstance(e.Instance)`. Both are in scope: the split is wrong without instance, and
correlating the award to its reservation is worth the two-line contract fix while we are editing
the same struct.

### 7.4 PRD open questions

1. **Self-lootable-only maps** — confirmed out of scope. It is a loot-*eligibility* rule; Atlas's
   equivalent gate lives in `drop.Model.CanBeReservedBy` (`drop/model.go:98`), which this design does
   not touch. Whether Atlas should grow an `isSelfLootableOnly` concept is a separate feature.
2. **Party-member location freshness** — accepted as-is. `atlas-parties` is the only source of member
   location in the system, and a member mid-map-transfer may be briefly mis-classified in either
   direction. FR-6 guarantees the picker is never the victim of this, so the failure mode is bounded
   to "a partner occasionally misses one share during a map change". Tightening it would require a
   live per-channel presence query, which is a much larger change for a marginal gain. No mitigation.
3. **Pet pickup** — no additional behavior. `petSlot` is carried on the reservation and consumed by
   the existing `PICKED_UP` path; the split is computed from the same reservation regardless of who
   physically picked up. Confirmed by `ReserveDrop`'s signature
   (`drop/registry.go:148`) — `petSlot` affects only the drop model, never the recipient.
4. **Zero-share drops** — confirmed: nobody is credited, matching Cosmic's `gainMeso(0)`. See §7.1
   for how the drop is still consumed.

---

## 8. Known limitation: at-least-once redelivery

`MESO_AWARDED` carries no idempotency key beyond `TransactionId`, and `atlas-character` performs no
dedupe on credit. A Kafka redelivery of one award double-credits that recipient. This is **not a new
exposure** — the existing `RESERVED`-driven credit has exactly the same property today — and making
meso credit idempotent is a service-wide concern (it would also cover `RequestChangeMeso`, sack
consumption, and saga compensation). Explicitly out of scope; recorded here so the plan does not
silently claim a guarantee the design does not provide.

---

## 9. Rollout

Deployment order is **`atlas-character` first, then `atlas-drops`**.

- New `atlas-character` + old `atlas-drops`: the new `MESO_AWARDED` handler is idle and the old
  `RESERVED` credit path is gone — meso pickup awards **nothing** until drops is rolled. This window
  must be short.
- Old `atlas-character` + new `atlas-drops`: `MESO_AWARDED` is ignored and `RESERVED` still credits
  the full amount to the picker — the pre-feature behavior, no double-credit.

The second ordering is strictly safer, so if the rollout can only guarantee one ordering, roll
**`atlas-drops` first**. Neither ordering ever over-awards. Both are transient; the fleet converges
within one deploy cycle.

`atlas-drops` needs an `atlas-parties` service-discovery entry (`RootUrlFor(ctx, "PARTIES")`) in the
compose and k8s manifests under `deploy/` **before** the new `atlas-drops` image is rolled;
otherwise every meso pickup takes the FR-7 degrade and logs an error per drop.

---

## 10. Files

**`atlas-drops`** — `party/{requests,rest,model,processor}.go` + `party/mock/processor.go` (new);
`drop/split.go` + `drop/split_test.go` (new); `drop/processor.go` (option pattern, `Reserve`);
`drop/producer.go` (`mesoAwardedEventStatusProvider`); `kafka/message/drop/kafka.go` (type + body);
`drop/processor_test.go`; `deploy/` manifests.

**`atlas-character`** — `kafka/message/drop/kafka.go` (type, body, `TransactionId`);
`kafka/consumer/drop/consumer.go` (new handler; delete `handleDropReservation`);
`character/processor.go` (`AttemptMesoPickUp` → `AwardPickedUpMeso`) + its mock and tests.

**Unchanged** — `atlas-parties`, `atlas-channel`, `atlas-saga-orchestrator`, `atlas-rates`,
`atlas-reactors`, `atlas-inventory`.

---

## 11. Acceptance

The PRD §10 acceptance criteria stand unmodified, with two additions from §7.1 and §7.2:

- [ ] A drop worth less than one meso per member is still removed from the map, and no recipient
      receives a "+0 mesos" chat line.
- [ ] A picker whose credit overflows still completes the pickup; the drop is removed from the map.
