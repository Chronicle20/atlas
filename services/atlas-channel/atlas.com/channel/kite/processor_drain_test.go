package kite

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), ten)
}

// kitesDoc renders a JSON:API "kites" document for kite ids [from, to]. meta
// describes the current page/total so requests.DrainProvider can decide
// whether to keep paging.
func kitesDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"id":"%d","type":"kites","attributes":{"characterId":%d,"name":"n","templateId":5390000,"message":"hi","x":0,"y":0}}`,
			id, id)
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestInMapModelProviderDrainsBeyondOnePage proves InMapModelProvider (via
// requests.DrainProvider) fetches every page of kites-in-a-map-instance
// rather than stopping after the first response. The fixture server serves
// 3 pages of 250 (750 total), so only a genuine drain picks up kite id 750,
// which lives on page 3.
func TestInMapModelProviderDrainsBeyondOnePage(t *testing.T) {
	var sawPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPaths = append(sawPaths, r.URL.Path)
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch number {
		case 2:
			_, _ = w.Write([]byte(kitesDoc(251, 500, 750, 2, 250, 3)))
		case 3:
			_, _ = w.Write([]byte(kitesDoc(501, 750, 750, 3, 250, 3)))
		default:
			_, _ = w.Write([]byte(kitesDoc(1, 250, 750, 1, 250, 3)))
		}
	}))
	defer srv.Close()
	t.Setenv("KITES_SERVICE_URL", srv.URL+"/")

	inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	f := field.NewBuilder(0, 1, 104040000).SetInstance(inst).Build()

	ms, err := NewProcessor(testLogger(t), testContext(t)).InMapModelProvider(f)()
	if err != nil {
		t.Fatalf("InMapModelProvider: %v", err)
	}
	if len(ms) != 750 {
		t.Fatalf("expected 750 kites (full 3-page drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.Id() == 750 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("kite id 750 (page 3) must be present; single-fetch impl would miss it")
	}

	if len(sawPaths) < 3 {
		t.Fatalf("expected at least 3 page requests, got %d: %v", len(sawPaths), sawPaths)
	}
	wantPath := fmt.Sprintf("/worlds/0/channels/1/maps/104040000/instances/%s/kites", inst.String())
	for _, p := range sawPaths {
		if p != wantPath {
			t.Errorf("requested %q, want %q", p, wantPath)
		}
	}
}

func TestInMapModelProviderRequestsInstanceScopedPath(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	t.Setenv("KITES_SERVICE_URL", srv.URL+"/")

	inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	f := field.NewBuilder(0, 1, 104040000).SetInstance(inst).Build()

	if _, err := NewProcessor(testLogger(t), testContext(t)).InMapModelProvider(f)(); err != nil {
		t.Fatalf("InMapModelProvider: %v", err)
	}
	want := fmt.Sprintf("/worlds/0/channels/1/maps/104040000/instances/%s/kites", inst.String())
	if seen != want {
		t.Errorf("requested %q, want %q", seen, want)
	}
}
