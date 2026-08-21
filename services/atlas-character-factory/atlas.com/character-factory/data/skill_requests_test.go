package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenantModel "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestGetSkillsByIds_DecodesEffectX stands up an httptest server returning a
// realistic JSON:API document for atlas-data's `data/skills` resource and
// drives it through the real GetSkillsByIds decode path (requestSkillsByIds
// -> jsonapi.Unmarshal -> SkillRestModel -> SkillInfo). The fixture uses the
// exact wire attribute name ("effects"/"x") atlas-data's own
// skill/effect.RestModel emits (json:"x" on RestModel.X), not a struct this
// test constructs in Go -- so a wire-shape mismatch on the new "x" field
// would fail this test instead of silently zeroing every SP investment.
//
// Every other factory-level test bypasses this decode via dmock.ProcessorMock;
// this is the only test that exercises the real HTTP decode path.
func TestGetSkillsByIds_DecodesEffectX(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/data/skills") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"type": "skills",
					"id": "1000001",
					"attributes": {
						"name": "Test Skill",
						"maxLevel": 3,
						"effects": [
							{"x": 0},
							{"x": 12},
							{"x": 24}
						]
					}
				}
			]
		}`))
	}))
	defer srv.Close()

	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	tenantId := uuid.New()
	tm, err := tenantModel.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}
	ctx := tenantModel.WithContext(context.Background(), tm)

	p := NewProcessor(logrus.New())
	skills, err := p.GetSkillsByIds(ctx, []uint32{1000001})
	if err != nil {
		t.Fatalf("GetSkillsByIds: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("len(skills) = %d, want 1", len(skills))
	}

	got := skills[0]
	if got.Id != 1000001 {
		t.Errorf("Id = %d, want 1000001", got.Id)
	}
	if got.Name != "Test Skill" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Skill")
	}
	if got.MaxLevel != 3 {
		t.Errorf("MaxLevel = %d, want 3", got.MaxLevel)
	}
	wantEffectX := []int16{0, 12, 24}
	if len(got.EffectX) != len(wantEffectX) {
		t.Fatalf("len(EffectX) = %d, want %d", len(got.EffectX), len(wantEffectX))
	}
	for i, want := range wantEffectX {
		if got.EffectX[i] != want {
			t.Errorf("EffectX[%d] = %d, want %d", i, got.EffectX[i], want)
		}
	}
}
