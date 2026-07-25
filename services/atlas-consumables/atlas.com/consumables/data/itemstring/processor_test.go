package itemstring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestGetName decodes an item-strings JSON:API document from atlas-data and
// returns the name attribute.
func TestGetName(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"item-strings","id":"2041303","attributes":{"name":"Ilbi Throwing-Stars"}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")

	name, err := NewProcessor(logrus.New(), context.Background()).GetName(2041303)
	require.NoError(t, err)
	require.Equal(t, "Ilbi Throwing-Stars", name)
	require.True(t, strings.HasSuffix(gotPath, "/data/item-strings/2041303"), "path: %s", gotPath)
}

// TestGetNameError surfaces a transport error to the caller.
func TestGetNameError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")

	_, err := NewProcessor(logrus.New(), context.Background()).GetName(2041303)
	require.Error(t, err)
}
