package character

import (
	"atlas-mts/kafka/message/character"
	"atlas-mts/listing"
	"atlas-mts/test"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// seedListing persists an active listing for sellerId with the given seller name
// and returns its id.
func seedListing(t *testing.T, db *gorm.DB, sellerId uint32, sellerName string) uuid.UUID {
	t.Helper()
	m, err := listing.NewBuilder(test.TestTenantId, 0, sellerId).
		SetSellerName(sellerName).
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("equip").
		SetSubCategory("one-handed-sword").
		Build()
	if err != nil {
		t.Fatalf("build seed listing: %v", err)
	}
	created, err := listing.CreateListing(db, m)
	if err != nil {
		t.Fatalf("create seed listing: %v", err)
	}
	return created.Id()
}

// sellerNameOf reads back the stored seller_name for a listing by id.
func sellerNameOf(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	got, err := listing.GetById(id.String())(db)()
	if err != nil {
		t.Fatalf("GetById(%s): %v", id, err)
	}
	return got.SellerName()
}

func nameChangedEvent(characterId uint32, oldName string, newName string) character.StatusEvent[character.StatusEventNameChangedBody] {
	return character.StatusEvent[character.StatusEventNameChangedBody]{
		TransactionId: uuid.New(),
		WorldId:       0,
		CharacterId:   characterId,
		Type:          character.StatusEventTypeNameChanged,
		Body: character.StatusEventNameChangedBody{
			OldName: oldName,
			NewName: newName,
		},
	}
}

// TestNameChangedUpdatesEverySellerListing asserts a seller holding several
// listings has ALL of them renamed, regardless of state — see updateSellerName
// for the rationale (seller_name is a current display identity, not a
// point-in-time sale record).
func TestNameChangedUpdatesEverySellerListing(t *testing.T) {
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration).WithContext(test.CreateTestContext())
	defer test.CleanupTestDB(t, db)

	id1 := seedListing(t, db, 1, "Yankee")
	id2 := seedListing(t, db, 1, "Yankee")

	handleCharacterNameChanged(db)(logger, test.CreateTestContext(), nameChangedEvent(1, "Yankee", "Zulu"))

	if got := sellerNameOf(t, db, id1); got != "Zulu" {
		t.Errorf("listing %s seller = %s, want Zulu", id1, got)
	}
	if got := sellerNameOf(t, db, id2); got != "Zulu" {
		t.Errorf("listing %s seller = %s, want Zulu", id2, got)
	}
}

// TestNameChangedRenamesSoldAndCancelledListingsToo asserts the rename is NOT
// scoped to active listings: sold/cancelled/expired rows for the same seller are
// renamed identically, per the controller-ruled "current identity, not a
// point-in-time snapshot" decision.
func TestNameChangedRenamesSoldAndCancelledListingsToo(t *testing.T) {
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration).WithContext(test.CreateTestContext())
	defer test.CleanupTestDB(t, db)

	active := seedListing(t, db, 2, "Yankee")
	m, err := listing.NewBuilder(test.TestTenantId, 0, 2).
		SetSellerName("Yankee").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateSold).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("equip").
		SetSubCategory("one-handed-sword").
		Build()
	if err != nil {
		t.Fatalf("build sold listing: %v", err)
	}
	sold, err := listing.CreateListing(db, m)
	if err != nil {
		t.Fatalf("create sold listing: %v", err)
	}

	handleCharacterNameChanged(db)(logger, test.CreateTestContext(), nameChangedEvent(2, "Yankee", "Zulu"))

	if got := sellerNameOf(t, db, active); got != "Zulu" {
		t.Errorf("active listing %s seller = %s, want Zulu", active, got)
	}
	if got := sellerNameOf(t, db, sold.Id()); got != "Zulu" {
		t.Errorf("sold listing %s seller = %s, want Zulu", sold.Id(), got)
	}
}

// TestNameChangedNoListingsIsNoop asserts a NAME_CHANGED for a seller with no
// listings does not error.
func TestNameChangedNoListingsIsNoop(t *testing.T) {
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration).WithContext(test.CreateTestContext())
	defer test.CleanupTestDB(t, db)

	handleCharacterNameChanged(db)(logger, test.CreateTestContext(), nameChangedEvent(999, "Nobody", "StillNobody"))
}

// TestNameChangedIgnoresOtherEventTypes asserts the handler's type guard skips
// any status event on the topic that is not NAME_CHANGED, leaving listings
// untouched.
func TestNameChangedIgnoresOtherEventTypes(t *testing.T) {
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration).WithContext(test.CreateTestContext())
	defer test.CleanupTestDB(t, db)

	id := seedListing(t, db, 3, "Yankee")

	ev := nameChangedEvent(3, "Yankee", "Zulu")
	ev.Type = "LOGIN"
	handleCharacterNameChanged(db)(logger, test.CreateTestContext(), ev)

	if got := sellerNameOf(t, db, id); got != "Yankee" {
		t.Errorf("listing %s seller = %s, want Yankee (unchanged)", id, got)
	}
}

// TestNameChangedRedeliveryIsIdempotent asserts replaying the same event twice
// leaves the listing at NewName, with no error on the second delivery.
func TestNameChangedRedeliveryIsIdempotent(t *testing.T) {
	logger := logrus.New()
	db := test.SetupTestDB(t, listing.Migration).WithContext(test.CreateTestContext())
	defer test.CleanupTestDB(t, db)

	id := seedListing(t, db, 4, "Yankee")
	ev := nameChangedEvent(4, "Yankee", "Zulu")

	handleCharacterNameChanged(db)(logger, test.CreateTestContext(), ev)
	handleCharacterNameChanged(db)(logger, test.CreateTestContext(), ev)

	if got := sellerNameOf(t, db, id); got != "Zulu" {
		t.Errorf("listing %s seller = %s, want Zulu", id, got)
	}
}
