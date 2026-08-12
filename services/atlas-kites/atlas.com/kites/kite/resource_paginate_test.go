package kite

import (
	"atlas-kites/character"
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

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupKitesRegistries(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
	character.InitRegistry(rc)
}

type kitesTestServerInformation struct{}

func (t *kitesTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *kitesTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &kitesTestServerInformation{}

func setupKitesRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&kitesTestServerInformation{})(r, l)
	return r
}

func kitesRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetKitesInMapPaginates drives GET
// /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/kites through the real
// resource router against Redis-backed kite + character-in-map registries.
// It seeds 5 kites for 5 different characters, all placed in the same field,
// and requests page[size]=2, asserting three pages of 2/2/1 with stable
// ordering by kite id (not the registry's unordered iteration order) and no
// duplicate kite ids across pages.
func TestGetKitesInMapPaginates(t *testing.T) {
	setupKitesRegistries(t)

	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	worldId := world.Id(1)
	channelId := channel.Id(1)
	mapId := mapconst.Id(100000000)
	instanceId := uuid.Nil
	f := field.NewBuilder(worldId, channelId, mapId).SetInstance(instanceId).Build()

	cp := character.NewProcessor(logrus.New(), ctx)

	// Seed out of kite-id order so a naive registry-iteration-order response
	// would fail the ordering assertions below.
	type seed struct {
		characterId uint32
		kiteId      uint32
	}
	seeds := []seed{
		{characterId: 500, kiteId: 5},
		{characterId: 300, kiteId: 3},
		{characterId: 100, kiteId: 1},
		{characterId: 400, kiteId: 4},
		{characterId: 200, kiteId: 2},
	}
	for _, s := range seeds {
		cp.Enter(f, s.characterId)
		m := NewBuilder(s.kiteId, f, s.characterId).
			SetName(fmt.Sprintf("kite-%d", s.kiteId)).
			SetTemplateId(1).
			SetMessage("hello").
			SetPosition(0, 0).
			SetCreatedAt(time.Now()).
			Build()
		require.NoError(t, getRegistry().Put(ctx, m))
	}

	srv := httptest.NewServer(setupKitesRouter())
	defer srv.Close()

	fetchPage := func(t *testing.T, pageNumber int) *jsonapi.Document {
		url := fmt.Sprintf("%s/worlds/%d/channels/%d/maps/%d/instances/%s/kites?page[number]=%d&page[size]=2", srv.URL, worldId, channelId, mapId, instanceId, pageNumber)
		req := kitesRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		return &doc
	}

	seen := make(map[string]bool)

	t.Run("Page1_TwoItems_IdsOneAndTwo", func(t *testing.T) {
		doc := fetchPage(t, 1)
		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		require.NotNil(t, doc.Meta)
		assert.EqualValues(t, 5, doc.Meta["total"])
		page := doc.Meta["page"].(map[string]interface{})
		assert.EqualValues(t, 3, page["last"])

		assert.Equal(t, "1", doc.Data.DataArray[0].ID)
		assert.Equal(t, "2", doc.Data.DataArray[1].ID)
		for _, d := range doc.Data.DataArray {
			seen[d.ID] = true
		}
	})

	t.Run("Page2_TwoItems_IdsThreeAndFour", func(t *testing.T) {
		doc := fetchPage(t, 2)
		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		assert.Equal(t, "3", doc.Data.DataArray[0].ID)
		assert.Equal(t, "4", doc.Data.DataArray[1].ID)
		for _, d := range doc.Data.DataArray {
			require.False(t, seen[d.ID], "kite id %s duplicated across pages", d.ID)
			seen[d.ID] = true
		}
	})

	t.Run("Page3_OneItem_IdFive", func(t *testing.T) {
		doc := fetchPage(t, 3)
		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 1)

		assert.Equal(t, "5", doc.Data.DataArray[0].ID)
		for _, d := range doc.Data.DataArray {
			require.False(t, seen[d.ID], "kite id %s duplicated across pages", d.ID)
			seen[d.ID] = true
		}
	})

	assert.Len(t, seen, 5, "all 5 kite ids must appear exactly once across the three pages")
}

func TestGetKiteByCharacterId_NotFound(t *testing.T) {
	setupKitesRegistries(t)

	tenantId := uuid.New()
	_, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	srv := httptest.NewServer(setupKitesRouter())
	defer srv.Close()

	url := fmt.Sprintf("%s/kites/999", srv.URL)
	req := kitesRequestWithTenant(http.MethodGet, url, tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
