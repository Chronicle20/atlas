package pet

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestToEntityIsInverseOfMake pins ToEntity as the exact inverse of Make for
// every persisted column, including reviveTransactionId — the field the Water
// of Life revive distinguishes redelivery by. A field added to Model and to
// Entity but forgotten in ToEntity would fail here rather than silently write
// a NULL.
func TestToEntityIsInverseOfMake(t *testing.T) {
	tenantId := uuid.New()
	txId := uuid.New()
	expiration := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)

	m, err := NewBuilder(0, 7000000, 5000017, "Test Pet", 1).
		SetLevel(3).
		SetCloseness(41).
		SetFullness(88).
		SetExpiration(expiration).
		SetSlot(2).
		SetFlag(5).
		SetPurchaseBy(99).
		SetReviveTransactionId(&txId).
		Build()
	if err != nil {
		t.Fatalf("build model: %v", err)
	}

	e := m.ToEntity(tenantId)
	if e.TenantId != tenantId {
		t.Errorf("TenantId = %v, want %v", e.TenantId, tenantId)
	}
	if e.Id != 0 {
		t.Errorf("Id = %d, want 0 (assigned by auto-increment on insert)", e.Id)
	}
	if len(e.Excludes) != 0 {
		t.Errorf("Excludes = %v, want empty (owned by setExcludes, not the create cascade)", e.Excludes)
	}

	rt, err := Make(e)
	if err != nil {
		t.Fatalf("make model from entity: %v", err)
	}

	if rt.CashId() != m.CashId() {
		t.Errorf("CashId = %d, want %d", rt.CashId(), m.CashId())
	}
	if rt.TemplateId() != m.TemplateId() {
		t.Errorf("TemplateId = %d, want %d", rt.TemplateId(), m.TemplateId())
	}
	if rt.Name() != m.Name() {
		t.Errorf("Name = %q, want %q", rt.Name(), m.Name())
	}
	if rt.OwnerId() != m.OwnerId() {
		t.Errorf("OwnerId = %d, want %d", rt.OwnerId(), m.OwnerId())
	}
	if rt.Level() != m.Level() {
		t.Errorf("Level = %d, want %d", rt.Level(), m.Level())
	}
	if rt.Closeness() != m.Closeness() {
		t.Errorf("Closeness = %d, want %d", rt.Closeness(), m.Closeness())
	}
	if rt.Fullness() != m.Fullness() {
		t.Errorf("Fullness = %d, want %d", rt.Fullness(), m.Fullness())
	}
	if !rt.Expiration().Equal(m.Expiration()) {
		t.Errorf("Expiration = %v, want %v", rt.Expiration(), m.Expiration())
	}
	if rt.Slot() != m.Slot() {
		t.Errorf("Slot = %d, want %d", rt.Slot(), m.Slot())
	}
	if rt.Flag() != m.Flag() {
		t.Errorf("Flag = %d, want %d", rt.Flag(), m.Flag())
	}
	if rt.PurchaseBy() != m.PurchaseBy() {
		t.Errorf("PurchaseBy = %d, want %d", rt.PurchaseBy(), m.PurchaseBy())
	}
	if rt.ReviveTransactionId() == nil || *rt.ReviveTransactionId() != txId {
		t.Errorf("ReviveTransactionId = %v, want %v", rt.ReviveTransactionId(), txId)
	}
}

// TestToEntityNeverRevivedStaysNil guards the NULL case: a pet that has never
// been revived must project a nil pointer, not a zero uuid, or the redelivery
// gate in Revive would treat uuid.Nil as a real prior transaction.
func TestToEntityNeverRevivedStaysNil(t *testing.T) {
	m, err := NewBuilder(0, 7000000, 5000017, "Test Pet", 1).
		SetExpiration(time.Now().Add(time.Hour)).
		SetSlot(-1).
		Build()
	if err != nil {
		t.Fatalf("build model: %v", err)
	}

	e := m.ToEntity(uuid.New())
	if e.ReviveTransactionId != nil {
		t.Errorf("ReviveTransactionId = %v, want nil", *e.ReviveTransactionId)
	}
}
