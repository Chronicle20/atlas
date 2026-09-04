package environment

import (
	"atlas-maps/rest"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInformation struct{}

func (testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = testServerInformation{}

const (
	testWorldId   = "0"
	testChannelId = "1"
	testMapId     = "910010000"
)

// environmentTestField builds the field.Model that corresponds to the URL
// vars set on the test request, so seeded registry entries and the request
// the handler parses agree on the same field.
func environmentTestField(instanceId uuid.UUID) field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(instanceId).Build()
}

// decodeAttributes unmarshals a jsonapi.Data's raw attributes into a map for
// assertion.
func decodeAttributes(t *testing.T, d jsonapi.Data) map[string]interface{} {
	t.Helper()
	var attrs map[string]interface{}
	require.NoError(t, json.Unmarshal(d.Attributes, &attrs))
	return attrs
}

// newEnvironmentRequest builds an httptest request carrying a fresh tenant's
// context and the mux URL vars the ParseWorldId/ParseChannelId/ParseMapId/
// ParseInstanceId nest expects.
func newEnvironmentRequest(t *testing.T, method string, instanceId uuid.UUID) (*http.Request, *rest.HandlerDependency, *rest.HandlerContext) {
	t.Helper()

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	l, _ := test.NewNullLogger()

	req := httptest.NewRequest(method, "/environment", nil)
	req = req.WithContext(ctx)
	req = mux.SetURLVars(req, map[string]string{
		"worldId":    testWorldId,
		"channelId":  testChannelId,
		"mapId":      testMapId,
		"instanceId": instanceId.String(),
	})

	dep := server.NewHandlerDependency(l, ctx)
	hc := server.NewHandlerContext(testServerInformation{})
	d := rest.HandlerDependency(dep)
	c := rest.HandlerContext(hc)
	return req, &d, &c
}

func TestGetEnvironmentInMap_Empty(t *testing.T) {
	instanceId := uuid.New()
	req, d, c := newEnvironmentRequest(t, http.MethodGet, instanceId)
	rr := httptest.NewRecorder()

	handleGetEnvironmentInMap(d, c)(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.NotNil(t, doc.Data.DataArray)
	require.Len(t, doc.Data.DataArray, 0)
}

func TestGetEnvironmentInMap_ReturnsTrackedInOrder(t *testing.T) {
	instanceId := uuid.New()
	req, d, c := newEnvironmentRequest(t, http.MethodGet, instanceId)
	f := environmentTestField(instanceId)

	p := NewProcessor(d.Logger(), d.Context())
	_, err := p.Set(f, field.ObjectKindObstacle, "a", 1)
	require.NoError(t, err)
	_, err = p.Set(f, field.ObjectKindEnvironment, "b", 2)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handleGetEnvironmentInMap(d, c)(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 2)

	attrs0 := decodeAttributes(t, doc.Data.DataArray[0])
	require.Equal(t, "OBSTACLE:a", doc.Data.DataArray[0].ID)
	require.Equal(t, "OBSTACLE", attrs0["kind"])
	require.Equal(t, "a", attrs0["name"])
	require.EqualValues(t, 1, attrs0["state"])

	attrs1 := decodeAttributes(t, doc.Data.DataArray[1])
	require.Equal(t, "ENVIRONMENT:b", doc.Data.DataArray[1].ID)
	require.EqualValues(t, 2, attrs1["state"])
}

func TestGetEnvironmentInMap_TenantIsolation(t *testing.T) {
	instanceId := uuid.New()
	f := environmentTestField(instanceId)

	tenA, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctxA := tenant.WithContext(context.Background(), tenA)
	l, _ := test.NewNullLogger()
	_, err = NewProcessor(l, ctxA).Set(f, field.ObjectKindObstacle, "a", 1)
	require.NoError(t, err)

	req, d, c := newEnvironmentRequest(t, http.MethodGet, instanceId)
	rr := httptest.NewRecorder()

	handleGetEnvironmentInMap(d, c)(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 0)
}

func postEnvironment(t *testing.T, instanceId uuid.UUID, input RestModel) (*httptest.ResponseRecorder, *rest.HandlerDependency, field.Model) {
	t.Helper()
	req, d, c := newEnvironmentRequest(t, http.MethodPost, instanceId)
	f := environmentTestField(instanceId)
	rr := httptest.NewRecorder()

	handleSetEnvironmentInMap(d, c, input)(rr, req)

	return rr, d, f
}

func TestPostEnvironment_BlankName(t *testing.T) {
	instanceId := uuid.New()
	rr, d, f := postEnvironment(t, instanceId, RestModel{Kind: "ENVIRONMENT", Name: "", State: 1})

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Len(t, NewProcessor(d.Logger(), d.Context()).GetAll(f), 0)
}

func TestPostEnvironment_WhitespaceName(t *testing.T) {
	instanceId := uuid.New()
	rr, d, f := postEnvironment(t, instanceId, RestModel{Kind: "ENVIRONMENT", Name: "   ", State: 1})

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Len(t, NewProcessor(d.Logger(), d.Context()).GetAll(f), 0)
}

func TestPostEnvironment_BadKind(t *testing.T) {
	instanceId := uuid.New()
	rr, d, f := postEnvironment(t, instanceId, RestModel{Kind: "GATE", Name: "gate01", State: 1})

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Len(t, NewProcessor(d.Logger(), d.Context()).GetAll(f), 0)
}

func TestPostEnvironment_Accepted(t *testing.T) {
	instanceId := uuid.New()
	rr, d, f := postEnvironment(t, instanceId, RestModel{Kind: "OBSTACLE", Name: "obs3", State: 2})

	require.Equal(t, http.StatusAccepted, rr.Code)
	entries := NewProcessor(d.Logger(), d.Context()).GetAll(f)
	require.Equal(t, []ObjectEntry{{Kind: field.ObjectKindObstacle, Name: "obs3", State: 2}}, entries)
}

func TestPostEnvironment_BlankKindDefaults(t *testing.T) {
	instanceId := uuid.New()
	rr, d, f := postEnvironment(t, instanceId, RestModel{Kind: "", Name: "gate01", State: 1})

	require.Equal(t, http.StatusAccepted, rr.Code)
	entries := NewProcessor(d.Logger(), d.Context()).GetAll(f)
	require.Len(t, entries, 1)
	require.Equal(t, field.ObjectKindEnvironment, entries[0].Kind)
}

func TestDeleteEnvironment_Untracked(t *testing.T) {
	instanceId := uuid.New()
	req, d, c := newEnvironmentRequest(t, http.MethodDelete, instanceId)
	rr := httptest.NewRecorder()

	handleResetEnvironmentInMap(d, c)(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
}

func TestDeleteEnvironment_ClearsTracked(t *testing.T) {
	instanceId := uuid.New()
	req, d, c := newEnvironmentRequest(t, http.MethodDelete, instanceId)
	f := environmentTestField(instanceId)
	p := NewProcessor(d.Logger(), d.Context())
	_, err := p.Set(f, field.ObjectKindObstacle, "a", 1)
	require.NoError(t, err)
	_, err = p.Set(f, field.ObjectKindEnvironment, "b", 2)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handleResetEnvironmentInMap(d, c)(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Len(t, p.GetAll(f), 0)
}
