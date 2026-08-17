package handler

import (
	"atlas-channel/asset"
	"atlas-channel/saga"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestBuildDestroyPetItemSaga pins the destroy shape: exactly one step, by
// SLOT (never by template — a character can hold several pets of one template
// and only the clicked one is dead), cash compartment, quantity 1.
func TestBuildDestroyPetItemSaga(t *testing.T) {
	txn := uuid.New()
	now := time.Now()
	s := buildDestroyPetItemSaga(txn, now, 100, 3, 5000012)

	if s.TransactionId != txn {
		t.Errorf("transactionId: got %s, want %s", s.TransactionId, txn)
	}
	if s.SagaType != saga.InventoryTransaction {
		t.Errorf("sagaType: got %s, want %s", s.SagaType, saga.InventoryTransaction)
	}
	if len(s.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(s.Steps))
	}
	if s.Steps[0].Action != saga.DestroyAssetFromSlot {
		t.Errorf("action: got %s, want %s (by slot, not by template)", s.Steps[0].Action, saga.DestroyAssetFromSlot)
	}
	p, ok := s.Steps[0].Payload.(saga.DestroyAssetFromSlotPayload)
	if !ok {
		t.Fatalf("payload type: %T", s.Steps[0].Payload)
	}
	if p.CharacterId != 100 || p.Slot != 3 || p.TemplateId != 5000012 || p.Quantity != 1 {
		t.Errorf("payload mismatch: %+v", p)
	}
	if p.InventoryType != 5 {
		t.Errorf("inventoryType: got %d, want 5 (cash)", p.InventoryType)
	}
}

// TestFindPetBySerialNumber pins the lookup the client's DESTROY_PET_ITEM_REQUEST
// depends on, including the pet-id fallback: a pet with no cash serial goes out
// on the wire keyed by its pet id (model.Asset.PetSerialNumber), so that is the
// value that comes back.
func TestFindPetBySerialNumber(t *testing.T) {
	cid := uuid.New()
	build := func(id uint32, slot int16, templateId uint32, petId uint32, sn uint64) asset.Model {
		m, err := asset.NewModelBuilder(id, cid, templateId).
			SetSlot(slot).
			SetPetId(petId).
			SetPetSerialNumber(sn).
			Build()
		if err != nil {
			t.Fatalf("build asset: %v", err)
		}
		return m
	}

	serialled := build(1, 1, 5000012, 7, 999)
	fallback := build(2, 2, 5000047, 8, 0)
	notAPet := build(3, 3, 5040004, 0, 0)
	assets := []asset.Model{serialled, fallback, notAPet}

	if got, ok := findPetBySerialNumber(assets, 999); !ok || got.Id() != serialled.Id() {
		t.Errorf("serial lookup: got %v ok=%t, want asset %d", got.Id(), ok, serialled.Id())
	}
	if got, ok := findPetBySerialNumber(assets, 8); !ok || got.Id() != fallback.Id() {
		t.Errorf("pet-id fallback: got %v ok=%t, want asset %d", got.Id(), ok, fallback.Id())
	}
	if _, ok := findPetBySerialNumber(assets, 12345); ok {
		t.Error("unknown serial resolved to an asset")
	}
	// The fallback must not turn a non-pet into a match on serial 0.
	if _, ok := findPetBySerialNumber([]asset.Model{notAPet}, 0); ok {
		t.Error("non-pet asset matched")
	}
}
