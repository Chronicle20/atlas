package information

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func TestExtract_CarriesLevelAndName(t *testing.T) {
	m, err := Extract(RestModel{Id: 100100, Name: "Blue Snail", Hp: 8, Experience: 3, Level: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Hp() != 8 {
		t.Errorf("expected Hp() == 8, got %d", m.Hp())
	}
	if m.Experience() != 3 {
		t.Errorf("expected Experience() == 3, got %d", m.Experience())
	}
	if m.Level() != 2 {
		t.Errorf("expected Level() == 2, got %d", m.Level())
	}
	if m.Name() != "Blue Snail" {
		t.Errorf("expected Name() == \"Blue Snail\", got %q", m.Name())
	}
}

func TestBuilder_SetsLevelAndName(t *testing.T) {
	m, err := NewBuilder().SetHp(1000).SetExperience(500).SetLevel(125).SetName("Zakum").Build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Hp() != 1000 {
		t.Errorf("expected Hp() == 1000, got %d", m.Hp())
	}
	if m.Experience() != 500 {
		t.Errorf("expected Experience() == 500, got %d", m.Experience())
	}
	if m.Level() != 125 {
		t.Errorf("expected Level() == 125, got %d", m.Level())
	}
	if m.Name() != "Zakum" {
		t.Errorf("expected Name() == \"Zakum\", got %q", m.Name())
	}
}

// TestRequestById_RoundTrip stands up an httptest server returning a
// realistic JSON:API document for a monster-information resource,
// INCLUDING a relationships block, and drives it through the real
// requests.GetRequest decode path. This proves the SetToOneReferenceID/
// SetToManyReferenceIDs stubs added per libs/atlas-rest/CLAUDE.md let
// api2go decode a response that carries relationships, and pins the
// wire-tag mapping for hp/experience/level/name.
func TestRequestById_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/data/monsters/100100") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "monsters",
				"id": "100100",
				"attributes": {
					"name": "Blue Snail",
					"hp": 8,
					"experience": 3,
					"level": 2
				},
				"relationships": {
					"drops": {
						"data": [
							{"type": "drops", "id": "1"}
						]
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	rm, err := requestById(ctx, 100100)(logrus.New(), ctx)
	if err != nil {
		t.Fatalf("requestById: %v", err)
	}
	if rm.Id != 100100 {
		t.Errorf("Id = %d, want 100100", rm.Id)
	}
	if rm.Name != "Blue Snail" {
		t.Errorf("Name = %q, want Blue Snail", rm.Name)
	}
	if rm.Hp != 8 {
		t.Errorf("Hp = %d, want 8", rm.Hp)
	}
	if rm.Experience != 3 {
		t.Errorf("Experience = %d, want 3", rm.Experience)
	}
	if rm.Level != 2 {
		t.Errorf("Level = %d, want 2", rm.Level)
	}
}

func TestTransformRoundTrip(t *testing.T) {
	m, err := NewBuilder().SetHp(11).SetExperience(22).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("transform: %v", err)
	}
	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", got, m)
	}
}
