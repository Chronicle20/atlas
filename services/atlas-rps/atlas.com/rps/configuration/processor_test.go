package configuration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// rpsRewardsBody is a faithful JSON:API document for the atlas-tenants
// rps-rewards configuration resource. atlas-tenants serves configuration
// resources uniformly as a COLLECTION (`"data": [{...}]`), so the fixture is
// an array. `ladder` is a plain nested JSON array attribute (not a JSON:API
// relationship).
const rpsRewardsBody = `{
  "data": [
    {
      "type": "rps-rewards",
      "id": "rps-rewards",
      "attributes": {
        "entryCostMeso": 1000,
        "ladder": [
          {"rung": 1, "itemId": 0, "quantity": 0, "meso": 2000},
          {"rung": 2, "itemId": 4000000, "quantity": 1, "meso": 0},
          {"rung": 3, "itemId": 4000001, "quantity": 5, "meso": 5000}
        ]
      }
    }
  ]
}`

// rpsRewardsEmptyBody is the collection response for a tenant with no
// rps-rewards configuration record.
const rpsRewardsEmptyBody = `{"data": []}`

func TestGetLadder_DecodesCollectionDocument(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(rpsRewardsBody))
	}))
	defer srv.Close()

	// RootUrl("TENANTS") reads TENANTS_SERVICE_URL per call.
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	p := NewProcessor(logrus.New(), context.Background())

	tenantId := uuid.New()
	ladder, err := p.GetLadder(tenantId)
	if err != nil {
		t.Fatalf("GetLadder() error = %v", err)
	}

	if ladder.EntryCostMeso != 1000 {
		t.Errorf("EntryCostMeso = %d, want 1000", ladder.EntryCostMeso)
	}

	if len(ladder.Rungs) != 3 {
		t.Fatalf("len(Rungs) = %d, want 3", len(ladder.Rungs))
	}

	want := []struct {
		rung     int
		itemId   uint32
		quantity uint32
		meso     uint32
	}{
		{1, 0, 0, 2000},
		{2, 4000000, 1, 0},
		{3, 4000001, 5, 5000},
	}
	for i, w := range want {
		r := ladder.Rungs[i]
		if r.Rung != w.rung {
			t.Errorf("Rungs[%d].Rung = %d, want %d", i, r.Rung, w.rung)
		}
		if uint32(r.ItemId) != w.itemId {
			t.Errorf("Rungs[%d].ItemId = %d, want %d", i, uint32(r.ItemId), w.itemId)
		}
		if r.Quantity != w.quantity {
			t.Errorf("Rungs[%d].Quantity = %d, want %d", i, r.Quantity, w.quantity)
		}
		if r.Meso != w.meso {
			t.Errorf("Rungs[%d].Meso = %d, want %d", i, r.Meso, w.meso)
		}
	}

	wantPath := "/tenants/" + tenantId.String() + "/configurations/rps-rewards"
	if gotPath != wantPath {
		t.Errorf("request path = %q, want %q", gotPath, wantPath)
	}
}

func TestGetLadder_EmptyCollectionReturnsSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(rpsRewardsEmptyBody))
	}))
	defer srv.Close()

	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	p := NewProcessor(logrus.New(), context.Background())

	_, err := p.GetLadder(uuid.New())
	if !errors.Is(err, ErrNoRewardConfig) {
		t.Fatalf("GetLadder() error = %v, want ErrNoRewardConfig", err)
	}
}

func TestGetLadder_PrizeAtResolvesByRungNotPosition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(rpsRewardsBody))
	}))
	defer srv.Close()

	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	p := NewProcessor(logrus.New(), context.Background())

	ladder, err := p.GetLadder(uuid.New())
	if err != nil {
		t.Fatalf("GetLadder() error = %v", err)
	}

	prize, ok := ladder.PrizeAt(2)
	if !ok {
		t.Fatalf("PrizeAt(2) ok = false, want true")
	}
	if uint32(prize.ItemId) != 4000000 || prize.Quantity != 1 {
		t.Errorf("PrizeAt(2) = %+v, want ItemId 4000000 Quantity 1", prize)
	}
}
