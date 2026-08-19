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

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
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

// serveLocation stands up an atlas-maps stub returning one character-locations
// document. attrs is spliced in verbatim so a test can omit "state" entirely.
func serveLocation(t *testing.T, attrs string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/characters/1234/location"), "path: %s", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "character-locations",
				"id": "1234",
				"attributes": {` + attrs + `}
			}
		}`))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetBaseURLForTest(srv.URL))
}

// TestGet_DecodesState pins the wire contract atlas-maps publishes.
func TestGet_DecodesState(t *testing.T) {
	serveLocation(t, `
		"worldId": 0,
		"channelId": 7,
		"mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"state": "IN_CASH_SHOP"`)

	m, err := Get(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, characterconst.PresenceStateInCashShop, m.State())
	require.Equal(t, uint8(7), uint8(m.ChannelId()))
	require.Equal(t, uint32(100000000), uint32(m.MapId()))
	require.Equal(t, uuid.Nil, m.Instance())
}

// TestGet_DecodesInField — channel 7 deliberately, not 0 or 1: the bug being
// fixed is a hard-coded 0 on the channel arm, and the client adds one for
// display, so a fixture on 0 or 1 passes against the broken code.
func TestGet_DecodesInField(t *testing.T) {
	serveLocation(t, `
		"worldId": 0,
		"channelId": 7,
		"mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"state": "IN_FIELD"`)

	m, err := Get(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, characterconst.PresenceStateInField, m.State())
	require.Equal(t, uint8(7), uint8(m.ChannelId()))
}

// TestGet_AbsentStateIsOffline covers an atlas-maps that has not been
// redeployed yet: /find must degrade to "not findable", never to a fabricated
// channel. The channel is still decoded — it is simply not trusted.
func TestGet_AbsentStateIsOffline(t *testing.T) {
	serveLocation(t, `
		"worldId": 0,
		"channelId": 7,
		"mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000"`)

	m, err := Get(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, characterconst.PresenceStateOffline, m.State())
	require.Equal(t, uint8(7), uint8(m.ChannelId()))
}

// TestGet_UnrecognisedStateIsOffline — same failure direction for a value this
// build does not know.
func TestGet_UnrecognisedStateIsOffline(t *testing.T) {
	serveLocation(t, `
		"worldId": 0,
		"channelId": 7,
		"mapId": 100000000,
		"instance": "00000000-0000-0000-0000-000000000000",
		"state": "IN_ORBIT"`)

	m, err := Get(logrus.New(), context.Background(), 1234)
	require.NoError(t, err)
	require.Equal(t, characterconst.PresenceStateOffline, m.State())
}

// TestGet_NotFoundIsErrNotFound — 404 means "no row at all", i.e. a character
// who has never logged in. Task 8 logs it as a distinct branch from OFFLINE.
func TestGet_NotFoundIsErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetBaseURLForTest(srv.URL))

	_, err := Get(logrus.New(), context.Background(), 1234)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound), "expected ErrNotFound, got %v", err)
}

// TestGet_InfrastructureErrorIsNotErrNotFound — a 5xx must stay distinguishable
// from a missing row: /find logs the two at different levels.
func TestGet_InfrastructureErrorIsNotErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(SetBaseURLForTest(srv.URL))

	_, err := Get(logrus.New(), context.Background(), 1234)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotFound), "expected non-ErrNotFound, got ErrNotFound")
}
