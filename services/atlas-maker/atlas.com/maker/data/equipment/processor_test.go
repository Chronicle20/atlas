package equipment_test

import (
	"atlas-maker/data/equipment"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// TestGetByIdDecodesReqLevel stands up an httptest server emulating
// atlas-data's equipment lookup and asserts the processor decodes reqLevel.
func TestGetByIdDecodesReqLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"statistics","id":"1302000","attributes":{"reqLevel":10}}}`))
	}))
	defer func() { srv.Close() }()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := equipment.NewProcessor(testLogger(), context.Background()).GetById(item.Id(1302000))
	require.NoError(t, err)
	assert.Equal(t, item.Id(1302000), m.Id())
	assert.EqualValues(t, 10, m.ReqLevel())
}

// TestGetByIdNotFound proves a 404 from atlas-data surfaces as
// requests.ErrNotFound, distinguishable from a genuine read failure.
func TestGetByIdNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer func() { srv.Close() }()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	_, err := equipment.NewProcessor(testLogger(), context.Background()).GetById(item.Id(9999999))
	require.Error(t, err)
	assert.True(t, errors.Is(err, requests.ErrNotFound))
}
