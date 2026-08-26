package character

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// jsonAPIResponseTmpl is a JSON:API document scoped to just the fields the
// atlas-reactors client consumes. atlas-character emits many more
// attributes; the unmarshal layer simply ignores fields not present on
// RestModel.
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
	baseURLProvider = func(_ context.Context) (string, error) {
		return strings.TrimRight(url, "/") + "/", nil
	}
	return func() { baseURLProvider = prev }
}

func TestSnapshot(t *testing.T) {
	const characterId = uint32(1)

	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantX   int16
		wantY   int16
		wantErr bool
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, jsonAPIResponseTmpl, characterId, 250, -300)
			},
			wantX:   250,
			wantY:   -300,
			wantErr: false,
		},
		{
			name: "not found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
		{
			name: "malformed body",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "not json")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			defer withBaseURL(srv.URL)()

			logger, _ := test.NewNullLogger()
			p := NewProcessor(logger, context.Background())

			gotX, gotY, err := p.Position(characterId)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantX, gotX)
			require.Equal(t, tt.wantY, gotY)
		})
	}
}
