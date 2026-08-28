package environment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tm
}

// TestGetAll_ParsesCollection verifies GetAll decodes a JSON:API collection
// of environment-objects resources into a slice of Model, preserving
// order.
func TestGetAll_ParsesCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[
			{"type":"environment-objects","id":"OBSTACLE:obs3","attributes":{"kind":"OBSTACLE","name":"obs3","state":2}},
			{"type":"environment-objects","id":"ENVIRONMENT:gate01","attributes":{"kind":"ENVIRONMENT","name":"gate01","state":1}}
		]}`))
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()
	ms, err := NewProcessor(logrus.New(), ctx).GetAll(f)
	require.NoError(t, err)
	require.Len(t, ms, 2)
	require.Equal(t, "OBSTACLE", ms[0].Kind())
	require.Equal(t, "obs3", ms[0].Name())
	require.Equal(t, uint32(2), ms[0].State())
	require.Equal(t, "ENVIRONMENT", ms[1].Kind())
	require.Equal(t, "gate01", ms[1].Name())
	require.Equal(t, uint32(1), ms[1].State())
}

// TestGetAll_EmptyCollection verifies GetAll returns an empty, non-nil-error
// slice when atlas-maps reports no environment objects for the field.
func TestGetAll_EmptyCollection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()
	ms, err := NewProcessor(logrus.New(), ctx).GetAll(f)
	require.NoError(t, err)
	require.Len(t, ms, 0)
}

// TestGetAll_ServerError verifies a 500 from atlas-maps surfaces as a
// non-nil error.
func TestGetAll_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()
	_, err := NewProcessor(logrus.New(), ctx).GetAll(f)
	require.Error(t, err)
}

// TestGetAll_RequestsInstancePath verifies GetAll builds the
// instance-scoped environment path atlas-maps registers.
func TestGetAll_RequestsInstancePath(t *testing.T) {
	var recordedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910010000)).SetInstance(uuid.Nil).Build()
	_, err := NewProcessor(logrus.New(), ctx).GetAll(f)
	require.NoError(t, err)

	want := "/worlds/0/channels/1/maps/910010000/instances/00000000-0000-0000-0000-000000000000/environment"
	require.True(t, strings.HasSuffix(recordedPath, want), "path: %s", recordedPath)
}
