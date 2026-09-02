package character

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

// TestRestModel_Decode asserts the JSON:API decode of a captured
// characters/{id} payload: name, gender, skinColor, face, hair, jobId,
// level and gm all populate.
func TestRestModel_Decode(t *testing.T) {
	payload := []byte(`{
		"data": {
			"type": "characters",
			"id": "1001",
			"attributes": {
				"name": "Statue",
				"gender": 1,
				"skinColor": 3,
				"face": 20000,
				"hair": 30030,
				"jobId": 112,
				"level": 120,
				"gm": 1
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

	if m.Id() != 1001 {
		t.Errorf("Id() = %d, want 1001", m.Id())
	}
	if m.Name() != "Statue" {
		t.Errorf("Name() = %q, want Statue", m.Name())
	}
	if m.Gender() != 1 {
		t.Errorf("Gender() = %d, want 1", m.Gender())
	}
	if m.SkinColor() != 3 {
		t.Errorf("SkinColor() = %d, want 3", m.SkinColor())
	}
	if m.Face() != 20000 {
		t.Errorf("Face() = %d, want 20000", m.Face())
	}
	if m.Hair() != 30030 {
		t.Errorf("Hair() = %d, want 30030", m.Hair())
	}
	if uint16(m.JobId()) != 112 {
		t.Errorf("JobId() = %d, want 112", m.JobId())
	}
	if m.Level() != 120 {
		t.Errorf("Level() = %d, want 120", m.Level())
	}
	if !m.Gm() {
		t.Errorf("Gm() = false, want true")
	}
}

// TestGetById_HttpDecode exercises the real HTTP round-trip: a
// httptest.NewServer serves a captured characters/{id} JSON:API fixture and
// Processor.GetById decodes it through the actual requests.GetRequest path,
// not a hand-built RestModel.
func TestGetById_HttpDecode(t *testing.T) {
	wantPath := "/characters/1001"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != wantPath {
			t.Errorf("unexpected path %s, want %s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"1001","attributes":{
			"name":"Statue","gender":1,"skinColor":3,"face":20000,"hair":30030,
			"jobId":112,"level":120,"gm":1
		}}}`))
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	m, err := NewProcessor(logrus.New(), testCtx(t)).GetById(1001)
	if err != nil {
		t.Fatalf("GetById returned error: %v", err)
	}
	if m.Id() != 1001 || m.Name() != "Statue" || m.Level() != 120 {
		t.Errorf("m = %+v, want id=1001 name=Statue level=120", m)
	}
}
