package definition

import (
	"atlas-events/event/registry"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func setupDefinitionRouter(db *gorm.DB) *mux.Router {
	r := mux.NewRouter()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ri := InitResource(&testServerInformation{})(db)
	ri(r, l)
	return r
}

func requestWithTenant(method, url string, body []byte, tenantId uuid.UUID) *http.Request {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
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

// resourceTestHandler validates any configuration and never fails
// ConcurrencyKey — the REST layer only needs a resolvable handler so Create
// (used to seed fixtures) does not itself become the thing under test.
type resourceTestHandler struct{ t string }

func (h resourceTestHandler) Type() string                                { return h.t }
func (h resourceTestHandler) ValidateConfiguration(json.RawMessage) error { return nil }
func (h resourceTestHandler) ConcurrencyKey(context.Context, json.RawMessage) (string, error) {
	return "slot", nil
}

func (h resourceTestHandler) ConcurrencyKeyIsConstant() bool { return true }
func (h resourceTestHandler) Evaluate(context.Context, registry.Definition, registry.Work) (*registry.Seed, error) {
	return nil, nil
}

func (h resourceTestHandler) Start(context.Context, registry.Occurrence) (registry.Progress, error) {
	return registry.Progress{}, nil
}

func (h resourceTestHandler) Advance(context.Context, registry.Occurrence, registry.Work) (registry.Progress, error) {
	return registry.Progress{}, nil
}

func testDefinition(theType, name string) Model {
	m, err := NewBuilder(theType, name).SetConfiguration(json.RawMessage(`{}`)).Build()
	if err != nil {
		panic(err)
	}
	return m
}

func TestGetAllDefinitionsPaginates(t *testing.T) {
	registryReset(t)
	registry.Register(resourceTestHandler{t: "RES"})

	db := newTestDB(t)
	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	p := NewProcessor(testLogger(t), ctx, db)
	_, err = p.Create(testDefinition("RES", "One"))
	require.NoError(t, err)
	_, err = p.Create(testDefinition("RES", "Two"))
	require.NoError(t, err)
	_, err = p.Create(testDefinition("RES", "Three"))
	require.NoError(t, err)

	srv := httptest.NewServer(setupDefinitionRouter(db))
	defer srv.Close()

	t.Run("FirstPageOfTwo", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions?page[number]=1&page[size]=2", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))

		require.NotNil(t, doc.Data)
		require.Len(t, doc.Data.DataArray, 2)
		assert.EqualValues(t, 3, doc.Meta["total"])
	})

	t.Run("InvalidPageParamIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions?page[size]=0", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestGetAllDefinitionsFilters(t *testing.T) {
	registryReset(t)
	registry.Register(resourceTestHandler{t: "RES"})
	registry.Register(resourceTestHandler{t: "OTHER"})

	db := newTestDB(t)
	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	p := NewProcessor(testLogger(t), ctx, db)
	resEnabled, err := p.Create(testDefinition("RES", "Enabled RES"))
	require.NoError(t, err)
	resEnabled, err = p.SetEnabled(resEnabled.Id(), true)
	require.NoError(t, err)
	require.True(t, resEnabled.Enabled())

	resDisabled, err := p.Create(testDefinition("RES", "Disabled RES"))
	require.NoError(t, err)
	require.False(t, resDisabled.Enabled())

	otherEnabled, err := p.Create(testDefinition("OTHER", "Enabled OTHER"))
	require.NoError(t, err)
	otherEnabled, err = p.SetEnabled(otherEnabled.Id(), true)
	require.NoError(t, err)
	require.True(t, otherEnabled.Enabled())

	srv := httptest.NewServer(setupDefinitionRouter(db))
	defer srv.Close()

	decodeIds := func(t *testing.T, resp *http.Response) []string {
		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		ids := make([]string, 0, len(doc.Data.DataArray))
		for _, d := range doc.Data.DataArray {
			ids = append(ids, d.ID)
		}
		return ids
	}

	t.Run("FilterByTypeReturnsOnlyThatType", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions?filter[type]=RES", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		ids := decodeIds(t, resp)
		assert.ElementsMatch(t, []string{resEnabled.Id().String(), resDisabled.Id().String()}, ids)
	})

	t.Run("FilterByTypeAndEnabledReturnsOnlyEnabledSubset", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions?filter[type]=RES&filter[enabled]=true", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		ids := decodeIds(t, resp)
		assert.ElementsMatch(t, []string{resEnabled.Id().String()}, ids)
	})

	t.Run("FilterEnabledWithoutTypeIsBadRequest", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions?filter[enabled]=true", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("UnfilteredReturnsEverything", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions?page[size]=10", srv.URL)
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		ids := decodeIds(t, resp)
		assert.ElementsMatch(t, []string{resEnabled.Id().String(), resDisabled.Id().String(), otherEnabled.Id().String()}, ids)
	})
}

func TestGetDefinitionHandler(t *testing.T) {
	registryReset(t)
	registry.Register(resourceTestHandler{t: "RES"})

	db := newTestDB(t)
	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	p := NewProcessor(testLogger(t), ctx, db)
	created, err := p.Create(testDefinition("RES", "One"))
	require.NoError(t, err)

	srv := httptest.NewServer(setupDefinitionRouter(db))
	defer srv.Close()

	t.Run("Found", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions/%s", srv.URL, created.Id())
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		require.NotNil(t, doc.Data)
		require.NotNil(t, doc.Data.DataObject)
		assert.Equal(t, created.Id().String(), doc.Data.DataObject.ID)

		attrs := doc.Data.DataObject.Attributes
		var rm RestModel
		require.NoError(t, json.Unmarshal(attrs, &rm))
		assert.True(t, rm.SingleOccurrence, "RES has a constant concurrency key")
	})

	t.Run("NotFound", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions/%s", srv.URL, uuid.New())
		req := requestWithTenant(http.MethodGet, url, nil, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestPatchDefinitionHandler(t *testing.T) {
	registryReset(t)
	registry.Register(resourceTestHandler{t: "RES"})

	db := newTestDB(t)
	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	p := NewProcessor(testLogger(t), ctx, db)
	created, err := p.Create(testDefinition("RES", "One"))
	require.NoError(t, err)
	require.False(t, created.Enabled())

	srv := httptest.NewServer(setupDefinitionRouter(db))
	defer srv.Close()

	t.Run("EnabledOnlyIsAccepted", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions/%s", srv.URL, created.Id())
		body := []byte(fmt.Sprintf(`{"data":{"id":%q,"type":%q,"attributes":{"enabled":true}}}`, created.Id(), Resource))
		req := requestWithTenant(http.MethodPatch, url, body, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc jsonapi.Document
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
		var rm RestModel
		require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &rm))
		assert.True(t, rm.Enabled)
	})

	// FR-API2: PATCH accepts only enabled; any other attribute is rejected
	// with a JSON:API error rather than silently ignored.
	t.Run("OtherAttributeIsRejected", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions/%s", srv.URL, created.Id())
		body := []byte(fmt.Sprintf(`{"data":{"id":%q,"type":%q,"attributes":{"name":"Renamed"}}}`, created.Id(), Resource))
		req := requestWithTenant(http.MethodPatch, url, body, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("MixedAttributesAreRejected", func(t *testing.T) {
		url := fmt.Sprintf("%s/events/definitions/%s", srv.URL, created.Id())
		body := []byte(fmt.Sprintf(`{"data":{"id":%q,"type":%q,"attributes":{"enabled":false,"name":"Renamed"}}}`, created.Id(), Resource))
		req := requestWithTenant(http.MethodPatch, url, body, tenantId)

		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
