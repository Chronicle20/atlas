package character

// TestExtendEquipSlot proves atlas-cashshop's REST client for atlas-character's
// ENABLE_EQUIP_SLOT subresource (task-240 task 23) round-trips the request the
// write route expects and decodes the response's expiresAt, against a real
// httptest fixture rather than a mock -- covering EquipSlotExtensionRestModel/
// ExtendEquipSlotInputRestModel (rest.go) and requestExtendEquipSlot
// (requests.go).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func TestExtendEquipSlot(t *testing.T) {
	const characterId = uint32(42)
	slotIndex := int16(-59)
	days := uint16(30)
	transactionId := uuid.New()
	expiresAt := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	var capturedBody ExtendEquipSlotInputRestModel
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, fmt.Sprintf("/api/characters/%d/equip-slot-extensions", characterId), r.URL.Path)

		var payload struct {
			Data struct {
				Attributes ExtendEquipSlotInputRestModel `json:"attributes"`
			} `json:"data"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		capturedBody = payload.Data.Attributes

		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprintf(w, `{"data":{"type":"equip-slot-extensions","id":"1","attributes":{"characterId":%d,"slotIndex":%d,"expiresAt":"%s"}}}`,
			characterId, slotIndex, expiresAt.Format(time.RFC3339))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")

	ctx := databasetest.TenantContext(uuid.New())
	l, _ := testlog.NewNullLogger()

	got, err := NewProcessor(l, ctx).ExtendEquipSlot(characterId, slotIndex, days, transactionId)
	require.NoError(t, err)
	require.True(t, expiresAt.Equal(got))

	require.Equal(t, slotIndex, capturedBody.SlotIndex)
	require.Equal(t, days, capturedBody.Days)
	require.Equal(t, transactionId, capturedBody.TransactionId)
}
