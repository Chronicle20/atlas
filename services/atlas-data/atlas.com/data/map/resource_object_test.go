package _map

import (
	"atlas-data/map/object"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMapObjects_Endpoint covers GET /data/maps/{mapId}/objects end to end:
// the objects survive the jsonapi document round trip through storage and are
// served with the object's name as the resource id.
func TestMapObjects_Endpoint(t *testing.T) {
	db := setupStorageTestDB(t)
	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	s := NewStorage(testLogger(t), db)
	m := RestModel{
		Id:         _map.Id(103000800),
		Name:       "First Time Together",
		StreetName: "Kerning City",
		Objects: []object.RestModel{
			{Name: "gate", State: 1},
		},
	}
	_, err := s.Add(ctx)(m)()
	require.NoError(t, err)

	router := buildMapsRouter(t, db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/maps/103000800/objects", ts.URL)
	resp, err := http.DefaultClient.Do(mapsRequest(url, tn.Id()))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc struct {
		Data []struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Attributes struct {
				Name  string `json:"name"`
				State uint32 `json:"state"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	require.Len(t, doc.Data, 1)
	assert.Equal(t, "objects", doc.Data[0].Type)
	assert.Equal(t, "gate", doc.Data[0].Id)
	assert.Equal(t, "gate", doc.Data[0].Attributes.Name)
	assert.EqualValues(t, 1, doc.Data[0].Attributes.State)
}
