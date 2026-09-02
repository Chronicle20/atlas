package object_test

import (
	"atlas-maps/data/map/object"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// objectDoc renders a JSON:API document for named map objects [from, to] on a
// single map. meta describes the current page/total so requests.DrainProvider
// can decide whether to keep paging.
func objectDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b,
			`{"id":"object-%d","type":"objects","attributes":{"name":"object-%d","state":%d}}`,
			id, id, id,
		)
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), ten)
}

// TestGetDefaultStateReturnsDeclaredState proves GetDefaultState surfaces the
// declared default state atlas-data reports for a named object.
func TestGetDefaultStateReturnsDeclaredState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gate","type":"objects","attributes":{"name":"gate","state":1}}],"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	state, err := object.NewProcessor(l, ctx).GetDefaultState(1, "gate")
	if err != nil {
		t.Fatal(err)
	}
	if state != 1 {
		t.Fatalf("state != 1, got %d", state)
	}
}

// TestGetDefaultStateReturnsErrUnknownObjectForUndeclaredName proves
// GetDefaultState returns ErrUnknownObject when the map declares no object
// with the requested name.
func TestGetDefaultStateReturnsErrUnknownObjectForUndeclaredName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gate","type":"objects","attributes":{"name":"gate","state":1}}],"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	_, err := object.NewProcessor(l, ctx).GetDefaultState(1, "barricade")
	if !errors.Is(err, object.ErrUnknownObject) {
		t.Fatalf("expected ErrUnknownObject, got %v", err)
	}
}

// TestGetDefaultStateDrainsBeyondOnePage proves the named-object provider
// (via requests.DrainProvider) fetches every page of a map's declared
// objects rather than stopping after the first response. The fixture serves
// 260 objects across two pages of 250 -- only a genuine drain picks up
// object-260, which lives on page 2.
func TestGetDefaultStateDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(objectDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(objectDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ctx := newTestContext(t)
	l, _ := test.NewNullLogger()

	state, err := object.NewProcessor(l, ctx).GetDefaultState(1, "object-260")
	if err != nil {
		t.Fatalf("expected object-260 (page 2) to be found via drain, got error: %v", err)
	}
	if state != 260 {
		t.Fatalf("state != 260, got %d", state)
	}
}
