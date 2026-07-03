package storage_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-asset-expiration/storage"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus/hooks/test"
)

// storageAssetDoc renders a JSON:API document for storage assets [from, to].
// meta describes the current page/total so requests.DrainProvider can
// decide whether to keep paging.
func storageAssetDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"storage_assets","attributes":{"templateId":4000000,"slot":%d,"expiration":"2024-01-01T00:00:00Z"}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetAssetsDrainsBeyondOnePage proves GetAssets (via requests.DrainProvider)
// fetches every page of an account's storage assets rather than stopping
// after the first response. atlas-storage's storage assets list is now
// paginated (task-117); the fixture server serves 300 assets across two
// pages of 250 -- only a genuine drain picks up asset id 300, which lives
// on page 2. Expiration checks (checkStorage) must see every asset or a
// truncated first-page-only fetch would silently stop expiring items past
// the 250th in storage.
func TestGetAssetsDrainsBeyondOnePage(t *testing.T) {
	accountId := uint32(7)
	worldId := world.Id(0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(storageAssetDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(storageAssetDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("STORAGE_SERVICE_URL", srv.URL+"/")

	ctx := context.Background()
	l, _ := test.NewNullLogger()

	assets, err := storage.GetAssets(l)(ctx)(accountId, worldId)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 300 {
		t.Fatalf("expected 300 assets (full drain), got %d; a single-fetch implementation would return 250", len(assets))
	}

	foundLast := false
	for _, a := range assets {
		if a.Id == "300" {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("asset id 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
