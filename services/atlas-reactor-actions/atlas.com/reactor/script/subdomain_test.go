package script

import (
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func subdomainTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return te
}

func TestReactorSubdomainBuildTouchRules(t *testing.T) {
	te := subdomainTestTenant(t)

	tests := []struct {
		name       string
		entityID   string
		attributes string
		want       struct {
			touchRules int
			touchId    string
		}
	}{
		{
			name:       "touchRules present",
			entityID:   "6109013",
			attributes: `{"reactorId":"6109013","description":"d","hitRules":[],"actRules":[],"touchRules":[{"id":"t1","conditions":[],"operations":[]}]}`,
			want: struct {
				touchRules int
				touchId    string
			}{
				touchRules: 1,
				touchId:    "t1",
			},
		},
		{
			name:       "touchRules absent",
			entityID:   "2001",
			attributes: `{"reactorId":"2001","description":"d","hitRules":[],"actRules":[]}`,
			want: struct {
				touchRules int
				touchId    string
			}{
				touchRules: 0,
			},
		},
		{
			name:       "touchRules empty array",
			entityID:   "2001",
			attributes: `{"reactorId":"2001","description":"d","hitRules":[],"actRules":[],"touchRules":[]}`,
			want: struct {
				touchRules int
				touchId    string
			}{
				touchRules: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envelope := `{"data":{"type":"reactor-action","id":"` + tt.entityID + `","attributes":` + tt.attributes + `}}`
			attrs, err := ReactorSubdomain{}.Decode([]byte(envelope))
			if err != nil {
				t.Fatalf("Decode() unexpected error = %v", err)
			}

			models, err := ReactorSubdomain{}.Build(te, tt.entityID, attrs)
			if err != nil {
				t.Fatalf("Build() unexpected error = %v", err)
			}
			if len(models) != 1 {
				t.Fatalf("Build() len(models) = %v, want 1", len(models))
			}

			m := models[0]
			if len(m.TouchRules()) != tt.want.touchRules {
				t.Errorf("Build() len(TouchRules) = %v, want %v", len(m.TouchRules()), tt.want.touchRules)
			}
			if tt.want.touchId != "" && len(m.TouchRules()) > 0 && m.TouchRules()[0].Id() != tt.want.touchId {
				t.Errorf("Build() TouchRules[0].Id = %v, want %v", m.TouchRules()[0].Id(), tt.want.touchId)
			}
		})
	}
}
