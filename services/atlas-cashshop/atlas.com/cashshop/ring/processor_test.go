package ring

// TestGetByCharacterIdEnrichesCashIdAndPartnerName proves task 8's read-time
// enrichment (design.md §5): ring.ProcessorImpl.GetByCharacterId adds
// CashId, PartnerCashId, and PartnerName on top of what GetByCharacterId
// (provider.go) reads off cash_rings, without ever storing them on Entity.
// Every one of the three resolutions -- this half's own asset, the sibling
// half, the partner's name -- fails soft to the zero value: a lookup
// failure must not turn into an error, since PRD FR-5's channel-side
// fallback is downstream of this.
import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"

	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/character"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ringAssetDatabase mirrors testDatabase (administrator_test.go), adding
// the asset table the enrichment step reads CashId off of.
func ringAssetDatabase(t *testing.T) (*gorm.DB, tenant.Model) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration, asset.Migration)
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return db, tm
}

func seedAsset(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uint32, cashId int64) {
	t.Helper()
	e := asset.Entity{
		Id:            id,
		TenantId:      tenantId,
		CompartmentId: uuid.New(),
		CashId:        cashId,
		TemplateId:    1112800,
		Quantity:      1,
		PurchasedBy:   1,
		Expiration:    time.Now(),
		CreatedAt:     time.Now(),
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("seedAsset: %v", err)
	}
}

// startCharacterServer serves GET /api/characters/{id} for the given
// name fixtures, mirroring cashshop.startRingCharacterServer.
func startCharacterServer(t *testing.T, names map[uint32]string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var id uint32
		if _, err := fmt.Sscanf(r.URL.Path, "/api/characters/%d", &id); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		name, ok := names[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"characters","id":"%d","attributes":{"accountId":1,"jobId":0,"name":"%s"}}}`, id, name)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")
}

func newRingProcessor(db *gorm.DB, tm tenant.Model) *ProcessorImpl {
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tm)
	chaP := character.NewProcessor(l, ctx)
	return NewProcessor(l, ctx, db, chaP).(*ProcessorImpl)
}

func TestGetByCharacterIdEnrichesCashIdAndPartnerName(t *testing.T) {
	t.Run("both halves present", func(t *testing.T) {
		db, tm := ringAssetDatabase(t)
		startCharacterServer(t, map[uint32]string{42: "Buyer", 77: "Partner"})

		_, err := CreatePair(db, tm.Id(), TypeFriendship,
			Half{CharacterId: 42, AssetId: 2001, ItemTemplateId: 1112800},
			Half{CharacterId: 77, AssetId: 2002, ItemTemplateId: 1112800},
		)
		if err != nil {
			t.Fatalf("CreatePair: %v", err)
		}
		seedAsset(t, db, tm.Id(), 2001, 9001)
		seedAsset(t, db, tm.Id(), 2002, 9002)

		p := newRingProcessor(db, tm)

		buyerRows, err := p.GetByCharacterId(42)
		if err != nil {
			t.Fatalf("GetByCharacterId(42): %v", err)
		}
		if len(buyerRows) != 1 {
			t.Fatalf("GetByCharacterId(42) = %d rows, want 1", len(buyerRows))
		}
		if buyerRows[0].CashId() != 9001 {
			t.Errorf("buyer CashId = %d, want 9001", buyerRows[0].CashId())
		}
		if buyerRows[0].PartnerCashId() != 9002 {
			t.Errorf("buyer PartnerCashId = %d, want 9002", buyerRows[0].PartnerCashId())
		}
		if buyerRows[0].PartnerName() != "Partner" {
			t.Errorf("buyer PartnerName = %q, want Partner", buyerRows[0].PartnerName())
		}

		partnerRows, err := p.GetByCharacterId(77)
		if err != nil {
			t.Fatalf("GetByCharacterId(77): %v", err)
		}
		if len(partnerRows) != 1 {
			t.Fatalf("GetByCharacterId(77) = %d rows, want 1", len(partnerRows))
		}
		if partnerRows[0].CashId() != 9002 {
			t.Errorf("partner CashId = %d, want 9002", partnerRows[0].CashId())
		}
		if partnerRows[0].PartnerCashId() != 9001 {
			t.Errorf("partner PartnerCashId = %d, want 9001", partnerRows[0].PartnerCashId())
		}
		if partnerRows[0].PartnerName() != "Buyer" {
			t.Errorf("partner PartnerName = %q, want Buyer", partnerRows[0].PartnerName())
		}
	})

	t.Run("sibling row missing", func(t *testing.T) {
		db, tm := ringAssetDatabase(t)
		startCharacterServer(t, map[uint32]string{42: "Buyer", 77: "Partner"})

		e := Entity{
			Id:                 uuid.New(),
			TenantId:           tm.Id(),
			PairId:             uuid.New(),
			CharacterId:        42,
			PartnerCharacterId: 77,
			AssetId:            2001,
			ItemTemplateId:     1112800,
			RingType:           string(TypeFriendship),
			State:              string(StateActive),
			CreatedAt:          time.Now(),
		}
		if err := db.Create(&e).Error; err != nil {
			t.Fatalf("seed lone half: %v", err)
		}
		seedAsset(t, db, tm.Id(), 2001, 9001)

		p := newRingProcessor(db, tm)

		rows, err := p.GetByCharacterId(42)
		if err != nil {
			t.Fatalf("GetByCharacterId(42): %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("GetByCharacterId(42) = %d rows, want 1", len(rows))
		}
		if rows[0].CashId() != 9001 {
			t.Errorf("CashId = %d, want 9001", rows[0].CashId())
		}
		if rows[0].PartnerCashId() != 0 {
			t.Errorf("PartnerCashId = %d, want 0 (no sibling)", rows[0].PartnerCashId())
		}
	})

	t.Run("character service unavailable", func(t *testing.T) {
		db, tm := ringAssetDatabase(t)
		// Start and immediately close a server: the URL is well-formed but
		// nothing is listening, so every request fails with a connection
		// error -- unlike an unset env var, this cannot silently resolve to
		// the legacy "" root.
		dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		dead.Close()
		t.Setenv("CHARACTERS_SERVICE_URL", dead.URL+"/api/")

		_, err := CreatePair(db, tm.Id(), TypeFriendship,
			Half{CharacterId: 42, AssetId: 2001, ItemTemplateId: 1112800},
			Half{CharacterId: 77, AssetId: 2002, ItemTemplateId: 1112800},
		)
		if err != nil {
			t.Fatalf("CreatePair: %v", err)
		}
		seedAsset(t, db, tm.Id(), 2001, 9001)
		seedAsset(t, db, tm.Id(), 2002, 9002)

		p := newRingProcessor(db, tm)

		rows, err := p.GetByCharacterId(42)
		if err != nil {
			t.Fatalf("GetByCharacterId(42): %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("GetByCharacterId(42) = %d rows, want 1", len(rows))
		}
		if rows[0].PartnerName() != "" {
			t.Errorf("PartnerName = %q, want empty (character service unavailable)", rows[0].PartnerName())
		}
	})
}
