package backeffect

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

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type backEffectTestServerInformation struct{}

func (t *backEffectTestServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *backEffectTestServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &backEffectTestServerInformation{}

func setupBackEffectRouter() *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	InitResource(&backEffectTestServerInformation{})(r, l)
	return r
}

func backEffectRequestWithTenant(method, url string, tenantId uuid.UUID) *http.Request {
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

// TestGetBackEffectsInMap_ReturnsEntries drives GET .../backEffects through
// the real resource router against the in-process registry, verifying the
// JSON:API collection envelope reflects the entries set via the processor.
func TestGetBackEffectsInMap_ReturnsEntries(t *testing.T) {
	tenantId := uuid.New()
	ten, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	f := field.NewBuilder(1, 1, 100000000).Build()
	p := NewProcessor(logrus.New(), ctx)
	p.Set(f, BackEffectEntry{Effect: 0, FieldId: 100000000, PageId: 1, Duration: 1000})
	p.Set(f, BackEffectEntry{Effect: 1, FieldId: 100000000, PageId: 2, Duration: 0})

	srv := httptest.NewServer(setupBackEffectRouter())
	defer srv.Close()

	url := fmt.Sprintf("%s/worlds/1/channels/1/maps/100000000/instances/%s/backEffects", srv.URL, f.Instance())
	req := backEffectRequestWithTenant(http.MethodGet, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 2)

	assert.Equal(t, "1", doc.Data.DataArray[0].ID)
	assert.Equal(t, "backEffect", doc.Data.DataArray[0].Type)

	var attrs0 RestModel
	require.NoError(t, json.Unmarshal(doc.Data.DataArray[0].Attributes, &attrs0))
	assert.EqualValues(t, 0, attrs0.Effect)
	assert.EqualValues(t, 1000, attrs0.Duration)

	assert.Equal(t, "2", doc.Data.DataArray[1].ID)
	var attrs1 RestModel
	require.NoError(t, json.Unmarshal(doc.Data.DataArray[1].Attributes, &attrs1))
	assert.EqualValues(t, 1, attrs1.Effect)
}

// TestGetBackEffectsInMap_EmptyIsTwoHundred asserts the deviation from PRD
// §5 settled by design §3.3: a field with no active back effects returns
// 200 with an empty `data` array, never 404.
func TestGetBackEffectsInMap_EmptyIsTwoHundred(t *testing.T) {
	tenantId := uuid.New()
	_, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	srv := httptest.NewServer(setupBackEffectRouter())
	defer srv.Close()

	url := fmt.Sprintf("%s/worlds/1/channels/1/maps/100000001/instances/%s/backEffects", srv.URL, uuid.Nil)
	req := backEffectRequestWithTenant(http.MethodGet, url, tenantId)

	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

	require.NotNil(t, doc.Data)
	assert.Len(t, doc.Data.DataArray, 0)
}
