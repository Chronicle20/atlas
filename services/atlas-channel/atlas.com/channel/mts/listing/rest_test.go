package listing

import (
	"reflect"
	"testing"
	"time"
)

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field set by Extract survives a Transform -> Extract round
// trip. EndsAt is a *time.Time on both sides (the codemod's SKIP reason);
// both the nil (fixed-price sale) and non-nil (auction) cases are covered,
// and the non-nil case confirms Transform copies the pointed-to value rather
// than aliasing the Model's pointer.
func TestTransformRoundTrip(t *testing.T) {
	base := RestModel{
		Id:            "1",
		WorldId:       0,
		ItcSn:         1001,
		SellerId:      2002,
		SellerName:    "seller",
		SaleType:      "FIXED",
		State:         "ACTIVE",
		TemplateId:    1302000,
		Quantity:      1,
		Strength:      10,
		Dexterity:     11,
		Intelligence:  12,
		Luck:          13,
		HP:            14,
		MP:            15,
		WeaponAttack:  16,
		MagicAttack:   17,
		WeaponDefense: 18,
		MagicDefense:  19,
		Accuracy:      20,
		Avoidability:  21,
		Hands:         22,
		Speed:         23,
		Jump:          24,
		Slots:         25,
		Level:         26,
		ItemLevel:     27,
		ItemExp:       28,
		RingId:        29,
		ViciousCount:  30,
		Flags:         31,
		Owner:         "owner",
		ListValue:     100000,
		BuyNowPrice:   200000,
		Category:      "WEAPON",
		SubCategory:   "SWORD",
		ContractFee:   500,
		CurrentBid:    600,
		HighBidderId:  700,
		MinIncrement:  800,
		BidCount:      1,
	}

	t.Run("nil EndsAt", func(t *testing.T) {
		rm := base
		rm.EndsAt = nil

		m, err := Extract(rm)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		rm2, err := Transform(m)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if rm2.Id != rm.Id {
			t.Errorf("Id mismatch. Expected %s, got %s", rm.Id, rm2.Id)
		}
		if rm2.EndsAt != nil {
			t.Errorf("expected nil EndsAt to survive round trip, got %v", rm2.EndsAt)
		}

		m2, err := Extract(rm2)
		if err != nil {
			t.Fatalf("Extract (second pass) failed: %v", err)
		}
		if !reflect.DeepEqual(m, m2) {
			t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
		}
	})

	t.Run("non-nil EndsAt", func(t *testing.T) {
		endsAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		rm := base
		rm.EndsAt = &endsAt

		m, err := Extract(rm)
		if err != nil {
			t.Fatalf("Extract failed: %v", err)
		}

		rm2, err := Transform(m)
		if err != nil {
			t.Fatalf("Transform failed: %v", err)
		}

		if rm2.Id != rm.Id {
			t.Errorf("Id mismatch. Expected %s, got %s", rm.Id, rm2.Id)
		}
		if rm2.EndsAt == nil || !rm2.EndsAt.Equal(endsAt) {
			t.Errorf("expected EndsAt %v, got %v", endsAt, rm2.EndsAt)
		}
		if rm2.EndsAt == rm.EndsAt {
			t.Errorf("expected Transform to copy EndsAt into a new pointer, not alias the source")
		}

		m2, err := Extract(rm2)
		if err != nil {
			t.Fatalf("Extract (second pass) failed: %v", err)
		}
		if !reflect.DeepEqual(m, m2) {
			t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
		}
	})
}
