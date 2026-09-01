package map_

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Register(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

// TestRestModel_Decode_MapArea asserts the JSON:API decode of a captured
// data/maps/{id} payload's mapArea bounds.
func TestRestModel_Decode_MapArea(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "maps",
			"id": "100000000",
			"attributes": {
				"mapArea": {"x": -500, "y": -300, "width": 1000, "height": 600}
			}
		}
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(payload, &rm); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if m.Id() != 100000000 {
		t.Errorf("Id() = %d, want 100000000", m.Id())
	}
	area := m.MapArea()
	if area == nil {
		t.Fatalf("MapArea() = nil, want non-nil")
	}
	if area.X() != -500 || area.Y() != -300 || area.Width() != 1000 || area.Height() != 600 {
		t.Errorf("MapArea() = %+v, want {-500 -300 1000 600}", area)
	}
}

// TestGroundResultRestModel_Decode asserts the ground response decodes into
// GroundResult values.
func TestGroundResultRestModel_Decode(t *testing.T) {
	payload := []byte(`{
		"data": [
			{"type": "grounds", "id": "0", "attributes": {"x": 0, "y": -50, "fh": 1, "found": true}},
			{"type": "grounds", "id": "1", "attributes": {"x": 30000, "y": 0, "fh": 0, "found": false}}
		]
	}`)

	var rms []GroundResultRestModel
	if err := jsonapi.Unmarshal(payload, &rms); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	if len(rms) != 2 {
		t.Fatalf("len(rms) = %d, want 2", len(rms))
	}

	first, err := ExtractGroundResult(rms[0])
	if err != nil {
		t.Fatalf("ExtractGroundResult returned error: %v", err)
	}
	if !first.Found() || first.Y() != -50 || first.Fh() != 1 {
		t.Errorf("first = %+v, want found=true y=-50 fh=1", first)
	}

	second, err := ExtractGroundResult(rms[1])
	if err != nil {
		t.Fatalf("ExtractGroundResult returned error: %v", err)
	}
	if second.Found() {
		t.Errorf("second.Found() = true, want false")
	}
}

// TestGetById_HttpDecode exercises the real HTTP round-trip: a
// httptest.NewServer serves a captured data/maps/{id} JSON:API fixture and
// Processor.GetById decodes it through the actual requests.GetRequest path.
func TestGetById_HttpDecode(t *testing.T) {
	wantPath := "/data/maps/100000000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"maps","id":"100000000","attributes":{
			"mapArea":{"x":-500,"y":-300,"width":1000,"height":600}
		}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), testCtx(t)).GetById(_map.Id(100000000))
	if err != nil {
		t.Fatalf("GetById returned error: %v", err)
	}
	area := m.MapArea()
	if area == nil || area.X() != -500 || area.Width() != 1000 {
		t.Errorf("MapArea() = %+v, want x=-500 width=1000", area)
	}
}

// TestGround_HttpDecode exercises the real HTTP round-trip of the POST
// data/maps/{id}/ground call, decoding a captured response into
// GroundResult values.
func TestGround_HttpDecode(t *testing.T) {
	wantPath := "/data/maps/100000000/ground"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"type":"grounds","id":"0","attributes":{"x":0,"y":-50,"fh":1,"found":true}}
		]}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	results, err := NewProcessor(logrus.New(), testCtx(t)).Ground(_map.Id(100000000), []GroundPoint{NewGroundPoint(0, 0)})
	if err != nil {
		t.Fatalf("Ground returned error: %v", err)
	}
	if len(results) != 1 || !results[0].Found() || results[0].Y() != -50 {
		t.Errorf("results = %+v, want one found=true y=-50 entry", results)
	}
}
