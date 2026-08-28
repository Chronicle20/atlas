# STOP_PORTION Consume Lock Enforcement — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refuse a standard-consumer item consume in `atlas-consumables` when the character carries an unexpired `STOP_PORTION` buff, and have `atlas-channel` explicitly unstick the client on the resulting `POTION_LOCKED` error.

**Architecture:** One pre-reservation gate inside `ProcessorImpl.RequestItemConsume`, predicated on the existing `usesStandardConsumer(itemId)`. It reads live buffs through the already-injectable `buff.Processor`, fails open on a read error, and on a lock emits the existing `ERROR` event with a new `POTION_LOCKED` type and returns a new `ErrPotionLocked` sentinel — never touching `CancelItemReservation`, because no reservation exists yet. `atlas-channel` gains an explicit branch for that wire value, reached through a newly extracted pure classifier so the routing is unit-testable.

**Tech Stack:** Go 1.27, `atlas-consumables` + `atlas-channel` services, `atlas-kafka` producer/consumer, `logrus`, `testify`, `producertest` capturing producer.

**Spec:** `docs/tasks/task-280-stop-potion-consume-lock/design.md` (PRD: `docs/tasks/task-280-stop-potion-consume-lock/prd.md`)

## Global Constraints

- The stat constant is spelled `TemporaryStatTypeStopPortion` = `"STOP_PORTION"` (`libs/atlas-constants/character/temporary_stat.go:81`). Reference the constant; never write the literal in `atlas-consumables`.
- Buff-read failure is **fail open** (FR-4): log at Warn, treat the character as unlocked, let the consume proceed.
- The gate MUST live only in `atlas-consumables`. No channel-side lock check (FR-1).
- The gate MUST NOT call `CancelItemReservation` — it runs before `RequestReserve` (FR-5).
- Magnitude of the `STOP_PORTION` stat is never consulted; presence is the whole predicate (FR-3).
- Out-of-scope items (anything `usesStandardConsumer` does not match: cash 5xx, morph coupons, town warp, pet food, monster cards, owl scanner) MUST issue **no** buffs read (FR-2).
- The locked-consume log line is **Debug**, never Error (FR-5.4).
- Wire value: `POTION_LOCKED`, additive to the existing `ErrorType*` group. The two `kafka/message/consumable/kafka.go` files are hand-mirrored contracts; add the constant to both, do not introduce a shared package.
- Module roots (the `go build` / `go test` cwd):
  - `services/atlas-consumables/atlas.com/consumables` (module `atlas-consumables`)
  - `services/atlas-channel/atlas.com/channel` (module `atlas-channel`)

---

## Task 1: `IsPotionLocked` predicate

### Files

- `services/atlas-consumables/atlas.com/consumables/character/buff/model.go` — add `IsPotionLocked` immediately below `IsZombified`
- `services/atlas-consumables/atlas.com/consumables/character/buff/model_test.go` — add `TestIsPotionLocked`
- `libs/atlas-constants/character/temporary_stat.go` — read-only; source of `TemporaryStatTypeStopPortion`

Patterns to copy: `services/atlas-consumables/atlas.com/consumables/character/buff/model.go:63-77` (`IsZombified` — identical shape) and `services/atlas-consumables/atlas.com/consumables/character/buff/model_test.go:11-79` (`TestIsZombified` — identical table shape).

Module root: `services/atlas-consumables/atlas.com/consumables`.

**Interfaces:**
- Consumes: `buff.Model` (`Expired()`, `Changes()`), `charconst.TemporaryStatTypeStopPortion`.
- Produces: `func IsPotionLocked(bs []Model) bool` in package `buff` — used by Task 3.

**Do not** generalise `IsZombified` into a shared `HasStat` helper. The design rejects that explicitly (§3.1); `IsZombified` is pinned by task-256 and must not be rewritten.

- [x] **Step 1: Write the failing test**

Append `TestIsPotionLocked` to `character/buff/model_test.go`. Table-driven, same
struct shape and same `for … t.Run` footer as the existing `TestIsZombified`
(copy lines 11-79 and change the stat and the assertion). Imports already
present in the file cover everything needed.

| case name | buffs | want |
|---|---|---|
| `unexpired stop portion` | one buff: changes `{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}`, `createdAt=time.Now()`, `expiresAt=time.Now().Add(time.Minute)`, `noExpiry=false` | `true` |
| `expired stop portion` | same, `expiresAt=time.Now().Add(-time.Second)` | `false` |
| `no-expiry stop portion` | same, `expiresAt=time.Time{}`, `noExpiry=true` | `true` |
| `unexpired non-stop-portion` | one buff: `{Type: charconst.TemporaryStatTypeSpeed, Amount: 20}`, unexpired | `false` |
| `stop portion not first change` | one unexpired buff with changes `[{Speed,20},{StopPortion,1}]` | `true` |
| `empty slice` | `nil` | `false` |
| `expired stop portion alongside unexpired speed` | buff 1 `{StopPortion,1}` expired; buff 2 `{Speed,20}` unexpired | `false` |

```go
func TestIsPotionLocked(t *testing.T) {
	tests := []struct {
		name  string
		buffs []Model
		want  bool
	}{
		{
			name: "unexpired stop portion",
			buffs: []Model{
				NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}}, time.Now(), time.Now().Add(time.Minute), false),
			},
			want: true,
		},
		// ... remaining six rows exactly as tabled above ...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPotionLocked(tt.buffs); got != tt.want {
				t.Fatalf("IsPotionLocked() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

Write all seven rows out in full; the ellipsis above is shorthand for this plan only.

- [x] **Step 2: Run the test to verify it fails**

Run from `services/atlas-consumables/atlas.com/consumables`:

```bash
go test ./character/buff/ -run TestIsPotionLocked -v
```

Expected: build failure — `undefined: IsPotionLocked`.

- [x] **Step 3: Write the implementation**

In `character/buff/model.go`, directly after `IsZombified`:

```go
// IsPotionLocked reports whether bs contains an unexpired buff carrying a
// STOP_PORTION stat change -- the Seal-style debuff that forbids potion use.
// Magnitude is never consulted: the WZ `x` value is a client-side display
// input, so presence is the entire predicate. Slice-level for the same
// reason IsZombified is: every caller already holds the drained list.
// See task-280 FR-3.
func IsPotionLocked(bs []Model) bool {
	for _, b := range bs {
		if b.Expired() {
			continue
		}
		for _, c := range b.changes {
			if c.Type == charconst.TemporaryStatTypeStopPortion {
				return true
			}
		}
	}
	return false
}
```

- [x] **Step 4: Run the tests to verify they pass**

```bash
go test ./character/buff/ -v
```

Expected: PASS, including the untouched `TestIsZombified` and `TestExpiredHonoursNoExpiry`.

- [x] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/character/buff/model.go \
        services/atlas-consumables/atlas.com/consumables/character/buff/model_test.go
git commit -m "feat(consumables): add IsPotionLocked buff predicate"
```

---

## Task 2: `ErrPotionLocked` sentinel and the `POTION_LOCKED` wire value

### Files

- `services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go` — add `ErrorTypePotionLocked` to the `ErrorType*` group (currently lines 118-122)
- `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` — add `ErrPotionLocked` to the `var (...)` block at lines 63-66; add a third `errors.Is` arm to `consumeErrorType` (lines 442-450)
- `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go` — add the classification and event-payload tests

Patterns to copy: `services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go:613-626` (the three existing `TestConsumeErrorType_*` cases) and `services/atlas-consumables/atlas.com/consumables/consumable/producer_reward_test.go:21-29` (provider-emits-one-message shape).

Module root: `services/atlas-consumables/atlas.com/consumables`.

**Interfaces:**
- Produces, for Task 3:
  - `consumable.ErrorTypePotionLocked` (package `atlas-consumables/kafka/message/consumable`) = `"POTION_LOCKED"`
  - `ErrPotionLocked error` (package `consumable`)
  - `consumeErrorType(ErrPotionLocked) == consumable.ErrorTypePotionLocked`

- [x] **Step 1: Write the failing tests**

Append to `consumable/processor_test.go`. The file already imports
`"atlas-consumables/kafka/message/consumable"`, `errors`, `encoding/json`?  — it
does **not** import `encoding/json`; add it. `ts` (`atlas-constants/character`)
and `assert` are already imported.

```go
// task-280 FR-6: the potion-lock rejection must classify to its own wire
// value rather than falling into ErrorTypeConsumeFailed, so the channel can
// route it explicitly instead of via its catch-all.
func TestConsumeErrorType_PotionLocked(t *testing.T) {
	assert.Equal(t, consumable.ErrorTypePotionLocked, consumeErrorType(ErrPotionLocked))
	assert.Equal(t, "POTION_LOCKED", consumable.ErrorTypePotionLocked)
}

// The ERROR event carrying POTION_LOCKED must be exactly one message whose
// body.error is the wire value. Asserted through the provider directly, per
// the producer_reward_test.go precedent -- no broker involved.
func TestErrorEventProviderPotionLocked(t *testing.T) {
	msgs, err := ErrorEventProvider(ts.Id(7), consumable.ErrorTypePotionLocked)()
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)

	var e consumable.Event[consumable.ErrorBody]
	assert.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	assert.Equal(t, consumable.EventTypeError, e.Type)
	assert.Equal(t, ts.Id(7), e.CharacterId)
	assert.Equal(t, "POTION_LOCKED", e.Body.Error)
}
```

- [x] **Step 2: Run the tests to verify they fail**

```bash
go test ./consumable/ -run 'TestConsumeErrorType_PotionLocked|TestErrorEventProviderPotionLocked' -v
```

Expected: build failure — `undefined: ErrPotionLocked`, `undefined: consumable.ErrorTypePotionLocked`.

- [x] **Step 3: Write the implementation**

In `kafka/message/consumable/kafka.go`, add to the `ErrorType*` group (keep the
block gofmt-aligned; adding this longer name re-aligns the group):

```go
	ErrorTypePetCannotConsume = "PET_CANNOT_CONSUME"
	ErrorTypePetCannotLearn   = "PET_CANNOT_LEARN"
	ErrorTypeInventoryFull    = "INVENTORY_FULL"
	ErrorTypeVegaInvalid      = "VEGA_INVALID"
	ErrorTypeConsumeFailed    = "CONSUME_FAILED"
	// ErrorTypePotionLocked is emitted when a consume is refused before
	// reservation because the character carries an unexpired STOP_PORTION
	// buff. atlas-channel routes it to an unstick with no client message.
	// See task-280 FR-6.
	ErrorTypePotionLocked = "POTION_LOCKED"
```

In `consumable/processor.go`, extend the sentinel block:

```go
var (
	ErrPetCannotConsume = errors.New("pet cannot consume")
	ErrPetCannotLearn   = errors.New("pet cannot learn")
	// ErrPotionLocked is returned by RequestItemConsume when a
	// standard-consumer item is refused because the character carries an
	// unexpired STOP_PORTION buff. See task-280.
	ErrPotionLocked = errors.New("potion use locked")
)
```

and add the arm to `consumeErrorType`, above the default return:

```go
	if errors.Is(err, ErrPotionLocked) {
		return consumable.ErrorTypePotionLocked
	}
	return consumable.ErrorTypeConsumeFailed
```

- [x] **Step 4: Run the tests to verify they pass**

```bash
go build ./... && go test ./consumable/ -run 'TestConsumeErrorType|TestErrorEventProvider' -v
```

Expected: PASS, including the pre-existing `TestConsumeErrorType_GenericFailure`, `_PetCannotConsume`, `_PetCannotLearn`.

- [x] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/message/consumable/kafka.go \
        services/atlas-consumables/atlas.com/consumables/consumable/processor.go \
        services/atlas-consumables/atlas.com/consumables/consumable/processor_test.go
git commit -m "feat(consumables): add ErrPotionLocked sentinel and POTION_LOCKED wire value"
```

---

## Task 3: The pre-reservation gate in `RequestItemConsume`

### Files

- `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` — add the `bp` field to `ProcessorImpl` (struct at lines 86-93, constructor at 95-105); add `resolvePotionLocked` next to `resolveZombified` (line 176); add the gate inside `RequestItemConsume` (line 300); add `rejectPotionLocked`
- `services/atlas-consumables/atlas.com/consumables/consumable/processor_potion_lock_test.go` — **new file**
- `services/atlas-consumables/atlas.com/consumables/character/buff/mock/processor.go` — read-only; `buffmock.ProcessorMock` with `GetByCharacterIdFunc`
- `services/atlas-consumables/atlas.com/consumables/compartment/mock/processor.go` — read-only; `compmock.ProcessorMock` with `RequestReserveFunc`

Patterns to copy: `services/atlas-consumables/atlas.com/consumables/consumable/processor.go:176-183` (`resolveZombified` — the fail-open shape to mirror) and `services/atlas-consumables/atlas.com/consumables/consumable/morph_coupon_test.go:1-33` (import block and mock-wiring style for this package).

Module root: `services/atlas-consumables/atlas.com/consumables`.

**Interfaces:**
- Consumes: `buff.IsPotionLocked` (Task 1); `ErrPotionLocked`, `consumeErrorType`, `consumable.ErrorTypePotionLocked` (Task 2); the existing `ErrorEventProvider(character.Id, string)` at `consumable/producer.go:18`.
- Produces: `ProcessorImpl.bp buff.Processor`; `func resolvePotionLocked(l logrus.FieldLogger, bp buff.Processor, characterId uint32) bool`; `func (p *ProcessorImpl) rejectPotionLocked(characterId uint32, itemId item2.Id) error`.

Notes the implementer needs:

- The `consumable` test package installs a capturing Kafka producer once in
  `consumable/testmain_test.go` (`emitted = producertest.InstallCapturing()`),
  so emissions succeed and are inspectable. Call `emitted.Reset()` at the top
  of any test that asserts on `emitted`.
- `consumer.GetManager().RegisterHandler(...)` inside `RequestItemConsume`
  returns an error for an unregistered topic and the code already discards it —
  no broker or consumer setup is needed for these tests.
- Do **not** move `ApplyConsumableEffect` / `CancelConsumableEffect` /
  `ApplyItemEffects` onto the new `bp` field. They keep constructing their own
  `buff.Processor` (design §3.2).
- Item ids for the table, from `item2.GetClassification(id) == id/10000`:
  `2000000` → 200 (standard, in scope), `2210000` → 221 (transformation, in
  scope), `2030000` → 203 (town warp, out of scope), `2120000` → 212 (pet food,
  out of scope), `2380000` → 238 (monster card, out of scope).

- [x] **Step 1: Write the failing test**

Create `consumable/processor_potion_lock_test.go`, package `consumable`.
Construct the processor directly as `&ProcessorImpl{l: …, ctx: context.Background(), bp: …, cpp: …}`
— all four are unexported fields in the same package, so no constructor or
test-only helper is needed. Import shape mirrors `morph_coupon_test.go`:
`context`, `testing`, `time`, `errors`, `buffmock "atlas-consumables/character/buff/mock"`,
`compmock "atlas-consumables/compartment/mock"`, `"atlas-consumables/character/buff"`,
`"atlas-consumables/character/buff/stat"`, `"atlas-consumables/kafka/message/consumable"`,
`"github.com/sirupsen/logrus"`, `"github.com/sirupsen/logrus/hooks/test"`,
`"github.com/stretchr/testify/assert"`, `charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"`,
`"github.com/Chronicle20/atlas/libs/atlas-constants/channel"`,
`inventory2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"`,
`item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"`,
`"github.com/google/uuid"`, `"encoding/json"`.

Build the locked buff with the package's own exported constructor:

```go
lockedBuffs := []buff.Model{
	buff.NewBuff(1, 1, 0, []stat.Model{{Type: charconst.TemporaryStatTypeStopPortion, Amount: 1}}, time.Now(), time.Now().Add(time.Minute), false),
}
```

Five test functions:

**`TestRequestItemConsume_LockedInScopeRejects`** — `bp` returns `lockedBuffs`;
`cpp.RequestReserveFunc` sets `reserved = true` and returns nil. Subtests over
in-scope ids:

| subtest | itemId | classification | expect return | expect reserve |
|---|---|---|---|---|
| `standard potion` | `2000000` | 200 | `ErrPotionLocked` (via `errors.Is`) | not called |
| `transformation potion` | `2210000` | 221 | `ErrPotionLocked` | not called |

Also assert, once, that `emitted.Messages(consumable.EnvEventTopic)` holds
exactly one message whose JSON unmarshals to `Type == "ERROR"` and
`Body.Error == "POTION_LOCKED"` (call `emitted.Reset()` first;
`consumable.EnvEventTopic` is `"EVENT_TOPIC_CONSUMABLE_STATUS"`, which is also
the capture key since the env var is unset in tests).

Call shape for every case:

```go
err := p.RequestItemConsume(channel.Model{}, 555, 3, item2.Id(tt.itemId), 1, 0)
```

**`TestRequestItemConsume_OutOfScopeIssuesNoBuffRead`** — FR-2. `bp.GetByCharacterIdFunc`
sets `read = true` and returns `lockedBuffs`. Subtests:

| subtest | itemId | classification | expect `read` |
|---|---|---|---|
| `town warp scroll` | `2030000` | 203 | `false` |
| `pet food` | `2120000` | 212 | `false` |
| `monster card` | `2380000` | 238 | `false` |

Assert `read == false` in each. Do not assert on the return value: these ids
take their own consumer branches and reserve normally.

**`TestRequestItemConsume_UnlockedInScopeReserves`** — `bp` returns an empty
`[]buff.Model{}`; `cpp.RequestReserveFunc` records its arguments. Item
`2000000`, slot `3`, quantity `1`, characterId `555`. Assert: `err == nil`;
`RequestReserveFunc` fired exactly once; `it == inventory2.TypeValueUse`;
`reserves` is one `compartment.Reserves{Slot: 3, ItemId: 2000000, Quantity: 1}`;
`expiry == 30*time.Second`.

**`TestRequestItemConsume_BuffReadErrorFailsOpen`** — FR-4.
`bp.GetByCharacterIdFunc` returns `(nil, errors.New("buffs down"))`. Item
`2000000`. Assert: `err == nil`; `RequestReserveFunc` fired; and via a
`test.NewNullLogger()` hook that exactly one entry is at `logrus.WarnLevel`
(pattern: `morph_coupon_test.go:440`). Assert the Warn message mentions the
character id.

**`TestResolvePotionLocked`** — direct unit coverage of the free function with
`buffmock.ProcessorMock`:

| subtest | `GetByCharacterIdFunc` returns | want |
|---|---|---|
| `locked` | `lockedBuffs, nil` | `true` |
| `unlocked` | `[]buff.Model{}, nil` | `false` |
| `read error` | `nil, errors.New("boom")` | `false` |

- [x] **Step 2: Run the test to verify it fails**

```bash
go test ./consumable/ -run 'PotionLock|RequestItemConsume_' -v
```

Expected: build failure — `unknown field bp in struct literal`, `undefined: resolvePotionLocked`.

- [x] **Step 3: Write the implementation**

Add the field and its construction:

```go
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  character.Processor
	ip  inventory.Processor
	cpp compartment.Processor
	cdp consumable3.Processor
	bp  buff.Processor
}
```

```go
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		cp:  character.NewProcessor(l, ctx),
		ip:  inventory.NewProcessor(l, ctx),
		cpp: compartment.NewProcessor(l, ctx),
		cdp: consumable3.NewProcessor(l, ctx),
		bp:  buff.NewProcessor(l, ctx),
	}
	return p
}
```

Add `resolvePotionLocked` immediately after `resolveZombified`:

```go
// resolvePotionLocked reports whether the character currently carries an
// unexpired STOP_PORTION buff. Takes bp as a parameter rather than reading
// p.bp so the fail-open branch is testable with the buff mock alone -- the
// same reason resolveZombified is shaped this way.
//
// A read failure is fail-open (task-280 FR-4): during an atlas-buffs outage
// the lock goes unenforced rather than denying every potion. GetByCharacterId
// already normalizes 404 to the empty slice, so "character has no buffs yet"
// never reaches the Warn branch.
func resolvePotionLocked(l logrus.FieldLogger, bp buff.Processor, characterId uint32) bool {
	bs, err := bp.GetByCharacterId(characterId)
	if err != nil {
		l.WithError(err).Warnf("Unable to read buffs for character [%d]; treating potion use as unlocked.", characterId)
		return false
	}
	return buff.IsPotionLocked(bs)
}
```

Insert the gate in `RequestItemConsume`, between the inventory-type check and
the `var itemConsumer ItemConsumer` declaration:

```go
	it, ok := inventory2.TypeFromItemId(itemId)
	if !ok {
		return errors.New("invalid item id")
	}

	// task-280: refuse a standard-consumer item while STOP_PORTION is active.
	// Placed above RequestReserve/RegisterHandler deliberately -- there is no
	// reservation to cancel on this path, so the rejection never has to issue
	// a spurious CancelItemReservation against a transaction atlas-compartment
	// never saw (FR-5). Short-circuit order is the FR-2 guarantee that
	// out-of-scope items issue no buffs read at all.
	if usesStandardConsumer(itemId) && resolvePotionLocked(p.l, p.bp, characterId) {
		return p.rejectPotionLocked(characterId, itemId)
	}

	var itemConsumer ItemConsumer
```

Add `rejectPotionLocked` directly below `RequestItemConsume`:

```go
// rejectPotionLocked emits the ERROR event that releases the client's item-use
// exclusive request and returns the sentinel. Debug, not Error: a locked
// consume is an expected in-game condition (task-280 FR-5.4). It deliberately
// does not route through ConsumeError, whose first act is
// cpp.CancelItemReservation -- no reservation exists on this path.
//
// The wire value is derived via consumeErrorType so sentinel and wire value
// have exactly one binding and cannot drift (FR-6).
func (p *ProcessorImpl) rejectPotionLocked(characterId uint32, itemId item2.Id) error {
	p.l.Debugf("Character [%d] attempted to consume item [%d] while potion use is locked. Rejecting.", characterId, itemId)
	err := producer.ProviderImpl(p.l)(p.ctx)(consumable.EnvEventTopic)(ErrorEventProvider(ts.Id(characterId), consumeErrorType(ErrPotionLocked)))
	if err != nil {
		p.l.WithError(err).Errorf("Unable to issue potion-locked error on event topic. Character [%d] likely going to be stuck.", characterId)
	}
	return ErrPotionLocked
}
```

- [x] **Step 4: Run the tests to verify they pass**

```bash
go build ./... && go test ./consumable/... ./character/... -v
```

Expected: PASS. The pre-existing zombify cases in `processor_test.go`,
`morph_coupon_test.go`, and every other suite in the package must pass
untouched — the `bp` field addition is purely additive.

- [x] **Step 5: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/consumable/processor.go \
        services/atlas-consumables/atlas.com/consumables/consumable/processor_potion_lock_test.go
git commit -m "feat(consumables): gate standard consumes on STOP_PORTION before reservation"
```

---

## Task 4: `atlas-channel` explicit `POTION_LOCKED` routing

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go` — add `ErrorTypePotionLocked` to the `ErrorType*` group (currently lines 90-92)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go` — extract `consumableErrorAction`; rewrite `handleErrorConsumableEvent` (lines 94-152) to switch on it
- `services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer_test.go` — **new file**

Patterns to copy: `services/atlas-channel/atlas.com/channel/kafka/consumer/cashshop/consumer_failure_routing_test.go:1-60` (the precedent for pinning an extracted pure routing classifier with a table).

Module root: `services/atlas-channel/atlas.com/channel`.

**Interfaces:**
- Consumes: `consumable2.ErrorTypePotionLocked` — this task adds the channel-side mirror of the constant Task 2 added on the producer side. Both must read `"POTION_LOCKED"`.
- Produces: `type errorAction int`; `func consumableErrorAction(errorType string) errorAction` in package `consumable` (channel side).

Behavioural constraint: this is a **pure extraction plus one new arm**. Every
existing error type must keep its exact current effect — `PET_CANNOT_CONSUME`
→ pet cash-food error packet; `INVENTORY_FULL` → status message then
`StatChanged`; `VEGA_INVALID` → `VegaScrollInvalid` then `StatChanged`;
everything else → bare `StatChanged` unstick. The new `POTION_LOCKED` arm
performs the bare `StatChanged` unstick and sends **no** client message (PRD
Q3). The channel-side lock check is explicitly forbidden — this task adds
routing only, never a buff read.

The channel `kafka.go` has no `ErrorTypeConsumeFailed` constant today; do not
add one. The test row for `CONSUME_FAILED` uses the string literal.

- [x] **Step 1: Write the failing test**

Create `kafka/consumer/consumable/consumer_test.go`, package `consumable`.
Imports: `"testing"` and `consumable2 "atlas-channel/kafka/message/consumable"`.

```go
// TestConsumableErrorAction pins the ERROR-event routing table.
// task-280 FR-7: POTION_LOCKED must be RECOGNIZED -- it may not be served by
// the catch-all, whose response is free to change when a future error type
// needs a different default. The remaining rows are regression pins: the
// extraction of consumableErrorAction out of handleErrorConsumableEvent must
// not move any existing type to a different arm.
func TestConsumableErrorAction(t *testing.T) {
	tests := []struct {
		name      string
		errorType string
		want      errorAction
	}{
		{"pet cannot consume", consumable2.ErrorTypePetCannotConsume, actionPetCashFoodError},
		{"inventory full", consumable2.ErrorTypeInventoryFull, actionInventoryFull},
		{"vega invalid", consumable2.ErrorTypeVegaInvalid, actionVegaInvalid},
		{"potion locked", consumable2.ErrorTypePotionLocked, actionUnstick},
		{"consume failed falls through", "CONSUME_FAILED", actionUnstick},
		{"empty falls through", "", actionUnstick},
		{"unrecognized falls through", "SOMETHING_ELSE", actionUnstick},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := consumableErrorAction(tt.errorType); got != tt.want {
				t.Fatalf("consumableErrorAction(%q) = %v, want %v", tt.errorType, got, tt.want)
			}
		})
	}
}

// The wire value must match atlas-consumables' hand-mirrored copy exactly;
// a typo in either would silently route POTION_LOCKED to the catch-all.
func TestPotionLockedWireValue(t *testing.T) {
	if consumable2.ErrorTypePotionLocked != "POTION_LOCKED" {
		t.Errorf("ErrorTypePotionLocked = %q, want \"POTION_LOCKED\"", consumable2.ErrorTypePotionLocked)
	}
}
```

Note that `actionPotionLocked` is deliberately **not** a distinct action value:
the response is the same unstick, and inventing a fourth action with an
identical body would be dead weight. Recognition is proven by the explicit
`case` in the classifier, which the next step adds.

- [x] **Step 2: Run the test to verify it fails**

```bash
go test ./kafka/consumer/consumable/ -v
```

Expected: build failure — `undefined: errorAction`, `undefined: consumableErrorAction`, `undefined: consumable2.ErrorTypePotionLocked`.

- [x] **Step 3: Write the implementation**

In `kafka/message/consumable/kafka.go`, add to the `ErrorType*` group:

```go
	ErrorTypePetCannotConsume = "PET_CANNOT_CONSUME"
	ErrorTypeInventoryFull    = "INVENTORY_FULL"
	ErrorTypeVegaInvalid      = "VEGA_INVALID"
	// ErrorTypePotionLocked is atlas-consumables' pre-reservation refusal of a
	// consume while STOP_PORTION is active. Hand-mirrored from
	// services/atlas-consumables/.../kafka/message/consumable/kafka.go; the
	// two spellings must agree. See task-280.
	ErrorTypePotionLocked = "POTION_LOCKED"
```

In `kafka/consumer/consumable/consumer.go`, add the classifier just above
`handleErrorConsumableEvent`:

```go
// errorAction is the client-facing reaction to a consumable ERROR event.
// Extracted from handleErrorConsumableEvent's if-chain so the routing is
// unit-testable without a server.Model, a writer.Producer, and the session
// registry -- which is what lets a test prove POTION_LOCKED is recognized
// rather than merely reaching the catch-all (task-280 FR-7).
type errorAction int

const (
	// actionUnstick sends StatChanged with an empty update list, releasing the
	// item-use exclusive-request lock and nothing else. This is the response
	// for POTION_LOCKED (no client message, PRD Q3), for CONSUME_FAILED, and
	// for any unrecognized type.
	actionUnstick errorAction = iota
	actionPetCashFoodError
	actionInventoryFull
	actionVegaInvalid
)

func consumableErrorAction(errorType string) errorAction {
	switch errorType {
	case consumable2.ErrorTypePetCannotConsume:
		return actionPetCashFoodError
	case consumable2.ErrorTypeInventoryFull:
		return actionInventoryFull
	case consumable2.ErrorTypeVegaInvalid:
		return actionVegaInvalid
	case consumable2.ErrorTypePotionLocked:
		// Explicit rather than left to the default: the default's response is
		// free to change when a future error type needs a different one, and
		// POTION_LOCKED must not silently inherit it.
		return actionUnstick
	default:
		return actionUnstick
	}
}
```

Rewrite `handleErrorConsumableEvent`'s body below the tenant guard, keeping
each arm's existing statements verbatim:

```go
		sp := session.NewProcessor(l, ctx)
		unstick := session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)

		var err error
		switch consumableErrorAction(e.Body.Error) {
		case actionPetCashFoodError:
			err = sp.IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), session.Announce(l)(ctx)(wp)(petpkt.PetCashFoodResultWriter)(petpkt.NewPetCashFoodResultError().Encode))
		case actionInventoryFull:
			err = sp.IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
				if aerr := session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageDropPickUpInventoryFullBody())(s); aerr != nil {
					return aerr
				}
				return unstick(s)
			})
		case actionVegaInvalid:
			// INVALID (0x42 on both verified versions) closes the dialog with
			// the client's own "This item cannot be used." notice -- required,
			// since the dialog is excl-request-blocked after sending; then
			// enable-actions.
			err = sp.IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), func(s session.Model) error {
				if verr := session.Announce(l)(ctx)(wp)(cashpkt.VegaScrollWriter)(cashpkt.VegaScrollInvalidBody())(s); verr != nil {
					return verr
				}
				return unstick(s)
			})
		default:
			// POTION_LOCKED, CONSUME_FAILED, and any unrecognized type: the
			// minimum bar is re-enabling client actions so the item-use
			// exclusive-request lock doesn't stay stuck.
			err = sp.IfPresentByCharacterId(sc.Channel())(uint32(e.CharacterId), unstick)
		}
		if err != nil {
			l.WithError(err).Errorf("Unable to process error event for character [%d].", e.CharacterId)
		}
```

- [x] **Step 4: Run the tests to verify they pass**

```bash
go build ./... && go test ./kafka/consumer/consumable/... -v
```

Expected: PASS on all nine cases.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/consumable/kafka.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer_test.go
git commit -m "feat(channel): route POTION_LOCKED consumable errors through an explicit unstick arm"
```

---

## Task 5: Repo-wide verification

### Files

- `tools/verify.sh` — read-only; the gate

No source changes. This task exists so the plan ends on a green flagless gate
rather than on a module-local `go test`.

- [x] **Step 1: Run the full gate**

```bash
tools/verify.sh
```

Expected: exit 0. `--quick` / `--no-docker` do NOT satisfy this — they skip the
bake and `-race`.

- [x] **Step 2: If it fails**

Fix inside the owning task's files, re-run the module-local tests for that
service, then re-run the flagless gate. Do not amend a passing commit to
silence an unrelated failure; report it instead.

- [x] **Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix(task-280): address verification findings"
```

Skip this step if the gate was green on the first run.
