package world_test

import (
	"atlas-channel/world"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	worldconstants "github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// worldDoc renders a JSON:API document for world ids [from, to]. meta
// describes the current page/total so requests.DrainProvider can decide
// whether to keep paging.
//
// The shape mirrors what atlas-world's GET /worlds actually emits for this
// caller: AllProvider deliberately omits atlas-login's ?include=channels, and
// without it atlas-world attaches no ChannelDecorator, so RestModel
// .GetReferencedIDs() returns nil and NO `relationships` block appears on the
// wire. That absence is the point of serving a faithful fixture here — a
// relationships block the target struct cannot unmarshal is the documented
// atlas-rest failure mode (libs/atlas-rest/CLAUDE.md).
func worldDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		_, _ = fmt.Fprintf(&b,
			`{"id":"%d","type":"worlds","attributes":{"name":"World%d","state":0,"message":"","eventMessage":"","recommended":false,"recommendedMessage":"","capacityStatus":0,"expRate":1,"mesoRate":1,"itemDropRate":1,"questExpRate":1}}`,
			id, id,
		)
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

func worldTestContext(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), ten)
}

// TestGetAllDrainsBeyondOnePage proves GetAll (via requests.DrainProvider)
// fetches every page rather than stopping after the first response, and that
// the JSON:API payload really decodes into a populated world.Model.
//
// This is not academic for this caller. The cash-shop world-transfer name list
// is indexed BY WORLD ID on the wire (see transferWorldNameList in
// socket/handler/cash_shop_check_transfer_world_possible.go), so a truncated
// drain does not merely shorten the list -- every world past the cut vanishes
// from the combo while the surviving ones keep their indices, and the player
// silently cannot reach them.
//
// The fixture spans ids 0..255 across two pages, which is the full range the
// domain can express: world.Id is a byte, so 256 worlds is the ceiling and an
// id above it would alias on Extract's int->byte conversion rather than
// round-trip. Pages are 200 + 56 rather than 250 + 6 because the drain
// terminates on the meta `last` value, not on a short page.
func TestGetAllDrainsBeyondOnePage(t *testing.T) {
	var sawPages []int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		if number == 0 {
			number = 1
		}
		sawPages = append(sawPages, number)
		if got := r.URL.Query().Get("page[size]"); got != "250" {
			t.Errorf("page[size] = %q, want 250 (atlas-world's paginate.MaxPageSize -- a larger value is silently clamped)", got)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(worldDoc(200, 255, 256, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(worldDoc(0, 199, 256, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("WORLDS_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	ms, err := world.NewProcessor(l, worldTestContext(t)).GetAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(ms) != 256 {
		t.Fatalf("got %d worlds, want 256 (full drain); a single-page implementation returns 200", len(ms))
	}
	if len(sawPages) < 2 {
		t.Fatalf("server saw pages %v, want at least two requests", sawPages)
	}

	// Prove the decode populated the fields the caller actually reads --
	// Id() and Name() -- rather than yielding 256 zero-valued models.
	names := make(map[worldconstants.Id]string, len(ms))
	for _, m := range ms {
		names[m.Id()] = m.Name()
	}
	if got, ok := names[255]; !ok {
		t.Fatal("world id 255 (page 2) must be present; a single-page impl would miss it")
	} else if got != "World255" {
		t.Fatalf("world 255 name = %q, want %q -- the JSON:API attributes did not decode", got, "World255")
	}
	if len(names) != 256 {
		t.Fatalf("got %d distinct world ids, want 256 -- ids must round-trip without aliasing", len(names))
	}
}

// A single short page must terminate the drain rather than paging forever.
func TestGetAllStopsOnASinglePage(t *testing.T) {
	requests := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(worldDoc(0, 1, 2, 1, 250, 1)))
	}))
	defer srv.Close()
	t.Setenv("WORLDS_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	ms, err := world.NewProcessor(l, worldTestContext(t)).GetAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(ms) != 2 {
		t.Fatalf("got %d worlds, want 2", len(ms))
	}
	if requests != 1 {
		t.Fatalf("server saw %d requests, want exactly 1 -- last=1 must terminate the drain", requests)
	}
	if ms[0].Id() != 0 || ms[0].Name() != "World0" {
		t.Fatalf("world[0] = {id %d, name %q}, want {0, World0}", ms[0].Id(), ms[0].Name())
	}
}

// An empty world set must surface as an empty slice and no error -- the
// world-transfer handler distinguishes that case itself (refusing with
// UNKNOWN_ERROR rather than sending the client a crash-inducing empty list),
// so the client must not turn it into a transport failure.
func TestGetAllEmptyWorldSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[],"meta":{"total":0,"page":{"number":1,"size":250,"last":1}}}`))
	}))
	defer srv.Close()
	t.Setenv("WORLDS_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	ms, err := world.NewProcessor(l, worldTestContext(t)).GetAll()
	if err != nil {
		t.Fatalf("an empty world set must not be an error: %v", err)
	}
	if len(ms) != 0 {
		t.Fatalf("got %d worlds, want 0", len(ms))
	}
}
