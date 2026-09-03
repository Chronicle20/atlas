package drops_test

import (
	"atlas-saga-orchestrator/drops"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestClearDropsIssuesDeleteToTheRightField proves Processor.ClearDrops
// issues a DELETE against the field-scoped drops resource path (world,
// channel, map, instance) atlas-drops exposes.
func TestClearDropsIssuesDeleteToTheRightField(t *testing.T) {
	var capturedPath string
	var capturedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("DROPS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(922000000)).Build()

	p := drops.NewProcessor(l, ctx)
	if err := p.ClearDrops(f); err != nil {
		t.Fatalf("ClearDrops returned error: %v", err)
	}

	if capturedMethod != http.MethodDelete {
		t.Fatalf("expected method DELETE, got %q", capturedMethod)
	}
	wantPath := fmt.Sprintf("/worlds/%d/channels/%d/maps/%d/instances/%s/drops",
		f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
	if capturedPath != wantPath {
		t.Fatalf("expected path %q, got %q", wantPath, capturedPath)
	}
}

// TestClearDropsPropagatesUpstreamFailure proves a non-2xx/204 response from
// atlas-drops surfaces as an error rather than being swallowed.
func TestClearDropsPropagatesUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("DROPS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(922000000)).Build()

	p := drops.NewProcessor(l, ctx)
	if err := p.ClearDrops(f); err == nil {
		t.Fatal("expected error for upstream 500, got nil")
	}
}
