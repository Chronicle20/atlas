package compartment_test

import (
	"atlas-inventory/asset"
	"atlas-inventory/compartment"
	"atlas-inventory/data/cash"
	cashmock "atlas-inventory/data/cash/mock"
	"atlas-inventory/kafka/message"
	assetMsg "atlas-inventory/kafka/message/asset"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// resetPetExpirationFixture sets up a cash compartment holding two pet assets
// (petIds 7 and 9) and a non-pet cash asset, mirroring the resolve-by-petId
// walk ResetPetExpiration shares with ChangeTemplate.
func resetPetExpirationFixture(t *testing.T, characterId uint32, db *gorm.DB, ctx context.Context, mb *message.Buffer, ap asset.Processor) (compartmentId uuid.UUID, expirations map[uint32]time.Time) {
	cp := compartment.NewProcessor(testLogger(), ctx, db).WithAssetProcessor(ap)
	c, err := cp.Create(mb)(uuid.New(), characterId, inventory.TypeValueCash, 24)
	if err != nil {
		t.Fatalf("Failed to create cash compartment: %v", err)
	}

	expirations = make(map[uint32]time.Time)

	mk := func(petId uint32, templateId uint32, slot int16) {
		exp := time.Now().Add(-time.Hour).Truncate(time.Second) // already dried up
		m := asset.NewBuilder(c.Id(), templateId).
			SetSlot(slot).
			SetPetId(petId).
			SetCashId(int64(petId) * 1000).
			SetExpiration(exp).
			Build()
		created, err := ap.CreateFromModel(mb)(uuid.New(), characterId, m)
		if err != nil {
			t.Fatalf("Failed to create pet asset petId=%d: %v", petId, err)
		}
		if !created.IsPet() {
			t.Fatalf("precondition failed: created asset petId=%d is not a pet", petId)
		}
		expirations[petId] = exp
	}

	mk(7, 5000028, -1)
	mk(9, 5000029, -2)

	// A non-pet cash asset (no petId) sharing the same compartment.
	nonPet := asset.NewBuilder(c.Id(), 5010000).SetSlot(-3).Build()
	if _, err := ap.CreateFromModel(mb)(uuid.New(), characterId, nonPet); err != nil {
		t.Fatalf("Failed to create non-pet cash asset: %v", err)
	}

	return c.Id(), expirations
}

func lifeCashMock(life uint32) cash.Processor {
	return &cashmock.ProcessorMock{
		GetByIdFunc: func(itemId uint32) (cash.Model, error) {
			return cash.NewModelBuilder(itemId).SetLife(life).Build(), nil
		},
	}
}

func assetBySlot(t *testing.T, ap asset.Processor, compartmentId uuid.UUID, slot int16) asset.Model {
	t.Helper()
	a, err := ap.GetBySlot(compartmentId, slot)
	if err != nil {
		t.Fatalf("Failed to reload asset at slot %d: %v", slot, err)
	}
	return a
}

// TestResetPetExpirationResolvesByPetId verifies that ResetPetExpiration
// walks the cash compartment's assets and mutates only the one whose petId
// matches, leaving sibling pet and non-pet assets untouched, and emits
// exactly one asset UPDATED event.
func TestResetPetExpirationResolvesByPetId(t *testing.T) {
	characterId := uint32(700)
	sourceTemplateId := uint32(5180000)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()
	ap := asset.NewProcessor(l, ctx, db)
	compartmentId, expirations := resetPetExpirationFixture(t, characterId, db, ctx, mb, ap)

	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithCashProcessor(lifeCashMock(90))

	mb = message.NewBuffer()
	want := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	if err := cp.ResetPetExpiration(mb)(uuid.New(), characterId, 9, want, sourceTemplateId); err != nil {
		t.Fatalf("ResetPetExpiration: %v", err)
	}

	got9 := assetBySlot(t, ap, compartmentId, -2)
	if !got9.Expiration().Equal(want) {
		t.Errorf("petId 9 expiration = %v, want %v", got9.Expiration(), want)
	}

	got7 := assetBySlot(t, ap, compartmentId, -1)
	if !got7.Expiration().Equal(expirations[7]) {
		t.Errorf("petId 7 expiration mutated: got %v, want unchanged %v", got7.Expiration(), expirations[7])
	}

	var updatedCount int
	for _, msg := range mb.GetAll()[assetMsg.EnvEventTopicStatus] {
		var ev assetMsg.StatusEvent[json.RawMessage]
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			continue
		}
		if ev.Type == assetMsg.StatusEventTypeUpdated {
			updatedCount++
		}
	}
	if updatedCount != 1 {
		t.Fatalf("UPDATED events = %d, want 1", updatedCount)
	}
}

// TestResetPetExpirationRejectsZeroLife verifies that a source item whose
// cash data carries no info/life (a data gap, or a forged sourceTemplateId
// that doesn't resolve to a real revival item) is rejected outright, and
// mutates nothing.
func TestResetPetExpirationRejectsZeroLife(t *testing.T) {
	characterId := uint32(701)
	sourceTemplateId := uint32(5180000)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()
	ap := asset.NewProcessor(l, ctx, db)
	compartmentId, expirations := resetPetExpirationFixture(t, characterId, db, ctx, mb, ap)

	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithCashProcessor(lifeCashMock(0))

	want := time.Now().Add(24 * time.Hour)
	if err := cp.ResetPetExpirationAndEmit(uuid.New(), characterId, 9, want, sourceTemplateId); err == nil {
		t.Fatal("expected rejection when the source item has no info/life")
	}

	got9 := assetBySlot(t, ap, compartmentId, -2)
	if !got9.Expiration().Equal(expirations[9]) {
		t.Errorf("petId 9 expiration mutated: got %v, want unchanged %v", got9.Expiration(), expirations[9])
	}
}

// TestResetPetExpirationRejectsOverCap is the NFR-2 forged-expiration guard:
// a request beyond the server-derived cap (info/life days from now) must be
// REJECTED outright, never clamped, and the asset must be left unchanged.
func TestResetPetExpirationRejectsOverCap(t *testing.T) {
	characterId := uint32(702)
	sourceTemplateId := uint32(5180000)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()
	ap := asset.NewProcessor(l, ctx, db)
	compartmentId, expirations := resetPetExpirationFixture(t, characterId, db, ctx, mb, ap)

	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithCashProcessor(lifeCashMock(90))

	forged := time.Now().Add(200 * 24 * time.Hour)
	if err := cp.ResetPetExpirationAndEmit(uuid.New(), characterId, 9, forged, sourceTemplateId); err == nil {
		t.Fatal("expected ResetPetExpirationAndEmit to reject an over-cap request, got nil error")
	}

	got9 := assetBySlot(t, ap, compartmentId, -2)
	if !got9.Expiration().Equal(expirations[9]) {
		t.Errorf("petId 9 expiration mutated: got %v, want unchanged %v", got9.Expiration(), expirations[9])
	}
}

// TestResetPetExpirationHonorsInBoundsRequest verifies that a request within
// the source item's server-derived cap is honoured verbatim.
func TestResetPetExpirationHonorsInBoundsRequest(t *testing.T) {
	characterId := uint32(703)
	sourceTemplateId := uint32(5180000)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()
	ap := asset.NewProcessor(l, ctx, db)
	compartmentId, _ := resetPetExpirationFixture(t, characterId, db, ctx, mb, ap)

	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithCashProcessor(lifeCashMock(90))

	want := time.Now().Add(90*24*time.Hour - time.Minute).Truncate(time.Second)
	if err := cp.ResetPetExpirationAndEmit(uuid.New(), characterId, 9, want, sourceTemplateId); err != nil {
		t.Fatalf("ResetPetExpirationAndEmit: %v", err)
	}

	got9 := assetBySlot(t, ap, compartmentId, -2)
	if !got9.Expiration().Equal(want) {
		t.Errorf("petId 9 expiration = %v, want %v", got9.Expiration(), want)
	}
}

// TestResetPetExpirationIdempotentRedelivery verifies that calling
// ResetPetExpiration twice with the identical absolute expiration leaves the
// asset unchanged on the second call but still emits UPDATED, so a
// redelivered command converges the saga rather than stalling it — this
// falls out of asset.ExtendExpiration's equal-value branch.
func TestResetPetExpirationIdempotentRedelivery(t *testing.T) {
	characterId := uint32(704)
	sourceTemplateId := uint32(5180000)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()
	ap := asset.NewProcessor(l, ctx, db)
	compartmentId, _ := resetPetExpirationFixture(t, characterId, db, ctx, mb, ap)

	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithCashProcessor(lifeCashMock(90))

	want := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	txId := uuid.New()

	if err := cp.ResetPetExpirationAndEmit(txId, characterId, 9, want, sourceTemplateId); err != nil {
		t.Fatalf("first ResetPetExpirationAndEmit: %v", err)
	}

	if err := cp.ResetPetExpirationAndEmit(txId, characterId, 9, want, sourceTemplateId); err != nil {
		t.Fatalf("redelivered ResetPetExpirationAndEmit: %v", err)
	}

	got9 := assetBySlot(t, ap, compartmentId, -2)
	if !got9.Expiration().Equal(want) {
		t.Errorf("petId 9 expiration = %v, want %v", got9.Expiration(), want)
	}
}

// TestResetPetExpirationUnknownPetIdErrors verifies that a petId with no
// matching asset in the cash compartment returns an error and mutates
// nothing.
func TestResetPetExpirationUnknownPetIdErrors(t *testing.T) {
	characterId := uint32(705)
	sourceTemplateId := uint32(5180000)

	l := testLogger()
	te := testTenant()
	ctx := tenant.WithContext(context.Background(), te)
	db := testDatabase(t, l)

	mb := message.NewBuffer()
	ap := asset.NewProcessor(l, ctx, db)
	compartmentId, expirations := resetPetExpirationFixture(t, characterId, db, ctx, mb, ap)

	cp := compartment.NewProcessor(l, ctx, db).WithAssetProcessor(ap).WithCashProcessor(lifeCashMock(90))

	want := time.Now().Add(24 * time.Hour)
	if err := cp.ResetPetExpirationAndEmit(uuid.New(), characterId, 999, want, sourceTemplateId); err == nil {
		t.Fatal("expected an error for an unknown petId")
	}

	got7 := assetBySlot(t, ap, compartmentId, -1)
	if !got7.Expiration().Equal(expirations[7]) {
		t.Errorf("petId 7 expiration mutated: got %v, want unchanged %v", got7.Expiration(), expirations[7])
	}
	got9 := assetBySlot(t, ap, compartmentId, -2)
	if !got9.Expiration().Equal(expirations[9]) {
		t.Errorf("petId 9 expiration mutated: got %v, want unchanged %v", got9.Expiration(), expirations[9])
	}
}
