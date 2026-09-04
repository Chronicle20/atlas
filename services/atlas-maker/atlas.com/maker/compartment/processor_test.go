package compartment_test

import (
	"atlas-maker/compartment"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// TestGetByTypeDecodesAssets stands up an httptest server emulating
// atlas-inventory's per-type compartment read (with an included "assets"
// relationship) and asserts the processor decodes the compartment and its
// assets.
func TestGetByTypeDecodesAssets(t *testing.T) {
	var gotPath string
	compartmentId := uuid.New()
	body := `{
  "data": {
    "type": "compartments",
    "id": "` + compartmentId.String() + `",
    "attributes": {"type": 4, "capacity": 100},
    "relationships": {
      "assets": {"data": [{"type": "assets", "id": "1"}]}
    }
  },
  "included": [
    {"type": "assets", "id": "1", "attributes": {"templateId": 4010000, "quantity": 25}}
  ]
}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(body))
	}))
	defer func() { srv.Close() }()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	m, err := compartment.NewProcessor(testLogger(), context.Background()).GetByType(1001, inventory.TypeValueETC)
	require.NoError(t, err)

	assert.Equal(t, "/characters/1001/inventory/compartments?type=4", gotPath)
	assert.Equal(t, compartmentId, m.Id())
	assert.EqualValues(t, 100, m.Capacity())
	require.Len(t, m.Assets(), 1)
	assert.Equal(t, item.Id(4010000), m.Assets()[0].TemplateId())
	assert.EqualValues(t, 25, m.Assets()[0].Quantity())
	assert.EqualValues(t, 25, m.QuantityOf(item.Id(4010000)))
}

// TestGetByTypeNotFound proves a 404 from atlas-inventory surfaces as
// requests.ErrNotFound, distinguishable from a genuine read failure.
func TestGetByTypeNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer func() { srv.Close() }()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	_, err := compartment.NewProcessor(testLogger(), context.Background()).GetByType(9999999, inventory.TypeValueEquip)
	require.Error(t, err)
	assert.True(t, errors.Is(err, requests.ErrNotFound))
}

// TestCanAccommodateRoundTripsMultipleItems asserts a multi-item request
// round-trips (the server-decoded request carries every item this call was
// given) and that Accommodated == false is surfaced when atlas-inventory
// rejects the set, with a per-item Results breakdown decoded without error.
func TestCanAccommodateRoundTripsMultipleItems(t *testing.T) {
	var gotBody struct {
		Data struct {
			Attributes struct {
				Items []struct {
					ItemId   uint32 `json:"itemId"`
					Quantity uint32 `json:"quantity"`
				} `json:"items"`
			} `json:"attributes"`
		} `json:"data"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
  "data": {
    "type": "inventoryAccommodations",
    "id": "1001",
    "attributes": {
      "accommodated": false,
      "results": [
        {"itemId": 4010000, "quantity": 5, "accommodated": true},
        {"itemId": 1382005, "quantity": 1, "accommodated": false}
      ]
    }
  }
}`))
	}))
	defer func() { srv.Close() }()
	t.Setenv("INVENTORY_SERVICE_URL", srv.URL+"/")

	items := []compartment.AccommodationItem{
		{ItemId: item.Id(4010000), Quantity: 5},
		{ItemId: item.Id(1382005), Quantity: 1},
	}
	ok, err := compartment.NewProcessor(testLogger(), context.Background()).CanAccommodate(1001, items)
	require.NoError(t, err)
	assert.False(t, ok)

	require.Len(t, gotBody.Data.Attributes.Items, 2)
	assert.EqualValues(t, 4010000, gotBody.Data.Attributes.Items[0].ItemId)
	assert.EqualValues(t, 5, gotBody.Data.Attributes.Items[0].Quantity)
	assert.EqualValues(t, 1382005, gotBody.Data.Attributes.Items[1].ItemId)
	assert.EqualValues(t, 1, gotBody.Data.Attributes.Items[1].Quantity)
}
