package position

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

// jsonAPIResponseTmpl is a JSON:API document as emitted by atlas-character.
// It carries mapId and hp attributes that RestModel deliberately does not
// declare; the unmarshal layer simply ignores fields not present on it.
const jsonAPIResponseTmpl = `{
    "data": {
        "type": "characters",
        "id": "%d",
        "attributes": {
            "mapId": 100000000,
            "hp": 875,
            "x": %d,
            "y": %d
        }
    }
}`

func withBaseURL(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) {
		return strings.TrimRight(url, "/") + "/", nil
	}
	return func() { baseURLProvider = prev }
}

func TestProcessor_GetPosition_ProjectsCoordinates(t *testing.T) {
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

	gotX, gotY, err := p.GetPosition(characterId)
	require.NoError(t, err)
	require.Equal(t, wantX, gotX)
	require.Equal(t, wantY, gotY)
}

func TestProcessor_GetPosition_PropagatesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	defer withBaseURL(srv.URL)()

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	p := NewProcessor(logger, context.Background())

	_, _, err := p.GetPosition(9999)
	require.ErrorIs(t, err, requests.ErrNotFound)
}
