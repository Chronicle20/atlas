# chore/todo-saga-race — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate the character-creation saga race that deletes characters when inventory compartments commit after the first `AwardAsset` step dispatches.

**Architecture:** Insert a passive `AwaitInventoryCreated` saga step between `create_character` and the first asset/equipment step. atlas-inventory's compartment-creation transaction emits `INVENTORY_STATUS.CREATED` (now carrying the saga `TransactionId`) on commit. A new consumer in atlas-saga-orchestrator on `EVENT_TOPIC_INVENTORY_STATUS` advances the await step. Mirrors the existing `AwaitCharacterCreated` shape; no new command topics.

**Tech Stack:** Go, Kafka (segmentio/kafka-go), Postgres (GORM), `libs/atlas-saga` shared lib, JSON wire format with `omitempty` for backward compatibility.

---

## Sequencing

Per `CLAUDE.md` §Build & Verification:

1. **Task 1** — `libs/atlas-saga` (pure Go, no Docker).
2. **Tasks 2–5** — `services/atlas-inventory` (additive wire-format change; `omitempty` makes it safe to deploy ahead of step 3).
3. **Tasks 6–10** — `services/atlas-saga-orchestrator` (consumes the new event type, dispatches new action).
4. **Task 11** — `services/atlas-character-factory` (emits sagas that include the new step).
5. **Task 12** — Integration verification.

No `go.mod` or `Dockerfile` modifications are expected. Verify Docker builds after each service is touched anyway (per `CLAUDE.md`).

---

## Task 1: Add `AwaitInventoryCreated` action, payload, and unmarshal case to `libs/atlas-saga`

**Files:**
- Modify: `libs/atlas-saga/model.go:134-135`
- Modify: `libs/atlas-saga/payloads.go:620` (insert after `AwaitCharacterCreatedPayload`)
- Modify: `libs/atlas-saga/unmarshal.go:401` (insert after `AwaitCharacterCreated` case)
- Test: `libs/atlas-saga/unmarshal_test.go` (add new test function)

- [ ] **Step 1: Write the failing test in `libs/atlas-saga/unmarshal_test.go`**

Append this function to the existing test file:

```go
func TestUnmarshalAwaitInventoryCreatedStep(t *testing.T) {
	raw := `{
		"stepId": "await_inventory_created-1",
		"status": "pending",
		"action": "await_inventory_created",
		"payload": {
			"characterId": 12345
		},
		"createdAt": "2026-05-15T00:00:00Z",
		"updatedAt": "2026-05-15T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if step.Action != AwaitInventoryCreated {
		t.Fatalf("expected action AwaitInventoryCreated, got %q", step.Action)
	}
	p, ok := step.Payload.(AwaitInventoryCreatedPayload)
	if !ok {
		t.Fatalf("expected AwaitInventoryCreatedPayload, got %T", step.Payload)
	}
	if p.CharacterId != 12345 {
		t.Errorf("characterId: expected 12345, got %d", p.CharacterId)
	}
}

func TestUnmarshalAwaitInventoryCreatedStep_ZeroCharacterId(t *testing.T) {
	// Mirrors the sentinel-payload shape that character-factory emits before
	// orchestrator result-forwarding substitutes the real characterId.
	raw := `{
		"stepId": "await_inventory_created-1",
		"status": "pending",
		"action": "await_inventory_created",
		"payload": {"characterId": 0},
		"createdAt": "2026-05-15T00:00:00Z",
		"updatedAt": "2026-05-15T00:00:00Z"
	}`

	var step Step[any]
	if err := json.Unmarshal([]byte(raw), &step); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	p, ok := step.Payload.(AwaitInventoryCreatedPayload)
	if !ok {
		t.Fatalf("expected AwaitInventoryCreatedPayload, got %T", step.Payload)
	}
	if p.CharacterId != 0 {
		t.Errorf("expected sentinel characterId=0, got %d", p.CharacterId)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-saga
go test -run TestUnmarshalAwaitInventoryCreatedStep ./...
```

Expected: FAIL with `undefined: AwaitInventoryCreated` and `undefined: AwaitInventoryCreatedPayload`.

- [ ] **Step 3: Add the `Action` constant to `libs/atlas-saga/model.go`**

Find the `// Character creation actions` block (line 133):

```go
	// Character creation actions
	CreateCharacter       Action = "create_character"
	AwaitCharacterCreated Action = "await_character_created"
```

Replace with:

```go
	// Character creation actions
	CreateCharacter        Action = "create_character"
	AwaitCharacterCreated  Action = "await_character_created"
	AwaitInventoryCreated  Action = "await_inventory_created"
```

- [ ] **Step 4: Add the payload struct to `libs/atlas-saga/payloads.go`**

Insert after the `AwaitCharacterCreatedPayload` struct (line 620):

```go
// AwaitInventoryCreatedPayload represents the payload required to await
// inventory-compartment creation. The orchestrator's result-forwarding
// substitutes CharacterId=0 with the actual id emitted by handleCharacterCreatedEvent.
type AwaitInventoryCreatedPayload struct {
	CharacterId uint32 `json:"characterId"`
}
```

- [ ] **Step 5: Add the unmarshal case to `libs/atlas-saga/unmarshal.go`**

Find the existing `case AwaitCharacterCreated:` block (line 396):

```go
	case AwaitCharacterCreated:
		var payload AwaitCharacterCreatedPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

Insert immediately after:

```go
	case AwaitInventoryCreated:
		var payload AwaitInventoryCreatedPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
cd libs/atlas-saga
go test -race ./...
go vet ./...
go build ./...
```

Expected: PASS, no vet warnings, build clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-saga
git commit -m "feat(atlas-saga): add AwaitInventoryCreated action and payload"
```

---

## Task 2: Add `TransactionId`, `CREATION_FAILED` constant, and body struct to atlas-inventory wire format

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go`

- [ ] **Step 1: Replace the file contents with the additive wire-format change**

Open `services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go` and replace its full contents:

```go
package inventory

import "github.com/google/uuid"

const (
	EnvEventTopicStatus           = "EVENT_TOPIC_INVENTORY_STATUS"
	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeCreationFailed = "CREATION_FAILED"
	StatusEventTypeDeleted        = "DELETED"
)

// StatusEvent is the on-wire shape of an inventory status event. TransactionId
// is added with omitempty so existing consumers (atlas-cashshop) continue to
// decode the same struct; non-saga emitters serialise without the field.
type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
}

// CreationFailedStatusEventBody carries the free-form error message for
// telemetry. The orchestrator does not inspect Reason; it only flips the
// step to Failed.
type CreationFailedStatusEventBody struct {
	Reason string `json:"reason"`
}

type DeletedStatusEventBody struct {
}
```

- [ ] **Step 2: Verify build**

```bash
cd services/atlas-inventory/atlas.com/inventory
go build ./...
go vet ./...
```

Expected: PASS. Other files in this service still compile; they reference the constants and struct names that exist.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go
git commit -m "feat(atlas-inventory): add TransactionId field and CREATION_FAILED event type"
```

---

## Task 3: Update atlas-inventory producer to carry `transactionId`

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/inventory/producer.go`

- [ ] **Step 1: Replace the file with the new signatures and add the failure-event provider**

Replace `services/atlas-inventory/atlas.com/inventory/inventory/producer.go` contents:

```go
package inventory

import (
	"atlas-inventory/kafka/message/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func CreatedEventStatusProvider(transactionId uuid.UUID, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.StatusEvent[inventory.CreatedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          inventory.StatusEventTypeCreated,
		Body:          inventory.CreatedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func CreationFailedEventStatusProvider(transactionId uuid.UUID, characterId uint32, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.StatusEvent[inventory.CreationFailedStatusEventBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		Type:          inventory.StatusEventTypeCreationFailed,
		Body:          inventory.CreationFailedStatusEventBody{Reason: reason},
	}
	return producer.SingleMessageProvider(key, value)
}

func DeletedEventStatusProvider(characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &inventory.StatusEvent[inventory.DeletedStatusEventBody]{
		CharacterId: characterId,
		Type:        inventory.StatusEventTypeDeleted,
		Body:        inventory.DeletedStatusEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 2: Verify**

```bash
cd services/atlas-inventory/atlas.com/inventory
go build ./...
go vet ./...
```

Expected: build will FAIL because `processor.go:103` still calls `CreatedEventStatusProvider(characterId)`. We fix that in Task 4. **Do not commit yet.**

---

## Task 4: Thread `transactionId` through atlas-inventory processor and emit on failure path

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/inventory/processor.go:77-112`

- [ ] **Step 1: Update `Create` to forward `transactionId` to the success-path emit and add a failure-path emit**

Replace lines 77–112 of `services/atlas-inventory/atlas.com/inventory/inventory/processor.go` (the `Create` method):

```go
func (p *ProcessorImpl) Create(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32) (Model, error) {
	return func(transactionId uuid.UUID, characterId uint32) (Model, error) {
		p.l.Debugf("Attempting to create inventory for character [%d].", characterId)
		var i Model
		txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			// Check if inventory already exists for character.
			var err error
			i, err = p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if i.Equipable().Capacity() != 0 {
				return errors.New("already exists")
			}

			// Generate inventory model by creating new compartments.
			b := NewBuilder(characterId)
			for _, it := range inventory.Types {
				var c compartment.Model
				c, err = p.compartmentProcessor.WithTransaction(tx).Create(mb)(transactionId, characterId, it, 24)
				if err != nil {
					return err
				}
				b.SetCompartment(c)
			}
			i = b.Build()
			return mb.Put(inventory2.EnvEventTopicStatus, CreatedEventStatusProvider(transactionId, characterId))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to create inventory for character [%d].", characterId)
			// Emit creation-failed on a fresh buffer — the in-flight one is
			// discarded because the transaction was rolled back.
			if emitErr := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
				return buf.Put(inventory2.EnvEventTopicStatus, CreationFailedEventStatusProvider(transactionId, characterId, txErr.Error()))
			}); emitErr != nil {
				p.l.WithError(emitErr).Errorf("Unable to emit inventory creation_failed for character [%d].", characterId)
			}
			return Model{}, txErr
		}
		p.l.Infof("Created inventory for character [%d].", characterId)
		return i, nil
	}
}
```

- [ ] **Step 2: Verify**

```bash
cd services/atlas-inventory/atlas.com/inventory
go build ./...
go vet ./...
go test -race ./...
```

Expected: build PASSES, tests PASS.

- [ ] **Step 3: Commit Tasks 3+4 together**

```bash
git add services/atlas-inventory/atlas.com/inventory/inventory/producer.go services/atlas-inventory/atlas.com/inventory/inventory/processor.go
git commit -m "feat(atlas-inventory): thread transactionId through inventory create and emit CREATION_FAILED"
```

---

## Task 5: atlas-inventory character consumer uses saga `transactionId` on reactive create

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/character/consumer.go:43-53`

- [ ] **Step 1: Replace `uuid.New()` with `e.TransactionId`**

In `handleStatusEventCreated`, change:

```go
		_, err := inventory.NewProcessor(l, ctx, db).CreateAndEmit(uuid.New(), e.CharacterId)
```

to:

```go
		_, err := inventory.NewProcessor(l, ctx, db).CreateAndEmit(e.TransactionId, e.CharacterId)
```

The upstream `character.StatusEvent` already carries `TransactionId` (verified at `services/atlas-inventory/atlas.com/inventory/kafka/message/character/kafka.go:15`).

- [ ] **Step 2: Remove the now-unused `uuid` import if no other usage**

Check whether the file still uses `uuid.New()` or `uuid.UUID` elsewhere:

```bash
grep -n "uuid\." services/atlas-inventory/atlas.com/inventory/kafka/consumer/character/consumer.go
```

If `handleStatusEventDeleted` (line 60) still calls `uuid.New()`, leave the import. Otherwise remove the `"github.com/google/uuid"` import.

(Per design we are NOT changing the deleted path. Leave the import; line 60 still uses `uuid.New()`.)

- [ ] **Step 3: Verify**

```bash
cd services/atlas-inventory/atlas.com/inventory
go build ./...
go vet ./...
go test -race ./...
```

Expected: PASS.

- [ ] **Step 4: Docker build (verifies lib drift)**

From the worktree root:

```bash
docker build -f services/atlas-inventory/Dockerfile .
```

Expected: PASS. atlas-inventory's `go.mod` did not change, but per `CLAUDE.md` we verify Docker after any service edit.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/kafka/consumer/character/consumer.go
git commit -m "fix(atlas-inventory): use saga TransactionId on reactive inventory create"
```

---

## Task 6: Add `EventKindInventoryCreated`/`EventKindInventoryCreationFailed` and acceptance entry to atlas-saga-orchestrator

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go`

- [ ] **Step 1: Write the failing coverage assertion**

Open `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go`.

In `allActions` (line 12), find:

```go
	sharedsaga.CreateCharacter, sharedsaga.AwaitCharacterCreated,
```

Replace with:

```go
	sharedsaga.CreateCharacter, sharedsaga.AwaitCharacterCreated, sharedsaga.AwaitInventoryCreated,
```

Append a new test function at the bottom of the file:

```go
func TestStepAcceptsEvent_AwaitInventoryCreated(t *testing.T) {
	if !StepAcceptsEvent(sharedsaga.AwaitInventoryCreated, EventKindInventoryCreated) {
		t.Errorf("StepAcceptsEvent(AwaitInventoryCreated, EventKindInventoryCreated) = false; want true")
	}
	if !StepAcceptsEvent(sharedsaga.AwaitInventoryCreated, EventKindInventoryCreationFailed) {
		t.Errorf("StepAcceptsEvent(AwaitInventoryCreated, EventKindInventoryCreationFailed) = false; want true")
	}
	if StepAcceptsEvent(sharedsaga.AwaitInventoryCreated, EventKindCompartmentCreated) {
		t.Errorf("StepAcceptsEvent(AwaitInventoryCreated, EventKindCompartmentCreated) = true; want false (compartment.created is a sub-event, not the rollup)")
	}
}
```

- [ ] **Step 2: Run the test — expect compile failure**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go test -run TestStepAcceptsEvent_AwaitInventoryCreated ./saga/...
```

Expected: COMPILE FAIL — `undefined: EventKindInventoryCreated`, `undefined: EventKindInventoryCreationFailed`.

- [ ] **Step 3: Add the two `EventKind` constants and acceptance-table entry**

In `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go`, find the `// Compartment (character inventory).` block (line 58–64). Insert a new block immediately after it (before the `// Storage.` block on line 66):

```go
	// Inventory (rollup of all compartments for a character).
	EventKindInventoryCreated        EventKind = "inventory.created"
	EventKindInventoryCreationFailed EventKind = "inventory.creation_failed"
```

Then find `sharedsaga.AwaitCharacterCreated` in the acceptance table (line 162):

```go
	sharedsaga.AwaitCharacterCreated: {EventKindCharacterCreated, EventKindCharacterCreationFailed},
```

Insert immediately after:

```go
	sharedsaga.AwaitInventoryCreated: {EventKindInventoryCreated, EventKindInventoryCreationFailed},
```

- [ ] **Step 4: Run the test — expect pass**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go test -run TestStepAcceptsEvent_AwaitInventoryCreated ./saga/...
go test -run TestAcceptanceTable_EveryActionRepresented ./saga/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go
git commit -m "feat(atlas-saga-orchestrator): add inventory.created/creation_failed event kinds"
```

---

## Task 7: Create the inventory Kafka message struct in atlas-saga-orchestrator

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/inventory/kafka.go`

- [ ] **Step 1: Create the new file**

```go
package inventory

import "github.com/google/uuid"

// Mirrors services/atlas-inventory/atlas.com/inventory/kafka/message/inventory/kafka.go
// so the orchestrator can deserialise events produced by atlas-inventory.

const (
	EnvEventTopicInventoryStatus  = "EVENT_TOPIC_INVENTORY_STATUS"
	StatusEventTypeCreated        = "CREATED"
	StatusEventTypeCreationFailed = "CREATION_FAILED"
	StatusEventTypeDeleted        = "DELETED"
)

type StatusEvent[E any] struct {
	TransactionId uuid.UUID `json:"transactionId,omitempty"`
	CharacterId   uint32    `json:"characterId"`
	Type          string    `json:"type"`
	Body          E         `json:"body"`
}

type CreatedStatusEventBody struct {
}

type CreationFailedStatusEventBody struct {
	Reason string `json:"reason"`
}

type DeletedStatusEventBody struct {
}
```

- [ ] **Step 2: Verify the package builds**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go build ./kafka/message/inventory/...
go vet ./kafka/message/inventory/...
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/inventory
git commit -m "feat(atlas-saga-orchestrator): add inventory kafka message struct"
```

---

## Task 8: Create the inventory Kafka consumer + handlers in atlas-saga-orchestrator

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory/consumer.go`
- Test: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory/consumer_test.go`

- [ ] **Step 1: Write the failing handler test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory/consumer_test.go`:

```go
package inventory

import (
	inventory2 "atlas-saga-orchestrator/kafka/message/inventory"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// TestHandleInventoryCreatedEvent_TypeGuard verifies the handler ignores events
// of the wrong type. This is a smoke test for the type-guard branch — the
// AcceptEvent integration is exercised in createandequip_integration_test.go.
func TestHandleInventoryCreatedEvent_TypeGuard(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := context.Background()
	e := inventory2.StatusEvent[inventory2.CreatedStatusEventBody]{
		TransactionId: uuid.New(),
		CharacterId:   100,
		Type:          inventory2.StatusEventTypeDeleted, // wrong type
		Body:          inventory2.CreatedStatusEventBody{},
	}
	// Should return immediately without panicking. AcceptEvent will not be
	// called because the type guard fails first.
	handleInventoryCreatedEvent(logrus.FieldLogger(l), ctx, e)
}

func TestHandleInventoryCreationFailedEvent_TypeGuard(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := context.Background()
	e := inventory2.StatusEvent[inventory2.CreationFailedStatusEventBody]{
		TransactionId: uuid.New(),
		CharacterId:   100,
		Type:          inventory2.StatusEventTypeCreated, // wrong type
		Body:          inventory2.CreationFailedStatusEventBody{Reason: "boom"},
	}
	handleInventoryCreationFailedEvent(logrus.FieldLogger(l), ctx, e)
}
```

- [ ] **Step 2: Run the test — expect compile fail (handlers don't exist yet)**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go test ./kafka/consumer/inventory/...
```

Expected: COMPILE FAIL — `undefined: handleInventoryCreatedEvent`, `undefined: handleInventoryCreationFailedEvent`.

- [ ] **Step 3: Create the consumer file**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory/consumer.go`:

```go
package inventory

import (
	consumer2 "atlas-saga-orchestrator/kafka/consumer"
	inventory2 "atlas-saga-orchestrator/kafka/message/inventory"
	"atlas-saga-orchestrator/saga"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("inventory_status_event")(inventory2.EnvEventTopicInventoryStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(inventory2.EnvEventTopicInventoryStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryCreatedEvent))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleInventoryCreationFailedEvent))); err != nil {
			return err
		}
		return nil
	}
}

func handleInventoryCreatedEvent(l logrus.FieldLogger, ctx context.Context, e inventory2.StatusEvent[inventory2.CreatedStatusEventBody]) {
	if e.Type != inventory2.StatusEventTypeCreated {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindInventoryCreated); !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_id":   e.CharacterId,
	}).Debug("Inventory created, advancing AwaitInventoryCreated step.")
	_ = p.StepCompleted(e.TransactionId, true)
}

func handleInventoryCreationFailedEvent(l logrus.FieldLogger, ctx context.Context, e inventory2.StatusEvent[inventory2.CreationFailedStatusEventBody]) {
	if e.Type != inventory2.StatusEventTypeCreationFailed {
		return
	}
	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.TransactionId, saga.EventKindInventoryCreationFailed); !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"transaction_id": e.TransactionId.String(),
		"character_id":   e.CharacterId,
		"reason":         e.Body.Reason,
	}).Error("Inventory creation failed, failing AwaitInventoryCreated step.")
	_ = p.StepCompleted(e.TransactionId, false)
}
```

- [ ] **Step 4: Run the tests**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go test -race ./kafka/consumer/inventory/...
go vet ./kafka/consumer/inventory/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/inventory
git commit -m "feat(atlas-saga-orchestrator): consume inventory status events"
```

---

## Task 9: Wire the new inventory consumer into `main.go`

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go`

- [ ] **Step 1: Add the import**

In the import block (line 5–18), add the new package next to `character`:

```go
	"atlas-saga-orchestrator/kafka/consumer/character"
	"atlas-saga-orchestrator/kafka/consumer/compartment"
	"atlas-saga-orchestrator/kafka/consumer/consumable"
```

becomes:

```go
	"atlas-saga-orchestrator/kafka/consumer/character"
	"atlas-saga-orchestrator/kafka/consumer/compartment"
	"atlas-saga-orchestrator/kafka/consumer/consumable"
	inventoryConsumer "atlas-saga-orchestrator/kafka/consumer/inventory"
```

- [ ] **Step 2: Register the consumer**

Find the `character.InitConsumers(l)(cmf)(consumerGroupId)` call (line 93). After the existing block of `InitConsumers` calls (line 89–102), add:

```go
	inventoryConsumer.InitConsumers(l)(cmf)(consumerGroupId)
```

Anywhere in that block — alphabetical sort places it after `guild` and before `pet`. Put it on a new line after `guild.InitConsumers(l)(cmf)(consumerGroupId)` (line 96).

- [ ] **Step 3: Register the handlers**

Find the corresponding `InitHandlers` calls (line 103+). After the `guild.InitHandlers` block (line 124–126), add:

```go
	if err := inventoryConsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
```

- [ ] **Step 4: Verify**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go build ./...
go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/main.go
git commit -m "feat(atlas-saga-orchestrator): register inventory consumer at startup"
```

---

## Task 10: Add `handleAwaitInventoryCreated` no-op handler and dispatch case

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`

- [ ] **Step 1: Add the dispatch case**

Open `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go`. Find the `GetHandler` switch case for `CreateCharacter` (line 751):

```go
	case CreateCharacter:
		return h.handleCreateCharacter, true
```

Insert immediately after:

```go
	case AwaitInventoryCreated:
		return h.handleAwaitInventoryCreated, true
```

- [ ] **Step 2: Add the no-op handler implementation**

At the bottom of the file (after the last existing handler function), append:

```go
// handleAwaitInventoryCreated is a no-op handler. The AwaitInventoryCreated
// step is passive: it is advanced by handleInventoryCreatedEvent (or failed by
// handleInventoryCreationFailedEvent) in kafka/consumer/inventory/consumer.go.
// This handler exists only to satisfy the dispatcher's unknown-action guard
// at saga/processor.go:947.
func (h *HandlerImpl) handleAwaitInventoryCreated(_ Saga, _ Step[any]) error {
	return nil
}
```

- [ ] **Step 3: Verify**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go test -race ./...
go vet ./...
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Docker build**

From the worktree root:

```bash
docker build -f services/atlas-saga-orchestrator/Dockerfile .
```

Expected: PASS. `libs/atlas-saga` already listed in the Dockerfile (verified at `Dockerfile:20,38,55,68`); no Dockerfile edits needed.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go
git commit -m "feat(atlas-saga-orchestrator): dispatch AwaitInventoryCreated as no-op handler"
```

---

## Task 11: Insert `await_inventory_created` step in character-factory sagas

**Files:**
- Modify: `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:184` (after `create_character` in `buildCharacterCreationSaga`)
- Modify: `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go:347` (after `create_character` in `buildPresetCharacterCreationSaga`)

- [ ] **Step 1: Write the failing test (assert step is present and ordered)**

Open `services/atlas-character-factory/atlas.com/character-factory/factory/processor_test.go`. Append:

```go
func TestBuildCharacterCreationSaga_HasAwaitInventoryCreatedStep(t *testing.T) {
	transactionId := uuid.New()
	input := RestModel{AccountId: 1, WorldId: 0, Name: "Test", JobIndex: 0, SubJobIndex: 0, Hp: 50, Mp: 5, MapId: 100000000}
	tmpl := template.RestModel{Items: []uint32{2010000}}

	sg := buildCharacterCreationSaga(transactionId, input, tmpl)

	// Find indices of the two steps that must be ordered.
	createIdx, awaitIdx, awardIdx := -1, -1, -1
	for i, st := range sg.Steps {
		switch st.Action {
		case saga.CreateCharacter:
			createIdx = i
		case saga.AwaitInventoryCreated:
			awaitIdx = i
		case saga.AwardAsset:
			if awardIdx == -1 {
				awardIdx = i
			}
		}
	}
	if createIdx == -1 {
		t.Fatalf("expected CreateCharacter step")
	}
	if awaitIdx == -1 {
		t.Fatalf("expected AwaitInventoryCreated step")
	}
	if awardIdx == -1 {
		t.Fatalf("expected at least one AwardAsset step")
	}
	if !(createIdx < awaitIdx && awaitIdx < awardIdx) {
		t.Fatalf("expected ordering CreateCharacter(%d) < AwaitInventoryCreated(%d) < AwardAsset(%d)", createIdx, awaitIdx, awardIdx)
	}

	// Verify the await step's payload has the sentinel CharacterId=0.
	pl, ok := sg.Steps[awaitIdx].Payload.(saga.AwaitInventoryCreatedPayload)
	if !ok {
		t.Fatalf("await step payload type: got %T, want saga.AwaitInventoryCreatedPayload", sg.Steps[awaitIdx].Payload)
	}
	if pl.CharacterId != 0 {
		t.Errorf("await step CharacterId: got %d, want 0 (sentinel)", pl.CharacterId)
	}
}
```

Open `services/atlas-character-factory/atlas.com/character-factory/factory/processor_preset_test.go`. Append:

```go
func TestBuildPresetCharacterCreationSaga_HasAwaitInventoryCreatedStep(t *testing.T) {
	transactionId := uuid.New()
	in := PresetCreateRestModel{AccountId: 1, WorldId: 0, Name: "Test", PresetId: uuid.New().String()}
	pr := preset.RestModel{Attributes: preset.RestAttributes{Stats: preset.RestStats{Hp: 50, Mp: 5}, MapId: 100000000, Inventory: []preset.RestInventory{{TemplateId: 2010000, Quantity: 1}}}}

	sg := buildPresetCharacterCreationSaga(transactionId, in, pr, map[uint32]data.SkillInfo{})

	createIdx, awaitIdx, awardIdx := -1, -1, -1
	for i, st := range sg.Steps {
		switch st.Action {
		case saga.CreateCharacter:
			createIdx = i
		case saga.AwaitInventoryCreated:
			awaitIdx = i
		case saga.AwardAsset:
			if awardIdx == -1 {
				awardIdx = i
			}
		}
	}
	if createIdx == -1 {
		t.Fatalf("expected CreateCharacter step")
	}
	if awaitIdx == -1 {
		t.Fatalf("expected AwaitInventoryCreated step")
	}
	if awardIdx == -1 {
		t.Fatalf("expected at least one AwardAsset step")
	}
	if !(createIdx < awaitIdx && awaitIdx < awardIdx) {
		t.Fatalf("expected ordering CreateCharacter(%d) < AwaitInventoryCreated(%d) < AwardAsset(%d)", createIdx, awaitIdx, awardIdx)
	}
}
```

- [ ] **Step 2: Run the tests — expect failure**

```bash
cd services/atlas-character-factory/atlas.com/character-factory
go test -run "AwaitInventoryCreated" ./factory/...
```

Expected: FAIL — `AwaitInventoryCreated` step missing.

- [ ] **Step 3: Insert the await step in `buildCharacterCreationSaga`**

Find `buildCharacterCreationSaga` (line 174). After the `// Step 1: Create character` block ending at line 206:

```go
	// Step 1: Create character
	builder.AddStep("create_character", saga.Pending, saga.CreateCharacter, saga.CharacterCreatePayload{
		...
		MapId:        input.MapId,
	})

	// Steps 2-N: Award assets for template items (characterId=0, forwarded by orchestrator)
```

Insert between the `create_character` AddStep call and the `// Steps 2-N:` comment:

```go
	// Step 2: Await inventory-compartment creation. Passive step advanced by
	// kafka/consumer/inventory/consumer.go in atlas-saga-orchestrator. Required
	// to close the race where AwardAsset dispatches before atlas-inventory's
	// compartments are committed. CharacterId=0 sentinel is replaced by
	// forwardCharacterCreationResult after create_character completes.
	builder.AddStep("await_inventory_created", saga.Pending, saga.AwaitInventoryCreated, saga.AwaitInventoryCreatedPayload{
		CharacterId: 0,
	})

```

- [ ] **Step 4: Insert the await step in `buildPresetCharacterCreationSaga`**

Find `buildPresetCharacterCreationSaga` (line 333). After the `// Step 1: create_character` block ending at line 370:

```go
	// Step 1: create_character
	builder.AddStep("create_character", saga.Pending, saga.CreateCharacter, saga.CharacterCreatePayload{
		...
		Meso:         a.Meso,
	})

	// Steps 2..N+1: award_asset for each inventory item
```

Insert between the `create_character` AddStep call and the `// Steps 2..N+1:` comment:

```go
	// Step 2: Await inventory-compartment creation. See buildCharacterCreationSaga
	// for the rationale; this is the preset variant.
	builder.AddStep("await_inventory_created", saga.Pending, saga.AwaitInventoryCreated, saga.AwaitInventoryCreatedPayload{
		CharacterId: 0,
	})

```

- [ ] **Step 5: Run the new tests — expect pass**

```bash
cd services/atlas-character-factory/atlas.com/character-factory
go test -run "AwaitInventoryCreated" ./factory/...
```

Expected: PASS.

- [ ] **Step 6: Run the full factory test suite**

```bash
cd services/atlas-character-factory/atlas.com/character-factory
go test -race ./...
go vet ./...
go build ./...
```

Expected: PASS. Existing tests in `processor_test.go` reference step indices/counts and may need adjustment — if a test counts `len(sg.Steps)` or asserts a specific step at a fixed index, update its expected count or index to account for the new step inserted at position 2. (Read the test failures; fix each by adjusting the expected count or by referencing the step by `Action` rather than by index.)

- [ ] **Step 7: Docker build**

From the worktree root:

```bash
docker build -f services/atlas-character-factory/Dockerfile .
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-character-factory/atlas.com/character-factory
git commit -m "feat(atlas-character-factory): insert await_inventory_created step in character creation sagas"
```

---

## Task 12: Integration test — race-condition repro

**Files:**
- Create: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/await_inventory_created_integration_test.go`

- [ ] **Step 1: Write the test**

Create `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/await_inventory_created_integration_test.go`:

```go
package saga

import (
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// TestAwaitInventoryCreated_BlocksAwardAssetUntilEvent verifies the race fix:
// AwardAsset must not dispatch until INVENTORY_STATUS.CREATED arrives.
//
// The pre-fix behaviour: orchestrator advanced create_character → award_item_0
// on CHARACTER_STATUS.CREATED. With the fix, the orchestrator advances
// create_character → await_inventory_created on CHARACTER_STATUS.CREATED,
// then advances await_inventory_created → award_item_0 only on
// INVENTORY_STATUS.CREATED.
func TestAwaitInventoryCreated_BlocksAwardAssetUntilEvent(t *testing.T) {
	l, _ := test.NewNullLogger()
	tnt, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tnt)

	txId := uuid.New()
	sg := sharedsaga.Saga{
		TransactionId: txId,
		SagaType:      sharedsaga.CharacterCreation,
		InitiatedBy:   "test",
		Steps: []sharedsaga.Step[any]{
			{StepId: "create_character", Status: sharedsaga.Pending, Action: sharedsaga.CreateCharacter, Payload: sharedsaga.CharacterCreatePayload{AccountId: 1, Name: "Test"}},
			{StepId: "await_inventory_created", Status: sharedsaga.Pending, Action: sharedsaga.AwaitInventoryCreated, Payload: sharedsaga.AwaitInventoryCreatedPayload{CharacterId: 0}},
			{StepId: "award_item_0", Status: sharedsaga.Pending, Action: sharedsaga.AwardAsset, Payload: sharedsaga.AwardItemActionPayload{CharacterId: 0, Item: sharedsaga.ItemPayload{TemplateId: 2010000, Quantity: 1}}},
		},
	}

	// Persist saga via the in-memory store (set by test setup hooks
	// already used by createandequip_integration_test.go).
	p := NewProcessor(logrus.FieldLogger(l), ctx)
	assert.NoError(t, p.Create(sg))

	// Mark create_character completed with a real characterId, exactly as
	// handleCharacterCreatedEvent would.
	_ = p.StepCompletedWithResult(txId, true, map[string]any{"characterId": uint32(42)})

	got, ok := p.GetById(txId)
	assert.True(t, ok)

	// Assertion 1: create_character is completed, await_inventory_created is
	// pending, award_item_0 is still pending (NOT completed, NOT advanced
	// past await).
	assertStepStatus(t, got, "create_character", sharedsaga.Completed)
	assertStepStatus(t, got, "await_inventory_created", sharedsaga.Pending)
	assertStepStatus(t, got, "award_item_0", sharedsaga.Pending)

	// Assertion 2: result-forwarding substituted CharacterId=0 with 42 in
	// the await step's payload AND the award step's payload.
	awaitPl := findStep(t, got, "await_inventory_created").Payload.(sharedsaga.AwaitInventoryCreatedPayload)
	assert.Equal(t, uint32(42), awaitPl.CharacterId)
	awardPl := findStep(t, got, "award_item_0").Payload.(sharedsaga.AwardItemActionPayload)
	assert.Equal(t, uint32(42), awardPl.CharacterId)

	// Simulate INVENTORY_STATUS.CREATED arriving.
	if _, ok := p.AcceptEvent(txId, EventKindInventoryCreated); !ok {
		t.Fatalf("AcceptEvent(EventKindInventoryCreated) returned false; expected true for pending AwaitInventoryCreated step")
	}
	_ = p.StepCompleted(txId, true)

	// Assertion 3: now await is completed and award_item_0 has been dispatched
	// (still Pending — it transitions to Completed on EventKindAssetCreated,
	// which is out of this test's scope; what matters is the dispatcher was
	// able to advance past the await step).
	got, _ = p.GetById(txId)
	assertStepStatus(t, got, "await_inventory_created", sharedsaga.Completed)
}

func TestAwaitInventoryCreated_FailEventCompensates(t *testing.T) {
	l, _ := test.NewNullLogger()
	tnt, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tnt)

	txId := uuid.New()
	sg := sharedsaga.Saga{
		TransactionId: txId,
		SagaType:      sharedsaga.CharacterCreation,
		InitiatedBy:   "test",
		Steps: []sharedsaga.Step[any]{
			{StepId: "create_character", Status: sharedsaga.Completed, Action: sharedsaga.CreateCharacter, Payload: sharedsaga.CharacterCreatePayload{AccountId: 1, Name: "Test"}},
			{StepId: "await_inventory_created", Status: sharedsaga.Pending, Action: sharedsaga.AwaitInventoryCreated, Payload: sharedsaga.AwaitInventoryCreatedPayload{CharacterId: 42}},
		},
	}
	p := NewProcessor(logrus.FieldLogger(l), ctx)
	assert.NoError(t, p.Create(sg))

	if _, ok := p.AcceptEvent(txId, EventKindInventoryCreationFailed); !ok {
		t.Fatalf("AcceptEvent(EventKindInventoryCreationFailed) returned false")
	}
	_ = p.StepCompleted(txId, false)

	got, _ := p.GetById(txId)
	assertStepStatus(t, got, "await_inventory_created", sharedsaga.Failed)
	assert.True(t, got.Failing(), "saga should be in Failing state after inventory.creation_failed")
}

// --- test helpers ---

func assertStepStatus(t *testing.T, sg sharedsaga.Saga, stepId string, want sharedsaga.Status) {
	t.Helper()
	for _, st := range sg.Steps {
		if st.StepId == stepId {
			if st.Status != want {
				t.Errorf("step %q status: got %q, want %q", stepId, st.Status, want)
			}
			return
		}
	}
	t.Fatalf("step %q not found in saga", stepId)
}

func findStep(t *testing.T, sg sharedsaga.Saga, stepId string) sharedsaga.Step[any] {
	t.Helper()
	for _, st := range sg.Steps {
		if st.StepId == stepId {
			return st
		}
	}
	t.Fatalf("step %q not found", stepId)
	return sharedsaga.Step[any]{}
}
```

- [ ] **Step 2: Run the test**

```bash
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator
go test -race -run "AwaitInventoryCreated" ./saga/...
```

Expected: PASS. If the test surfaces assumptions about `NewProcessor`/`Create`/`AcceptEvent`/`StepCompleted` signatures that don't match this codebase, read those methods (see `saga/processor.go`) and adjust the test setup — DO NOT change the production code. If `tenant.Create` has a different signature, use the existing pattern from `createandequip_integration_test.go:42-50` instead.

- [ ] **Step 3: Final whole-service verification**

From the worktree root:

```bash
cd libs/atlas-saga && go test -race ./... && go vet ./... && go build ./... && cd -
cd services/atlas-inventory/atlas.com/inventory && go test -race ./... && go vet ./... && go build ./... && cd -
cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... && go vet ./... && go build ./... && cd -
cd services/atlas-character-factory/atlas.com/character-factory && go test -race ./... && go vet ./... && go build ./... && cd -

docker build -f services/atlas-inventory/Dockerfile .
docker build -f services/atlas-saga-orchestrator/Dockerfile .
docker build -f services/atlas-character-factory/Dockerfile .
```

Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/await_inventory_created_integration_test.go
git commit -m "test(atlas-saga-orchestrator): integration coverage for AwaitInventoryCreated"
```

---

## Self-Review (run before requesting code review)

1. **Spec coverage** — every section of `design.md` mapped to a task:
   - §3.1 `libs/atlas-saga` → Task 1
   - §3.2 `services/atlas-inventory` → Tasks 2, 3, 4, 5
   - §3.3 `services/atlas-saga-orchestrator` → Tasks 6, 7, 8, 9, 10
   - §3.4 `services/atlas-character-factory` → Task 11
   - §3.5 Kafka topic config → no-op (existing topic), verified in Task 12 by running the consumer
   - §4 Failure handling → exercised by Task 12 (`TestAwaitInventoryCreated_FailEventCompensates`)
   - §5 Testing → Tasks 1 (unit), 6 (acceptance unit), 8 (handler smoke), 11 (saga builder), 12 (integration)
   - §6 Backward compatibility → `omitempty` on `TransactionId`; no consumer updates required (verified by passing existing tests)
   - §7 Out of scope → explicitly NOT addressed (left as-is)
   - §8 Sequencing → tasks ordered 1 → 12 with Docker builds at lib-dep boundaries

2. **Placeholder scan** — no `TODO`, `TBD`, "fill in", "similar to", or steps without explicit commands/code.

3. **Type consistency** —
   - `AwaitInventoryCreated` (Action) and `AwaitInventoryCreatedPayload` (struct with `CharacterId uint32`) are referenced identically in Tasks 1, 6, 11, 12.
   - `EventKindInventoryCreated` / `EventKindInventoryCreationFailed` (string-typed `EventKind`) referenced identically in Tasks 6, 8, 12.
   - `StatusEventTypeCreationFailed = "CREATION_FAILED"` (string) referenced identically in atlas-inventory and atlas-saga-orchestrator mirror struct (Tasks 2, 7, 8).
   - Producer signature `CreatedEventStatusProvider(transactionId uuid.UUID, characterId uint32)` matches caller in Task 4 processor.

## Execution Handoff

Plan complete and saved to `docs/tasks/chore-todo-saga-race/plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** — `/execute-task chore-todo-saga-race` dispatches a fresh subagent per task with two-stage review.

**2. Inline Execution** — run `superpowers:executing-plans` to batch tasks in this session.

Recommend option 1 because the plan crosses four services with Docker dependencies.
