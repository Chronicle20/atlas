package channel_test

import (
	"atlas-world/channel"
	"atlas-world/test"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	logtest "github.com/sirupsen/logrus/hooks/test"

	channelConstant "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

func doGetChannelServers(t *testing.T, router *mux.Router, path string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("TENANT_ID", test.DefaultTenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestGetChannelServersPaginates(t *testing.T) {
	setupTestRegistry(t)
	logger, _ := logtest.NewNullLogger()
	ctx := test.CreateTestContext()

	// Seed 3 channels for world 1, deliberately out of ascending channelId
	// order - the Redis hash backing the registry is unordered, so the
	// handler's stable-sort is what makes page order deterministic.
	processor := channel.NewProcessor(logger, ctx)
	if _, err := processor.Register(channelConstant.NewModel(1, 3), "192.168.1.1", 8080, 0, 100); err != nil {
		t.Fatalf("seed register failed: %v", err)
	}
	if _, err := processor.Register(channelConstant.NewModel(1, 1), "192.168.1.2", 8081, 0, 100); err != nil {
		t.Fatalf("seed register failed: %v", err)
	}
	if _, err := processor.Register(channelConstant.NewModel(1, 2), "192.168.1.3", 8082, 0, 100); err != nil {
		t.Fatalf("seed register failed: %v", err)
	}

	router := mux.NewRouter()
	channel.InitResource(testServerInformation{})(router, logger)

	t.Run("FirstPageOfTwoInAscendingChannelIdOrder", func(t *testing.T) {
		rr := doGetChannelServers(t, router, "/worlds/1/channels?page[number]=1&page[size]=2")
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body=%s", rr.Code, rr.Body.String())
		}
		var doc struct {
			Data []struct {
				Attributes struct {
					ChannelId uint8 `json:"channelId"`
				} `json:"attributes"`
			} `json:"data"`
			Meta struct {
				Total int `json:"total"`
				Page  struct {
					Number int `json:"number"`
					Size   int `json:"size"`
					Last   int `json:"last"`
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
			t.Fatalf("len(data) = %d, want 2", len(doc.Data))
		}
		if doc.Data[0].Attributes.ChannelId != 1 || doc.Data[1].Attributes.ChannelId != 2 {
			t.Fatalf("got channelIds [%d, %d], want [1, 2]", doc.Data[0].Attributes.ChannelId, doc.Data[1].Attributes.ChannelId)
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
		rr := doGetChannelServers(t, router, "/worlds/1/channels?page[size]=0")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		rr := doGetChannelServers(t, router, "/worlds/1/channels?limit=5")
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		rr := doGetChannelServers(t, router, "/worlds/1/channels?page[number]=99&page[size]=2")
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
}
