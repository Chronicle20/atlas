package listing_test

import (
	"atlas-mts/listing"
	"testing"

	"github.com/google/uuid"
)

func TestBuilder_RequiresTenantAndWorld(t *testing.T) {
	_, err := listing.NewBuilder(uuid.Nil, 0, 1001).Build()
	if err == nil {
		t.Fatal("expected error when tenantId is nil")
	}
}

func TestBuilder_BuildsFixedListing(t *testing.T) {
	tid := uuid.New()
	m, err := listing.NewBuilder(tid, 0, 1001).
		SetSellerName("alice").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(110).
		SetCommissionRate(0.10).
		SetCategory("equip").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.SaleType() != listing.SaleTypeFixed || m.State() != listing.StateActive {
		t.Fatalf("unexpected sale/state: %v/%v", m.SaleType(), m.State())
	}
	if m.ListValue() != 110 || m.SellerId() != 1001 || m.WorldId() != 0 {
		t.Fatalf("unexpected fields")
	}
}

// TestBuilder_SetOwnerRoundTrip asserts the item-tag owner name set via
// SetOwner survives Build() and is exposed by Model.Owner(), mirroring the
// existing Flags()/SetFlags() round trip.
func TestBuilder_SetOwnerRoundTrip(t *testing.T) {
	tid := uuid.New()
	m, err := listing.NewBuilder(tid, 0, 1001).
		SetSellerName("alice").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetOwner("Chronicle").
		Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Owner() != "Chronicle" {
		t.Fatalf("m.Owner() = %q, want %q", m.Owner(), "Chronicle")
	}
}
