package character

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
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

func withBaseURL(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string {
		return strings.TrimRight(url, "/") + "/"
	}
	return func() { baseURLProvider = prev }
}

func TestProcessor_Position_ReturnsCoordinatesFromAtlasCharacter(t *testing.T) {
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

	gotX, gotY, err := p.Position(characterId)
	require.NoError(t, err)
	require.Equal(t, wantX, gotX)
	require.Equal(t, wantY, gotY)
}

func TestProcessor_Position_PropagatesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	defer withBaseURL(srv.URL)()

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	p := NewProcessor(logger, context.Background())

	_, _, err := p.Position(9999)
	require.ErrorIs(t, err, requests.ErrNotFound)
}
