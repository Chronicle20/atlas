package location

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestGetField_HappyPath verifies that GetField correctly decodes a JSON:API
// response from atlas-maps and returns a fully-populated field.Model.
func TestGetField_HappyPath(t *testing.T) {
	instanceUUID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/characters/1234/location"), "path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "character-locations",
				"id": "1234",
				"attributes": {
					"worldId": 2,
					"channelId": 3,
					"mapId": 100000000,
					"instance": "` + instanceUUID.String() + `"
				}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	f, err := GetField(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, uint8(2), uint8(f.WorldId()))
	require.Equal(t, uint8(3), uint8(f.ChannelId()))
	require.Equal(t, uint32(100000000), uint32(f.MapId()))
	require.Equal(t, instanceUUID, f.Instance())
}

// TestGetField_NotFound verifies that a 404 from atlas-maps is mapped to
// the package-level ErrNotFound sentinel so callers can distinguish
// first-login conditions from infrastructure errors.
func TestGetField_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	_, err := GetField(logrus.New(), context.Background(), 1234)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got %v", err)
}

// TestGetField_InfrastructureError verifies that a 5xx from atlas-maps
// returns a non-nil error that is NOT ErrNotFound — callers should treat
// these as infrastructure failures.
func TestGetField_InfrastructureError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	_, err := GetField(logrus.New(), context.Background(), 1234)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotFound), "expected non-ErrNotFound, got ErrNotFound")
}
