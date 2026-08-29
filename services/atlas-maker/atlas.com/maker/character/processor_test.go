package character_test

import (
	"atlas-maker/character"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// TestGetByIdDecodesLevelAndMeso stands up an httptest server emulating
// atlas-character's by-id lookup and asserts the processor decodes level and
// meso.
func TestGetByIdDecodesLevelAndMeso(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"1001","attributes":{"level":40,"meso":1000000}}}`))
	}))
	defer func() { srv.Close() }()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	m, err := character.NewProcessor(testLogger(), context.Background()).GetById(1001)
	require.NoError(t, err)

	assert.Equal(t, "/characters/1001", gotPath)
	assert.EqualValues(t, 1001, m.Id())
	assert.EqualValues(t, 40, m.Level())
	assert.EqualValues(t, 1000000, m.Meso())
}

// TestGetByIdNotFound proves a 404 from atlas-character surfaces as
// requests.ErrNotFound, distinguishable from a genuine read failure.
func TestGetByIdNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer func() { srv.Close() }()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	_, err := character.NewProcessor(testLogger(), context.Background()).GetById(9999999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, requests.ErrNotFound))
}
