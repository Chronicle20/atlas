package maps

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// F2: no test previously exercised the HTTP/decode path for the maps
// client -- only pure functions were covered. This drives a real fixture
// of a paginated character-list page through the exact path
// CharacterIdsInMap uses (requests.DrainProvider -> jsonapi.Unmarshal),
// asserting the character ids come out correctly. Per libs/atlas-rest/CLAUDE.md,
// fixture-based decode tests are how F1-shaped defects (a missing api2go
// stub) get caught before merge -- this pins that maps.RestModel already
// has no such defect.
func charactersPageDoc(ids []uint32) string {
	data := ""
	for _, id := range ids {
		if data != "" {
			data += ","
		}
		data += fmt.Sprintf(`{"id":"%d","type":"characters"}`, id)
	}
	return fmt.Sprintf(`{"data":[%s],"meta":{"total":%d,"page":{"number":1,"size":250,"last":1}}}`, data, len(ids))
}

func TestCharacterIdsInMapDecodesPagedFixture(t *testing.T) {
	want := []uint32{100001, 100002, 100003}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(charactersPageDoc(want)))
	}))
	defer srv.Close()

	oldEnv, hadEnv := os.LookupEnv("MAPS_SERVICE_URL")
	if err := os.Setenv("MAPS_SERVICE_URL", srv.URL+"/"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if hadEnv {
			_ = os.Setenv("MAPS_SERVICE_URL", oldEnv)
		} else {
			_ = os.Unsetenv("MAPS_SERVICE_URL")
		}
	}()

	l, _ := test.NewNullLogger()
	f := field.NewBuilder(1, 2, 100000000).SetInstance(uuid.New()).Build()
	p := NewProcessor(l, context.Background())

	got, err := p.CharacterIdsInMap(f)
	if err != nil {
		t.Fatalf("CharacterIdsInMap: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, id := range want {
		if got[i] != id {
			t.Errorf("index %d: got %d, want %d", i, got[i], id)
		}
	}
}
