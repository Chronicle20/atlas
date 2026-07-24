package buff_test

import (
	"atlas-monsters/character/buff"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// buffsDoc renders a JSON:API "buffs" collection page carrying a single
// resource with the given sourceId/expiresAt attributes. meta describes the
// current page/total so requests.DrainProvider (GetByCharacterId's
// transport) can decide whether to keep paging -- mirrors the fixture
// pattern in ../../map/processor_drain_test.go.
func buffsDoc(id string, sourceId int32, expiresAt time.Time, total, number, size, last int) string {
	return fmt.Sprintf(
		`{"data":[{"id":"%s","type":"buffs","attributes":{"sourceId":%d,"expiresAt":"%s"}}],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		id, sourceId, expiresAt.Format(time.RFC3339), total, number, size, last,
	)
}

// TestGetByCharacterId_HTTPRoundTrip exercises GetByCharacterId's real
// unmarshal path (requests.DrainProvider -> JSON:API decode -> Extract),
// not an injected seam. An active (non-expired) SuperGmHide buff is served
// and the decoded Model must round-trip sourceId/expiresAt well enough for
// HasActiveGmHide to recognize it -- a struct-shape regression (e.g. a typo'd
// json tag) would decode to the zero value and fail this assertion.
func TestGetByCharacterId_HTTPRoundTrip_ActiveGmHide(t *testing.T) {
	const superGmHideId = int32(9101004) // skill.SuperGmHideId
	expiresAt := time.Now().Add(1 * time.Hour)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(buffsDoc("1", superGmHideId, expiresAt, 1, 1, 250, 1)))
	}))
	defer srv.Close()
	t.Setenv("BUFFS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(42)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 1 {
		t.Fatalf("expected 1 decoded buff, got %d", len(bs))
	}
	if bs[0].SourceId() != superGmHideId {
		t.Fatalf("SourceId() = %d, want %d (decode of \"sourceId\" attribute failed)", bs[0].SourceId(), superGmHideId)
	}
	if !buff.HasActiveGmHide(bs) {
		t.Fatal("HasActiveGmHide(bs) = false, want true for an unexpired SuperGmHide buff (expiresAt decode failed)")
	}
}

// TestGetByCharacterId_HTTPRoundTrip_ExpiredBuffNotActive proves the
// expiresAt decode is a real timestamp, not a fixed/zero value: an
// already-expired SuperGmHide buff must NOT read as active.
func TestGetByCharacterId_HTTPRoundTrip_ExpiredBuffNotActive(t *testing.T) {
	const superGmHideId = int32(9101004)
	expiresAt := time.Now().Add(-1 * time.Hour)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(buffsDoc("2", superGmHideId, expiresAt, 1, 1, 250, 1)))
	}))
	defer srv.Close()
	t.Setenv("BUFFS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(42)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 1 {
		t.Fatalf("expected 1 decoded buff, got %d", len(bs))
	}
	if buff.HasActiveGmHide(bs) {
		t.Fatal("HasActiveGmHide(bs) = true, want false for an expired buff (expiresAt decode returned wrong/zero time)")
	}
}
