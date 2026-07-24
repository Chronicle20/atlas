package teleport_rock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type teleportRockResourceTestServerInfo struct{}

func (t *teleportRockResourceTestServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t *teleportRockResourceTestServerInfo) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &teleportRockResourceTestServerInfo{}

// stubWorldIdOf returns a fixed worldId for every character, matching the
// WorldIdOf signature after B1's fix (request-scoped logger first param).
func stubWorldIdOf(fixed world.Id) WorldIdOf {
	return func(_ logrus.FieldLogger, _ context.Context, _ uint32) (world.Id, error) {
		return fixed, nil
	}
}

func setupTeleportRockResourceRouter(db *gorm.DB, worldIdOf WorldIdOf) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&teleportRockResourceTestServerInfo{})(db)(worldIdOf)
	ri(r, l)
	return r
}

func teleportRockRequestWithTenant(t *testing.T, method, url string, tenantId uuid.UUID, body []byte) *http.Request {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader([]byte{})
	}
	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func addMapBody(t *testing.T, list string, mapId uint32) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(AddMapInputRestModel{List: list, MapId: mapId})
	require.NoError(t, err)
	return b
}

// TestPostTeleportRockMap drives POST /characters/{id}/teleport-rock-maps
// through the real resource router (InitResource), covering the validation
// and status-mapping paths of handleAddTeleportRockMap end to end: an
// unrecognized list value, an ineligible map, a duplicate, and a full list,
// plus the success envelope (regularCapacity/vipCapacity + the added map).
func TestPostTeleportRockMap(t *testing.T) {
	fixedWorldId := world.Id(3)

	tests := []struct {
		name       string
		seed       []_map.Id // pre-seeded regular-list maps for this character
		list       string
		mapId      uint32
		wantStatus int
	}{
		{
			name:       "ValidAddToRegular",
			list:       ListTypeRegular,
			mapId:      100000000,
			wantStatus: http.StatusOK,
		},
		{
			name:       "UnknownListValue",
			list:       "bogus",
			mapId:      100000000,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "IneligibleMapId",
			list:       ListTypeRegular,
			mapId:      4000000, // sub-9-digit: fails EligibleForRegistration
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "DuplicateMap",
			seed:       []_map.Id{100000000},
			list:       ListTypeRegular,
			mapId:      100000000,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "FullList",
			seed:       []_map.Id{100000000, 101000000, 102000000, 103000000, 104000000},
			list:       ListTypeRegular,
			mapId:      105000000,
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := testDatabase(t)
			tenantId := uuid.New()
			characterId := uint32(42)

			router := setupTeleportRockResourceRouter(db, stubWorldIdOf(fixedWorldId))
			srv := httptest.NewServer(router)
			defer srv.Close()

			if len(tc.seed) > 0 {
				require.NoError(t, replaceList(db, tenantId, characterId, ListTypeRegular, tc.seed))
			}

			url := fmt.Sprintf("%s/characters/%d/teleport-rock-maps", srv.URL, characterId)
			req := teleportRockRequestWithTenant(t, http.MethodPost, url, tenantId, addMapBody(t, tc.list, tc.mapId))

			resp, err := (&http.Client{}).Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			require.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.wantStatus == http.StatusOK {
				var doc jsonapi.Document
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
				require.NotNil(t, doc.Data)

				var rm RestModel
				require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &rm))

				assert.Equal(t, RegularCapacity, rm.RegularCapacity)
				assert.Equal(t, VipCapacity, rm.VipCapacity)
				assert.Contains(t, rm.Regular, _map.Id(tc.mapId))
			}
		})
	}
}

// TestDeleteTeleportRockMap drives DELETE
// /characters/{id}/teleport-rock-maps/{list}/{mapId} through the real
// resource router, covering the success path (map removed from the list),
// not-found (404), and a malformed mapId path segment (400).
func TestDeleteTeleportRockMap(t *testing.T) {
	fixedWorldId := world.Id(3)

	t.Run("RemovesExistingMap", func(t *testing.T) {
		db := testDatabase(t)
		tenantId := uuid.New()
		characterId := uint32(42)
		require.NoError(t, replaceList(db, tenantId, characterId, ListTypeRegular, []_map.Id{100000000, 101000000}))

		router := setupTeleportRockResourceRouter(db, stubWorldIdOf(fixedWorldId))
		srv := httptest.NewServer(router)
		defer srv.Close()

		url := fmt.Sprintf("%s/characters/%d/teleport-rock-maps/%s/%d", srv.URL, characterId, ListTypeRegular, 100000000)
		req := teleportRockRequestWithTenant(t, http.MethodDelete, url, tenantId, nil)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)

		var rm RestModel
		require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &rm))

		assert.NotContains(t, rm.Regular, _map.Id(100000000))
		assert.Contains(t, rm.Regular, _map.Id(101000000))
	})

	t.Run("NotPresentMapIsNotFound", func(t *testing.T) {
		db := testDatabase(t)
		tenantId := uuid.New()
		characterId := uint32(42)

		router := setupTeleportRockResourceRouter(db, stubWorldIdOf(fixedWorldId))
		srv := httptest.NewServer(router)
		defer srv.Close()

		url := fmt.Sprintf("%s/characters/%d/teleport-rock-maps/%s/%d", srv.URL, characterId, ListTypeRegular, 100000000)
		req := teleportRockRequestWithTenant(t, http.MethodDelete, url, tenantId, nil)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("NonNumericMapIdIsBadRequest", func(t *testing.T) {
		db := testDatabase(t)
		tenantId := uuid.New()
		characterId := uint32(42)

		router := setupTeleportRockResourceRouter(db, stubWorldIdOf(fixedWorldId))
		srv := httptest.NewServer(router)
		defer srv.Close()

		url := fmt.Sprintf("%s/characters/%d/teleport-rock-maps/%s/not-a-number", srv.URL, characterId, ListTypeRegular)
		req := teleportRockRequestWithTenant(t, http.MethodDelete, url, tenantId, nil)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
