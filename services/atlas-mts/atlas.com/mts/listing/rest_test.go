package listing_test

import (
	"atlas-mts/listing"
	"testing"

	"github.com/google/uuid"
)

// TestTransformOwner asserts the item-tag owner name set on the listing model
// survives Transform into the REST DTO, mirroring the existing Flags field. The
// empty case covers an untagged item (the column's default-'' state).
func TestTransformOwner(t *testing.T) {
	tests := []struct {
		name  string
		owner string
	}{
		{name: "tagged", owner: "Chronicle"},
		{name: "untagged (default empty)", owner: ""},
		{name: "multi-word name", owner: "Dark Lord"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tid := uuid.New()
			m, err := listing.NewBuilder(tid, 0, 1001).
				SetSellerName("alice").
				SetSaleType(listing.SaleTypeFixed).
				SetState(listing.StateActive).
				SetTemplateId(1302000).
				SetQuantity(1).
				SetOwner(tt.owner).
				Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			rm, err := listing.Transform(m)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rm.Owner != tt.owner {
				t.Fatalf("rm.Owner = %q, want %q", rm.Owner, tt.owner)
			}
		})
	}
}
