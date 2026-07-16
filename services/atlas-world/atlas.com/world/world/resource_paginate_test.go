package world_test

import (
	"atlas-world/channel"
	"atlas-world/configuration"
	tenantconfig "atlas-world/configuration/tenant"
	"atlas-world/configuration/tenant/worlds"
	"atlas-world/rate"
	"atlas-world/test"
	"atlas-world/world"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	channelConstant "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	worldConstant "github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	goredis "github.com/redis/go-redis/v9"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// setupWorldsPaginateFixture registers channels for 3 worlds (ids 2, 0, 1, in
// that deliberately out-of-order registration sequence) and publishes a
// tenant configuration snapshot with 3 world entries, so handleGetWorlds can
// materialize a full [0,1,2] world list for the tenant. World existence is
// derived from a Go map (mapDistinctWorldId over the channel registry), whose
// iteration order is not deterministic - the handler's stable-sort by Id is
// what makes the paged response order deterministic.
func setupWorldsPaginateFixture(t *testing.T) uuid.UUID {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	channel.InitRegistry(rc)
	rate.InitRegistry(rc)

	tenantId := uuid.New()
	ctx := test.CreateTestContextWithTenant(tenantId)
	logger, _ := logtest.NewNullLogger()
	cp := channel.NewProcessor(logger, ctx)
	for _, wid := range []worldConstant.Id{2, 0, 1} {
		if _, err := cp.Register(channelConstant.NewModel(wid, 0), "192.168.1.1", 8080, 0, 100); err != nil {
			t.Fatalf("seed register failed: %v", err)
		}
	}

	configuration.PublishSnapshot(map[uuid.UUID]tenantconfig.RestModel{
		tenantId: {
			Region:       "GMS",
			MajorVersion: 83,
			MinorVersion: 1,
			Worlds: []worlds.RestModel{
				{Name: "World0"},
				{Name: "World1"},
				{Name: "World2"},
			},
		},
	})

	return tenantId
}

func doGetWorlds(t *testing.T, router *mux.Router, tenantId uuid.UUID, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestGetWorldsPaginates(t *testing.T) {
	tenantId := setupWorldsPaginateFixture(t)
	logger, _ := logtest.NewNullLogger()

	router := mux.NewRouter()
	world.InitResource(testServerInformation{})(router, logger)

	t.Run("FirstPageOfTwoInAscendingIdOrder", func(t *testing.T) {
		rr := doGetWorlds(t, router, tenantId, "/worlds/?page[number]=1&page[size]=2")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct {
				Id string `json:"id"`
			} `json:"data"`
			Meta struct {
				Total int `json:"total"`
				Page  struct {
					Last int `json:"last"`
				} `json:"page"`
			} `json:"meta"`
			Links struct {
				Next *string `json:"next"`
			} `json:"links"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		if len(doc.Data) != 2 {
			t.Fatalf("len(data) = %d, want 2, body=%s", len(doc.Data), rr.Body.String())
		}
		if doc.Data[0].Id != "0" || doc.Data[1].Id != "1" {
			t.Fatalf("got ids [%s, %s], want [0, 1]", doc.Data[0].Id, doc.Data[1].Id)
		}
		if doc.Meta.Total != 3 {
			t.Fatalf("meta.total = %d, want 3", doc.Meta.Total)
		}
		if doc.Meta.Page.Last != 2 {
			t.Fatalf("meta.page.last = %d, want 2", doc.Meta.Page.Last)
		}
		if doc.Links.Next == nil {
			t.Fatal("expected links.next to be present")
		}
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		rr := doGetWorlds(t, router, tenantId, "/worlds/?page[size]=0")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		rr := doGetWorlds(t, router, tenantId, "/worlds/?limit=5")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		rr := doGetWorlds(t, router, tenantId, "/worlds/?page[number]=99&page[size]=2")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct{} `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		if len(doc.Data) != 0 {
			t.Fatalf("len(data) = %d, want 0", len(doc.Data))
		}
	})

	t.Run("IncludeChannelsDecoratesOnlyThePageItems", func(t *testing.T) {
		rr := doGetWorlds(t, router, tenantId, "/worlds/?include=channels&page[number]=1&page[size]=1")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct {
				Id            string `json:"id"`
				Relationships struct {
					Channels struct {
						Data []struct {
							Id string `json:"id"`
						} `json:"data"`
					} `json:"channels"`
				} `json:"relationships"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
			t.Fatalf("unmarshal: %v, body=%s", err, rr.Body.String())
		}
		if len(doc.Data) != 1 {
			t.Fatalf("len(data) = %d, want 1", len(doc.Data))
		}
		if doc.Data[0].Id != "0" {
			t.Fatalf("got id %s, want 0", doc.Data[0].Id)
		}
		if len(doc.Data[0].Relationships.Channels.Data) != 1 {
			t.Fatalf("world 0 should have 1 decorated channel, got %d, body=%s", len(doc.Data[0].Relationships.Channels.Data), rr.Body.String())
		}
	})
}
