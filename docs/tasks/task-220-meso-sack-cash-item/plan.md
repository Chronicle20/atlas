# Meso Sack Cash Item Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make cash-slot type 19 (meso sacks, WZ classification `520`) work end to end — consume one sack, credit its WZ `info/meso` amount atomically, render the meso chat line, and unlock the client on every outcome.

**Architecture:** `atlas-data` parses `info/meso` into a first-class field on the cash document. `atlas-channel` grows a `CashSlotItemTypeCurrencySack` branch that resolves the amount server-side and creates a two-step `meso_sack_use` saga (`DestroyAsset` → `AwardMesos`). `atlas-character` emits a `MESO_OVERFLOW` error status event on the previously silent overflow path so the award step fails fast. `atlas-saga-orchestrator` gains a dedicated `meso_sack_use` compensator that refunds the sack and emits the saga-failed event with the **real** character id (the generic emitter would emit `characterId: 0`), plus the matching timeout classification.

**Tech Stack:** Go 1.x, multi-module workspace (`go.work`), Kafka (segmentio/kafka-go via `libs/atlas-kafka`), JSON:API REST (`libs/atlas-rest`), GORM + Postgres, testify + stdlib `testing`.

## Global Constraints

- **No wire change.** Design §3 IDA-verified all ten versions: the classification-520 arm of `CWvsContext::SendConsumeCashItemUseRequest` encodes **no sub-body**. Do not add a codec, do not touch `libs/atlas-packet`, do not touch any `services/atlas-configurations/seed-data/templates/*.json`.
- **Do not version-gate `GetCashSlotItemType`'s `19`.** The v48 (17) and v61 (18) client tables disagree with Atlas's `19`, and that is correct — the type never rides the wire (design §3.1(a)). A "fix" breaks those builds.
- **Fail closed on a missing/oversized amount.** `meso == 0` (absent node, or a Maple Point sack `5200009`/`5200010`) and `meso > math.MaxInt32` both reject: nothing consumed, nothing awarded, warn logged, client unlocked. `AwardMesosPayload.Amount` is `int32` while the WZ value is `uint32`; without the ceiling guard a large sack silently wraps negative and *takes* mesos.
- **Consume first, award second.** Every sibling cash-item-use arm destroys before it grants.
- **No unlock packet on the success path.** `atlas-character`'s `statChangedProvider` hard-codes `ExclRequestSent: true` (`services/atlas-character/atlas.com/character/character/producer.go:238`), and the meso credit already emits `STAT_CHANGED{TypeMeso}` — that packet *is* the unlock, correctly ordered by construction. An extra empty `StatChanged` would race the real one.
- **Saga type string:** `meso_sack_use`. **Error code string:** `MESO_OVERFLOW`.
- **Meso-ceiling copy:** `You cannot hold any more mesos.` — every other error code on this saga type gets `You are unable to use this item right now.`
- Use the project's Builder pattern in tests. No `*_testhelpers.go` files. No `// TODO`, no stubs.
- Preserve existing line endings; never write absolute home paths into committed files.

---

### Task 1: `atlas-data` parses `info/meso`

**Files:**
- Modify: `services/atlas-data/atlas.com/data/cash/rest.go:38-50` (the `RestModel` struct)
- Modify: `services/atlas-data/atlas.com/data/cash/reader.go:76-80` (the `info` scalar block)
- Test: `services/atlas-data/atlas.com/data/cash/reader_test.go` (append)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `cash.RestModel.Meso uint32`, serialized as `"meso"` with `omitempty`, on the `cash_items` JSON:API resource.

- [x] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/cash/reader_test.go`:

```go
// testMesoSackXML mirrors the real GMS 83.1 Item.wz/Cash/0520.img node set:
// icon/iconRaw/meso/cash only — no slotMax, no spec, no tradeBlock. 05200003
// carries an explicit meso of 0 and 05200004 omits the node entirely; both
// must land on Meso == 0 so the handler's fail-closed guard trips (FR-1.2).
const testMesoSackXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="0520.img">
  <imgdir name="05200000">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="meso" value="1000000"/>
    </imgdir>
  </imgdir>
  <imgdir name="05200001">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="meso" value="5000000"/>
    </imgdir>
  </imgdir>
  <imgdir name="05200002">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="meso" value="10000000"/>
    </imgdir>
  </imgdir>
  <imgdir name="05200003">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="meso" value="0"/>
    </imgdir>
  </imgdir>
  <imgdir name="05200004">
    <imgdir name="info">
      <int name="cash" value="1"/>
      <int name="maplepoint" value="10000"/>
    </imgdir>
  </imgdir>
</imgdir>
`

func TestReaderMesoSacks(t *testing.T) {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testMesoSackXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		id   int
		want uint32
	}{
		{5200000, 1000000},
		{5200001, 5000000},
		{5200002, 10000000},
		{5200003, 0}, // explicit zero
		{5200004, 0}, // node absent (Maple Point sack)
	}
	for _, tc := range cases {
		rm, ok := rmm[strconv.Itoa(tc.id)]
		if !ok {
			t.Fatalf("cash item %d missing from read result", tc.id)
		}
		if rm.Meso != tc.want {
			t.Errorf("Meso(%d) = %d, want %d", tc.id, rm.Meso, tc.want)
		}
	}
}

// The award amount is a first-class field, not an effect: folding it into Spec
// would feed it to the consumable pipeline. Spec must gain no "meso" key.
func TestReaderMesoNotFoldedIntoSpec(t *testing.T) {
	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testMesoSackXML)))
	rmm, err := model.CollectToMap[RestModel, string, RestModel](rms, RestModel.GetID, Identity)()
	if err != nil {
		t.Fatal(err)
	}
	if _, present := rmm[strconv.Itoa(5200000)].Spec[SpecType("meso")]; present {
		t.Fatal(`Spec gained a "meso" key; the award amount must stay a first-class field`)
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-data/atlas.com/data && go test ./cash/ -run 'TestReaderMeso' -v`
Expected: FAIL — `rm.Meso undefined (type RestModel has no field or method Meso)` (compile error).

- [x] **Step 3: Add the field**

In `services/atlas-data/atlas.com/data/cash/rest.go`, add `Meso` to `RestModel` immediately after `ProtectTime`:

```go
type RestModel struct {
	Id              uint32             `json:"-"`
	SlotMax         uint32             `json:"slotMax"`
	ProtectTime     uint32             `json:"protectTime,omitempty"`
	Meso            uint32             `json:"meso,omitempty"` // 0520 meso sacks: info/meso award amount
	StateChangeItem uint32             `json:"stateChangeItem,omitempty"`
	BgmPath         string             `json:"bgmPath,omitempty"`
	Spec            map[SpecType]int32 `json:"spec"`
	TimeWindows     []TimeWindow       `json:"timeWindows,omitempty"` // Active time windows from info/time
	PetSkills       []string           `json:"petSkills,omitempty"`
	PetSkillAdd     bool               `json:"petSkillAdd,omitempty"`
	TradeBlock      bool               `json:"tradeBlock"`
}
```

- [x] **Step 4: Parse the node**

In `services/atlas-data/atlas.com/data/cash/reader.go`, immediately after the `m.ProtectTime = ...` line:

```go
		m.ProtectTime = uint32(i.GetIntegerWithDefault("protectTime", 0))
		// 0520 meso sacks: the flat award amount. Absent node => 0, which the
		// channel handler treats as "reject, consume nothing" (FR-1.2/FR-2.4).
		// Deliberately NOT a Spec entry — Spec is the consumable effect map and
		// this is an award amount.
		m.Meso = uint32(i.GetIntegerWithDefault("meso", 0))
```

(The exact indentation must match the surrounding block — the assignments sit inside the `for _, cxml := range exml.ChildNodes` loop.)

- [x] **Step 5: Run the tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test ./cash/ -run 'TestReaderMeso' -v`
Expected: PASS (both tests).

- [x] **Step 6: Run the full module test + vet**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./...`
Expected: all PASS, vet silent.

- [x] **Step 7: Commit**

```bash
git add services/atlas-data/atlas.com/data/cash/rest.go \
        services/atlas-data/atlas.com/data/cash/reader.go \
        services/atlas-data/atlas.com/data/cash/reader_test.go
git commit -m "feat(task-220): parse cash info/meso into a first-class Meso field"
```

---

### Task 2: `atlas-channel` cash view model carries `meso`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/data/cash/rest.go:7-12`
- Test: `services/atlas-channel/atlas.com/channel/data/cash/rest_test.go` (create)

**Interfaces:**
- Consumes: the `"meso"` JSON attribute produced by Task 1.
- Produces: `cash.RestModel.Meso uint32`, reachable from the handler as `cashData.NewProcessor(l, ctx).GetById(itemId)` → `cd.Meso`.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/data/cash/rest_test.go`:

```go
package cash

import (
	"encoding/json"
	"testing"
)

// The channel mirror of atlas-data's cash_items resource is partial by design —
// it carries only the fields the channel actually uses. This pins that `meso`
// (the 0520 meso-sack award amount) is one of them: without it the type-19
// handler cannot resolve an amount and every sack use fails closed.
func TestRestModelDecodesMeso(t *testing.T) {
	var rm RestModel
	if err := json.Unmarshal([]byte(`{"stateChangeItem":0,"bgmPath":"","protectTime":0,"meso":1000000}`), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rm.Meso != 1000000 {
		t.Fatalf("Meso = %d, want 1000000", rm.Meso)
	}
}

// atlas-data omits `meso` (omitempty) for every non-sack cash item; the mirror
// must decode that as 0 rather than erroring.
func TestRestModelMesoAbsentIsZero(t *testing.T) {
	var rm RestModel
	if err := json.Unmarshal([]byte(`{"stateChangeItem":0,"bgmPath":"","protectTime":7}`), &rm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if rm.Meso != 0 {
		t.Fatalf("Meso = %d, want 0", rm.Meso)
	}
}
```

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/cash/ -v`
Expected: FAIL — `rm.Meso undefined`.

- [x] **Step 3: Add the field**

In `services/atlas-channel/atlas.com/channel/data/cash/rest.go`:

```go
type RestModel struct {
	Id              uint32 `json:"-"`
	StateChangeItem uint32 `json:"stateChangeItem"`
	BgmPath         string `json:"bgmPath"`
	ProtectTime     uint32 `json:"protectTime"`
	// Meso is the 0520 meso-sack award amount (atlas-data info/meso). Absent
	// or 0 means "no payout" and the type-19 handler rejects the use.
	Meso uint32 `json:"meso"`
}
```

- [x] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./data/cash/ -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/data/cash/rest.go \
        services/atlas-channel/atlas.com/channel/data/cash/rest_test.go
git commit -m "feat(task-220): carry meso on the channel cash view model"
```

---

### Task 3: `meso_sack_use` saga type + error code constants

**Files:**
- Modify: `libs/atlas-saga/model.go:42-44` (the `Type` const block)
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go:52-67` (the re-export const block)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go:45` (the re-export const block)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go:15-27`
- Test: `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka_test.go` (create)

**Interfaces:**
- Produces:
  - `sharedsaga.MesoSackUse Type = "meso_sack_use"` (`libs/atlas-saga`)
  - `saga.MesoSackUse` re-exported in both `atlas-channel` and `atlas-saga-orchestrator`
  - `saga.SagaTypeMesoSackUse = "meso_sack_use"` in `atlas-channel/kafka/message/saga`
  - `saga.ErrorCodeMesoOverflow = "MESO_OVERFLOW"` in `atlas-channel/kafka/message/saga`
- Later tasks rely on all four names.

- [x] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka_test.go`:

```go
package saga

import "testing"

// The saga consumer compares e.Body.SagaType against this string copy, not
// against the atlas-saga Type. The two are separate declarations in separate
// modules; if they drift, every meso_sack_use failure silently skips its
// client-notification arm and the player stays input-locked.
func TestSagaTypeMesoSackUseString(t *testing.T) {
	if SagaTypeMesoSackUse != "meso_sack_use" {
		t.Fatalf("SagaTypeMesoSackUse = %q, want %q", SagaTypeMesoSackUse, "meso_sack_use")
	}
}

// atlas-character emits this exact string as StatusEventMesoErrorBody.Error;
// the orchestrator threads it verbatim onto the saga-failed event's errorCode.
func TestErrorCodeMesoOverflowString(t *testing.T) {
	if ErrorCodeMesoOverflow != "MESO_OVERFLOW" {
		t.Fatalf("ErrorCodeMesoOverflow = %q, want %q", ErrorCodeMesoOverflow, "MESO_OVERFLOW")
	}
}
```

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/saga/ -v`
Expected: FAIL — `undefined: SagaTypeMesoSackUse`.

- [x] **Step 3: Add the shared saga type**

In `libs/atlas-saga/model.go`, in the `Type` const block, after `MegaphoneUse`:

```go
	PointReset          Type = "point_reset"
	MegaphoneUse        Type = "megaphone_use"
	MesoSackUse         Type = "meso_sack_use"
)
```

- [x] **Step 4: Re-export it in both services**

In `services/atlas-channel/atlas.com/channel/saga/model.go`, in the saga-types section of the re-export const block, after `NoteSend`:

```go
	NoteSend             = sharedsaga.NoteSend
	MesoSackUse          = sharedsaga.MesoSackUse
```

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go`, on the line after the `PointReset` re-export:

```go
	PointReset           = sharedsaga.PointReset
	MesoSackUse          = sharedsaga.MesoSackUse
```

(Match the existing alignment in each file — `gofumpt` via `tools/lint.sh` will re-align if it differs.)

- [x] **Step 5: Add the channel message constants**

In `services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go`, in the const block that already holds `ErrorCodeNotEnoughMesos` / `SagaTypePointReset`:

```go
	ErrorCodeNotEnoughMesos = "NOT_ENOUGH_MESOS"
	ErrorCodeInventoryFull  = "INVENTORY_FULL"
	ErrorCodeStorageFull    = "STORAGE_FULL"
	ErrorCodeUnknown        = "UNKNOWN"
	// ErrorCodeMesoOverflow is atlas-character's StatusEventErrorTypeMesoOverflow,
	// threaded onto the saga-failed event by the orchestrator's meso-error handler.
	ErrorCodeMesoOverflow = "MESO_OVERFLOW"
```

and, beside the other saga-type strings:

```go
	SagaTypePointReset       = "point_reset"
	SagaTypeMtsOperation     = "mts_operation"
	SagaTypeNoteSend         = "note_send"
	SagaTypeMesoSackUse      = "meso_sack_use"
```

- [x] **Step 6: Run the test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/message/saga/ -v`
Expected: PASS (both tests).

- [x] **Step 7: Build all three modules**

Run:
```bash
cd libs/atlas-saga && go build ./... && \
cd ../../services/atlas-channel/atlas.com/channel && go build ./... && \
cd ../../../atlas-saga-orchestrator/atlas.com/saga-orchestrator && go build ./...
```
Expected: clean.

- [x] **Step 8: Commit**

```bash
git add libs/atlas-saga/model.go \
        services/atlas-channel/atlas.com/channel/saga/model.go \
        services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka.go \
        services/atlas-channel/atlas.com/channel/kafka/message/saga/kafka_test.go \
        services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go
git commit -m "feat(task-220): add meso_sack_use saga type and MESO_OVERFLOW error code"
```

---

### Task 4: `atlas-character` emits `MESO_OVERFLOW`

**Files:**
- Modify: `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go:244` (const block)
- Modify: `services/atlas-character/atlas.com/character/character/producer.go:168-181` (add a sibling provider)
- Modify: `services/atlas-character/atlas.com/character/character/processor.go:824-859` (`RequestChangeMeso`)
- Test: `services/atlas-character/atlas.com/character/character/meso_overflow_test.go` (create)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `character2.StatusEventErrorTypeMesoOverflow = "MESO_OVERFLOW"`, and a `StatusEvent[StatusEventMesoErrorBody]` with `Type: ERROR`, `Body.Error: "MESO_OVERFLOW"`, `Body.Amount: <requested amount>` emitted to `EnvEventTopicCharacterStatus` on the overflow rejection. `RequestChangeMeso` still returns `ErrMesoOverflow` (unchanged — reject, never clamp).

- [x] **Step 1: Write the failing test**

Create `services/atlas-character/atlas.com/character/character/meso_overflow_test.go`:

```go
package character_test

import (
	"atlas-character/character"
	character2 "atlas-character/kafka/message/character"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RequestChangeMeso's overflow guard used to return ErrMesoOverflow with no
// emission at all, which stranded the award_mesos step of a meso_sack_use saga
// until the orchestrator's timeout backstop fired. It must now emit the same
// non-generic meso-error body the NOT_ENOUGH_MESO path uses, with a
// MESO_OVERFLOW code, so the step fails fast. The error return is unchanged —
// rejection, never clamping.
func TestRequestChangeMeso_OverflowEmitsMesoOverflowErrorEvent(t *testing.T) {
	capture := producertest.InstallCapturing()
	t.Cleanup(producertest.InstallNoop)

	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)
	c := createTestCharacter(t, tctx, db, 0)

	p := character.NewProcessor(testLogger(), tctx, db)
	// Two max-int32 credits from a zero base guarantee crossing MaxUint32 on
	// the second, which is the one that must emit.
	require.NoError(t, p.RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false))
	capture.Reset()

	before := outboxRowCount(t, db)
	err := p.RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false)
	require.ErrorIs(t, err, character.ErrMesoOverflow)

	// The rejection commits nothing: no MESO_CHANGED / STAT_CHANGED outbox rows.
	require.Equal(t, before, outboxRowCount(t, db))

	msgs := capture.Messages(character2.EnvEventTopicCharacterStatus)
	require.Len(t, msgs, 1, "overflow must emit exactly one status event")

	var e character2.StatusEvent[character2.StatusEventMesoErrorBody]
	require.NoError(t, json.Unmarshal(msgs[0].Value, &e))
	require.Equal(t, character2.StatusEventTypeError, e.Type)
	require.Equal(t, c.Id(), e.CharacterId)
	require.Equal(t, character2.StatusEventErrorTypeMesoOverflow, e.Body.Error)
	require.Equal(t, int32(2147483647), e.Body.Amount)

	// And the balance is untouched.
	got, gerr := p.GetById()(c.Id())
	require.NoError(t, gerr)
	require.Equal(t, uint32(2147483647), got.Meso())
}
```

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestRequestChangeMeso_OverflowEmits -v`
Expected: FAIL — `undefined: character2.StatusEventErrorTypeMesoOverflow`.

- [x] **Step 3: Add the error-type constant**

In `services/atlas-character/atlas.com/character/kafka/message/character/kafka.go`, beside `StatusEventErrorTypeNotEnoughMeso`:

```go
	StatusEventErrorTypeNotEnoughMeso = "NOT_ENOUGH_MESO"
	// StatusEventErrorTypeMesoOverflow rejects an award that would exceed the
	// uint32 meso ceiling. Shares StatusEventMesoErrorBody with NOT_ENOUGH_MESO
	// so the orchestrator's existing meso-error handler accepts it unchanged.
	StatusEventErrorTypeMesoOverflow = "MESO_OVERFLOW"
```

- [x] **Step 4: Add the provider**

In `services/atlas-character/atlas.com/character/character/producer.go`, directly below `notEnoughMesoErrorStatusEventProvider`:

```go
func mesoOverflowErrorStatusEventProvider(transactionId uuid.UUID, characterId uint32, worldId world.Id, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.StatusEvent[character2.StatusEventMesoErrorBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		WorldId:       worldId,
		Type:          character2.StatusEventTypeError,
		Body: character2.StatusEventMesoErrorBody{
			Error:  character2.StatusEventErrorTypeMesoOverflow,
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [x] **Step 5: Emit on the overflow path**

In `services/atlas-character/atlas.com/character/character/processor.go`, inside `RequestChangeMeso`, replace the overflow guard body:

```go
		if amount > 0 && uint32(amount) > (math.MaxUint32-c.Meso()) {
			p.l.Errorf("Transaction for character [%d] would result in a uint32 overflow. Rejecting transaction.", characterId)
			rejectEmit = func() error {
				return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvEventTopicCharacterStatus)(mesoOverflowErrorStatusEventProvider(transactionId, characterId, c.WorldId(), amount))
			}
			return ErrMesoOverflow
		}
```

and, after the transaction closure, extend the post-rollback emission block:

```go
	if errors.Is(txErr, ErrNotEnoughMeso) && rejectEmit != nil {
		_ = rejectEmit()
		return nil
	}
	// Deliberate asymmetry with the NOT_ENOUGH_MESO path above: overflow keeps
	// returning the error so the REST/command caller still logs a failure. The
	// emission is additive — the saga is driven by the event either way. Do not
	// "harmonise" these two into one branch.
	if errors.Is(txErr, ErrMesoOverflow) && rejectEmit != nil {
		_ = rejectEmit()
	}
	return txErr
```

- [x] **Step 6: Run the test to verify it passes**

Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestRequestChangeMeso -v`
Expected: PASS, including the pre-existing `TestRequestChangeMeso_OverflowReturnsError` and `TestRequestChangeMeso_NotEnoughMesoEmitsNoOutboxRowsAndReturnsNil`.

- [x] **Step 7: Run the full module test + vet**

Run: `cd services/atlas-character/atlas.com/character && go test -race ./... && go vet ./...`
Expected: all PASS, vet silent.

- [x] **Step 8: Commit**

```bash
git add services/atlas-character/atlas.com/character/kafka/message/character/kafka.go \
        services/atlas-character/atlas.com/character/character/producer.go \
        services/atlas-character/atlas.com/character/character/processor.go \
        services/atlas-character/atlas.com/character/character/meso_overflow_test.go
git commit -m "feat(task-220): emit MESO_OVERFLOW error event on the meso ceiling rejection"
```

---

### Task 5: Orchestrator threads the meso error code onto the step result

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/consumer.go:166-184` (`handleCharacterMesoErrorEvent`)

**Interfaces:**
- Consumes: `character2.StatusEventMesoErrorBody.Error` (`"MESO_OVERFLOW"` or `"NOT_ENOUGH_MESO"`) from Task 4.
- Produces: the failed step's result map now carries `{"errorCode": <the error string>}`, which Task 6's compensator reads.

**Verified, no change needed:** the acceptance table already maps `AwardMesos → {MesoChanged, MesoError}` (`saga/event_acceptance.go:132`) with `EventKindCharacterMesoError → OutcomeFailure` (`saga/event_acceptance.go:327`), so the new saga's `award_mesos` step is already covered (PRD FR-7.2). Do not edit `event_acceptance.go`.

- [x] **Step 1: Write the failing test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/meso_error_result_test.go`:

```go
//go:build test

package character

import (
	"atlas-saga-orchestrator/saga"
	character2 "atlas-saga-orchestrator/kafka/message/character"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// The meso-error handler used to call StepCompleted(tx, false) and drop
// Body.Error on the floor, so the meso_sack_use compensator had no way to tell
// a ceiling rejection from any other failure and would have rendered the
// generic message. It must thread the code onto the step's result map, exactly
// as handleCharacterApTransferErrorEvent already does.
func TestHandleCharacterMesoErrorEventThreadsErrorCode(t *testing.T) {
	l, _ := test.NewNullLogger()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.MesoSackUse).
		SetInitiatedBy("meso-error-result-test").
		AddStep("consume_meso_sack", saga.Completed, saga.DestroyAsset, saga.DestroyAssetPayload{
			CharacterId: 4001, TemplateId: 5200000, Quantity: 1,
		}).
		AddStep("award_mesos", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
			CharacterId: 4001, ActorId: 5200000, ActorType: "ITEM", Amount: 1000000, ShowEffect: true,
		}).
		Build()
	require.NoError(t, err)
	require.NoError(t, saga.GetCache().Put(ctx, s))
	t.Cleanup(func() { saga.GetCache().Remove(ctx, tx) })

	handleCharacterMesoErrorEvent(l, ctx, character2.StatusEvent[character2.StatusEventMesoErrorBody]{
		TransactionId: tx,
		CharacterId:   4001,
		Type:          character2.StatusEventTypeError,
		Body: character2.StatusEventMesoErrorBody{
			Error:  "MESO_OVERFLOW",
			Amount: 1000000,
		},
	})

	got, ok := saga.GetCache().GetById(ctx, tx)
	require.True(t, ok, "saga must still be resolvable")
	step, ok := got.StepAt(1)
	require.True(t, ok)
	require.NotNil(t, step.Result(), "failed award_mesos step must carry a result map")
	assert.Equal(t, "MESO_OVERFLOW", step.Result()["errorCode"])
}
```

> **If any helper name in this test does not exist** (`saga.GetCache().Put`, `Saga.StepAt`, `saga.NewBuilder().AddStep`), read `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/point_reset_compensation_test.go` and `saga/accept_event_test.go` and use the exact seeding idiom they use — do not invent one. `AddStep`, `StepAt`, `Result()`, `NewBuilder()` and `MesoSackUse` are all confirmed to exist; only the cache-seeding call should need checking.

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test ./kafka/consumer/character/ -run TestHandleCharacterMesoErrorEventThreadsErrorCode -v`
Expected: FAIL — `step.Result()` is nil (the handler calls `StepCompleted`, which stores no result).

- [x] **Step 3: Thread the code**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/consumer.go`, in `handleCharacterMesoErrorEvent`, replace the final line:

```go
	// Thread the machine-readable code onto the step result so a bespoke
	// compensator (meso_sack_use) can render specific client feedback instead of
	// a generic failure. Backward compatible: every existing consumer of this
	// step ignores the result map, and NOT_ENOUGH_MESO behaviour is otherwise
	// byte-identical.
	_ = p.StepCompletedWithResult(e.TransactionId, false, map[string]any{"errorCode": e.Body.Error})
```

- [x] **Step 4: Run the test to verify it passes**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test ./kafka/consumer/character/ -run TestHandleCharacterMesoErrorEventThreadsErrorCode -v`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/consumer.go \
        services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/character/meso_error_result_test.go
git commit -m "feat(task-220): thread the meso error code onto the failed step result"
```

---

### Task 6: Orchestrator `meso_sack_use` compensator

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go` — interface block (~line 56 and ~line 120), `CompensateFailedStep` dispatch (~line 276), and new functions at the end of the point-reset section (~line 1580)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go:179-190` (`EmitSagaFailed`)
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/meso_sack_compensation_test.go` (create)

**Interfaces:**
- Consumes: `saga.MesoSackUse` (Task 3), the `{"errorCode": ...}` result map (Task 5), the saga shape built in Task 8 (`consume_meso_sack` → `DestroyAsset`, `award_mesos` → `AwardMesos`).
- Produces:
  - `(*CompensatorImpl).compensateMesoSackUse(s Saga, failedStep Step[any]) error`
  - `(*CompensatorImpl).DispatchMesoSackRollbacks(s Saga)` (exported; tests drive it directly to avoid the Kafka path)
  - `mesoSackCharacterId(s Saga) uint32`, `mesoSackErrorCode(failedStep Step[any]) string`
  - `EmitSagaFailed` now resolves a real `characterId` for `MesoSackUse` sagas.

**Why `EmitSagaFailed` gains an arm rather than the compensator calling `EmitSagaFailedByIds` directly (design §5.2 refinement):** `EmitSagaFailed` populates `characterId` from `ExtractCharacterCreationIds`, which returns `0` for any saga without a `CreateCharacter` step (`saga/producer.go:138-152`). The channel resolves the session by that id, so `0` means the player gets silence *and* stays input-locked. `compensateNoteSend` works around this by calling `EmitSagaFailedByIds` directly — but the **timeout** path (`saga/timer.go:142`) calls `EmitSagaFailed`, so a compensator-local workaround leaves timed-out sacks unnotified. Putting the arm in `EmitSagaFailed` — exactly where the `MtsOperation` arm already lives — fixes both entry points at once.

- [x] **Step 1: Write the failing tests**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/meso_sack_compensation_test.go`:

```go
//go:build test

package saga

import (
	compmock "atlas-saga-orchestrator/compartment/mock"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	mesoSackCharId  = uint32(77001)
	mesoSackItemId  = uint32(5200000)
	mesoSackAmount  = int32(1000000)
	mesoSackWorldId = world.Id(0)
	mesoSackChannel = channel.Id(1)
)

func newMesoSackSaga(t *testing.T, tx uuid.UUID, destroyStatus Status) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(MesoSackUse).
		SetInitiatedBy("meso-sack-compensation-test").
		AddStep("consume_meso_sack", destroyStatus, DestroyAsset, DestroyAssetPayload{
			CharacterId: mesoSackCharId,
			TemplateId:  mesoSackItemId,
			Quantity:    1,
			RemoveAll:   false,
		}).
		AddStep("award_mesos", Failed, AwardMesos, AwardMesosPayload{
			CharacterId: mesoSackCharId,
			WorldId:     mesoSackWorldId,
			ChannelId:   mesoSackChannel,
			ActorId:     mesoSackItemId,
			ActorType:   "ITEM",
			Amount:      mesoSackAmount,
			ShowEffect:  true,
		}).
		Build()
	require.NoError(t, err)
	return s
}

// A failed award must refund the already-consumed sack — exactly once, with the
// destroyed template and quantity. Without this the player pays a cash item for
// nothing.
func TestMesoSackCompensationRefundsSack(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	type createCall struct {
		CharacterId uint32
		TemplateId  uint32
		Quantity    uint32
	}
	var calls []createCall
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, characterId uint32, templateId uint32, quantity uint32, _ time.Time) error {
			calls = append(calls, createCall{characterId, templateId, quantity})
			return nil
		},
	}

	s := newMesoSackSaga(t, uuid.New(), Completed)
	NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP).
		DispatchMesoSackRollbacks(s)

	require.Len(t, calls, 1, "consumed sack must be refunded exactly once")
	assert.Equal(t, mesoSackCharId, calls[0].CharacterId)
	assert.Equal(t, mesoSackItemId, calls[0].TemplateId)
	assert.Equal(t, uint32(1), calls[0].Quantity)
}

// A consume step that never completed committed nothing and has no inverse.
func TestMesoSackCompensationSkipsUncompletedConsume(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var count int
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			count++
			return nil
		},
	}

	s := newMesoSackSaga(t, uuid.New(), Failed)
	NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP).
		DispatchMesoSackRollbacks(s)

	assert.Equal(t, 0, count, "an uncompleted destroy must not be inverted")
}

// THE regression guard for the characterId-0 bug: EmitSagaFailed's default
// extractor only recognizes a CreateCharacter step, so a meso_sack_use failure
// would emit characterId 0, the channel's session lookup would miss, and the
// player would get silence AND stay input-locked.
func TestMesoSackFailedEventCarriesRealCharacterIdAndErrorCode(t *testing.T) {
	logger, _ := test.NewNullLogger()

	type emitted struct {
		SagaType    string
		CharacterId uint32
		ErrorCode   string
		FailedStep  string
	}
	var got []emitted
	restore := SetEmitSagaFailedForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, sagaType string, _ uint32, characterId uint32, errorCode string, _ string, failedStep string) error {
		got = append(got, emitted{sagaType, characterId, errorCode, failedStep})
		return nil
	})
	t.Cleanup(func() { SetEmitSagaFailedForTest(restore) })

	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error { return nil },
	}

	ctx := testTenantContext()
	tx := uuid.New()
	s := newMesoSackSaga(t, tx, Completed)
	require.NoError(t, GetCache().Put(ctx, s))
	t.Cleanup(func() { GetCache().Remove(ctx, tx) })
	require.True(t, GetCache().TryTransition(ctx, tx, SagaLifecyclePending, SagaLifecycleCompensating))

	failedStep, ok := s.StepAt(1)
	require.True(t, ok)
	failedStep = failedStep.WithResult(map[string]any{"errorCode": "MESO_OVERFLOW"})

	c := NewCompensator(logger, ctx).WithCompartmentProcessor(compP)
	require.NoError(t, c.(*CompensatorImpl).compensateMesoSackUse(s, failedStep))

	require.Len(t, got, 1, "exactly one saga-failed emission")
	assert.Equal(t, "meso_sack_use", got[0].SagaType)
	assert.Equal(t, mesoSackCharId, got[0].CharacterId, "characterId must be the payload's, never 0")
	assert.Equal(t, "MESO_OVERFLOW", got[0].ErrorCode)
	assert.Equal(t, "award_mesos", got[0].FailedStep)
}
```

> **Two helper calls to confirm before writing:** `Step[any].WithResult(...)` and `GetCache().Put(ctx, s)`. If either does not exist under that name, read `saga/model.go` / `saga/cache.go` and use the real one; if a step's result cannot be set from outside, seed the saga through the cache and drive `StepCompletedWithResult` instead (the idiom `saga/late_event_integration_test.go` uses). Do not fabricate an API.

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test ./saga/ -run TestMesoSack -v`
Expected: FAIL — `DispatchMesoSackRollbacks` / `compensateMesoSackUse` undefined.

- [x] **Step 3: Declare both on the `Compensator` interface**

In `saga/compensator.go`, add to the unexported-compensator list (after `compensatePointReset`):

```go
	compensatePointReset(s Saga, failedStep Step[any]) error
	compensateMesoSackUse(s Saga, failedStep Step[any]) error
```

and after the `DispatchPointResetRollbacks` declaration:

```go
	// DispatchMesoSackRollbacks reverse-walks the completed steps of a
	// meso_sack_use saga and refunds the consumed sack (DestroyAsset →
	// CreateItem). The failed award_mesos step committed nothing
	// (RequestChangeMeso rejects inside its own transaction) and has no
	// inverse. No lifecycle transitions, no Failed emission, no cache
	// eviction — callers handle those.
	DispatchMesoSackRollbacks(s Saga)
```

- [x] **Step 4: Add the dispatch arm**

In `CompensateFailedStep`, immediately after the `PointReset` arm:

```go
	// Meso-sack reverse-walk. Destroy-first, like point_reset: invert the
	// completed consume_meso_sack via re-award, then emit the saga-failed event
	// carrying the character id (EmitSagaFailed's meso-sack arm) and the
	// threaded error code, so atlas-channel can render the ceiling message and
	// release the client's exclusive-request gate.
	if s.SagaType() == MesoSackUse {
		return c.compensateMesoSackUse(s, failedStep)
	}
```

- [x] **Step 5: Implement the compensator**

Append after `DispatchPointResetRollbacks` in `saga/compensator.go`:

```go
// compensateMesoSackUse is the meso_sack_use reverse-walk compensator: on a
// failed award_mesos it refunds the already-consumed sack and emits exactly one
// StatusEventTypeFailed carrying atlas-character's machine-readable error code
// (threaded off the failed step's result map by handleCharacterMesoErrorEvent).
// TryTransition(Compensating → Failed) guards against a double-emit where the
// timeout backstop already emitted Failed.
func (c *CompensatorImpl) compensateMesoSackUse(s Saga, failedStep Step[any]) error {
	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"failed_step":    failedStep.StepId(),
		"failed_action":  failedStep.Action(),
		"tenant_id":      c.t.Id().String(),
	}).Info("MesoSackUse saga failing — dispatching reverse-walk compensation.")

	c.DispatchMesoSackRollbacks(s)

	if !GetCache().TryTransition(c.ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed) {
		c.l.WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Info("saga already in terminal Failed state; meso-sack emission skipped.")
		SagaTimers().Cancel(s.TransactionId())
		GetCache().Remove(c.ctx, s.TransactionId())
		return nil
	}

	SagaTimers().Cancel(s.TransactionId())
	GetCache().Remove(c.ctx, s.TransactionId())

	errorCode := mesoSackErrorCode(failedStep)
	reason := fmt.Sprintf("Meso sack use failed at step [%s] action [%s]", failedStep.StepId(), failedStep.Action())
	if err := EmitSagaFailed(c.l, c.ctx, s, errorCode, reason, failedStep.StepId()); err != nil {
		c.l.WithError(err).WithFields(logrus.Fields{
			"transaction_id": s.TransactionId().String(),
			"tenant_id":      c.t.Id().String(),
		}).Error("Failed to emit saga failed event after meso-sack compensation.")
		return err
	}

	c.l.WithFields(logrus.Fields{
		"transaction_id": s.TransactionId().String(),
		"tenant_id":      c.t.Id().String(),
	}).Info("Meso-sack reverse-walk compensation complete; saga terminated.")
	return nil
}

// mesoSackErrorCode reads the machine-readable code atlas-character supplied
// (MESO_OVERFLOW / NOT_ENOUGH_MESO) off the failed step's result map. A
// destroy-step failure or a timeout has no such map, so the channel renders the
// generic message rather than falsely claiming a meso ceiling.
func mesoSackErrorCode(failedStep Step[any]) string {
	if res := failedStep.Result(); res != nil {
		if v, ok := res["errorCode"].(string); ok && v != "" {
			return v
		}
	}
	return sagaMsg.ErrorCodeUnknown
}

// mesoSackCharacterId resolves the character to notify. The AwardMesos payload
// is present on every meso_sack_use saga by construction; the DestroyAsset
// payload is the belt-and-braces fallback (same shape as compensateNoteSend).
func mesoSackCharacterId(s Saga) uint32 {
	for _, step := range s.Steps() {
		if step.Action() == AwardMesos {
			if id := ExtractCharacterId(step); id != 0 {
				return id
			}
		}
	}
	for _, step := range s.Steps() {
		if id := ExtractCharacterId(step); id != 0 {
			return id
		}
	}
	return 0
}

// DispatchMesoSackRollbacks reverse-walks the saga's completed steps and
// refunds the consumed sack (DestroyAsset → RequestCreateItem). Pure dispatch
// half — no lifecycle transitions, no event emission, no cache eviction. Only
// Completed destroy steps are inverted; the failed award step committed nothing
// and has no inverse. The refund lands in the first free CASH slot, matching
// every other refund path — DestroyAsset is template-keyed, not slot-keyed.
// An error refunding one step does not abort the walk.
func (c *CompensatorImpl) DispatchMesoSackRollbacks(s Saga) {
	steps := s.Steps()
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		if step.Status() != Completed {
			continue
		}
		if step.Action() != DestroyAsset {
			continue
		}
		if payload, ok := step.Payload().(DestroyAssetPayload); ok {
			qty := payload.Quantity
			if qty == 0 {
				qty = 1
			}
			if err := c.compP.RequestCreateItem(s.TransactionId(), payload.CharacterId, payload.TemplateId, qty, time.Time{}); err != nil {
				c.l.WithError(err).WithFields(logrus.Fields{
					"transaction_id": s.TransactionId().String(),
					"step_id":        step.StepId(),
					"template_id":    payload.TemplateId,
				}).Error("Reverse-walk: meso sack DestroyAsset → CreateItem dispatch failed; continuing chain.")
			}
		}
	}
}
```

- [x] **Step 6: Give `EmitSagaFailed` a meso-sack arm**

In `saga/producer.go`, inside `EmitSagaFailed`, immediately after the `MtsOperation` arm:

```go
	// A meso_sack_use saga carries no CharacterCreation ids either. Without this
	// arm the FAILED event has characterId 0, the channel's session lookup
	// misses, and the player gets no message AND stays behind the client's
	// exclusive-request gate. Covers both entry points — the compensator and the
	// timeout backstop in timer.go, which also calls EmitSagaFailed.
	if s.SagaType() == MesoSackUse {
		return EmitSagaFailedByIds(l, ctx, s.TransactionId(), string(s.SagaType()), 0, mesoSackCharacterId(s), errorCode, reason, failedStep)
	}
```

- [x] **Step 7: Run the tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test ./saga/ -run TestMesoSack -v`
Expected: PASS (all three).

- [x] **Step 8: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/compensator.go \
        services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/producer.go \
        services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/meso_sack_compensation_test.go
git commit -m "feat(task-220): meso_sack_use compensator refunds the sack and emits a real characterId"
```

---

### Task 7: Classify `meso_sack_use` for the timeout backstop

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go:169-208` (the three classification lists) and `:216-250` (`dispatchTimeoutRollbacks`)

**Interfaces:**
- Consumes: `MesoSackUse` (Task 3), `DispatchMesoSackRollbacks` (Task 6).
- Produces: nothing new — closes a silent value-destroying hole.

**Why this task exists (not in design.md):** `saga/timer.go` maintains `reverseWalkSagaTypes`, `noReverseWalkSagaTypes`, `allSagaTypes` and a `dispatchTimeoutRollbacks` switch. Its own doc comment records the exact bug this prevents: a type routed to a bespoke compensator by `CompensateFailedStep` but absent from these lists rolls back **nothing** on timeout — the sack is consumed, the mesos never arrive, and no compensation runs. `TestEverySagaTypeIsClassified` will also fail once `MesoSackUse` exists and neither list names it.

- [x] **Step 1: Run the existing classification test to see it fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run TestEverySagaTypeIsClassified -v`
Expected: this may already PASS if the test only iterates `allSagaTypes`. Record the actual result before proceeding — if it passes, Step 2's new assertion is what makes the gap visible.

- [x] **Step 2: Write the failing test**

Append to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/meso_sack_compensation_test.go`:

```go
// timer.go's own doc comment records the defect this pins: a saga type routed
// to a bespoke compensator by CompensateFailedStep but missing from the timeout
// lists rolls back NOTHING when it times out. For meso_sack_use that means a
// consumed sack, no mesos, and no refund.
func TestMesoSackUseIsClassifiedForTimeout(t *testing.T) {
	var inAll, inReverse bool
	for _, ty := range allSagaTypes {
		if ty == MesoSackUse {
			inAll = true
		}
	}
	for _, ty := range reverseWalkSagaTypes {
		if ty == MesoSackUse {
			inReverse = true
		}
	}
	assert.True(t, inAll, "MesoSackUse must appear in allSagaTypes")
	assert.True(t, inReverse, "MesoSackUse must appear in reverseWalkSagaTypes")
}

// And the switch must actually have an arm — a type in the list with no case
// falls to default and returns false, dispatching no inverse.
func TestMesoSackTimeoutDispatchesRefund(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var count int
	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			count++
			return nil
		},
	}
	_ = compP

	s := newMesoSackSaga(t, uuid.New(), Completed)
	if !dispatchTimeoutRollbacks(logger, testTenantContext(), s) {
		t.Fatal("dispatchTimeoutRollbacks returned false: meso_sack_use has no timeout arm")
	}
}
```

> `dispatchTimeoutRollbacks` builds its own `NewCompensator(l, ctx)` with no compartment processor, so this test asserts only that the arm is reached (the return value), not the refund call count — that is already covered by `TestMesoSackCompensationRefundsSack`. Keep the `compP`/`count` locals out if the compiler flags them as unused; the assertion that matters is the `true` return.

- [x] **Step 3: Run the tests to verify they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test ./saga/ -run 'TestMesoSackUseIsClassified|TestMesoSackTimeoutDispatches' -v`
Expected: FAIL — `MesoSackUse must appear in allSagaTypes`, and `dispatchTimeoutRollbacks returned false`.

- [x] **Step 4: Classify the type**

In `saga/timer.go`, add `MesoSackUse` to `reverseWalkSagaTypes` (after `SkillBookUse`):

```go
	NoteSend,
	SkillBookUse,
	MesoSackUse,
}
```

and to `allSagaTypes`:

```go
var allSagaTypes = []Type{
	InventoryTransaction, QuestReward, TradeTransaction, TradeStaging,
	CharacterCreation, StorageOperation, CharacterRespawn, GachaponTransaction,
	PetEvolution, ItemTagUse, SealingLockUse, IncubatorUse, PointReset,
	MtsOperation, NoteSend, SkillBookUse, MesoSackUse,
}
```

- [x] **Step 5: Add the timeout dispatch arm**

In `dispatchTimeoutRollbacks`, after the `SkillBookUse` case:

```go
	case SkillBookUse:
		c.DispatchSkillBookUseRollbacks(s)
	case MesoSackUse:
		// Without this a timed-out sack use is pure loss: consume_meso_sack
		// completed, award_mesos never landed, and nothing puts the sack back.
		c.DispatchMesoSackRollbacks(s)
```

- [x] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test ./saga/ -run 'TestMesoSack|TestEverySagaTypeIsClassified' -v`
Expected: PASS.

- [x] **Step 7: Run the full module test suite, both tag sets**

Run:
```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && \
  go test -race ./... && go test -tags=test -race ./... && go vet ./...
```
Expected: all PASS, vet silent.

- [x] **Step 8: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/timer.go \
        services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/meso_sack_compensation_test.go
git commit -m "feat(task-220): classify meso_sack_use for the saga timeout reverse walk"
```

---

### Task 8: `atlas-channel` type-19 branch

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` — add `CashSlotItemTypeCurrencySack` to the const block (~line 660), use it at `GetCashSlotItemType`'s classification-520 return (~line 894), and add the dispatch arm beside the point-reset arm (~line 482)
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_meso_sack.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_meso_sack_test.go` (create)

**Interfaces:**
- Consumes: `cd.Meso` (Task 2), `saga.MesoSackUse` + `saga.AwardMesosPayload` + `saga.DestroyAssetPayload` (Task 3).
- Produces:
  - `CashSlotItemTypeCurrencySack = CashSlotItemType(19)`
  - `buildMesoSackUseSaga(transactionId uuid.UUID, now time.Time, characterId uint32, itemId item.Id, worldId world.Id, channelId channel.Id, amount int32) saga.Saga`
  - `handleMesoSackUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id)`
  - `cashItemDataFunc` — package-var test seam for the cash-data lookup

- [x] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_meso_sack_test.go`:

```go
package handler

import (
	cashData "atlas-channel/data/cash"
	"atlas-channel/saga"
	sagaMsg "atlas-channel/kafka/message/saga"
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// installCashItemDataSeam swaps the cash-data lookup for the test and returns a
// restore func (same package-var injection precedent as installCashItemInSlotSeam).
func installCashItemDataSeam(t *testing.T, meso uint32, err error) func() {
	t.Helper()
	orig := cashItemDataFunc
	cashItemDataFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (cashData.RestModel, error) {
		if err != nil {
			return cashData.RestModel{}, err
		}
		return cashData.RestModel{Meso: meso}, nil
	}
	return func() { cashItemDataFunc = orig }
}

// The saga shape is the FR-4 invariant: destroy-first, exactly two steps, the
// award carrying ShowEffect so the meso gain renders as the chat line.
func TestBuildMesoSackUseSaga(t *testing.T) {
	txn := uuid.New()
	now := time.Now()
	s := buildMesoSackUseSaga(txn, now, 4242, item.Id(5200000), world.Id(0), channel.Id(1), 1000000)

	if s.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", s.TransactionId, txn)
	}
	if s.SagaType != saga.MesoSackUse {
		t.Errorf("sagaType: got %s, want %s", s.SagaType, saga.MesoSackUse)
	}
	if len(s.Steps) != 2 {
		t.Fatalf("steps: got %d, want 2", len(s.Steps))
	}

	if s.Steps[0].StepId != "consume_meso_sack" {
		t.Errorf("step 1 id: got %q, want %q", s.Steps[0].StepId, "consume_meso_sack")
	}
	if s.Steps[0].Action != saga.DestroyAsset {
		t.Errorf("step 1 action: got %s, want %s (destroy-first is mandatory)", s.Steps[0].Action, saga.DestroyAsset)
	}
	dp, ok := s.Steps[0].Payload.(saga.DestroyAssetPayload)
	if !ok {
		t.Fatalf("step 1 payload type: %T", s.Steps[0].Payload)
	}
	if dp.CharacterId != 4242 || dp.TemplateId != 5200000 || dp.Quantity != 1 || dp.RemoveAll {
		t.Errorf("destroy payload mismatch: %+v", dp)
	}

	if s.Steps[1].StepId != "award_mesos" {
		t.Errorf("step 2 id: got %q, want %q", s.Steps[1].StepId, "award_mesos")
	}
	if s.Steps[1].Action != saga.AwardMesos {
		t.Errorf("step 2 action: got %s, want %s", s.Steps[1].Action, saga.AwardMesos)
	}
	ap, ok := s.Steps[1].Payload.(saga.AwardMesosPayload)
	if !ok {
		t.Fatalf("step 2 payload type: %T", s.Steps[1].Payload)
	}
	if ap.CharacterId != 4242 || ap.Amount != 1000000 || !ap.ShowEffect {
		t.Errorf("award payload mismatch: %+v", ap)
	}
	if ap.ActorId != 5200000 || ap.ActorType != "ITEM" {
		t.Errorf("actor mismatch: got %d/%q, want 5200000/ITEM", ap.ActorId, ap.ActorType)
	}
	if ap.WorldId != world.Id(0) || ap.ChannelId != channel.Id(1) {
		t.Errorf("field mismatch: got %d/%d, want 0/1", ap.WorldId, ap.ChannelId)
	}
}

// A resolvable, in-range amount creates exactly one saga and announces nothing:
// the success unlock rides on atlas-character's STAT_CHANGED{Meso}, which
// already carries ExclRequestSent. An extra empty StatChanged here would race it.
func TestMesoSackUseCreatesSagaAndAnnouncesNothing(t *testing.T) {
	const characterId = uint32(4242)
	const itemId = uint32(5200000)

	restoreData := installCashItemDataSeam(t, 1000000, nil)
	defer restoreData()
	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	handleMesoSackUse(logrus.New(), ctx, rec.producer())(s, item.Id(itemId))

	if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 1 {
		t.Fatalf("emitted %d saga commands, want exactly 1", got)
	}
	if rec.calls != 0 {
		t.Fatalf("announced %d packets on the success path, want 0", rec.calls)
	}
}

// Fail closed: a zero amount (Maple Point sack 5200009, or a tenant whose WZ was
// not re-ingested) must consume nothing, award nothing, and unlock the client.
// Same for an amount past the int32 ceiling of AwardMesosPayload.Amount — without
// that guard a large sack wraps negative and TAKES mesos — and for a lookup error.
func TestMesoSackUseRejectsAndUnlocks(t *testing.T) {
	cases := []struct {
		name string
		meso uint32
		err  error
	}{
		{"zero amount (maple point sack)", 0, nil},
		{"above int32 ceiling", uint32(math.MaxInt32) + 1, nil},
		{"cash data lookup failure", 0, errors.New("404")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restoreData := installCashItemDataSeam(t, tc.meso, tc.err)
			defer restoreData()
			captured, restoreProducer := installCapturingProducer()
			defer restoreProducer()

			s, ctx, cleanup := newCashItemUseTestSession(t, 4242)
			defer cleanup()

			rec := &gaugeProducerRecorder{}
			handleMesoSackUse(logrus.New(), ctx, rec.producer())(s, item.Id(5200009))

			if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 0 {
				t.Fatalf("emitted %d saga commands, want 0 (nothing may be consumed)", got)
			}
			if rec.calls != 1 {
				t.Fatalf("announced %d packets, want exactly 1 (the enable-actions unlock)", rec.calls)
			}
		})
	}
}

// The constant must stay 19 for every version: Atlas derives the type from the
// server-resolved template id and the type never rides the wire, so the v48 (17)
// and v61 (18) client tables are irrelevant. A version gate here would break the
// branch on those builds (design §3.1(a)).
func TestCurrencySackTypeIsNineteenOnEveryVersion(t *testing.T) {
	for _, v := range []struct {
		region string
		major  uint16
		minor  uint16
	}{
		{"GMS", 48, 1}, {"GMS", 61, 1}, {"GMS", 72, 1}, {"GMS", 79, 1}, {"GMS", 83, 1},
		{"GMS", 84, 1}, {"GMS", 87, 1}, {"GMS", 92, 1}, {"GMS", 95, 1}, {"JMS", 185, 1},
	} {
		ten := mustTenant(t, v.region, v.major, v.minor)
		for _, id := range []uint32{5200000, 5200001, 5200002, 5202000} {
			if got := GetCashSlotItemType(ten)(item.Id(id)); got != CashSlotItemTypeCurrencySack {
				t.Errorf("%s v%d: GetCashSlotItemType(%d) = %d, want %d", v.region, v.major, id, got, CashSlotItemTypeCurrencySack)
			}
		}
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'MesoSack|CurrencySack' -v`
Expected: FAIL — `undefined: cashItemDataFunc`, `undefined: buildMesoSackUseSaga`, `undefined: handleMesoSackUse`, `undefined: CashSlotItemTypeCurrencySack`.

- [x] **Step 3: Add the constant and use it**

In `character_cash_item_use.go`, add to the `CashSlotItemType` const block (keep it with the other named types, before the point-reset comment block):

```go
	CashSlotItemTypeCube          = CashSlotItemType(74)
	// CashSlotItemTypeCurrencySack is classification 520 (meso sacks). Atlas
	// returns 19 on EVERY version even though the v48 client's own table says
	// 17 and v61's says 18: the type is derived from the server-resolved
	// template id and never rides the wire, and no other classification maps to
	// 19 here. Do NOT version-gate this (design §3.1(a)).
	CashSlotItemTypeCurrencySack = CashSlotItemType(19)
```

and replace the classification-520 return in `GetCashSlotItemType`:

```go
		if category == item.ClassificationCurrencySack {
			return CashSlotItemTypeCurrencySack
		}
```

- [x] **Step 4: Add the dispatch arm**

In `CharacterCashItemUseHandleFunc`, immediately after the point-reset arm and before the vicious-hammer arm:

```go
		if it == CashSlotItemTypeCurrencySack {
			// No sub-body: the classification-520 arm of
			// CWvsContext::SendConsumeCashItemUseRequest encodes nothing beyond
			// the common header on all ten versions (design §3, per-version
			// addresses). Nothing to decode off r.
			handleMesoSackUse(l, ctx, wp)(s, itemId)
			return
		}
```

- [x] **Step 5: Create the arm**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_meso_sack.go`:

```go
package handler

import (
	cashData "atlas-channel/data/cash"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
)

// cashItemDataFunc is a test seam for the cash-item data lookup (package-var
// injection precedent: cashItemInSlotFunc in character_cash_item_use.go).
var cashItemDataFunc = func(l logrus.FieldLogger, ctx context.Context, itemId uint32) (cashData.RestModel, error) {
	return cashData.NewProcessor(l, ctx).GetById(itemId)
}

// buildMesoSackUseSaga assembles the meso_sack_use saga: destroy-first, exactly
// two steps. The sack is consumed by TEMPLATE, not by slot — the pre-branch
// guard in CharacterCashItemUseHandleFunc already proved the named CASH slot
// holds this template, the orchestrator's inverse for DestroyAsset is a plain
// RequestCreateItem, and a refund landing in the first free CASH slot matches
// every other refund path in the system.
func buildMesoSackUseSaga(transactionId uuid.UUID, now time.Time, characterId uint32, itemId item.Id, worldId world.Id, channelId channel.Id, amount int32) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.MesoSackUse,
		InitiatedBy:   "CASH_ITEM_USE",
		Steps: []saga.Step{
			{
				StepId: "consume_meso_sack",
				Status: saga.Pending,
				Action: saga.DestroyAsset,
				Payload: saga.DestroyAssetPayload{
					CharacterId: characterId,
					TemplateId:  uint32(itemId),
					Quantity:    1,
					RemoveAll:   false,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				StepId: "award_mesos",
				Status: saga.Pending,
				Action: saga.AwardMesos,
				Payload: saga.AwardMesosPayload{
					CharacterId: characterId,
					WorldId:     worldId,
					ChannelId:   channelId,
					ActorId:     uint32(itemId),
					ActorType:   "ITEM",
					Amount:      amount,
					ShowEffect:  true,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// handleMesoSackUse implements the CashSlotItemType 19 arm: classification 520
// meso sacks (5200000/5200001/5200002 on every version, plus the 5202xxx random
// family on v92/v95/JMS, which this pays at its flat info/meso value).
//
// The payout is resolved server-side from the WZ data keyed by the
// server-resolved template id — no client-supplied value influences it.
//
// Nothing is announced on the success path: the award's STAT_CHANGED{Meso} from
// atlas-character already carries ExclRequestSent, so that packet both renders
// the new balance and releases the client's exclusive-request gate, correctly
// ordered by construction.
func handleMesoSackUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id) {
	return func(s session.Model, itemId item.Id) {
		enableActions := func() {
			_ = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
		}

		cd, err := cashItemDataFunc(l, ctx, uint32(itemId))
		if err != nil {
			l.WithError(err).Warnf("Character [%d] used meso sack [%d] but its cash item data could not be resolved. Rejecting.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		// Fail closed. Zero covers both a Maple Point sack (5200009/5200010 carry
		// info/maplepoint and no info/meso) and a tenant whose WZ has not been
		// re-ingested since the Meso field was added. The ceiling covers the
		// int32 width of AwardMesosPayload.Amount: a larger value would wrap
		// negative and DEDUCT mesos. Never drop this guard.
		if cd.Meso == 0 {
			l.Warnf("Character [%d] used meso sack [%d] with no info/meso amount. Rejecting; nothing consumed.", s.CharacterId(), itemId)
			enableActions()
			return
		}
		if cd.Meso > uint32(math.MaxInt32) {
			l.Warnf("Character [%d] used meso sack [%d] whose amount [%d] exceeds the int32 award ceiling. Rejecting; nothing consumed.", s.CharacterId(), itemId, cd.Meso)
			enableActions()
			return
		}

		f := s.Field()
		l.Debugf("Character [%d] using meso sack [%d] for [%d] mesos.", s.CharacterId(), itemId, cd.Meso)
		_ = saga.NewProcessor(l, ctx).Create(buildMesoSackUseSaga(uuid.New(), time.Now(), s.CharacterId(), itemId, f.WorldId(), f.ChannelId(), int32(cd.Meso)))
	}
}
```

- [x] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'MesoSack|CurrencySack' -v`
Expected: PASS (all four).

- [x] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_meso_sack.go \
        services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_meso_sack_test.go
git commit -m "feat(task-220): implement the cash-slot type 19 meso sack branch"
```

---

### Task 9: `atlas-channel` renders the meso-sack failure

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go` — add a `mesoSackFailureMessage` mapper and a `SagaTypeMesoSackUse` arm in `handleFailedEvent` (beside the `SagaTypePointReset` arm, ~line 348)
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer_test.go` (append)

**Interfaces:**
- Consumes: `saga.SagaTypeMesoSackUse`, `saga.ErrorCodeMesoOverflow` (Task 3); the saga-failed event's `Body.CharacterId` (now non-zero thanks to Task 6) and `Body.ErrorCode` (Task 5).
- Produces: `mesoSackFailureMessage(errorCode string) string`.

- [x] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer_test.go`:

```go
// A meso_sack_use saga can fail three ways: the ceiling rejection, a destroy
// failure, and a timeout. Only the first may claim the meso limit as the reason
// — saying "you cannot hold any more mesos" after a timeout would be a lie.
func TestMesoSackFailureMessage(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{saga.ErrorCodeMesoOverflow, "You cannot hold any more mesos."},
		{saga.ErrorCodeUnknown, "You are unable to use this item right now."},
		{"SAGA_TIMEOUT", "You are unable to use this item right now."},
		{"", "You are unable to use this item right now."},
	}
	for _, tc := range cases {
		if got := mesoSackFailureMessage(tc.code); got != tc.want {
			t.Errorf("mesoSackFailureMessage(%q) = %q, want %q", tc.code, got, tc.want)
		}
	}
}
```

- [x] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/saga/ -run TestMesoSackFailureMessage -v`
Expected: FAIL — `undefined: mesoSackFailureMessage`.

- [x] **Step 3: Add the mapper**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go`, beside `getStorageErrorBodyProducer`:

```go
// mesoSackFailureMessage maps a meso_sack_use saga's errorCode to the pink text
// the player sees. Only MESO_OVERFLOW names the meso ceiling: the same saga can
// also fail on the destroy step or by timeout (SAGA_TIMEOUT), and claiming a
// ceiling then would be a lie.
func mesoSackFailureMessage(errorCode string) string {
	if errorCode == saga.ErrorCodeMesoOverflow {
		return "You cannot hold any more mesos."
	}
	return "You are unable to use this item right now."
}
```

- [x] **Step 4: Add the failure arm**

In `handleFailedEvent`, immediately after the `SagaTypePointReset` arm:

```go
		// Meso-sack failures: the orchestrator's compensator has already refunded
		// the sack. Tell the player why, then release the client's
		// exclusive-request gate — this is the ONLY unlock on the failure path
		// (the success path is unlocked by the award's STAT_CHANGED instead).
		if e.Body.SagaType == saga.SagaTypeMesoSackUse {
			msg := mesoSackFailureMessage(e.Body.ErrorCode)
			err = session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send meso-sack pink text.")
			}
			err = session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)(s)
			if err != nil {
				l.WithError(err).WithField("character_id", e.Body.CharacterId).Error("Failed to send enable-actions after meso-sack failure.")
			}
			return
		}
```

- [x] **Step 5: Run the test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/saga/ -v`
Expected: PASS (the new test and the four pre-existing ones).

- [x] **Step 6: Run the full module test + vet + build**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...`
Expected: all PASS, vet silent, build clean.

- [x] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer.go \
        services/atlas-channel/atlas.com/channel/kafka/consumer/saga/consumer_test.go
git commit -m "feat(task-220): render the meso-sack failure message and unlock the client"
```

---

### Task 10: Full verification, rollout notes, and code review

**Files:**
- Create: `docs/tasks/task-220-meso-sack-cash-item/rollout.md`

**Interfaces:**
- Consumes: everything above.
- Produces: a committed rollout/verification record.

- [x] **Step 1: Run every changed module's tests and vet**

Run from the worktree root:
```bash
for m in services/atlas-data/atlas.com/data \
         services/atlas-channel/atlas.com/channel \
         services/atlas-character/atlas.com/character \
         services/atlas-saga-orchestrator/atlas.com/saga-orchestrator \
         libs/atlas-saga; do
  echo "=== $m ==="
  ( cd "$m" && go build ./... && go vet ./... && go test -race ./... ) || exit 1
done
```
Expected: every module clean.

- [x] **Step 2: Run the tag-gated tests**

Run:
```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -tags=test -race ./...
```
Expected: PASS. (`go test -race ./...` alone does not compile the `//go:build test` files, so Tasks 5–7's tests only run here.)

- [x] **Step 3: Run the repo guards**

Run from the worktree root:
```bash
tools/redis-key-guard.sh && \
tools/goroutine-guard.sh && \
tools/skill-job-id-guard.sh && \
tools/buff-duration-guard.sh && \
tools/lint.sh --check
```
Expected: all exit 0. If `tools/lint.sh --check` reports formatting diffs, run `tools/lint.sh` (no flags) to fix in place, re-run `--check`, and amend the affected commit.

The template guards (`template-opcode-order`, `template-duplicate-binding`, `template-movement-types`), `service-registration-guard.sh`, `trade-contract-mirror-guard.sh` and `docker buildx bake` are **not** required: no template, no `services.json`/`deploy/k8s`/`docker-bake.hcl`/`go.work`, no trade contract, and no `go.mod` changed. Confirm that with `git diff --name-only main...HEAD` before skipping them.

- [x] **Step 4: Write the rollout record**

Create `docs/tasks/task-220-meso-sack-cash-item/rollout.md`:

```markdown
# task-220 Rollout — Meso Sack Cash Item

## Hard prerequisite: WZ re-ingest per tenant

`atlas-data` stores cash items as JSONB documents (`document.Storage[string, RestModel]`,
kind `CASH`). The new `meso` field is additive: **no existing document gains it on
deploy.** Until a tenant's WZ is re-ingested, `meso` is absent, the handler's
fail-closed guard trips, and every sack use is a logged rejection (loud in logs,
invisible to a smoke test that never uses a sack).

Re-ingest each tenant version, then verify:

    GET /api/data/cash/items/5200000
    GET /api/data/cash/items/5200001
    GET /api/data/cash/items/5200002

Each must return a non-zero `meso`. Record the observed values here per tenant —
do not assume they match GMS 83.1 (1,000,000 / 5,000,000 / 10,000,000); the
per-version amounts were unverified at design time.

| Tenant | 5200000 | 5200001 | 5200002 | Re-ingested |
|---|---|---|---|---|
| gms_v48 | | | | |
| gms_v61 | | | | |
| gms_v72 | | | | |
| gms_v79 | | | | |
| gms_v83 | | | | |
| gms_v84 | | | | |
| gms_v87 | | | | |
| gms_v92 | | | | |
| gms_v95 | | | | |
| jms_v185 | | | | |

Also record, where the item exists: `5202000` (v92/v95/JMS — pays its flat
`info/meso`; the client shows an amount-less "random" prompt on v92/v95, an
accepted cosmetic divergence) and `5200009`/`5200010` (v84/v87/v92/v95 — Maple
Point sacks; these must show no `meso` and must reject).

## Manual acceptance, once per tenant after re-ingest

- [ ] `5200000` on a fresh character: exactly 1 sack removed from CASH, the
      tenant's recorded amount credited, meso chat line renders, client responsive.
- [ ] `5200001` / `5200002`: same, at their recorded amounts.
- [ ] Near-ceiling character: mesos unchanged, sack still in inventory, pink text
      "You cannot hold any more mesos.", client responsive.
- [ ] `5200009` on a v87 tenant: nothing consumed, nothing awarded, warn logged
      naming the item id, client responsive.
- [ ] `5202000` on a v92 tenant: pays the flat `info/meso` amount.

## Known limitations (accepted, from the PRD's non-goals)

- Randomized payout (`mesomin`/`mesomax`/`mesostdev` on `5202000-2`) is not
  implemented; the flat `info/meso` value is paid. A gaussian roll would change
  only the handler's amount resolution — not the saga, not the wire.
- Maple Point sacks are rejected by the zero-amount guard rather than paid.
  Paying NX is a separate cash-shop concern.
- `gms_12` is out of scope: that template does not register
  `CharacterCashItemUseHandle` and no `gms_12` tenant exists.
- Whether JMS v185's `5202000` carries a base `info/meso` is unverified; the
  fail-closed guard covers it either way. The table above resolves it.
```

- [x] **Step 5: Commit the rollout record**

```bash
git add docs/tasks/task-220-meso-sack-cash-item/rollout.md
git commit -m "docs(task-220): rollout and per-tenant re-ingest verification record"
```

- [x] **Step 6: Run code review before opening the PR**

Invoke `superpowers:requesting-code-review`. Go files changed in four services plus one shared lib, so it dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no `atlas-ui` change → no frontend reviewer). Findings land in `docs/tasks/task-220-meso-sack-cash-item/audit.md`. Address them before the PR — do not skip this step even though the plan looks complete.

---

## Self-Review

**Spec coverage.** Every PRD FR and design section maps to a task:

| Requirement | Task |
|---|---|
| FR-1.1 / 1.2 / 1.3 (parse + serialize `info/meso`, absent ⇒ 0) | 1 |
| FR-1.4 (re-ingest prerequisite) | 10 (rollout.md) |
| FR-2.1 (`CashSlotItemTypeCurrencySack`) | 8 |
| FR-2.2 (no sub-body decode) | 8 |
| FR-2.3 (resolve amount via data/cash) | 8 |
| FR-2.4 (fail closed on 0 / oversized) | 8 |
| FR-2.5 (unlock on every rejection) | 8 |
| FR-2.6 (pre-branch slot/template guard unchanged) | 8 — the arm is dispatched from inside the existing guard's scope; no edit to it |
| FR-3.1 (channel cash model) | 2 |
| FR-4.1 / 4.2 (two-step saga, ShowEffect, DestroyAsset) | 8 |
| FR-4.3 (award failure compensates the consume) | 6, 7 |
| FR-4.4 (new saga type discriminates) | 3 |
| FR-5.1 (unlock on every outcome) | 8 (rejections), 9 (failure), success path unlocked by STAT_CHANGED |
| FR-5.2 (meso-ceiling message) | 9 |
| FR-5.3 (no new success writer) | 8 — nothing announced on success |
| FR-6.1 / 6.2 (wire verified, no codec, no template) | Global Constraints; design §3 |
| FR-7.1 (`MESO_OVERFLOW` emission) | 4 |
| FR-7.2 (orchestrator acceptance table) | 5 — verified unchanged at `event_acceptance.go:132`/`:327`; the errorCode threading is the only edit |
| FR-7.3 (reject, never clamp) | 4 |
| FR-8.1 (all ten versions, no raw `> N`) | 8 — `TestCurrencySackTypeIsNineteenOnEveryVersion`; no gate added anywhere |
| FR-8.2 (per-version item availability) | 10 (rollout table) |

**Additions beyond design.md, both load-bearing:**
1. **Task 7** — `saga/timer.go`'s three classification lists and `dispatchTimeoutRollbacks`. The design missed them. Its own doc comment records the exact defect: a bespoke-compensated type absent from these lists rolls back nothing on timeout, so a timed-out sack use is pure player loss.
2. **Task 6, Step 6** — the `MesoSackUse` arm goes in `EmitSagaFailed` rather than the compensator calling `EmitSagaFailedByIds` directly (design §5.2's shape). Same result for the compensator path, and it additionally fixes the timeout path, which calls `EmitSagaFailed` and would otherwise still emit `characterId: 0`.

**Placeholder scan.** No TBD/TODO. Every code step carries the literal code. Three steps carry an explicit "confirm this helper exists before writing, and if not read file X for the real idiom" instruction (Task 5 Step 1, Task 6 Step 1, Task 7 Step 2) — those are the only APIs in the plan not directly read from source during planning, and each names the file to check and forbids inventing one.

**Type consistency.** `Meso uint32` in both `RestModel`s; `Amount int32` on `AwardMesosPayload` (hence the `math.MaxInt32` guard); step ids `consume_meso_sack` / `award_mesos` identical in Tasks 6, 7 and 8; `"meso_sack_use"` identical across `libs/atlas-saga`, both re-exports and the channel's message-layer copy (pinned by a test in Task 3); `"MESO_OVERFLOW"` identical in `atlas-character`'s kafka constants and the channel's `ErrorCodeMesoOverflow` (pinned by tests in Tasks 3 and 4).
