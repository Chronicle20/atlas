package listing_test

import (
	"atlas-mts/holding"
	"atlas-mts/listing"
	"atlas-mts/test"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// newCancelServer wires the listing routes onto an httptest server backed by a DB
// migrated with both the listing and holding schemas (Cancel writes a holding).
func newCancelServer(t *testing.T) (*httptest.Server, *gorm.DB, func()) {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	db := test.SetupTestDB(t, listing.Migration, holding.Migration)
	if err := db.Exec("DELETE FROM listings").Error; err != nil {
		t.Fatalf("reset listings: %v", err)
	}
	if err := db.Exec("DELETE FROM holdings").Error; err != nil {
		t.Fatalf("reset holdings: %v", err)
	}
	router := mux.NewRouter()
	listing.InitResource(testServerInfo{})(db)(router, logger)
	srv := httptest.NewServer(router)
	cleanup := func() {
		srv.Close()
		test.CleanupTestDB(t, db)
	}
	return srv, db, cleanup
}

func seedCancelListing(t *testing.T, db *gorm.DB, sellerId uint32) listing.Model {
	t.Helper()
	m, err := listing.NewBuilder(test.TestTenantId, 0, sellerId).
		SetSellerName("Seller").
		SetSaleType(listing.SaleTypeFixed).
		SetState(listing.StateActive).
		SetTemplateId(1302000).
		SetQuantity(1).
		SetListValue(1000).
		SetCommissionRate(0.10).
		SetCategory("equip").
		SetSubCategory("onehand").
		Build()
	if err != nil {
		t.Fatalf("build listing: %v", err)
	}
	stored, err := listing.CreateListing(db, m)
	if err != nil {
		t.Fatalf("seed listing: %v", err)
	}
	return stored
}

// TestCancelRoute_SellerSucceeds asserts the seller can cancel their own active
// listing: 204 No Content, the row is cancelled, and a seller holding is created.
func TestCancelRoute_SellerSucceeds(t *testing.T) {
	srv, db, cleanup := newCancelServer(t)
	defer cleanup()

	const sellerId = uint32(6660001)
	created := seedCancelListing(t, db, sellerId)

	client := &http.Client{}
	url := fmt.Sprintf("%s/worlds/0/listings/%s?characterId=%d", srv.URL, created.Id().String(), sellerId)
	resp, err := client.Do(withTenant(t, http.MethodDelete, url))
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("cancel status = %d, want 204", resp.StatusCode)
	}

	stored, err := listing.GetById(created.Id().String())(db.WithContext(test.CreateTestContext()))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateCancelled {
		t.Fatalf("expected listing cancelled, got %s", stored.State())
	}
}

// TestCancelRoute_NonSellerForbidden asserts a character who is not the listing's
// seller cannot cancel it: 403 Forbidden, the row stays active, no holding.
func TestCancelRoute_NonSellerForbidden(t *testing.T) {
	srv, db, cleanup := newCancelServer(t)
	defer cleanup()

	const sellerId = uint32(6660002)
	const intruderId = uint32(6669999)
	created := seedCancelListing(t, db, sellerId)

	client := &http.Client{}
	url := fmt.Sprintf("%s/worlds/0/listings/%s?characterId=%d", srv.URL, created.Id().String(), intruderId)
	resp, err := client.Do(withTenant(t, http.MethodDelete, url))
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-seller cancel status = %d, want 403", resp.StatusCode)
	}

	// listing remains active — the non-seller's attempt did not transition it
	stored, err := listing.GetById(created.Id().String())(db.WithContext(test.CreateTestContext()))()
	if err != nil {
		t.Fatalf("listing lookup: %v", err)
	}
	if stored.State() != listing.StateActive {
		t.Fatalf("expected listing to remain active after forbidden cancel, got %s", stored.State())
	}
}

// TestCancelRoute_NonActiveConflict asserts cancelling an already-settled listing
// is a clean 409 Conflict (race loser), not a 500.
func TestCancelRoute_NonActiveConflict(t *testing.T) {
	srv, db, cleanup := newCancelServer(t)
	defer cleanup()

	const sellerId = uint32(6660003)
	created := seedCancelListing(t, db, sellerId)

	// Simulate a concurrent buy winning: the listing is already sold.
	if _, err := listing.UpdateState(db.WithContext(test.CreateTestContext()), created.Id().String(), listing.StateActive, listing.StateSold); err != nil {
		t.Fatalf("simulate concurrent buy: %v", err)
	}

	client := &http.Client{}
	url := fmt.Sprintf("%s/worlds/0/listings/%s?characterId=%d", srv.URL, created.Id().String(), sellerId)
	resp, err := client.Do(withTenant(t, http.MethodDelete, url))
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("non-active cancel status = %d, want 409", resp.StatusCode)
	}
}

// TestCancelRoute_MissingCharacterIdBadRequest asserts a cancel without the
// characterId query param (the caller identity for the seller-only check) is a
// 400, not a panic or silent success.
func TestCancelRoute_MissingCharacterIdBadRequest(t *testing.T) {
	srv, db, cleanup := newCancelServer(t)
	defer cleanup()

	created := seedCancelListing(t, db, 6660004)

	client := &http.Client{}
	url := fmt.Sprintf("%s/worlds/0/listings/%s", srv.URL, created.Id().String())
	resp, err := client.Do(withTenant(t, http.MethodDelete, url))
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing characterId cancel status = %d, want 400", resp.StatusCode)
	}
}
