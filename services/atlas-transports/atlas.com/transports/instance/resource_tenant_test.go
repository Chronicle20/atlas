package instance

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestGetInstanceRouteStatusIsTenantScoped is the cross-tenant guard. An
// instance created under tenant A must never surface for tenant B: the
// handler reads the per-route Redis set, which is tenant-keyed, and
// FindOrCreateInstance writes under the creating tenant's id.
func TestGetInstanceRouteStatusIsTenantScoped(t *testing.T) {
	setupInstanceTestRegistry(t)

	route, err := NewRouteBuilder("tenant-scoped-route").
		SetStartMapId(_map.Id(100000000)).
		SetTransitMapIds([]_map.Id{100000100}).
		SetDestinationMapId(_map.Id(100000200)).
		SetCapacity(3).
		SetBoardingWindow(10 * time.Second).
		SetTravelDuration(30 * time.Second).
		Build()
	if err != nil {
		t.Fatalf("seed build failed: %v", err)
	}

	tenantA := uuid.New()
	tenantB := uuid.New()

	tmA, err := tenant.Create(tenantA, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("seed tenant create failed: %v", err)
	}
	ctxA := tenant.WithContext(context.Background(), tmA)

	reg := getInstanceRegistry()
	inst := reg.FindOrCreateInstance(ctxA, route, time.Now())
	reg.AddCharacter(ctxA, inst.InstanceId(), CharacterEntry{CharacterId: 1})

	logger, _ := logtest.NewNullLogger()
	router := mux.NewRouter()
	InitResource(testServerInformation{})(router, logger)

	path := "/transports/instance-routes/" + route.Id().String() + "/status"

	readIds := func(tenantId uuid.UUID) ([]string, int) {
		rr := doGetInstance(t, router, tenantId, path)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct {
				Id string `json:"id"`
			} `json:"data"`
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		ids := make([]string, 0, len(doc.Data))
		for _, d := range doc.Data {
			ids = append(ids, d.Id)
		}
		return ids, doc.Meta.Total
	}

	t.Run("CreatingTenantSeesTheInstance", func(t *testing.T) {
		ids, total := readIds(tenantA)
		if len(ids) != 1 || ids[0] != inst.InstanceId().String() {
			t.Fatalf("tenant A ids = %v, want [%s]", ids, inst.InstanceId())
		}
		if total != 1 {
			t.Fatalf("tenant A meta.total = %d, want 1", total)
		}
	})

	t.Run("OtherTenantSeesNothing", func(t *testing.T) {
		ids, total := readIds(tenantB)
		if len(ids) != 0 {
			t.Fatalf("tenant B ids = %v, want []", ids)
		}
		if total != 0 {
			t.Fatalf("tenant B meta.total = %d, want 0", total)
		}
	})
}
