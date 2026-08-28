package npc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

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

// TestRestModel_Decode_ImitateTrue asserts that imitate: true decodes and
// populates Imitate().
func TestRestModel_Decode_ImitateTrue(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "npcs",
			"id": "9010000",
			"attributes": {
				"name": "Maple Administrator",
				"imitate": true
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

	if m.Id() != 9010000 {
		t.Errorf("Id() = %d, want 9010000", m.Id())
	}
	if m.Name() != "Maple Administrator" {
		t.Errorf("Name() = %q, want Maple Administrator", m.Name())
	}
	if !m.Imitate() {
		t.Errorf("Imitate() = false, want true")
	}
}

// TestRestModel_Decode_ImitateAbsent asserts that an absent imitate field
// decodes to false, matching the server's default.
func TestRestModel_Decode_ImitateAbsent(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "npcs",
			"id": "1052",
			"attributes": {
				"name": "Snail"
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

	if m.Imitate() {
		t.Errorf("Imitate() = true, want false")
	}
}

// TestGetById_HttpDecode exercises the real HTTP round-trip: a
// httptest.NewServer serves a captured data/npcs/{id} JSON:API fixture and
// Processor.GetById decodes it through the actual requests.GetRequest path.
func TestGetById_HttpDecode(t *testing.T) {
	wantPath := "/data/npcs/9010000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"npcs","id":"9010000","attributes":{
			"name":"Maple Administrator","imitate":true
		}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), testCtx(t)).GetById(9010000)
	if err != nil {
		t.Fatalf("GetById returned error: %v", err)
	}
	if m.Id() != 9010000 || m.Name() != "Maple Administrator" || !m.Imitate() {
		t.Errorf("m = %+v, want id=9010000 name=Maple Administrator imitate=true", m)
	}
}
