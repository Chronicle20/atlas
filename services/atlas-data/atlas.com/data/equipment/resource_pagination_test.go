package equipment

import (
	"atlas-data/document"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestEquipmentSlots_PaginationEnvelope exercises the embedded-sub-list
// GET /data/equipment/{id}/slots route: an equipment item with 3 slot rows
// paginated at page[size]=2 returns 2 items, total=3, and a next link.
func TestEquipmentSlots_PaginationEnvelope(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()

	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)
	storage := document.NewStorage(l, db, GetModelRegistry(), "EQUIPMENT")
	_, err = storage.Add(ctx)(RestModel{
		Id:           1102000,
		WeaponAttack: 5,
		Slots:        7,
		EquipSlots: []SlotRestModel{
			{Id: "Ea", Name: "Earring-1", WZ: "Ea", Slot: -9},
			{Id: "Ea", Name: "Earring-2", WZ: "Ea", Slot: -10},
			{Id: "Ea", Name: "Earring-3", WZ: "Ea", Slot: -49},
		},
	})()
	require.NoError(t, err)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/equipment/1102000/slots?page[number]=1&page[size]=2", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc struct {
		Data  []interface{}          `json:"data"`
		Meta  map[string]interface{} `json:"meta"`
		Links map[string]interface{} `json:"links"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	assert.Len(t, doc.Data, 2)
	assert.EqualValues(t, 3, doc.Meta["total"])
	assert.NotNil(t, doc.Links["next"])
}

func TestEquipmentSlots_RejectsBadPageSize(t *testing.T) {
	db := setupResourceTestDB(t)
	tenantId := uuid.New()
	setupTestEquipmentData(t, db, tenantId)

	router := setupTestRouter(db)
	ts := httptest.NewServer(router)
	defer ts.Close()

	url := fmt.Sprintf("%s/data/equipment/1302000/slots?page[size]=abc", ts.URL)
	resp, err := http.DefaultClient.Do(createRequestWithTenant("GET", url, tenantId))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
