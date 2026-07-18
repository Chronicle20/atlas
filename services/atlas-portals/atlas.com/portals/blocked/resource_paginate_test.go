package blocked

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupBlockedRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
}

type blockedTestServerInformation struct{}

func (t *blockedTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *blockedTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &blockedTestServerInformation{}

func setupBlockedRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&blockedTestServerInformation{})(r, l)
	return r
}

func blockedRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetBlockedPortalsPaginates drives GET /portals/blocked?characterId=
// through the real resource router against the Redis SET-backed blocked
// registry (unordered membership), verifying the JSON:API paginated
// envelope AND that page 1's items come back in ascending (mapId, portalId)
// order (the stable-sort fix) rather than raw SET member order.
func TestGetBlockedPortalsPaginates(t *testing.T) {
	setupBlockedRegistry(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	characterId := uint32(1000)

	// seeded out of ascending order
	GetRegistry().Block(ctx, characterId, 300, 1)
	GetRegistry().Block(ctx, characterId, 100, 1)
	GetRegistry().Block(ctx, characterId, 200, 1)

	srv := httptest.NewServer(setupBlockedRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/portals/blocked?characterId=%d&page[number]=1&page[size]=2", srv.URL, characterId)
		req := blockedRequestWithTenant(http.MethodGet, url, tenantId)

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

		// seeded out of ascending mapId order (300, 100, 200): page 1 must
		// return 100:1 then 200:1, not registry read order.
		assert.Equal(t, "100:1", doc.Data.DataArray[0].ID)
		assert.Equal(t, "200:1", doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/portals/blocked?characterId=%d&page[size]=0", srv.URL, characterId)
		req := blockedRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/portals/blocked?characterId=%d&limit=5", srv.URL, characterId)
		req := blockedRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/portals/blocked?characterId=%d&page[number]=99&page[size]=2", srv.URL, characterId)
		req := blockedRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 0)

		require.NotNil(t, doc.Links)
		assert.NotContains(t, doc.Links, "next")
		require.Contains(t, doc.Links, "prev")
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=2")
	})

	t.Run("MissingCharacterIdIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/portals/blocked", srv.URL)
		req := blockedRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
