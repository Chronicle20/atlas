package character

import (
	"atlas-invites/invite"
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

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupInviteRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	invite.InitRegistry(client)
}

func setupInviteRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&testServerInformation{})(r, l)
	return r
}

func requestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetCharacterInvitesPaginates drives GET /characters/{characterId}/invites
// through the real resource router (InitResource) against a miniredis-backed
// registry, verifying the JSON:API paginated envelope over registry-sourced
// (not DB-sourced) data.
func TestGetCharacterInvitesPaginates(t *testing.T) {
	setupInviteRegistry(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	// 3 invites targeting character 700, created out of any Id-sort order
	// concern by construction (registry index order is not guaranteed) —
	// the handler's stable sort by Id is what must determine page contents.
	for i := 0; i < 3; i++ {
		invite.GetRegistry().Create(ctx, uint32(800+i), 0, 700, "PARTY", uint32(9000+i))
	}

	srv := httptest.NewServer(setupInviteRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/700/invites?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 3, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 2, page["last"])

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")

		// Determinism: page 1 must be the two lowest invite Ids, in
		// ascending order, regardless of registry lookup order.
		var first, second struct {
			OriginatorId uint32 `json:"originatorId"`
		}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &first))
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &second))
		assert.EqualValues(t, 800, first.OriginatorId)
		assert.EqualValues(t, 801, second.OriginatorId)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/700/invites?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/700/invites?limit=5", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/characters/700/invites?page[number]=99&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		assert.Len(t, doc.Data.DataArray, 0)

		require.NotNil(t, doc.Links)
		require.Contains(t, doc.Links, "prev")
		assert.Contains(t, doc.Links["prev"].Href, "page%5Bnumber%5D=2")
		assert.NotContains(t, doc.Links, "next")
	})
}
