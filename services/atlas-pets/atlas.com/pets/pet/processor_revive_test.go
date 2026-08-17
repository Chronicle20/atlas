package pet_test

import (
	"atlas-pets/data/cash"
	cashmock "atlas-pets/data/cash/mock"
	"atlas-pets/kafka/message"
	compartmentmsg "atlas-pets/kafka/message/compartment"
	pet2 "atlas-pets/kafka/message/pet"
	"atlas-pets/pet"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

const waterOfLifeSourceTemplateId = 5180000

// driedUpMochi builds (and persists) a pet fixture matching the brief's
// happy-path scenario: name "Mochi", level 7, closeness 300, fullness 42,
// slot -1 (unsummoned), and an expiration one hour in the past (dried up).
func driedUpMochi(t *testing.T, p pet.Processor, ownerId uint32) pet.Model {
	t.Helper()
	i, err := p.Create(message.NewBuffer())(mustBuild(t, pet.NewModelBuilder(0, 7100000, 5000008, "Mochi", ownerId).
		SetLevel(7).
		SetCloseness(300).
		SetFullness(42).
		SetSlot(-1).
		SetExpiration(time.Now().Add(-1*time.Hour))))
	if err != nil {
		t.Fatalf("Failed to create dried-up pet fixture: %v", err)
	}
	return i
}

func cashMockWithLife(life uint32) *cashmock.ProcessorMock {
	return &cashmock.ProcessorMock{
		GetByIdFunc: func(itemId uint32) (cash.Model, error) {
			return cash.NewModelBuilder(itemId).SetLife(life).Build(), nil
		},
	}
}

func TestReviveHappyPath(t *testing.T) {
	fip := &fakeInventoryProcessor{}
	cdp := cashMockWithLife(90)
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(
		pet.WithInventoryProcessor(fip),
		pet.WithCashDataProcessor(cdp),
	)

	i := driedUpMochi(t, p, 7000000)
	transactionId := uuid.New()

	if err := p.ReviveAndEmit(transactionId, i.OwnerId(), i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Failed to revive pet: %v", err)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet after revive: %v", err)
	}

	expected := time.Now().Add(90 * 24 * time.Hour)
	if diff := o.Expiration().Sub(expected); diff < -time.Minute || diff > time.Minute {
		t.Fatalf("Revived expiration not within 1 minute of now+90d. got=%v expected=%v", o.Expiration(), expected)
	}
	if o.Name() != i.Name() {
		t.Fatalf("Revive mutated Name. Expected %s, got %s", i.Name(), o.Name())
	}
	if o.Level() != i.Level() {
		t.Fatalf("Revive mutated Level. Expected %d, got %d", i.Level(), o.Level())
	}
	if o.Closeness() != i.Closeness() {
		t.Fatalf("Revive mutated Closeness. Expected %d, got %d", i.Closeness(), o.Closeness())
	}
	if o.Fullness() != i.Fullness() {
		t.Fatalf("Revive mutated Fullness. Expected %d, got %d", i.Fullness(), o.Fullness())
	}
	if o.Slot() != i.Slot() {
		t.Fatalf("Revive mutated Slot. Expected %d, got %d", i.Slot(), o.Slot())
	}
	if o.TemplateId() != i.TemplateId() {
		t.Fatalf("Revive mutated TemplateId. Expected %d, got %d", i.TemplateId(), o.TemplateId())
	}
	if o.Flag() != i.Flag() {
		t.Fatalf("Revive mutated Flag. Expected %d, got %d", i.Flag(), o.Flag())
	}
}

// TestReviveLifespanFromData is the FR-5.1 "not a constant" guard: the same
// fixture with a different WZ life value must produce a different expiration.
func TestReviveLifespanFromData(t *testing.T) {
	fip := &fakeInventoryProcessor{}
	cdp := cashMockWithLife(30)
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(
		pet.WithInventoryProcessor(fip),
		pet.WithCashDataProcessor(cdp),
	)

	i := driedUpMochi(t, p, 7000001)
	transactionId := uuid.New()

	if err := p.ReviveAndEmit(transactionId, i.OwnerId(), i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Failed to revive pet: %v", err)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet after revive: %v", err)
	}

	expected := time.Now().Add(30 * 24 * time.Hour)
	if diff := o.Expiration().Sub(expected); diff < -time.Minute || diff > time.Minute {
		t.Fatalf("Revived expiration not within 1 minute of now+30d. got=%v expected=%v", o.Expiration(), expected)
	}

	// Guard: 30-day and 90-day expirations must differ meaningfully, or this
	// test (and TestReviveHappyPath) proves nothing about the source of the
	// lifespan.
	ninety := time.Now().Add(90 * 24 * time.Hour)
	if diff := o.Expiration().Sub(ninety); diff > -60*24*time.Hour {
		t.Fatalf("30-day and 90-day revive expirations are not meaningfully different: got=%v", o.Expiration())
	}
}

func TestReviveZeroLifeIsRejected(t *testing.T) {
	fip := &fakeInventoryProcessor{}
	cdp := cashMockWithLife(0)
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(
		pet.WithInventoryProcessor(fip),
		pet.WithCashDataProcessor(cdp),
	)

	i := driedUpMochi(t, p, 7000002)
	transactionId := uuid.New()

	mb := message.NewBuffer()
	if err := p.Revive(mb)(transactionId, i.OwnerId(), i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Revive with zero life returned an error instead of buffering REVIVE_FAILED: %v", err)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet: %v", err)
	}
	if !o.Expiration().Equal(i.Expiration()) {
		t.Fatalf("Zero-life revive should not have written the pet row. old=%v new=%v", i.Expiration(), o.Expiration())
	}
	if o.ReviveTransactionId() != nil {
		t.Fatalf("Zero-life revive should not have set revive_transaction_id")
	}

	assertReviveFailedEmitted(t, mb)
	if fip.rpeCalled {
		t.Fatalf("Zero-life revive should not cascade RESET_PET_EXPIRATION")
	}
}

func TestReviveCascadesResetPetExpiration(t *testing.T) {
	// The REAL inventory processor is used here (not the fake) so the
	// assertion is against the genuine buffered Kafka command, not a
	// test-double's recorded call args. ChangeTemplate/ResetPetExpiration only
	// buffer into mb -- no network call is made.
	cdp := cashMockWithLife(90)
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(
		pet.WithCashDataProcessor(cdp),
	)

	i := driedUpMochi(t, p, 7000003)
	transactionId := uuid.New()

	mb := message.NewBuffer()
	if err := p.Revive(mb)(transactionId, i.OwnerId(), i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Failed to revive pet: %v", err)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet after revive: %v", err)
	}

	// The cascade command is on the compartment command topic in the buffer.
	ke := mb.GetAll()
	cs, ok := ke[compartmentmsg.EnvCommandTopic]
	if !ok || len(cs) == 0 {
		t.Fatalf("Expected a RESET_PET_EXPIRATION command on topic %s", compartmentmsg.EnvCommandTopic)
	}
	var found bool
	for _, msg := range cs {
		var env struct {
			Type string                                       `json:"type"`
			Body compartmentmsg.ResetPetExpirationCommandBody `json:"body"`
		}
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			t.Fatalf("Failed to unmarshal compartment command: %v", err)
		}
		if env.Type != compartmentmsg.CommandResetPetExpiration {
			continue
		}
		found = true
		if env.Body.PetId != i.Id() {
			t.Fatalf("RESET_PET_EXPIRATION command carried wrong petId. Expected %d, got %d", i.Id(), env.Body.PetId)
		}
		if !env.Body.Expiration.Equal(o.Expiration()) {
			t.Fatalf("RESET_PET_EXPIRATION command carried an expiration different from the persisted pet row. cascade=%v persisted=%v", env.Body.Expiration, o.Expiration())
		}
		if env.Body.SourceTemplateId != waterOfLifeSourceTemplateId {
			t.Fatalf("RESET_PET_EXPIRATION command carried wrong sourceTemplateId. Expected %d, got %d", waterOfLifeSourceTemplateId, env.Body.SourceTemplateId)
		}
	}
	if !found {
		t.Fatalf("No RESET_PET_EXPIRATION command found on topic %s", compartmentmsg.EnvCommandTopic)
	}
}

// TestReviveRedeliveryIsIdempotent is idempotency row A: a redelivered REVIVE
// (same transactionId as the stored one) must perform no write, but must
// re-cascade RESET_PET_EXPIRATION with the STORED expiration and re-emit
// REVIVED.
func TestReviveRedeliveryIsIdempotent(t *testing.T) {
	fip := &fakeInventoryProcessor{}
	cdp := cashMockWithLife(90)
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(
		pet.WithInventoryProcessor(fip),
		pet.WithCashDataProcessor(cdp),
	)

	i := driedUpMochi(t, p, 7000004)
	transactionId := uuid.New()

	mb1 := message.NewBuffer()
	if err := p.Revive(mb1)(transactionId, i.OwnerId(), i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Failed initial revive: %v", err)
	}
	first, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet after first revive: %v", err)
	}

	// Reset the fake's capture before the redelivery, so the redelivery's own
	// cascade args can be asserted in isolation.
	fip.rpeCalled = false

	mb2 := message.NewBuffer()
	if err := p.Revive(mb2)(transactionId, i.OwnerId(), i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Failed redelivered revive: %v", err)
	}
	second, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet after redelivered revive: %v", err)
	}

	if !second.Expiration().Equal(first.Expiration()) {
		t.Fatalf("Redelivery performed a write. first=%v second=%v", first.Expiration(), second.Expiration())
	}
	if !fip.rpeCalled {
		t.Fatalf("Redelivery did not re-cascade RESET_PET_EXPIRATION")
	}
	if !fip.rpeExpiration.Equal(first.Expiration()) {
		t.Fatalf("Redelivery re-cascaded with a different expiration than the stored one. stored=%v cascaded=%v", first.Expiration(), fip.rpeExpiration)
	}

	assertRevivedEmitted(t, mb2)
}

// TestReviveRejectsSecondDifferentTransactionOnLivePet is idempotency row B:
// a genuinely SECOND Water of Life used on an already-revived (still-live)
// pet, carrying a DIFFERENT transaction id, must be rejected so the saga
// refunds it.
func TestReviveRejectsSecondDifferentTransactionOnLivePet(t *testing.T) {
	fip := &fakeInventoryProcessor{}
	cdp := cashMockWithLife(90)
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(
		pet.WithInventoryProcessor(fip),
		pet.WithCashDataProcessor(cdp),
	)

	i := driedUpMochi(t, p, 7000005)
	firstTransactionId := uuid.New()

	if err := p.ReviveAndEmit(firstTransactionId, i.OwnerId(), i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Failed initial revive: %v", err)
	}
	first, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet after first revive: %v", err)
	}

	fip.rpeCalled = false
	secondTransactionId := uuid.New()
	mb := message.NewBuffer()
	if err := p.Revive(mb)(secondTransactionId, i.OwnerId(), i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Second revive on a live pet returned an error instead of buffering REVIVE_FAILED: %v", err)
	}

	second, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet: %v", err)
	}
	if !second.Expiration().Equal(first.Expiration()) {
		t.Fatalf("Rejected second revive performed a write. first=%v second=%v", first.Expiration(), second.Expiration())
	}
	assertReviveFailedEmitted(t, mb)
	if fip.rpeCalled {
		t.Fatalf("Rejected second revive should not cascade RESET_PET_EXPIRATION")
	}
}

func TestReviveRejectsWrongOwner(t *testing.T) {
	fip := &fakeInventoryProcessor{}
	cdp := cashMockWithLife(90)
	p := pet.NewProcessor(testLogger(), testContext(), testDatabase(t)).With(
		pet.WithInventoryProcessor(fip),
		pet.WithCashDataProcessor(cdp),
	)

	i := driedUpMochi(t, p, 7000006)
	transactionId := uuid.New()

	mb := message.NewBuffer()
	otherActorId := i.OwnerId() + 1
	if err := p.Revive(mb)(transactionId, otherActorId, i.Id(), waterOfLifeSourceTemplateId); err != nil {
		t.Fatalf("Revive by non-owner returned an error instead of buffering REVIVE_FAILED: %v", err)
	}

	o, err := p.GetById(i.Id())
	if err != nil {
		t.Fatalf("Failed to retrieve pet: %v", err)
	}
	if !o.Expiration().Equal(i.Expiration()) {
		t.Fatalf("Non-owner revive should not have written the pet row. old=%v new=%v", i.Expiration(), o.Expiration())
	}
	assertReviveFailedEmitted(t, mb)
	if fip.rpeCalled {
		t.Fatalf("Non-owner revive should not cascade RESET_PET_EXPIRATION")
	}
}

func assertReviveFailedEmitted(t *testing.T, mb *message.Buffer) {
	t.Helper()
	ke := mb.GetAll()
	se, ok := ke[pet2.EnvStatusEventTopic]
	if !ok || len(se) == 0 {
		t.Fatalf("Expected a status event on topic %s", pet2.EnvStatusEventTopic)
	}
	for _, msg := range se {
		var env struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			t.Fatalf("Failed to unmarshal status event: %v", err)
		}
		if env.Type == pet2.StatusEventTypeReviveFailed {
			return
		}
	}
	t.Fatalf("Expected a %s status event, none found", pet2.StatusEventTypeReviveFailed)
}

func assertRevivedEmitted(t *testing.T, mb *message.Buffer) {
	t.Helper()
	ke := mb.GetAll()
	se, ok := ke[pet2.EnvStatusEventTopic]
	if !ok || len(se) == 0 {
		t.Fatalf("Expected a status event on topic %s", pet2.EnvStatusEventTopic)
	}
	for _, msg := range se {
		var env struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			t.Fatalf("Failed to unmarshal status event: %v", err)
		}
		if env.Type == pet2.StatusEventTypeRevived {
			return
		}
	}
	t.Fatalf("Expected a %s status event, none found", pet2.StatusEventTypeRevived)
}
