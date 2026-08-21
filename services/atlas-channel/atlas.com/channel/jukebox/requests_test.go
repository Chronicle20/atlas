package jukebox

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

// TestGetActiveDecodesTheJukeboxResource verifies GetActive decodes the
// atlas-maps jukebox JSON:API resource into a populated RestModel.
func TestGetActiveDecodesTheJukeboxResource(t *testing.T) {
	instanceId := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"jukebox","id":"5100000","attributes":{"itemId":5100000,"playerName":"Chronicle"}}}`))
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(instanceId).Build()
	m, err := NewProcessor(logrus.New(), ctx).GetActive(f)
	require.NoError(t, err)
	require.Equal(t, uint32(5100000), m.ItemId)
	require.Equal(t, "Chronicle", m.PlayerName)
}

// TestGetActiveRequestsTheInstanceScopedPath verifies GetActive builds the
// instance-scoped jukebox path atlas-maps registers.
func TestGetActiveRequestsTheInstanceScopedPath(t *testing.T) {
	instanceId := uuid.New()
	var recordedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recordedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"jukebox","id":"5100000","attributes":{"itemId":5100000,"playerName":"Chronicle"}}}`))
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(instanceId).Build()
	_, err := NewProcessor(logrus.New(), ctx).GetActive(f)
	require.NoError(t, err)

	want := "/worlds/0/channels/1/maps/100000000/instances/" + instanceId.String() + "/jukebox"
	require.True(t, strings.HasSuffix(recordedPath, want), "path: %s", recordedPath)
}

// TestGetActiveReturnsErrorWhenNothingPlaying verifies a 404 from atlas-maps
// (no active jukebox playback in the field) surfaces as a non-nil error;
// the caller in Task 8 treats any error as "no song".
func TestGetActiveReturnsErrorWhenNothingPlaying(t *testing.T) {
	instanceId := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(instanceId).Build()
	_, err := NewProcessor(logrus.New(), ctx).GetActive(f)
	require.Error(t, err)
}
