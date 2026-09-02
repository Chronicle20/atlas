package environment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// serveMapObjects stands up an atlas-data stub for
// GET /data/maps/{mapId}/objects and points the DATA service root at it.
func serveMapObjects(t *testing.T, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/objects") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")
}

func newDataBackedProcessor(t *testing.T) (Processor, field.Model) {
	t.Helper()
	l, _ := test.NewNullLogger()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)
	return NewProcessor(l, ctx), newTestField()
}

// TestProcessorSetRetainsDeclaredDefault pins the fix for the Kerning PQ gate:
// the object's visible state is 0 while the map declares a default of 1, so a
// reset must restore 1, not 0.
func TestProcessorSetRetainsDeclaredDefault(t *testing.T) {
	serveMapObjects(t, `{"data":[
		{"type":"objects","id":"gate","attributes":{"name":"gate","state":1}},
		{"type":"objects","id":"chest","attributes":{"name":"chest","state":0}}
	]}`)

	p, f := newDataBackedProcessor(t)

	entry, err := p.Set(f, field.ObjectKindEnvironment, "gate", 0)
	require.NoError(t, err)
	require.Equal(t, uint32(0), entry.State)
	require.Equal(t, uint32(1), entry.DefaultState)

	cleared := p.Reset(f)
	require.Equal(t, []ObjectEntry{{Kind: field.ObjectKindEnvironment, Name: "gate", State: 0, DefaultState: 1}}, cleared)
}

// TestProcessorSetResolvesDefaultOncePerObject asserts a re-set reuses the
// default already retained rather than re-resolving it from atlas-data.
func TestProcessorSetResolvesDefaultOncePerObject(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[{"type":"objects","id":"gate","attributes":{"name":"gate","state":1}}]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	p, f := newDataBackedProcessor(t)

	_, err := p.Set(f, field.ObjectKindEnvironment, "gate", 1)
	require.NoError(t, err)
	_, err = p.Set(f, field.ObjectKindEnvironment, "gate", 2)
	require.NoError(t, err)

	require.Equal(t, 1, calls)

	cleared := p.Reset(f)
	require.Len(t, cleared, 1)
	require.Equal(t, uint32(2), cleared[0].State)
	require.Equal(t, uint32(1), cleared[0].DefaultState)
}

// TestProcessorSetFallsBackToZeroForUndeclaredObject: an object the map does
// not declare is not addressable by any default, and a reset must never fail.
func TestProcessorSetFallsBackToZeroForUndeclaredObject(t *testing.T) {
	serveMapObjects(t, `{"data":[{"type":"objects","id":"gate","attributes":{"name":"gate","state":1}}]}`)

	p, f := newDataBackedProcessor(t)

	entry, err := p.Set(f, field.ObjectKindEnvironment, "unknown", 3)
	require.NoError(t, err)
	require.Equal(t, uint32(0), entry.DefaultState)
}

// TestProcessorSetFallsBackToZeroWhenDataUnreachable: an unreachable
// atlas-data costs the declared default, not the set.
func TestProcessorSetFallsBackToZeroWhenDataUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	p, f := newDataBackedProcessor(t)

	entry, err := p.Set(f, field.ObjectKindEnvironment, "gate", 1)
	require.NoError(t, err)
	require.Equal(t, uint32(0), entry.DefaultState)
}

// TestProcessorSetSkipsDataForObstacles: FieldObstacleAllReset means "all
// off", so an obstacle's default is 0 without consulting atlas-data.
func TestProcessorSetSkipsDataForObstacles(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":[{"type":"objects","id":"obs3","attributes":{"name":"obs3","state":1}}]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	p, f := newDataBackedProcessor(t)

	entry, err := p.Set(f, field.ObjectKindObstacle, "obs3", 2)
	require.NoError(t, err)
	require.Equal(t, uint32(0), entry.DefaultState)
	require.Equal(t, 0, calls)
}
