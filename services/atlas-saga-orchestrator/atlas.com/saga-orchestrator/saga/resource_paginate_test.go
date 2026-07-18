package saga

import (
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

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type sagaTestServerInformation struct{}

func (t *sagaTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *sagaTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &sagaTestServerInformation{}

func setupSagaRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&sagaTestServerInformation{})(r, l)
	return r
}

func sagaRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

func seedSaga(t *testing.T, ctx context.Context, transactionId uuid.UUID) {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("step-1", Pending, AwardMesos, nil).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))
}

// TestGetSagasPaginates drives GET /sagas through the real resource router
// against the in-memory saga cache (a Go map, whose iteration order is
// explicitly randomized at runtime), verifying the JSON:API paginated
// envelope AND that page 1's items come back in ascending
// transaction-id-string order (the stable-sort fix) rather than map
// iteration order.
func TestGetSagasPaginates(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	// Deliberately fixed, out-of-lexicographic-order UUIDs so the sort is
	// the only thing that can put them in order.
	idHigh := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	idLow := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	idMid := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	seedSaga(t, ctx, idHigh)
	seedSaga(t, ctx, idLow)
	seedSaga(t, ctx, idMid)

	srv := httptest.NewServer(setupSagaRouter())
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/sagas?page[number]=1&page[size]=2", srv.URL)
		req := sagaRequestWithTenant(http.MethodGet, url, tenantId)

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

		// seeded high, low, mid: page 1 must return low then mid in
		// ascending lexicographic order, not seed/map-iteration order.
		assert.Equal(t, idLow.String(), doc.Data.DataArray[0].ID)
		assert.Equal(t, idMid.String(), doc.Data.DataArray[1].ID)
	})

	t.Run("PageSizeZeroIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/sagas?page[size]=0", srv.URL)
		req := sagaRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("LegacyLimitParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/sagas?limit=5", srv.URL)
		req := sagaRequestWithTenant(http.MethodGet, url, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("PastLastPageReturnsEmptyWithPrevAtLast", func(t *testing.T) {
		url := fmt.Sprintf("%s/sagas?page[number]=99&page[size]=2", srv.URL)
		req := sagaRequestWithTenant(http.MethodGet, url, tenantId)

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
}
