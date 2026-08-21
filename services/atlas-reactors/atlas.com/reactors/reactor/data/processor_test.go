package data

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

// jsonAPIResponseTmpl is a representative atlas-data reactor JSON:API
// document. It includes activateByTouch and a populated touchAreaInfo so the
// real requests.GetRequest[RestModel] unmarshal path is exercised end to end
// (rest_test.go only exercises Extract/MarshalJSON in-process).
const jsonAPIResponseTmpl = `{
    "data": {
        "type": "reactors",
        "id": "%d",
        "attributes": {
            "name": "TRUNK",
            "tl": {"x": -100, "y": -50},
            "br": {"x": 100, "y": 50},
            "activateByTouch": true,
            "touchAreaInfo": {
                "0": {"tl": {"x": -53, "y": 24}, "br": {"x": 62, "y": 69}}
            },
            "stateInfo": {},
            "timeoutInfo": {},
            "timeoutNextStateInfo": {}
        }
    }
}`

func withDataServiceURL(t *testing.T, url string) {
	t.Setenv("DATA_SERVICE_URL", strings.TrimRight(url, "/")+"/")
}

func TestSnapshot(t *testing.T) {
	const reactorId = uint32(1)

	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr bool
	}{
		{
			name: "success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/vnd.api+json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, jsonAPIResponseTmpl, reactorId)
			},
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

			withDataServiceURL(t, srv.URL)

			logger, _ := test.NewNullLogger()
			p := NewProcessor(logger, context.Background())

			m, err := p.GetById(reactorId)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.True(t, m.ActivateByTouch())
			a, ok := m.TouchArea(0)
			require.True(t, ok)
			require.Equal(t, int16(-53), a.TL().X())
			require.Equal(t, int16(24), a.TL().Y())
			require.Equal(t, int16(62), a.BR().X())
			require.Equal(t, int16(69), a.BR().Y())
		})
	}
}
