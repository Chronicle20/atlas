package monsterbook

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// TestCollectionRestModel_Unmarshal asserts the JSON:API stubs are wired so
// api2go.Unmarshal succeeds and every documented attribute round-trips into
// the wire model. This is the regression guard for EXT-01..03 (missing
// reference stubs causing decode failures even when no relationships are
// present).
func TestCollectionRestModel_Unmarshal(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "monster-book",
			"id": "42",
			"attributes": {
				"bookLevel": 3,
				"normalCount": 5,
				"specialCount": 2,
				"totalUniqueCards": 7,
				"coverCardId": 2380000,
				"expBonusPercent": 3
			}
		}
	}`)

	var rm CollectionRestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}

	if rm.GetID() != "42" {
		t.Errorf("GetID() = %q, want %q", rm.GetID(), "42")
	}
	if rm.Id != 42 {
		t.Errorf("Id = %d, want 42", rm.Id)
	}
	if rm.BookLevel != 3 {
		t.Errorf("BookLevel = %d, want 3", rm.BookLevel)
	}
	if rm.NormalCount != 5 {
		t.Errorf("NormalCount = %d, want 5", rm.NormalCount)
	}
	if rm.SpecialCount != 2 {
		t.Errorf("SpecialCount = %d, want 2", rm.SpecialCount)
	}
	if rm.TotalUniqueCards != 7 {
		t.Errorf("TotalUniqueCards = %d, want 7", rm.TotalUniqueCards)
	}
	if rm.CoverCardId != item.Id(2380000) {
		t.Errorf("CoverCardId = %d, want 2380000", rm.CoverCardId)
	}
	if rm.ExpBonusPercent != 3 {
		t.Errorf("ExpBonusPercent = %d, want 3", rm.ExpBonusPercent)
	}
}

// TestCollectionRestModel_ReferenceStubs documents that the JSON:API
// reference interface methods exist and return empty/no-op values. The
// presence of these methods is what api2go.Unmarshal checks for via
// type assertion when walking a document; if the upstream ever adds
// a `relationships` block, this guarantees decode does not error.
func TestCollectionRestModel_ReferenceStubs(t *testing.T) {
	var rm CollectionRestModel
	if refs := rm.GetReferences(); len(refs) != 0 {
		t.Errorf("GetReferences() len = %d, want 0", len(refs))
	}
	if ids := rm.GetReferencedIDs(); len(ids) != 0 {
		t.Errorf("GetReferencedIDs() len = %d, want 0", len(ids))
	}
	if err := rm.SetToOneReferenceID("any", "id"); err != nil {
		t.Errorf("SetToOneReferenceID: %v", err)
	}
	if err := rm.SetToManyReferenceIDs("any", []string{"a", "b"}); err != nil {
		t.Errorf("SetToManyReferenceIDs: %v", err)
	}
}

// TestExtract asserts the wire→domain mapping preserves every attribute,
// independent of any HTTP plumbing.
func TestExtract(t *testing.T) {
	rm := CollectionRestModel{
		Id:               42,
		BookLevel:        3,
		NormalCount:      5,
		SpecialCount:     2,
		TotalUniqueCards: 7,
		CoverCardId:      item.Id(2380000),
		ExpBonusPercent:  3,
	}
	c, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if c.BookLevel() != 3 || c.NormalCount() != 5 || c.SpecialCount() != 2 ||
		c.TotalUniqueCards() != 7 || c.CoverCardId() != item.Id(2380000) ||
		c.ExpBonusPercent() != 3 {
		t.Fatalf("Extract round-trip mismatch: %#v", c)
	}
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestGetByCharacterId_RoundTrip stands up an httptest server returning a
// canned monster-book JSON:API document and asserts NewProcessor.GetByCharacterId
// decodes it into a populated Collection. This is the integration test
// libs/atlas-rest/CLAUDE.md prescribes for every external client.
func TestGetByCharacterId_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/characters/42/monster-book") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "monster-book",
				"id": "42",
				"attributes": {
					"bookLevel": 3,
					"normalCount": 5,
					"specialCount": 2,
					"totalUniqueCards": 7,
					"coverCardId": 2380000,
					"expBonusPercent": 3
				}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	col, err := NewProcessor(logrus.New(), ctx).GetByCharacterId(character.Id(42))
	if err != nil {
		t.Fatalf("GetByCharacterId: %v", err)
	}
	if col.BookLevel() != 3 {
		t.Errorf("BookLevel = %d, want 3", col.BookLevel())
	}
	if col.CoverCardId() != item.Id(2380000) {
		t.Errorf("CoverCardId = %d, want 2380000", col.CoverCardId())
	}
	if col.NormalCount() != 5 || col.SpecialCount() != 2 || col.TotalUniqueCards() != 7 || col.ExpBonusPercent() != 3 {
		t.Errorf("collection mismatch: %#v", col)
	}
}

// TestGetByCharacterId_NotFound asserts a 404 from the upstream surfaces as
// requests.ErrNotFound (per libs/atlas-rest/CLAUDE.md), so callers can
// distinguish "no collection yet" from a deploy-time bug.
func TestGetByCharacterId_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).GetByCharacterId(character.Id(42))
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("expected requests.ErrNotFound, got %T: %v", err, err)
	}
}

func TestCardRestModel_Unmarshal(t *testing.T) {
	body := []byte(`{
		"data": [
			{"type":"monster-book-card","id":"2380005","attributes":{"level":2,"isSpecial":false}},
			{"type":"monster-book-card","id":"2382000","attributes":{"level":5,"isSpecial":true}}
		]
	}`)
	var rms []CardRestModel
	if err := jsonapi.Unmarshal(body, &rms); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}
	if len(rms) != 2 {
		t.Fatalf("len = %d, want 2", len(rms))
	}
	if rms[0].CardId != item.Id(2380005) || rms[0].Level != 2 || rms[0].IsSpecial {
		t.Errorf("card[0] = %+v", rms[0])
	}
	if rms[1].CardId != item.Id(2382000) || rms[1].Level != 5 || !rms[1].IsSpecial {
		t.Errorf("card[1] = %+v", rms[1])
	}
}

func TestGetCardsByCharacterId_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/characters/42/monster-book/cards") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"type":"monster-book-card","id":"2380005","attributes":{"level":2,"isSpecial":false}},
				{"type":"monster-book-card","id":"2382000","attributes":{"level":5,"isSpecial":true}}
			]
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	cards, err := NewProcessor(logrus.New(), ctx).GetCardsByCharacterId(character.Id(42))
	if err != nil {
		t.Fatalf("GetCardsByCharacterId: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("len = %d, want 2", len(cards))
	}
	if cards[0].CardId() != item.Id(2380005) || cards[0].Level() != 2 {
		t.Errorf("card[0] = cardId %d level %d", cards[0].CardId(), cards[0].Level())
	}
}

func TestGetCardsByCharacterId_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).GetCardsByCharacterId(character.Id(42))
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("expected requests.ErrNotFound, got %T: %v", err, err)
	}
}

func TestExtractIncludesCoverMonsterId(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "monster-book",
			"id": "42",
			"attributes": {
				"bookLevel": 3,
				"normalCount": 5,
				"specialCount": 2,
				"totalUniqueCards": 7,
				"coverCardId": 2380000,
				"coverMonsterId": 100100,
				"expBonusPercent": 3
			}
		}
	}`)
	var rm CollectionRestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}
	if rm.CoverMonsterId != 100100 {
		t.Fatalf("CoverMonsterId = %d, want 100100", rm.CoverMonsterId)
	}
	c, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if c.CoverMonsterId() != 100100 {
		t.Fatalf("Collection.CoverMonsterId() = %d, want 100100", c.CoverMonsterId())
	}
	if c.CoverCardId() != item.Id(2380000) {
		t.Fatalf("Collection.CoverCardId() = %d, want 2380000 (must remain card id)", c.CoverCardId())
	}
}
