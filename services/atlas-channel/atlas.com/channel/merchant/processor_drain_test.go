package merchant_test

import (
	"atlas-channel/merchant"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// merchantDoc renders a JSON:API document for n shops, all owned by
// characterId, with deterministic uuids (derived from the given ids) so a
// test can assert a specific one made it into the drained result. meta
// describes the current page/total so requests.DrainProvider can decide
// whether to keep paging.
func merchantDoc(characterId uint32, ids []int, total, number, size, last int) string {
	var b strings.Builder
	for _, id := range ids {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		shopId := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("merchant-shop-%d", id)))
		b.WriteString(fmt.Sprintf(
			`{"id":"%s","type":"merchants","attributes":{"characterId":%d,"shopType":1,"state":2,"title":"shop-%d","worldId":0,"channelId":0,"mapId":910000001,"instanceId":"00000000-0000-0000-0000-000000000000","x":0,"y":0,"permitItemId":0,"mesoBalance":0,"listingCount":0}}`,
			shopId.String(), characterId, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

func idRange(from, to int) []int {
	var ids []int
	for i := from; i <= to; i++ {
		ids = append(ids, i)
	}
	return ids
}

// TestGetByCharacterIdDrainsBeyondOnePage proves GetByCharacterId (via
// requests.DrainProvider) fetches every page of a character's shops rather
// than stopping after the first response. atlas-merchant's GET
// /characters/{characterId}/merchants is now paginated (task-117); the
// fixture server serves 260 shops across two pages of 250 -- only a genuine
// drain picks up shop 260, which lives on page 2.
func TestGetByCharacterIdDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(merchantDoc(characterId, idRange(251, 260), 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(merchantDoc(characterId, idRange(1, 250), 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("MERCHANT_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := merchant.NewProcessor(l, ctx).GetByCharacterId(characterId)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 260 {
		t.Fatalf("expected 260 shops (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	lastId := uuid.NewSHA1(uuid.NameSpaceOID, []byte("merchant-shop-260"))
	foundLast := false
	for _, m := range ms {
		if m.Id() == lastId {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("shop 260 (page 2) must be present; single-fetch impl would miss it")
	}
}

// TestInFieldModelProviderDrainsBeyondOnePage is the field-scoped analogue
// of the above, covering InFieldModelProvider (used by session spawn on map
// load, which needs every shop visible on the field, not one page of it).
func TestInFieldModelProviderDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(99)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(merchantDoc(characterId, idRange(251, 251), 251, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(merchantDoc(characterId, idRange(1, 250), 251, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("MERCHANT_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(0, 0, 910000001).Build()

	ms, err := merchant.NewProcessor(l, ctx).InFieldModelProvider(f)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 251 {
		t.Fatalf("expected 251 shops (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}
}
