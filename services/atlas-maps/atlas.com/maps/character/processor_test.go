package character

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// jsonAPIResponseTmpl is a JSON:API document scoped to just the fields the
// atlas-maps client consumes. atlas-character emits many more attributes;
// the unmarshal layer simply ignores fields not present on RestModel.
const jsonAPIResponseTmpl = `{
    "data": {
        "type": "characters",
        "id": "%d",
        "attributes": {
            "mapId": 100000000,
            "x": %d,
            "y": %d
        }
    }
}`

// jsonAPIResponseWithHpTmpl additionally sets hp, for the Snapshot test that
// asserts HP is projected alongside position.
const jsonAPIResponseWithHpTmpl = `{
    "data": {
        "type": "characters",
        "id": "%d",
        "attributes": {
            "mapId": 100000000,
            "x": %d,
            "y": %d,
            "hp": %d
        }
    }
}`

func withBaseURL(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string {
		return strings.TrimRight(url, "/") + "/"
	}
	return func() { baseURLProvider = prev }
}

func TestProcessor_Snapshot_ReturnsCoordinatesFromAtlasCharacter(t *testing.T) {
	const wantX, wantY = int16(123), int16(-456)
	const characterId = uint32(1001)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/characters/1001", r.URL.Path)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, jsonAPIResponseTmpl, characterId, wantX, wantY)
	}))
	defer srv.Close()

	defer withBaseURL(srv.URL)()

	logger, _ := test.NewNullLogger()
	p := NewProcessor(logger, context.Background())

	gotX, gotY, _, err := p.Snapshot(characterId)
	require.NoError(t, err)
	require.Equal(t, wantX, gotX)
	require.Equal(t, wantY, gotY)
}

func TestProcessor_Snapshot_PropagatesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	defer withBaseURL(srv.URL)()

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	p := NewProcessor(logger, context.Background())

	_, _, _, err := p.Snapshot(9999)
	require.ErrorIs(t, err, requests.ErrNotFound)
}

// The mist tick needs HP as well as position: a dead character must not be
// healed by a Recovery Aura (FR-5.3), and atlas-character's ChangeMP clamps
// to max MP but does not check HP.
func TestSnapshot_ProjectsPositionAndHp(t *testing.T) {
	const wantX, wantY, wantHp = int16(120), int16(-40), uint16(875)
	const characterId = uint32(1001)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, jsonAPIResponseWithHpTmpl, characterId, wantX, wantY, wantHp)
	}))
	t.Cleanup(srv.Close)

	defer withBaseURL(srv.URL)()

	logger, _ := test.NewNullLogger()
	x, y, hp, err := NewProcessor(logger, context.Background()).Snapshot(characterId)

	require.NoError(t, err)
	require.Equal(t, wantX, x)
	require.Equal(t, wantY, y)
	require.Equal(t, wantHp, hp)
}
