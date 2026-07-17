package reactor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-reactors/reactor/data"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type reactorTestServerInformation struct{}

func (t *reactorTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *reactorTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &reactorTestServerInformation{}

func setupReactorRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&reactorTestServerInformation{})(r, l)
	return r
}

func reactorRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// TestGetInMapPaginates drives GET
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/reactors through the real
// resource router against the Redis-backed reactor registry, verifying the
// JSON:API paginated envelope AND that page 1's items come back in
// ascending reactor-id order (the stable-sort-by-id fix) rather than the
// registry's unordered Redis SET membership order.
func TestGetInMapPaginates(t *testing.T) {
	setupTestRegistry(t)

	ten := setupTestTenant()

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := mapconst.Id(100000000)
	instanceId := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

	var ids []uint32
	for i := 0; i < 3; i++ {
		builder := NewModelBuilder(ten, f, 100, "reactor1").
			SetState(0).
			SetPosition(100, 200).
			SetDelay(0).
			SetDirection(0).
			SetData(data.Model{})
		m, err := GetRegistry().Create(ten, builder)
		require.NoError(t, err)
		ids = append(ids, m.Id())
	}

	srv := httptest.NewServer(setupReactorRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/reactors?page[number]=1&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId)
		req := reactorRequestWithTenant(http.MethodGet, url, ten.Id())

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")

		assert.Equal(t, fmt.Sprintf("%d", ids[0]), doc.Data.DataArray[0].ID)
		assert.Equal(t, fmt.Sprintf("%d", ids[1]), doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/reactors?page[size]=0", srv.URL, worldId, channelId, mapId, instanceId)
		req := reactorRequestWithTenant(http.MethodGet, url, ten.Id())

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/reactors?limit=5", srv.URL, worldId, channelId, mapId, instanceId)
		req := reactorRequestWithTenant(http.MethodGet, url, ten.Id())

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("NameFilterAppliedBeforePagination", func(t *testing.T) {
		builder := NewModelBuilder(ten, f, 200, "special-reactor").
			SetState(0).
			SetPosition(300, 300).
			SetDelay(0).
			SetDirection(0).
			SetData(data.Model{})
		_, err := GetRegistry().Create(ten, builder)
		require.NoError(t, err)

		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/reactors?name=special-reactor", srv.URL, worldId, channelId, mapId, instanceId)
		req := reactorRequestWithTenant(http.MethodGet, url, ten.Id())

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 1, doc.Meta["total"], "meta.total must reflect the filtered count, not the unfiltered one")
		require.Len(t, doc.Data.DataArray, 1)
	})
}
