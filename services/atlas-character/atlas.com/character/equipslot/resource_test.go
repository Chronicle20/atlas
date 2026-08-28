package equipslot

// TestPostExtendEquipSlot proves the write route task 22's InitResource doc
// comment deferred (task-240 task 23, R2): a POST persists a real row via
// Extend (task 22's domain layer, unchanged), and the JSON:API response
// carries the SAME slotIndex the caller sent (R1 -- this route never
// resolves or invents that value) and a real expiry.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type equipSlotResourceTestServerInfo struct{}

func (t *equipSlotResourceTestServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t *equipSlotResourceTestServerInfo) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &equipSlotResourceTestServerInfo{}

func setupEquipSlotResourceRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&equipSlotResourceTestServerInfo{})(db)
	ri(r, l)
	return r
}

func equipSlotRequestWithTenant(t *testing.T, method, url string, tenantId uuid.UUID, body []byte) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "95")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func extendBody(t *testing.T, slotIndex int16, days uint16) []byte {
	t.Helper()
	return extendBodyWithTransaction(t, slotIndex, days, uuid.Nil)
}

func extendBodyWithTransaction(t *testing.T, slotIndex int16, days uint16, transactionId uuid.UUID) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(ExtendInputRestModel{SlotIndex: slotIndex, Days: days, TransactionId: transactionId})
	require.NoError(t, err)
	return b
}

func TestPostExtendEquipSlot(t *testing.T) {
	db := testDB(t)
	tenantId := uuid.New()
	characterId := uint32(42)
	S := testSlotIndex(t)

	router := setupEquipSlotResourceRouter(db)
	req := equipSlotRequestWithTenant(t, http.MethodPost, "/characters/42/equip-slot-extensions", tenantId, extendBody(t, S, 30))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body: %s", rr.Body.String())

	active, err := GetActive(db, tenantId, characterId)
	require.NoError(t, err)
	require.Len(t, active, 1, "the POST must persist a row via Extend")
	assert.Equal(t, S, active[0].SlotIndex(), "the persisted slotIndex must be exactly what the caller sent")
	assert.WithinDuration(t, time.Now().Add(30*24*time.Hour), active[0].ExpiresAt(), time.Minute)

	var payload struct {
		Data struct {
			Attributes struct {
				CharacterId uint32    `json:"characterId"`
				SlotIndex   int16     `json:"slotIndex"`
				ExpiresAt   time.Time `json:"expiresAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &payload))
	assert.Equal(t, characterId, payload.Data.Attributes.CharacterId)
	assert.Equal(t, S, payload.Data.Attributes.SlotIndex, "the response must echo the caller's slotIndex, not a resolved/invented one")
	assert.WithinDuration(t, time.Now().Add(30*24*time.Hour), payload.Data.Attributes.ExpiresAt, time.Minute)
}

// TestPostExtendEquipSlot_RedeliveredTransactionIdDoesNotDoubleExtend proves
// the write route's idempotency guard (task-240 task 24c): atlas-cashshop's
// EXTEND_EQUIP_SLOT outbox command is at-least-once, so a POST redelivered
// with the SAME transactionId must not add days a second time. Driven
// through the route (not around it, i.e. not calling Extend directly) so it
// proves the handler actually threads TransactionId from the wire body into
// the dedupe check.
func TestPostExtendEquipSlot_RedeliveredTransactionIdDoesNotDoubleExtend(t *testing.T) {
	db := testDB(t)
	tenantId := uuid.New()
	characterId := uint32(42)
	S := testSlotIndex(t)
	txId := uuid.New()

	router := setupEquipSlotResourceRouter(db)

	first := equipSlotRequestWithTenant(t, http.MethodPost, "/characters/42/equip-slot-extensions", tenantId, extendBodyWithTransaction(t, S, 30, txId))
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, first)
	require.Equal(t, http.StatusOK, rr1.Code, "body: %s", rr1.Body.String())

	// Redeliver the identical command: same character, slot, days, AND
	// transaction id -- exactly what an at-least-once outbox redelivery
	// looks like on the wire.
	second := equipSlotRequestWithTenant(t, http.MethodPost, "/characters/42/equip-slot-extensions", tenantId, extendBodyWithTransaction(t, S, 30, txId))
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, second)
	require.Equal(t, http.StatusOK, rr2.Code, "a redelivery must not fail the caller -- it is success-without-effect")

	active, err := GetActive(db, tenantId, characterId)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.WithinDuration(t, time.Now().Add(30*24*time.Hour), active[0].ExpiresAt(), time.Minute, "the redelivery must not add another 30 days")

	var payload struct {
		Data struct {
			Attributes struct {
				ExpiresAt time.Time `json:"expiresAt"`
			} `json:"attributes"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr2.Body.Bytes(), &payload))
	assert.WithinDuration(t, time.Now().Add(30*24*time.Hour), payload.Data.Attributes.ExpiresAt, time.Minute)
}
