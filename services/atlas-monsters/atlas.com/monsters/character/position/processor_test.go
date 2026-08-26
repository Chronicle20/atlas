package position

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
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

func TestProcessor_GetPosition(t *testing.T) {
	const wantX, wantY = int16(123), int16(-456)

	tests := []struct {
		name        string
		characterId uint32
		handler     func(t *testing.T) http.HandlerFunc
		wantErr     error
		wantX       int16
		wantY       int16
	}{
		{
			name:        "projects coordinates",
			characterId: 1001,
			handler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, http.MethodGet, r.Method)
					require.Equal(t, "/characters/1001", r.URL.Path)
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusOK)
					_, _ = fmt.Fprintf(w, jsonAPIResponseTmpl, uint32(1001), wantX, wantY)
				}
			},
			wantX: wantX,
			wantY: wantY,
		},
		{
			name:        "propagates not found",
			characterId: 9999,
			handler: func(t *testing.T) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantErr: requests.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler(t))
			defer srv.Close()

			defer withBaseURL(srv.URL)()

			logger := logrus.New()
			logger.SetLevel(logrus.PanicLevel)
			p := NewProcessor(logger, context.Background())

			gotX, gotY, err := p.GetPosition(tt.characterId)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantX, gotX)
			require.Equal(t, tt.wantY, gotY)
		})
	}
}
