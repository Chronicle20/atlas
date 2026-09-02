package _map

import (
	"atlas-data/point"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func seedGroundMap(t *testing.T, db *gorm.DB, ctx context.Context, mapId uint32) {
	t.Helper()
	tree := NewFootholdTree(-32000, -32000, 32000, 32000).Insert([]FootholdRestModel{
		{Id: 1, First: &point.RestModel{X: -100, Y: -50}, Second: &point.RestModel{X: 100, Y: -50}},
	})
	l, _ := test.NewNullLogger()
	s := NewStorage(l, db)
	m := RestModel{Id: _map.Id(mapId), Name: "Test", StreetName: "Test", FootholdTree: *tree}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)
}

func groundRequestBody(points string) []byte {
	return []byte(fmt.Sprintf(`{"data":{"type":"grounds","id":"0","attributes":{"points":%s}}}`, points))
}

func postGroundRequest(url string, body []byte, tenantId string) *http.Request {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId)
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func TestHandleGetMapGroundRequest(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	seedGroundMap(t, db, ctx, 100000000)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps/100000000/ground", ts.URL)

	t.Run("single point over a flat foothold", func(t *testing.T) {
		body := groundRequestBody(`[{"x":0,"y":-100}]`)
		req := postGroundRequest(url, body, tn.Id().String())
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 1)

		var attrs map[string]interface{}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &attrs))
		assert.Equal(t, float64(-50), attrs["y"])
		assert.Equal(t, float64(1), attrs["fh"])
		assert.Equal(t, true, attrs["found"])
	})

	t.Run("two points, one over empty space", func(t *testing.T) {
		body := groundRequestBody(`[{"x":0,"y":-100},{"x":30000,"y":-100}]`)
		req := postGroundRequest(url, body, tn.Id().String())
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)

		var first map[string]interface{}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &first))
		assert.Equal(t, true, first["found"])
		assert.Equal(t, float64(-50), first["y"])

		var second map[string]interface{}
		require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &second))
		assert.Equal(t, false, second["found"])
		assert.Equal(t, float64(0), second["y"])
		assert.Equal(t, float64(0), second["fh"])
	})

	t.Run("empty list is bad request", func(t *testing.T) {
		body := groundRequestBody(`[]`)
		req := postGroundRequest(url, body, tn.Id().String())
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
