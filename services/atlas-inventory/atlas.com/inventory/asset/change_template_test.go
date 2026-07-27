package asset_test

import (
	"atlas-inventory/asset"
	"atlas-inventory/kafka/message"
	assetMsg "atlas-inventory/kafka/message/asset"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func changeTemplateTestDatabase(t *testing.T, l logrus.FieldLogger) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := asset.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

// TestChangeTemplatePreservesIdentity verifies that ChangeTemplate swaps only the
// templateId of a pet asset in place — slot, petId, and cashId are preserved — and
// that an UPDATED status event is buffered (never DELETED), which is what keeps the
// pet alive in atlas-pets.
func TestChangeTemplatePreservesIdentity(t *testing.T) {
	l, _ := test.NewNullLogger()
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), te)
	db := changeTemplateTestDatabase(t, l)

	mb := message.NewBuffer()
	ap := asset.NewProcessor(l, ctx, db)

	const (
		characterId  = uint32(1)
		origTemplate = uint32(5000028)
		newTemplate  = uint32(5000029)
		petId        = uint32(77)
		cashId       = int64(123456789)
		slot         = int16(-1)
	)
	compartmentId := uuid.New()

	// A pet asset is cash (template 5000028 -> type 5) with petId > 0.
	petModel := asset.NewBuilder(compartmentId, origTemplate).
		SetSlot(slot).
		SetPetId(petId).
		SetCashId(cashId).
		Build()

	created, err := ap.CreateFromModel(mb)(uuid.New(), characterId, petModel)
	if err != nil {
		t.Fatalf("Failed to create pet asset: %v", err)
	}
	if !created.IsPet() {
		t.Fatalf("precondition failed: created asset is not a pet (template %d, petId %d)", created.TemplateId(), created.PetId())
	}

	// Fresh buffer so we only inspect events from ChangeTemplate.
	mb = message.NewBuffer()

	if err := ap.ChangeTemplate(mb)(uuid.New(), characterId, created.Id(), newTemplate); err != nil {
		t.Fatalf("ChangeTemplate failed: %v", err)
	}

	got, err := ap.GetById(created.Id())
	if err != nil {
		t.Fatalf("Failed to reload asset: %v", err)
	}
	if got.TemplateId() != newTemplate {
		t.Fatalf("templateId not changed: got %d, want %d", got.TemplateId(), newTemplate)
	}
	if got.Slot() != slot {
		t.Fatalf("slot not preserved: got %d, want %d", got.Slot(), slot)
	}
	if got.PetId() != petId {
		t.Fatalf("petId not preserved: got %d, want %d", got.PetId(), petId)
	}
	if got.CashId() != cashId {
		t.Fatalf("cashId not preserved: got %d, want %d", got.CashId(), cashId)
	}

	// An UPDATED status event must be buffered, never DELETED.
	events := mb.GetAll()[assetMsg.EnvEventTopicStatus]
	var sawUpdated bool
	for _, msg := range events {
		var ev assetMsg.StatusEvent[json.RawMessage]
		if err := json.Unmarshal(msg.Value, &ev); err != nil {
			continue
		}
		if ev.Type == assetMsg.StatusEventTypeDeleted {
			t.Fatalf("ChangeTemplate emitted DELETED — pet would die in atlas-pets")
		}
		if ev.Type == assetMsg.StatusEventTypeUpdated {
			sawUpdated = true
			if ev.TemplateId != newTemplate {
				t.Fatalf("UPDATED event templateId = %d, want %d", ev.TemplateId, newTemplate)
			}
		}
	}
	if !sawUpdated {
		t.Fatalf("expected an UPDATED status event, got %d events", len(events))
	}
}
