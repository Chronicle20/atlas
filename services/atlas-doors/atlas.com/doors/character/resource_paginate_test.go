package character

import (
	"atlas-doors/door"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupDoorsCharRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	door.InitIdAllocator(rc)
	door.InitRegistry(rc)
}

type doorsCharTestServerInformation struct{}

func (t *doorsCharTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *doorsCharTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &doorsCharTestServerInformation{}

func setupDoorsCharRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&doorsCharTestServerInformation{})(r, l)
	return r
}

func doorsCharRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetDoorsByOwnerPaginates drives GET /characters/{characterId}/doors
// through the real resource router against the Redis-backed door registry,
// verifying the JSON:API paginated envelope AND that page 1's items come
// back in ascending pair-id (areaDoorId) order (the stable-sort fix) rather
// than the registry's unordered Redis SET membership order.
func TestGetDoorsByOwnerPaginates(t *testing.T) {
	setupDoorsCharRegistry(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	ownerId := character.Id(42)
	f := field.NewBuilder(1, 1, 100000000).Build()

	now := time.Now()
	for _, id := range []uint32{300, 100, 200} {
		m := door.NewBuilder().
			SetAreaDoorId(id).
			SetTownDoorId(id + 1).
			SetOwnerCharacterId(ownerId).
			SetField(f).
			SetTownMapId(104000000).
			SetSlot(0).
			SetDeployTime(now).
			SetExpiresAt(now.Add(60 * time.Second)).
			Build()
		require.NoError(t, door.GetRegistry().Put(ctx, ten, m))
	}

	srv := httptest.NewServer(setupDoorsCharRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/%d/doors?page[number]=1&page[size]=2", srv.URL, ownerId)
		req := doorsCharRequestWithTenant(http.MethodGet, url, tenantId)

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

		assert.Equal(t, "100", doc.Data.DataArray[0].ID)
		assert.Equal(t, "200", doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/%d/doors?page[size]=0", srv.URL, ownerId)
		req := doorsCharRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/%d/doors?limit=5", srv.URL, ownerId)
		req := doorsCharRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
