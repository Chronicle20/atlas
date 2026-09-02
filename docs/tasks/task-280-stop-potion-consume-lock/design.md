# STOP_PORTION Consume Lock Enforcement — Design

Version: v1
Status: Draft
Created: 2026-08-28
Inputs: `docs/tasks/task-280-stop-potion-consume-lock/prd.md` (approved)

---

## 1. Summary

One pre-reservation gate in `atlas-consumables`
`ProcessorImpl.RequestItemConsume`, predicated on `usesStandardConsumer(itemId)`,
that reads the character's live buffs and refuses the consume when an unexpired
`STOP_PORTION` stat change is present. Rejection returns a new `ErrPotionLocked`
sentinel and emits an `ERROR` event carrying `POTION_LOCKED`; `atlas-channel`
gains an explicit branch for that value that unsticks the client.

No new packet, no new endpoint, no persisted state. The change is four files in
`atlas-consumables` and two in `atlas-channel`, plus tests.

---

## 2. Architecture

### 2.1 Decision flow

```
REQUEST_ITEM_CONSUME (channel: item use / pet auto-pot / pet food / cash use)
  └─ Kafka COMMAND → atlas-consumables kafka/consumer/consumable/consumer.go
       └─ ProcessorImpl.RequestItemConsume
            ├─ inventory type resolution                  (unchanged)
            ├─ ► GATE: usesStandardConsumer(itemId)?      (NEW)
            │     └─ yes → resolvePotionLocked(l, bp, characterId)
            │               ├─ read error → Warn, false   (fail open, FR-4)
            │               ├─ locked     → reject        (FR-5)
            │               └─ not locked → fall through
            └─ classification dispatch → RegisterHandler → RequestReserve
                                                            (unchanged)
```

The gate sits between the inventory-type validity check and the classification
dispatch. Two things follow from that placement:

- **Validity precedes authority.** An unresolvable item id still returns
  `invalid item id`, not `POTION_LOCKED`. A locked character sending garbage
  gets the same answer they get today.
- **No reservation exists to cancel.** `RequestReserve` and
  `RegisterHandler` are both downstream of the gate, so the rejection path never
  touches `CancelItemReservation` (FR-5). This is the whole reason the gate is
  placed here rather than inside `ConsumeStandard`.

`transactionId := uuid.New()` and the `topic.EnvProvider` lookup happen above the
gate today. Both are side-effect-free locals; the rejected request simply
abandons an unused uuid. Hoisting them below the gate would be a larger diff for
no behavioural gain, so they stay where they are.

### 2.2 Why one gate covers every request shape

Every channel-side entrypoint — `character_item_use.go`, `pet_item_use.go`
(auto-pot, line 162), `pet_food.go`, `character_cash_item_use.go`,
`shopscanner/processor.go`, `skill/handler/common.go` — funnels into the same
`consumable.Processor.RequestItemConsume`, which produces the single
`REQUEST_ITEM_CONSUME` command consumed at
`kafka/consumer/consumable/consumer.go:68`. There is no second path into
`ProcessorImpl.RequestItemConsume`. The gate is therefore complete by
construction, and FR-1's "MUST NOT be duplicated in atlas-channel" holds without
any channel-side check. This trace satisfies the pet auto-pot acceptance
criterion; it is restated in §7.

### 2.3 Why the cash path needs no explicit exclusion

PRD §2 puts the cash-shop item-use path out of scope. It falls out of FR-2's
predicate rather than needing a carve-out: `usesStandardConsumer` matches only
classifications 200, 201, 202, 205 and `ClassificationConsumableTransformation`
(221) — all USE-inventory. Cash items (5xx, including the 530 morph coupon
routed by `routesToMorphCoupon`) and the owl scanner item (231) never satisfy
the predicate, so they never reach `resolvePotionLocked` and never issue a buffs
read. The scope boundary and the "no network call on paths the gate does not
govern" requirement are the same line of code.

---

## 3. Component design

### 3.1 `character/buff/model.go` — `IsPotionLocked`

A slice-level predicate mirroring `IsZombified` exactly: skip expired buffs,
scan `Changes()` for `charconst.TemporaryStatTypeStopPortion`, return on first
hit. Same `Expired()` semantics, so a `noExpiry` buff counts as active.

Magnitude is not consulted (FR-3) — the WZ `x` value is a client-side display
input, and presence is the entire predicate.

**Alternative rejected:** a generalised `HasStat(bs []Model, t TemporaryStatType) bool`
with `IsZombified`/`IsPotionLocked` as thin wrappers. Tempting, but it would
rewrite `IsZombified` — a function task-256 pinned with its own tests and
comment — to buy nothing at two call sites. Deferred until a third predicate
appears (`STOP_MOTION`, PRD Q4, is the likely trigger).

### 3.2 `consumable/processor.go` — dependency injection

`ProcessorImpl` gains a `bp buff.Processor` field alongside `cp`/`ip`/`cpp`/`cdp`,
constructed in `NewProcessor` via `buff.NewProcessor(l, ctx)`. This matches the
existing field shape and is what makes the gate exercisable from a test that
builds `&ProcessorImpl{...}` directly with `character/buff/mock` and
`compartment/mock`.

**Alternative rejected:** a `potionGateDeps` struct in the style of
`morphCouponDeps` (`morph_coupon.go`). That pattern earns its keep in
`consumeMorphCoupon`, which is a free function with five collaborators. The gate
has one collaborator and lives on a method that already owns four injected
processors; a second injection idiom in the same file is noise.

No existing call site changes. `ApplyConsumableEffect`, `CancelConsumableEffect`
and `ApplyItemEffects` continue to construct their own `buff.Processor` — moving
them onto the new field is a refactor this task does not need and PRD FR-8
argues against.

### 3.3 `consumable/processor.go` — `resolvePotionLocked`

A package-level free function with the same signature shape as
`resolveZombified`:

```go
func resolvePotionLocked(l logrus.FieldLogger, bp buff.Processor, characterId uint32) bool
```

Read error → `l.WithError(err).Warnf(...)` with the character id, return `false`
(FR-4). Success → `buff.IsPotionLocked(bs)`. A successful read of zero buffs is
`false`, not a failure — `GetByCharacterId` already normalises 404 to the empty
slice (`character/buff/processor.go`), so the "no buffs yet" case never reaches
the Warn branch.

Taking `bp` as a parameter rather than reading `p.bp` inside is deliberate and
copied from `resolveZombified`: it makes the fail-open branch directly testable
with the existing mock and no `ProcessorImpl` at all.

### 3.4 `consumable/processor.go` — the gate and the rejection

```go
if usesStandardConsumer(itemId) && resolvePotionLocked(p.l, p.bp, characterId) {
    return p.rejectPotionLocked(characterId, itemId)
}
```

Short-circuit evaluation is the FR-2 "no buffs read for out-of-scope items"
guarantee; the test asserts it by failing if the mock's `GetByCharacterIdFunc`
fires.

`rejectPotionLocked` logs at Debug (FR-5.4 — a locked consume is an expected
in-game condition, never an Error), emits the `ERROR` event through the existing
`ErrorEventProvider`, and returns `ErrPotionLocked`. It deliberately does **not**
route through `ConsumeError`, whose first act is
`cpp.CancelItemReservation` — there is no reservation, and calling it would
produce a spurious cancel against a transaction id `atlas-compartment` never
saw.

The emitted wire value is obtained as `consumeErrorType(ErrPotionLocked)` rather
than by naming `consumable.ErrorTypePotionLocked` directly. One binding between
sentinel and wire value, so FR-6's mapping cannot drift from the emission site.

`ErrPotionLocked` joins `ErrPetCannotConsume`/`ErrPetCannotLearn` in the
existing `var (...)` block; `consumeErrorType` gains a third `errors.Is` arm
above the `ErrorTypeConsumeFailed` default.

### 3.5 Kafka constants

`ErrorTypePotionLocked = "POTION_LOCKED"` is added to the `ErrorType*` group in
both `services/atlas-consumables/.../kafka/message/consumable/kafka.go` and
`services/atlas-channel/.../kafka/message/consumable/kafka.go`. The two files are
hand-mirrored contracts today (the channel copy carries a subset); this follows
that convention rather than introducing a shared package, which would be a
cross-service refactor well outside this task.

Additive and backward compatible: no producer emits the value before this
change, and a channel that predates it lands in its existing unrecognised-type
branch — which already unsticks (§3.6).

### 3.6 `atlas-channel` — error handling

`handleErrorConsumableEvent` gains an explicit branch:

```go
if e.Body.Error == consumable2.ErrorTypePotionLocked {
    // unstick only: StatChanged with an empty update list, excl-request released
}
```

**Honest note on what this buys.** The existing fall-through at the bottom of
`handleErrorConsumableEvent` already performs exactly this `StatChanged` unstick
for any unrecognised error type. Behaviourally, on the day this lands, an
explicit branch and no branch at all are indistinguishable. The branch is added
anyway for two reasons: PRD FR-7 requires the type not to be served by the
fall-through, and the fall-through is a catch-all whose response is free to
change when some future error type needs a different default — at which point
`POTION_LOCKED` would silently inherit it. The explicit branch pins the
intended response.

No client-visible message (PRD Q3). A legitimate client suppresses the use
locally and never reaches this path; a modified one gets nothing but its lock
released.

---

## 4. Resolved open questions

**Q2 — can the lock gate and `resolveZombified` share one buffs read? No, and
not for a reason design can fix.** The two reads are in different process
invocations separated by a Kafka round trip:

```
RequestItemConsume  ──► RequestReserve ──► [Kafka: compartment status]
                                              └─► ConsumeStandard ──► [Kafka: saga]
                                                     └─► ApplyConsumableEffect
                                                            └─► ApplyItemEffects
                                                                   └─► resolveZombified   (processor.go:273)
```

`resolveZombified` runs only after the reservation has been committed
downstream; the gate must run before the reservation is created. There is no
call site where both are in scope, so there is nothing to share. Per PRD §8 the
duplication is accepted and recorded here: an in-scope consume costs **two**
paginated `GET {BUFFS}/characters/{id}/buffs` reads across its full lifecycle
instead of one. Both are already on the hot potion path today for the zombify
half; this doubles a read that was already there rather than introducing a new
class of call. Caching buff state in `atlas-consumables` to collapse them would
introduce a staleness window on exactly the authority boundary this task exists
to close, and is rejected.

**Q1 — item scope** is carried forward as specified (FR-2). No IDA or WZ
evidence was produced during design either; the scope remains a product
decision, and §2.3 records that the classification predicate is the single point
to revisit if evidence later contradicts it.

**Q3 / Q4** are unchanged: no client message, `STOP_MOTION` out of scope.

---

## 5. Files changed

| File | Change |
|---|---|
| `services/atlas-consumables/.../character/buff/model.go` | `IsPotionLocked` |
| `services/atlas-consumables/.../character/buff/model_test.go` | predicate cases |
| `services/atlas-consumables/.../consumable/processor.go` | `bp` field; `ErrPotionLocked`; `resolvePotionLocked`; gate in `RequestItemConsume`; `rejectPotionLocked`; `consumeErrorType` arm |
| `services/atlas-consumables/.../kafka/message/consumable/kafka.go` | `ErrorTypePotionLocked` |
| `services/atlas-consumables/.../consumable/processor_potion_lock_test.go` | new (§6) |
| `services/atlas-channel/.../kafka/message/consumable/kafka.go` | `ErrorTypePotionLocked` |
| `services/atlas-channel/.../kafka/consumer/consumable/consumer.go` | `POTION_LOCKED` branch |
| `services/atlas-channel/.../kafka/consumer/consumable/consumer_test.go` | new (§6) |

Untouched: `libs/atlas-packet`, `libs/atlas-constants`, `atlas-buffs`,
`atlas-monsters`, `atlas-ui`, `socket/handler/pet_item_use.go`, `consumables/cash/`.

---

## 6. Test strategy

Three seams, each hermetic — no Kafka broker, no live `atlas-buffs`.

**Predicate** (`character/buff/model_test.go`) — table-driven, mirroring
`TestIsZombified`: unexpired `STOP_PORTION` → true; expired → false; `noExpiry`
→ true; unrelated stat → false; empty slice → false; `STOP_PORTION` alongside
other changes in one buff → true.

**Gate** (`consumable/processor_potion_lock_test.go`) — builds
`&ProcessorImpl{l: logger, ctx: ctx, bp: &buffmock.ProcessorMock{...}, cpp: &cmpmock.ProcessorMock{...}}`
and calls `RequestItemConsume`:

- in-scope item (e.g. 2000000, classification 200) + locked → returns
  `ErrPotionLocked`; `RequestReserveFunc` never fired.
- out-of-scope item (town-warp scroll, pet food, monster card) + locked →
  `GetByCharacterIdFunc` never fired. Asserted with a bool the mock sets.
- in-scope + unlocked → `RequestReserveFunc` fired once with the expected
  transaction shape (inventory type USE, slot, item id, quantity).
- `GetByCharacterIdFunc` returns an error → proceeds; `RequestReserveFunc`
  fired. Warn is asserted via a `logrus/hooks/test` hook.

The locked path calls the Kafka producer. With no broker configured the emission
errors, is logged, and `ErrPotionLocked` is still returned — the test asserts the
return value and the absence of the reserve, not the emission. The emission
itself is pinned separately, below.

**Event payload** — `ErrorEventProvider(ts.Id(c), consumable.ErrorTypePotionLocked)()`
is invoked directly and its single message asserted to carry
`type == "ERROR"` and `body.error == "POTION_LOCKED"`, following the
`producer_reward_test.go` precedent. Paired with a
`consumeErrorType(ErrPotionLocked) == ErrorTypePotionLocked` case (plus
regression cases for the two pet errors and the default) this covers the
"exactly one ERROR event with `POTION_LOCKED`" criterion without a broker.

**Channel routing** (`kafka/consumer/consumable/consumer_test.go`, new file) —
`handleErrorConsumableEvent` is not directly unit-testable: it needs a
`server.Model`, a `writer.Producer` and the session registry. The branch
decision is extracted into a pure classifier:

```go
type errorAction int
const (actionUnstick errorAction = iota; actionPetCashFoodError; actionInventoryFull; actionVegaInvalid)
func consumableErrorAction(errorType string) errorAction
```

`handleErrorConsumableEvent` switches on its result; the test tables
`PET_CANNOT_CONSUME`, `INVENTORY_FULL`, `VEGA_INVALID`, `POTION_LOCKED`,
`CONSUME_FAILED`, and `""`. This is the smallest refactor that lets a test prove
`POTION_LOCKED` is *recognised* rather than reaching the catch-all — which is
precisely the FR-7 assertion, and which no test of the current
if-chain-with-inline-effects could make.

**Regression** — the existing zombify cases in `consumable/processor_test.go`
and `morph_coupon_test.go` must pass untouched (PRD acceptance). The `bp` field
addition is additive; nothing they exercise changes.

---

## 7. Pet auto-pot trace (acceptance evidence)

`socket/handler/pet_item_use.go:162` calls
`consumable.NewProcessor(l, ctx).RequestItemConsume(...)` — the *channel-side*
processor, whose only body is
`producer.ProviderImpl(...)(consumable2.EnvCommandTopic)(RequestItemConsumeCommandProvider(...))`
(`channel/consumable/processor.go:43-50`). That command is consumed at
`consumables/kafka/consumer/consumable/consumer.go:68`, which calls the gated
`ProcessorImpl.RequestItemConsume`. A repo-wide grep for `RequestItemConsume`
shows no other implementation and no path that bypasses the command. The auto-pot
path is therefore gated transitively, with no second code path to gate.

---

## 8. Risks

- **Fail-open is a deliberate hole.** During an `atlas-buffs` outage the lock is
  unenforced. PRD FR-4 chose availability over authority here; the alternative —
  denying every potion when buffs is down — is a far worse outage. Recorded, not
  mitigated.
- **Added latency on the hot potion path.** One extra `atlas-buffs` read per
  in-scope consume, on the path players hit most often in combat. Fail-open with
  no retry bounds the worst case to one timeout, but a slow (not down) buffs
  service will slow every potion use. This is the cost of a server-authoritative
  check and there is no cheaper form of it without caching (rejected, §4).
- **`STOP_PORTION` spelling.** The constant carries the historical R
  (`"STOP_PORTION"`). It is referenced, never re-spelled, and never written as a
  literal in `atlas-consumables` (FR-3).
- **Contract mirroring.** The `ErrorType*` constant now exists in two
  hand-maintained copies. Nothing enforces they agree; a typo in either would
  silently route `POTION_LOCKED` to the channel fall-through. The channel test
  table pins the channel-side spelling.
