# Party Split of Picked-Up Meso — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split a picked-up meso drop evenly among the picker and every online party member co-located with the drop, resolved in `atlas-drops` and credited per-recipient by `atlas-character`.

**Architecture:** `atlas-drops` gains a read-only `atlas-parties` REST client and a pure `splitMeso` function. On a successful reservation of a drop with `meso > 0`, `Reserve` resolves the picker's party, computes the recipient set, and puts one new `MESO_AWARDED` status event per recipient into the same `message.Buffer` as the existing `RESERVED` event. `atlas-character` stops crediting meso on `RESERVED` and instead credits each `MESO_AWARDED` recipient in its own transaction, emitting `MESO_CHANGED` (`ShowEffect: true`) + `STAT_CHANGED`, and completes the drop pickup only for the award flagged `Picker: true`.

**Tech Stack:** Go 1.x, JSON:API over `atlas-rest/requests`, Kafka via `atlas-kafka/producer` + the transactional outbox (`atlas-outbox`), GORM, `miniredis` for the drops registry in tests, `producertest` for Kafka capture in tests.

**Spec:** [design.md](./design.md) (PRD: [prd.md](./prd.md))

## Global Constraints

- Module roots: `services/atlas-drops/atlas.com/drops` (module `atlas-drops`) and `services/atlas-character/atlas.com/character` (module `atlas-character`). Run `go build ./... && go test ./...` from the module root of whichever service the task touches. Never run repo-wide verification from an implementer.
- Never reach into another service's package. `atlas-drops` gets its own `party/` package; it does not import `atlas-channel`'s.
- Immutable models with unexported fields and value receivers; construction through a Builder. No `*_testhelpers.go` files.
- No stubs, no `// TODO`, no unimplemented status responses.
- Preserve existing line endings; use repo-relative paths in any committed doc.
- `RESERVED` (`StatusEventReservedBody`) is **unchanged** — no fields added or removed on either side of the mirror.
- Every emitted drop status event is keyed `producer.CreateKey(int(dropId))`.
- Behavioral derivations cite repo source. No Cosmic citations in code comments.

---

## Task 1: `atlas-drops/party` — read-only party client

### Files

- `services/atlas-drops/atlas.com/drops/party/model.go` — **new file**; immutable `Model`/`MemberModel` plus builders
- `services/atlas-drops/atlas.com/drops/party/rest.go` — **new file**; JSON:API `RestModel`/`MemberRestModel` + `Extract`
- `services/atlas-drops/atlas.com/drops/party/requests.go` — **new file**; `parties?filter[members.id]=%d` request
- `services/atlas-drops/atlas.com/drops/party/processor.go` — **new file**; `Processor.GetByMemberId`
- `services/atlas-drops/atlas.com/drops/party/mock/processor.go` — **new file**; `ProcessorMock`
- `services/atlas-drops/atlas.com/drops/party/rest_test.go` — **new file**; `Extract` tests

Patterns to copy:
- `services/atlas-monster-death/atlas.com/monster/party/requests.go` (whole file — same request, same `PARTIES` root)
- `services/atlas-monster-death/atlas.com/monster/party/processor.go:31-34` (the `SliceProvider` → `FirstProvider` body)
- `services/atlas-monster-death/atlas.com/monster/party/mock/processor.go` (whole file)
- `services/atlas-channel/atlas.com/channel/party/rest.go:16-170` (the `members` relationship plumbing and `ExtractMember`)
- `services/atlas-channel/atlas.com/channel/party/model.go` (its `modelBuilder` at the bottom of the file — the builder shape)

Read-only reference (do not edit): `services/atlas-parties/atlas.com/parties/party/rest.go:95-103` — the authoritative member field names on the wire.

Module root: `services/atlas-drops/atlas.com/drops`. `github.com/jtumidanski/api2go`, `github.com/google/uuid`, `atlas-constants`, `atlas-model`, and `atlas-rest` are already in `go.mod`; no dependency changes.

**Interfaces:**
- Produces (consumed by Tasks 2 and 3):
  - `party.Model` with `Id() uint32`, `Members() []party.MemberModel`
  - `party.MemberModel` with `Id() uint32`, `Field() field.Model`, `Online() bool`
  - `party.NewBuilder() *modelBuilder` → `.SetId(uint32)`, `.SetMembers([]MemberModel)`, `.Build() Model`
  - `party.NewMemberBuilder() *memberBuilder` → `.SetId(uint32)`, `.SetField(field.Model)`, `.SetOnline(bool)`, `.Build() MemberModel`
  - `party.Processor` interface with `GetByMemberId(memberId uint32) (Model, error)`; `party.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`
  - `mock.ProcessorMock{GetByMemberIdFunc func(memberId uint32) (party.Model, error)}`

Deliberately **not** carried on `MemberModel`: `name`, `level`, `jobId`, `leaderId`. Drops has no use for them (design §3.1).

- [ ] **Step 1: Write the failing test**

Create `services/atlas-drops/atlas.com/drops/party/rest_test.go`, package `party`.

`TestExtract` — one test function, no table; builds a `RestModel` literal with two `MemberRestModel` entries and asserts the extracted `Model`.

Inputs:

| field | member A | member B |
|---|---|---|
| `Id` | 100 | 200 |
| `WorldId` | `world.Id(1)` | `world.Id(2)` |
| `ChannelId` | `channel.Id(1)` | `channel.Id(3)` |
| `MapId` | `_map.Id(100000000)` | `_map.Id(200000000)` |
| `Instance` | `uuid.MustParse("00000000-0000-0000-0000-000000000001")` | `uuid.Nil` |
| `Online` | `true` | `false` |

`RestModel.Id` is `7`.

Expected, asserted with `require`:

- `m.Id() == uint32(7)`
- `len(m.Members()) == 2`
- `m.Members()[0].Id() == uint32(100)`; `.Online() == true`; `.Field().WorldId() == world.Id(1)`; `.Field().ChannelId() == channel.Id(1)`; `.Field().MapId() == _map.Id(100000000)`; `.Field().Instance() == uuid.MustParse("00000000-0000-0000-0000-000000000001")`
- `m.Members()[1].Id() == uint32(200)`; `.Online() == false`; `.Field().WorldId() == world.Id(2)`; `.Field().ChannelId() == channel.Id(3)`; `.Field().MapId() == _map.Id(200000000)`; `.Field().Instance() == uuid.Nil`

`TestExtract_NoMembers` — a `RestModel` with `Members` nil extracts to a `Model` whose `Members()` is non-nil and has length 0 (the recipient resolution in Task 3 ranges over it directly).

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-drops/atlas.com/drops && go test ./party/... -run TestExtract -v`
Expected: FAIL — the `party` package does not exist / `Extract` undefined.

- [ ] **Step 3: Write `model.go`**

```go
package party

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Model is the subset of a party atlas-drops needs: the roster, with each
// member's location and online flag. Name, level, job, and leadership are
// deliberately absent — the meso split does not read them.
type Model struct {
	id      uint32
	members []MemberModel
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Members() []MemberModel {
	return m.members
}

type MemberModel struct {
	id     uint32
	field  field.Model
	online bool
}

func (m MemberModel) Id() uint32 {
	return m.id
}

func (m MemberModel) Field() field.Model {
	return m.field
}

func (m MemberModel) Online() bool {
	return m.online
}

type modelBuilder struct {
	id      uint32
	members []MemberModel
}

// NewBuilder returns a party model builder. The production path builds
// through Extract over the REST response; this exists for tests and any
// in-process construction.
func NewBuilder() *modelBuilder {
	return &modelBuilder{members: make([]MemberModel, 0)}
}

func (b *modelBuilder) SetId(v uint32) *modelBuilder             { b.id = v; return b }
func (b *modelBuilder) SetMembers(v []MemberModel) *modelBuilder { b.members = v; return b }

func (b *modelBuilder) Build() Model {
	return Model{
		id:      b.id,
		members: b.members,
	}
}

type memberBuilder struct {
	id     uint32
	field  field.Model
	online bool
}

func NewMemberBuilder() *memberBuilder {
	return &memberBuilder{}
}

func (b *memberBuilder) SetId(v uint32) *memberBuilder          { b.id = v; return b }
func (b *memberBuilder) SetField(v field.Model) *memberBuilder   { b.field = v; return b }
func (b *memberBuilder) SetOnline(v bool) *memberBuilder         { b.online = v; return b }

func (b *memberBuilder) Build() MemberModel {
	return MemberModel{
		id:     b.id,
		field:  b.field,
		online: b.online,
	}
}
```

- [ ] **Step 4: Write `rest.go`**

Copy the relationship plumbing verbatim from `services/atlas-channel/atlas.com/channel/party/rest.go:16-170` — `GetName`, `GetID`, `SetID`, `GetReferences`, `GetReferencedIDs`, `GetReferencedStructs`, `SetToManyReferenceIDs`, `SetReferencedStructs`, and the `MemberRestModel` identifier methods — and change only what follows.

`RestModel` drops `LeaderId`:

```go
type RestModel struct {
	Id      uint32            `json:"-"`
	Members []MemberRestModel `json:"-"`
}
```

`MemberRestModel` carries only the five fields the split reads:

```go
type MemberRestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Online    bool       `json:"online"`
}
```

In `SetToManyReferenceIDs`, append `MemberRestModel{Id: uint32(id)}` (the channel copy spells out zero values for fields this model does not have).

```go
func Extract(rm RestModel) (Model, error) {
	members := make([]MemberModel, 0)
	for _, m := range rm.Members {
		mm, err := ExtractMember(m)
		if err != nil {
			return Model{}, err
		}
		members = append(members, mm)
	}
	return Model{
		id:      rm.Id,
		members: members,
	}, nil
}

func ExtractMember(rm MemberRestModel) (MemberModel, error) {
	return MemberModel{
		id:     rm.Id,
		field:  field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(),
		online: rm.Online,
	}, nil
}
```

- [ ] **Step 5: Write `requests.go`**

Copy `services/atlas-monster-death/atlas.com/monster/party/requests.go` verbatim — same `Resource`/`ByMemberId` constants, same `requests.RootUrlFor(ctx, "PARTIES")`, same `requestByMemberId`. No deploy or service-discovery change is needed: `RootUrlFor` falls back to `BASE_SERVICE_URL` and the shared ingress already routes `/api/parties` (`deploy/shared/routes.conf:16-17`).

- [ ] **Step 6: Write `processor.go`**

```go
type Processor interface {
	GetByMemberId(memberId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByMemberId(memberId uint32) (Model, error) {
	rp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByMemberId(p.ctx, memberId), Extract, model.Filters[Model]())
	return model.FirstProvider(rp, model.Filters[Model]())()
}
```

- [ ] **Step 7: Write `mock/processor.go`**

Copy `services/atlas-monster-death/atlas.com/monster/party/mock/processor.go` verbatim, changing the import to `"atlas-drops/party"`.

- [ ] **Step 8: Run the tests to verify they pass**

Run: `cd services/atlas-drops/atlas.com/drops && go build ./... && go test ./party/... -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-drops/atlas.com/drops/party
git commit -m "feat(atlas-drops): add read-only atlas-parties client"
```

---

## Task 2: `splitMeso` — the pure split function

### Files

- `services/atlas-drops/atlas.com/drops/drop/split.go` — **new file**; `Recipient` + `splitMeso`
- `services/atlas-drops/atlas.com/drops/drop/split_test.go` — **new file**; table-driven tests
- `services/atlas-drops/atlas.com/drops/party/model.go` — read-only here; supplies `party.MemberModel` and `party.NewMemberBuilder()` from Task 1

Module root: `services/atlas-drops/atlas.com/drops`.

**Interfaces:**
- Consumes (Task 1): `party.MemberModel`, `party.NewMemberBuilder()`.
- Produces (Task 3): `drop.Recipient{CharacterId uint32; Amount uint32; Picker bool}` and `splitMeso(f field.Model, meso uint32, pickerId uint32, members []party.MemberModel) []Recipient`.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-drops/atlas.com/drops/drop/split_test.go`, package `drop`.

`TestSplitMeso` — table-driven. Shared fixtures declared once at the top of the function:

```go
instA := uuid.MustParse("00000000-0000-0000-0000-00000000000a")
instB := uuid.MustParse("00000000-0000-0000-0000-00000000000b")
f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).SetInstance(instA).Build()
member := func(id uint32, mf field.Model, online bool) party.MemberModel {
    return party.NewMemberBuilder().SetId(id).SetField(mf).SetOnline(online).Build()
}
onField := func(id uint32) party.MemberModel { return member(id, f, true) }
```

Picker id is `10` in every case except the sorting case.

| subtest name | meso | pickerId | members | expected `[]Recipient` (in order) |
|---|---|---|---|---|
| `no party` | 100 | 10 | `nil` | `{10, 100, true}` |
| `empty member list` | 100 | 10 | `[]party.MemberModel{}` | `{10, 100, true}` |
| `party of one is the picker` | 100 | 10 | `onField(10)` | `{10, 100, true}` |
| `party of three all eligible` | 100 | 10 | `onField(10), onField(20), onField(30)` | `{10, 33, true}, {20, 33, false}, {30, 33, false}` |
| `offline member excluded` | 100 | 10 | `onField(10), member(20, f, false)` | `{10, 100, true}` |
| `different world excluded` | 100 | 10 | `onField(10), member(20, field.NewBuilder(world.Id(2), channel.Id(1), _map.Id(100000000)).SetInstance(instA).Build(), true)` | `{10, 100, true}` |
| `different channel excluded` | 100 | 10 | `onField(10), member(20, field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(instA).Build(), true)` | `{10, 100, true}` |
| `different map excluded` | 100 | 10 | `onField(10), member(20, field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(200000000)).SetInstance(instA).Build(), true)` | `{10, 100, true}` |
| `different instance excluded` | 100 | 10 | `onField(10), member(20, field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).SetInstance(instB).Build(), true)` | `{10, 100, true}` |
| `duplicate member ids collapsed` | 100 | 10 | `onField(20), onField(20)` | `{10, 50, true}, {20, 50, false}` |
| `remainder discarded` | 100 | 10 | `onField(20), onField(30)` | `{10, 33, true}, {20, 33, false}, {30, 33, false}` |
| `meso less than recipient count` | 2 | 10 | `onField(20), onField(30)` | `{10, 0, true}, {20, 0, false}, {30, 0, false}` |
| `picker included despite offline own record` | 100 | 10 | `member(10, f, false), onField(20)` | `{10, 50, true}, {20, 50, false}` |
| `picker included despite stale field on own record` | 100 | 10 | `member(10, field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(200000000)).SetInstance(instA).Build(), true), onField(20)` | `{10, 50, true}, {20, 50, false}` |
| `sorted by character id` | 90 | 25 | `onField(30), onField(20)` | `{20, 30, false}, {25, 30, true}, {30, 30, false}` |

Per case, assert with `require.Equal(t, tc.expected, got)` — that pins order, amount, and picker flag in one comparison.

Two standalone assertions outside the table:

`TestSplitMeso_ExactlyOnePicker` — for the `party of three all eligible` inputs, count `Picker: true` in the result and `require.Equal(t, 1, n)`.

`TestSplitMeso_RemainderIsDiscarded` — for `meso=100`, three eligible recipients, sum every `Amount` and `require.Equal(t, uint32(99), total)` (explicitly **not** 100).

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-drops/atlas.com/drops && go test ./drop/... -run TestSplitMeso -v`
Expected: FAIL — `splitMeso` and `Recipient` undefined.

- [ ] **Step 3: Write `split.go`**

```go
package drop

import (
	"sort"

	"atlas-drops/party"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// Recipient is one character's share of a split meso drop.
type Recipient struct {
	CharacterId uint32
	Amount      uint32
	Picker      bool
}

// splitMeso divides meso evenly among the picker and every online party member
// whose recorded location matches the drop's field on all four dimensions.
// The picker is always included, even when their own party-member record is
// stale or reports offline, so the recipient set is never empty. members == nil
// (no party, or a failed party lookup) collapses to a single full-amount award
// to the picker — the degrade is the empty input, not a special case.
//
// Integer division: the remainder is discarded, nobody receives it. Recipients
// are returned sorted by character id with exactly one Picker: true.
func splitMeso(f field.Model, meso uint32, pickerId uint32, members []party.MemberModel) []Recipient {
	ids := []uint32{pickerId}
	seen := map[uint32]bool{pickerId: true}
	for _, m := range members {
		if seen[m.Id()] {
			continue
		}
		if !m.Online() {
			continue
		}
		mf := m.Field()
		if mf.WorldId() != f.WorldId() || mf.ChannelId() != f.ChannelId() || mf.MapId() != f.MapId() || mf.Instance() != f.Instance() {
			continue
		}
		seen[m.Id()] = true
		ids = append(ids, m.Id())
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	share := meso / uint32(len(ids))
	rs := make([]Recipient, 0, len(ids))
	for _, id := range ids {
		rs = append(rs, Recipient{CharacterId: id, Amount: share, Picker: id == pickerId})
	}
	return rs
}
```

The field predicate is re-expressed here rather than imported from `atlas-channel`'s `party.MemberInMap`: reaching across a service boundary is disallowed by repo convention, and the predicate is three lines.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd services/atlas-drops/atlas.com/drops && go build ./... && go test ./drop/... -run TestSplitMeso -v`
Expected: PASS, all subtests.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-drops/atlas.com/drops/drop/split.go services/atlas-drops/atlas.com/drops/drop/split_test.go
git commit -m "feat(atlas-drops): add pure meso split function"
```

---

## Task 3: `MESO_AWARDED` event and `Reserve` wiring

### Files

- `services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go` — add the type constant and body
- `services/atlas-drops/atlas.com/drops/drop/producer.go` — add `mesoAwardedEventStatusProvider`
- `services/atlas-drops/atlas.com/drops/drop/processor.go` — add the `ProcessorOption` pattern and the split branch in `Reserve`
- `services/atlas-drops/atlas.com/drops/drop/processor_test.go` — add the `Reserve` award tests
- `services/atlas-drops/atlas.com/drops/party/mock/processor.go` — read-only here; injected via `WithPartyProcessor`
- `services/atlas-drops/atlas.com/drops/drop/split.go` — read-only here; supplies `splitMeso`/`Recipient`

Patterns to copy:
- `services/atlas-drops/atlas.com/drops/drop/producer.go:100-122` (`reservedEventStatusProvider` — the provider shape and key)
- `services/atlas-mounts/atlas.com/mounts/mount/processor.go:48-63` (`ProcessorOption` / `With`)
- `services/atlas-drops/atlas.com/drops/drop/processor_test.go:21-40, 97-130` (miniredis registry + tenant context harness, existing `Reserve` test shape)

Module root: `services/atlas-drops/atlas.com/drops`.

**Interfaces:**
- Consumes: `party.Processor`, `mock.ProcessorMock` (Task 1); `splitMeso`, `Recipient` (Task 2).
- Produces (consumed by Task 4 across the service boundary): `EVENT_TOPIC_DROP_STATUS` events with `"type": "MESO_AWARDED"` and body `{"characterId": uint32, "amount": uint32, "picker": bool}`, inside the existing `StatusEvent[E]` envelope (`transactionId`, `worldId`, `channelId`, `mapId`, `instance`, `dropId`).

Note: `services/atlas-drops/atlas.com/drops/drop/mock/processor.go` is not updated. It has no `var _ drop.Processor` assertion, already omits `Consume`/`ConsumeAndEmit`, and has no callers — adding `With` to the interface does not break it. Leave it alone.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-drops/atlas.com/drops/drop/processor_test.go` (package `drop`). Reuse `setupProcessorTestRegistry`, `createTestContext`, `createTestLogger` already in that file.

Add one local helper at the top of the new block:

```go
func awardedFrom(t *testing.T, buf *message.Buffer) []messageDropKafka.StatusEvent[messageDropKafka.StatusEventMesoAwardedBody] {
	t.Helper()
	var out []messageDropKafka.StatusEvent[messageDropKafka.StatusEventMesoAwardedBody]
	for _, m := range buf.GetAll()[messageDropKafka.EnvEventTopicDropStatus] {
		var e messageDropKafka.StatusEvent[messageDropKafka.StatusEventMesoAwardedBody]
		if err := json.Unmarshal(m.Value, &e); err != nil {
			t.Fatalf("unable to decode buffered message: %v", err)
		}
		if e.Type == messageDropKafka.StatusEventTypeMesoAwarded {
			out = append(out, e)
		}
	}
	return out
}
```

(`messageDropKafka` is the existing import alias used in `producer.go` for `atlas-drops/kafka/message/drop`; add it plus `encoding/json` to the test file's imports.)

Every test spawns the drop with `NewModelBuilder(ten, f).SetMeso(<amount>)` (item tests keep `SetItem(1000000, 10)`), reserves through `p.With(WithPartyProcessor(&partymock.ProcessorMock{...}))`, and inspects `awardedFrom(t, reserveBuf)`.

Field for all tests: `f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()` (instance defaults to `uuid.Nil`; party members are built on the same field).

| test function | drop | party mock returns | expected |
|---|---|---|---|
| `TestProcessor_Reserve_MesoDrop_SplitsAmongCoLocatedPartyMembers` | `SetMeso(100)` | party with members 12345 (the picker), 22222, 33333 — all `f`, all online | 3 awards; `CharacterId`/`Amount` pairs exactly `{12345,33},{22222,33},{33333,33}` in ascending-id order `12345,22222,33333`; exactly one `Picker: true`, on 12345; each event's `DropId` equals the spawned drop id and `Type == StatusEventTypeMesoAwarded` |
| `TestProcessor_Reserve_MesoDrop_ExcludesMembersNotCoLocated` | `SetMeso(100)` | members 12345 on `f` online; 22222 on `f` but `online: false`; 33333 online on `_map.Id(200000000)` | 1 award: `{12345, 100, true}` |
| `TestProcessor_Reserve_ItemDrop_MakesNoPartyLookup` | `SetItem(1000000, 10)` (meso 0) | mock increments a `calls` counter and returns an empty party | `calls == 0`; `awardedFrom` is empty; the `RESERVED` event is still buffered |
| `TestProcessor_Reserve_PartyLookupError_AwardsFullAmountToPicker` | `SetMeso(100)` | `GetByMemberIdFunc` returns `party.Model{}, errors.New("unreachable")` | 1 award: `{12345, 100, true}` |
| `TestProcessor_Reserve_FailedReservation_EmitsNoAwards` | `SetMeso(100)`, reserved once by 11111 first, then reserved again by 22222 | mock increments a `calls` counter | second `Reserve` returns an error; `awardedFrom(t, reserveBuf2)` is empty; `calls == 0` on the second attempt |
| `TestProcessor_Reserve_ZeroShareSuppressesNonPickersOnly` | `SetMeso(2)` | members 12345 (picker), 22222, 33333 — all `f`, all online (share = 0) | exactly 1 award: `{12345, 0, true}`; no award for 22222 or 33333 |

Import the party mock as `partymock "atlas-drops/party/mock"` and the party package as `"atlas-drops/party"`; build member fixtures with `party.NewBuilder().SetId(1).SetMembers([]party.MemberModel{...}).Build()` and `party.NewMemberBuilder()`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-drops/atlas.com/drops && go test ./drop/... -run 'TestProcessor_Reserve_(MesoDrop|ItemDrop|PartyLookupError|FailedReservation_EmitsNoAwards|ZeroShare)' -v`
Expected: FAIL — `WithPartyProcessor`, `With`, and `StatusEventTypeMesoAwarded` undefined.

- [ ] **Step 3: Add the contract to `kafka/message/drop/kafka.go`**

Add to the existing status-type `const` block, after `StatusEventTypeConsumed`:

```go
	StatusEventTypeMesoAwarded        = "MESO_AWARDED"
```

Add the body next to `StatusEventReservedBody`:

```go
// StatusEventMesoAwardedBody is the body for MESO_AWARDED status events. One
// event is emitted per recipient of a split meso drop; exactly one carries
// Picker: true, and that recipient's handler completes the pickup.
type StatusEventMesoAwardedBody struct {
	CharacterId uint32 `json:"characterId"`
	Amount      uint32 `json:"amount"`
	Picker      bool   `json:"picker"`
}
```

`StatusEventReservedBody` is not touched.

- [ ] **Step 4: Add the provider to `drop/producer.go`**

```go
func mesoAwardedEventStatusProvider(transactionId uuid.UUID, f field.Model, dropId uint32, r Recipient) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(dropId))
	value := &messageDropKafka.StatusEvent[messageDropKafka.StatusEventMesoAwardedBody]{
		TransactionId: transactionId,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		MapId:         f.MapId(),
		Instance:      f.Instance(),
		DropId:        dropId,
		Type:          messageDropKafka.StatusEventTypeMesoAwarded,
		Body: messageDropKafka.StatusEventMesoAwardedBody{
			CharacterId: r.CharacterId,
			Amount:      r.Amount,
			Picker:      r.Picker,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Add the option pattern to `drop/processor.go`**

Add `With(opts ...ProcessorOption) Processor` as the first method of the `Processor` interface. Add the `pp` field, default it in `NewProcessor`, and add the option plumbing:

```go
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	pp  party.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		pp:  party.NewProcessor(l, ctx),
	}
}

type ProcessorOption func(*ProcessorImpl)

// WithPartyProcessor overrides the atlas-parties client. Tests inject a mock;
// production always uses the default from NewProcessor.
func WithPartyProcessor(pp party.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.pp = pp
	}
}

func (p *ProcessorImpl) With(opts ...ProcessorOption) Processor {
	clone := *p
	cp := &clone
	for _, opt := range opts {
		opt(cp)
	}
	return cp
}
```

Add `"atlas-drops/party"` to the imports.

- [ ] **Step 6: Rewrite `Reserve` in `drop/processor.go`**

Replace the existing method body (the interface signature is unchanged):

```go
// Reserve reserves a drop for a character. On a successful reservation of a
// meso drop, the amount is split among the picker and every online party
// member co-located with the drop; one MESO_AWARDED event per recipient goes
// into the same buffer as RESERVED, so reservation and awards emit as one
// batch. A failed reservation emits neither.
func (p *ProcessorImpl) Reserve(msgBuf *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error) {
	return func(transactionId uuid.UUID, f field.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error) {
		d, err := GetRegistry().ReserveDrop(p.t, dropId, characterId, partyId, petSlot)
		if err != nil {
			p.l.Debugf("Failed reserving [%d] for [%d].", dropId, characterId)
			_ = msgBuf.Put(drop.EnvEventTopicDropStatus, reservationFailureEventStatusProvider(transactionId, f, dropId, characterId))
			return d, err
		}
		p.l.Debugf("Reserving [%d] for [%d].", dropId, characterId)
		_ = msgBuf.Put(drop.EnvEventTopicDropStatus, reservedEventStatusProvider(transactionId, f, d, characterId))
		if d.Meso() == 0 {
			return d, nil
		}
		rs := splitMeso(f, d.Meso(), characterId, p.resolveMembers(characterId))
		p.l.Debugf("Splitting [%d] meso from drop [%d] among [%d] recipient(s).", d.Meso(), dropId, len(rs))
		for _, r := range rs {
			// A zero share is suppressed for everyone but the picker: the
			// picker's award is what completes the pickup, so it must be
			// emitted even at Amount 0 or the drop never leaves the map.
			if r.Amount == 0 && !r.Picker {
				continue
			}
			p.l.Debugf("Awarding [%d] meso from drop [%d] to character [%d].", r.Amount, dropId, r.CharacterId)
			_ = msgBuf.Put(drop.EnvEventTopicDropStatus, mesoAwardedEventStatusProvider(transactionId, f, dropId, r))
		}
		return d, nil
	}
}

// resolveMembers returns the picker's party roster, or nil when the lookup
// fails. It deliberately returns no error: an atlas-parties outage degrades to
// a full-amount award to the picker and must never fail the pickup, and this
// signature makes that impossible to get wrong at a call site.
func (p *ProcessorImpl) resolveMembers(characterId uint32) []party.MemberModel {
	m, err := p.pp.GetByMemberId(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to resolve the party for character [%d]. Awarding the full amount to them alone.", characterId)
		return nil
	}
	return m.Members()
}
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `cd services/atlas-drops/atlas.com/drops && go build ./... && go test ./... `
Expected: PASS, including the pre-existing `TestProcessor_Reserve_SuccessfulReservation` and `TestProcessor_Reserve_FailedReservation_BuffersFailureMessage`.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go services/atlas-drops/atlas.com/drops/drop/producer.go services/atlas-drops/atlas.com/drops/drop/processor.go services/atlas-drops/atlas.com/drops/drop/processor_test.go
git commit -m "feat(atlas-drops): emit MESO_AWARDED per recipient on meso reservation"
```

---

## Task 4: `atlas-character` credits each recipient

### Files

- `services/atlas-character/atlas.com/character/kafka/message/drop/kafka.go` — add `TransactionId` to `StatusEvent`, the `MESO_AWARDED` type constant, and `MesoAwardedStatusEventBody`
- `services/atlas-character/atlas.com/character/kafka/consumer/drop/consumer.go` — delete `handleDropReservation` and its registration; register `handleMesoAwarded`
- `services/atlas-character/atlas.com/character/character/processor.go` — replace `AttemptMesoPickUp` with `AwardPickedUpMeso` (interface line 130, implementation line 928)
- `services/atlas-character/atlas.com/character/character/meso_award_test.go` — **new file**; the award tests
- `services/atlas-character/atlas.com/character/character/meso_outbox_test.go` — read-only; supplies `outboxTestDb`, `createTestCharacter`, `outboxRowCount`
- `services/atlas-character/atlas.com/character/character/processor_test.go` — read-only; supplies `testDatabase`, `testTenant`, `testLogger`

Patterns to copy:
- `services/atlas-character/atlas.com/character/character/processor.go:880-925` (`RequestChangeMeso` — the transaction + outbox emit shape, and the style of the deliberate-asymmetry comment)
- `services/atlas-character/atlas.com/character/character/meso_overflow_test.go:24-40` (`producertest.InstallCapturing` + `outboxTestDb` + `createTestCharacter` setup)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go:55-74` (registering more than one handler on one topic — `atlas-character` registers only one, so follow its own `InitHandlers` shape and swap the handler)

Module root: `services/atlas-character/atlas.com/character`.

**Interfaces:**
- Consumes (Task 3, over Kafka): `EVENT_TOPIC_DROP_STATUS` `"MESO_AWARDED"` with body `{characterId, amount, picker}` and envelope `{transactionId, worldId, channelId, mapId, instance, dropId, type}`.
- Produces: `character.Processor.AwardPickedUpMeso(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, meso uint32, picker bool) error`. `AttemptMesoPickUp` is removed from the interface and the implementation; its only caller is the consumer changed in this task (verified: `grep -rn AttemptMesoPickUp` returns exactly `processor.go:130`, `processor.go:928`, `consumer.go:49`).

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-character/atlas.com/character/character/meso_award_test.go`, package `character_test`.

Every test starts with:

```go
capture := producertest.InstallCapturing()
t.Cleanup(producertest.InstallNoop)
tctx := tenant.WithContext(context.Background(), testTenant())
db := outboxTestDb(t)
```

and uses `createTestCharacter(t, tctx, db, <startingMeso>)` plus `character.NewProcessor(testLogger(), tctx, db)`.

Add one local helper for reading back the outbox payloads:

```go
func outboxEvents(t *testing.T, db *gorm.DB) []character2.StatusEvent[json.RawMessage] {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Order("id asc").Find(&rows).Error)
	out := make([]character2.StatusEvent[json.RawMessage], 0, len(rows))
	for _, r := range rows {
		var e character2.StatusEvent[json.RawMessage]
		require.NoError(t, json.Unmarshal(r.MessageValue, &e))
		out = append(out, e)
	}
	return out
}
```

and one for pick-up commands:

```go
func pickUpCommands(t *testing.T, capture *producertest.Capture) []dropmsg.Command[dropmsg.RequestPickUpCommandBody] {
	t.Helper()
	var out []dropmsg.Command[dropmsg.RequestPickUpCommandBody]
	for _, m := range capture.Messages(dropmsg.EnvCommandTopic) {
		var c dropmsg.Command[dropmsg.RequestPickUpCommandBody]
		require.NoError(t, json.Unmarshal(m.Value, &c))
		if c.Type == dropmsg.CommandTypeRequestPickUp {
			out = append(out, c)
		}
	}
	return out
}
```

(`dropmsg` is `atlas-character/kafka/message/drop`; `character2` is `atlas-character/kafka/message/character`.)

Field fixture: `f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.MustParse("00000000-0000-0000-0000-00000000000a")).Build()`.

| test function | starting meso | call | expected |
|---|---|---|---|
| `TestAwardPickedUpMeso_CreditsAndEmitsMesoChangedAndStatChanged` | 0 | `AwardPickedUpMeso(txId, f, c.Id(), 4242, 33, false)` | returns nil; balance is 33; outbox row count is `before+2`; the two new events are `character2.StatusEventTypeMesoChanged` then `character2.StatusEventTypeStatChanged`; the `MESO_CHANGED` body decodes to `Amount: int32(33)`, `ShowEffect: true`, `ActorId: uint32(4242)`, `ActorType: "DROP"`; both events' `TransactionId` equals `txId`; `pickUpCommands` is empty |
| `TestAwardPickedUpMeso_PickerCompletesThePickUp` | 0 | `AwardPickedUpMeso(txId, f, c.Id(), 4242, 33, true)` | returns nil; exactly one pick-up command; its `Body.DropId == uint32(4242)`, `Body.CharacterId == c.Id()`, `WorldId == world.Id(0)`, `ChannelId == channel.Id(1)`, `MapId == _map.Id(100000000)`, `Instance` equals the fixture instance |
| `TestAwardPickedUpMeso_NonPickerDoesNotCompleteThePickUp` | 0 | `AwardPickedUpMeso(txId, f, c.Id(), 4242, 33, false)` | balance is 33 and `pickUpCommands` is empty |
| `TestAwardPickedUpMeso_ZeroAmountRunsNoTransactionButCompletesThePickUp` | 500 | `AwardPickedUpMeso(txId, f, c.Id(), 4242, 0, true)` | returns nil; balance is still 500; outbox row count is unchanged; exactly one pick-up command |
| `TestAwardPickedUpMeso_OverflowSkipsTheCreditButStillCompletesThePickUp` | see below | `AwardPickedUpMeso(txId, f, c.Id(), 4242, 2147483647, true)` | returns `character.ErrMesoOverflow`; balance unchanged at 3147483647; outbox row count unchanged; exactly one pick-up command |
| `TestAwardPickedUpMeso_AmountAboveInt32IsRejected` | 0 | `AwardPickedUpMeso(txId, f, c.Id(), 4242, 2147483648, true)` | returns `character.ErrMesoOverflow`; balance still 0; outbox unchanged; exactly one pick-up command |

Overflow-case setup, copied from `meso_overflow_test.go:38-43`: `c := createTestCharacter(t, tctx, db, 1000000000)`, then `require.NoError(t, p.RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false))` to reach 3147483647, then `capture.Reset()` and re-read `before := outboxRowCount(t, db)` before the award call.

Regression guard for FR-15 — a second new test in the same file:

`TestReservedStatusEventNoLongerCreditsMeso` — asserts the `RESERVED` credit path is gone at the source level, since the handler itself no longer exists to call. Implement it as a compile-and-behavior guard on the consumer package instead: create `services/atlas-character/atlas.com/character/kafka/consumer/drop/consumer_test.go` (package `drop`) with

`TestHandleMesoAwarded_IgnoresNonMesoAwardedEvents` — call `handleMesoAwarded(nil)(testLogger, ctx, e)` with `e.Type = "RESERVED"` and assert it returns without panicking (a `nil` `*gorm.DB` would panic if the guard did not short-circuit). Build the logger with `test.NewNullLogger()` and the context with `tenant.WithContext(context.Background(), <tenant>)`; construct the tenant with `tenant.Create(uuid.New(), "GMS", 83, 1)`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/... -run TestAwardPickedUpMeso -v`
Expected: FAIL — `AwardPickedUpMeso` undefined.

- [ ] **Step 3: Extend the drop contract mirror**

In `services/atlas-character/atlas.com/character/kafka/message/drop/kafka.go`:

Add `TransactionId` as the first field of `StatusEvent` so the award correlates to the reservation that produced it, instead of the consumer fabricating one:

```go
type StatusEvent[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	MapId         _map.Id    `json:"mapId"`
	Instance      uuid.UUID  `json:"instance"`
	DropId        uint32     `json:"dropId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}
```

Extend the status const block and add the body:

```go
const (
	EnvEventTopicDropStatus    = "EVENT_TOPIC_DROP_STATUS"
	StatusEventTypeReserved    = "RESERVED"
	StatusEventTypeMesoAwarded = "MESO_AWARDED"
)

// MesoAwardedStatusEventBody mirrors atlas-drops' StatusEventMesoAwardedBody.
// One event per recipient of a split meso drop; exactly one carries
// Picker: true, and only that one completes the pickup.
type MesoAwardedStatusEventBody struct {
	CharacterId uint32 `json:"characterId"`
	Amount      uint32 `json:"amount"`
	Picker      bool   `json:"picker"`
}
```

`ReservedStatusEventBody` is left exactly as it is — `RESERVED` is still consumed by other services and its shape must not move.

- [ ] **Step 4: Replace the consumer handler**

Rewrite `services/atlas-character/atlas.com/character/kafka/consumer/drop/consumer.go`: delete `handleDropReservation` entirely (it has no remaining responsibility once meso moves to `MESO_AWARDED`), register `handleMesoAwarded` in its place, and drop the now-unused `github.com/google/uuid` import.

```go
func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(drop2.EnvEventTopicDropStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMesoAwarded(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleMesoAwarded(db *gorm.DB) message.Handler[drop2.StatusEvent[drop2.MesoAwardedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e drop2.StatusEvent[drop2.MesoAwardedStatusEventBody]) {
		if e.Type != drop2.StatusEventTypeMesoAwarded {
			return
		}
		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		if err := character.NewProcessor(l, ctx, db).AwardPickedUpMeso(e.TransactionId, f, e.Body.CharacterId, e.DropId, e.Body.Amount, e.Body.Picker); err != nil {
			l.WithError(err).Errorf("Unable to award [%d] meso from drop [%d] to character [%d].", e.Body.Amount, e.DropId, e.Body.CharacterId)
		}
	}
}
```

`SetInstance(e.Instance)` is load-bearing: the old handler built the field without it, and the recipient set is compared on instance.

- [ ] **Step 5: Replace `AttemptMesoPickUp` with `AwardPickedUpMeso`**

In `services/atlas-character/atlas.com/character/character/processor.go`, replace the interface entry at line 130:

```go
	AwardPickedUpMeso(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, meso uint32, picker bool) error
```

and replace the implementation at line 928:

```go
// actorTypeDrop identifies a picked-up drop as the actor behind a meso change.
// ActorType is a free-form string on the wire; atlas-channel's MESO_CHANGED
// consumer ignores it, so this value is purely additive.
const actorTypeDrop = "DROP"

// AwardPickedUpMeso credits one recipient's share of a split meso drop and,
// when that recipient is the picker, completes the pickup.
//
// Deliberate asymmetry, and the reason this replaced AttemptMesoPickUp: the
// pickup completion is NOT conditional on the credit succeeding. The old shape
// returned the transaction error before reaching RequestPickUp, so a picker at
// the meso ceiling left the drop on the map forever. The credit error is
// returned for logging only. Do not "harmonise" these two into one branch.
func (p *ProcessorImpl) AwardPickedUpMeso(transactionId uuid.UUID, f field.Model, characterId uint32, dropId uint32, meso uint32, picker bool) error {
	var txErr error
	if meso > 0 {
		txErr = database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetById()(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve character [%d] who is having their meso adjusted.", characterId)
				return err
			}
			// MESO_CHANGED carries the amount as an int32; an award that
			// cannot be represented there is rejected rather than emitted
			// as a negative delta.
			if meso > math.MaxInt32 {
				p.l.Errorf("Meso award of [%d] to character [%d] exceeds the int32 MESO_CHANGED amount. Rejecting transaction.", meso, characterId)
				return ErrMesoOverflow
			}
			if meso > (math.MaxUint32 - c.Meso()) {
				p.l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
				return ErrMesoOverflow
			}
			if err = dynamicUpdate(tx)(SetMeso(uint32(int64(c.Meso()) + int64(meso))))(c); err != nil {
				return err
			}
			return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
				if err = buf.Put(character2.EnvEventTopicCharacterStatus, mesoChangedStatusEventProvider(transactionId, characterId, c.WorldId(), int32(meso), dropId, actorTypeDrop, true)); err != nil {
					return err
				}
				return buf.Put(character2.EnvEventTopicCharacterStatus, statChangedProvider(transactionId, channel.NewModel(f.WorldId(), f.ChannelId()), characterId, []stat.Type{stat.TypeMeso}, nil))
			})
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to credit character [%d] with [%d] meso from drop [%d].", characterId, meso, dropId)
		}
	}
	if picker {
		if err := drop.NewProcessor(p.l, p.ctx).RequestPickUp(f, dropId, characterId); err != nil {
			p.l.WithError(err).Errorf("Unable to complete pick up of drop [%d] for character [%d].", dropId, characterId)
		}
	}
	return txErr
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-character/atlas.com/character && go build ./... && go test ./character/... ./kafka/... -v`
Expected: PASS, including the pre-existing meso tests in `meso_outbox_test.go` and `meso_overflow_test.go`.

- [ ] **Step 7: Run the full module test suite**

Run: `cd services/atlas-character/atlas.com/character && go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/message/drop/kafka.go services/atlas-character/atlas.com/character/kafka/consumer/drop/consumer.go services/atlas-character/atlas.com/character/kafka/consumer/drop/consumer_test.go services/atlas-character/atlas.com/character/character/processor.go services/atlas-character/atlas.com/character/character/meso_award_test.go
git commit -m "feat(atlas-character): credit each MESO_AWARDED recipient and stop crediting on RESERVED"
```

---

## Verification

After Task 4, from the worktree root:

```bash
tools/verify.sh
```

must exit 0 (flagless — `--quick`/`--no-docker` skip the bake and `-race` and do not count).

Then run the code-review step before opening a PR: `backend-guidelines-reviewer` over `services/atlas-drops` and `services/atlas-character`.
