package messenger

import (
	"atlas-messengers/character"
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

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupMessengersRegistries(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	character.InitRegistry(rc)
}

type messengersTestServerInformation struct{}

func (t *messengersTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *messengersTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &messengersTestServerInformation{}

func setupMessengersRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&messengersTestServerInformation{})(r, l)
	return r
}

func messengersRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// seedMessenger writes a messenger directly into the registry with an
// explicit id (bypassing the sequential id generator used by Create) so
// tests can seed out-of-ascending-order data, and creates a matching
// character so Transform's per-member live-state lookup (name/online/etc)
// succeeds.
func seedMessenger(t *testing.T, ctx context.Context, ten tenant.Model, ch channel.Model, id uint32, memberId uint32) {
	t.Helper()
	character.GetRegistry().Create(ctx, ch, memberId, fmt.Sprintf("Member%d", memberId))
	m, err := NewBuilder().
		SetTenantId(ten.Id()).
		SetId(id).
		AddMember(memberId, 0).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetRegistry().messengers.Put(ctx, ten, id, m))
}

// TestGetMessengersPaginates drives GET /messengers through the real
// resource router against the Redis-backed messenger registry, verifying
// the JSON:API paginated envelope AND that page 1's items come back in
// ascending messenger-id order (the stable-sort-by-id fix) rather than raw
// registry read order (which atlas-redis's TenantRegistry.GetAllValues does
// not contractually guarantee).
func TestGetMessengersPaginates(t *testing.T) {
	setupMessengersRegistries(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	ch := channel.NewModel(world.Id(1), channel.Id(1))

	// seeded out of ascending-id order: +300, +100, +200
	seedMessenger(t, ctx, ten, ch, StartMessengerId+300, 9300)
	seedMessenger(t, ctx, ten, ch, StartMessengerId+100, 9100)
	seedMessenger(t, ctx, ten, ch, StartMessengerId+200, 9200)

	srv := httptest.NewServer(setupMessengersRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers?page[number]=1&page[size]=2", srv.URL)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

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

		// seeded out of ascending-id order (+300, +100, +200): page 1 must
		// return +100 then +200, not registry read order.
		assert.Equal(t, fmt.Sprintf("%d", StartMessengerId+100), doc.Data.DataArray[0].ID)
		assert.Equal(t, fmt.Sprintf("%d", StartMessengerId+200), doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers?page[size]=0", srv.URL)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers?limit=5", srv.URL)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers?page[number]=99&page[size]=2", srv.URL)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

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

	t.Run("FilterByMemberIdKeepsShapeAndAcceptsPageParams", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers?filter[members.id]=9100&page[number]=1&page[size]=50", srv.URL)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 1)
		assert.Equal(t, fmt.Sprintf("%d", StartMessengerId+100), doc.Data.DataArray[0].ID)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 1, doc.Meta["total"])
	})
}

// seedMessengerWithMembers writes a messenger with multiple members directly
// into the registry, and creates a matching character for each member so
// Transform's per-member live-state lookup succeeds.
func seedMessengerWithMembers(t *testing.T, ctx context.Context, ten tenant.Model, ch channel.Model, id uint32, memberIds []uint32) {
	t.Helper()
	b := NewBuilder().SetTenantId(ten.Id()).SetId(id)
	for slot, memberId := range memberIds {
		character.GetRegistry().Create(ctx, ch, memberId, fmt.Sprintf("Member%d", memberId))
		b.AddMember(memberId, byte(slot))
	}
	m, err := b.Build()
	require.NoError(t, err)
	require.NoError(t, GetRegistry().messengers.Put(ctx, ten, id, m))
}

// TestGetMessengerMembersPaginates drives GET /messengers/{messengerId}/members
// (and the aliased .../relationships/members route) through the real
// resource router, verifying both now emit the JSON:API paginated envelope
// (task-117 fix for the STRICT MarshalResponse[[]] pagination gate) and that
// members are returned in deterministic ascending-id order even when seeded
// out of order.
func TestGetMessengerMembersPaginates(t *testing.T) {
	setupMessengersRegistries(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	ch := channel.NewModel(world.Id(1), channel.Id(1))

	messengerId := StartMessengerId + 400
	// seeded out of ascending-id order: +30, +10, +20 (MaxMembers is 3)
	seedMessengerWithMembers(t, ctx, ten, ch, messengerId, []uint32{9430, 9410, 9420})

	srv := httptest.NewServer(setupMessengersRouter())
	defer srv.Close()

	t.Run("MembersRouteReturnsPaginatedEnvelopeInAscendingIdOrder", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers/%d/members?page[number]=1&page[size]=2", srv.URL, messengerId)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

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

		require.NotNil(t, doc.Links)
		assert.Contains(t, doc.Links, "next")

		// seeded out of ascending-id order (+30, +10, +20): page 1 must
		// return +10 then +20, not registry/slice read order.
		assert.Equal(t, fmt.Sprintf("%d", uint32(9410)), doc.Data.DataArray[0].ID)
		assert.Equal(t, fmt.Sprintf("%d", uint32(9420)), doc.Data.DataArray[1].ID)
	})

	t.Run("RelationshipsMembersRouteReturnsSamePaginatedEnvelope", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers/%d/relationships/members?page[number]=1&page[size]=2", srv.URL, messengerId)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

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
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers/%d/members?page[size]=0", srv.URL, messengerId)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("DefaultPageSizeReturnsFullBoundedMemberSet", func(t *testing.T) {
		url := fmt.Sprintf("%s/messengers/%d/members", srv.URL, messengerId)
		req := messengersRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 3)
	})
}
