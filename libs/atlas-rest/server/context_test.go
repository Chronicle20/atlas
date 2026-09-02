package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

type optionalInputModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
}

func (m optionalInputModel) GetName() string { return "optional-inputs" }
func (m optionalInputModel) GetID() string   { return m.Id }
func (m *optionalInputModel) SetID(s string) error {
	m.Id = s
	return nil
}

func newOptionalInputDeps(t *testing.T) (*server.HandlerDependency, *server.HandlerContext) {
	t.Helper()
	l, _ := test.NewNullLogger()
	d := server.NewHandlerDependency(l, nil)
	c := server.NewHandlerContext(fakeServer{})
	return &d, &c
}

func runParseOptionalInput(t *testing.T, body string) (*httptest.ResponseRecorder, bool, optionalInputModel) {
	t.Helper()
	d, c := newOptionalInputDeps(t)

	var called bool
	var got optionalInputModel
	h := server.ParseOptionalInput[optionalInputModel](d, c, func(_ *server.HandlerDependency, _ *server.HandlerContext, model optionalInputModel) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			called = true
			got = model
			w.WriteHeader(http.StatusOK)
		}
	})

	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(http.MethodPost, "/", nil)
	} else {
		req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	}
	w := httptest.NewRecorder()
	h(w, req)
	return w, called, got
}

func TestParseOptionalInput_AbsentBodyDecodesToZeroValue(t *testing.T) {
	w, called, got := runParseOptionalInput(t, "")
	require.True(t, called, "handler must be reached for an absent body")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, optionalInputModel{}, got)
}

func TestParseOptionalInput_EmptyObjectDecodesToZeroValue(t *testing.T) {
	w, called, got := runParseOptionalInput(t, "{}")
	require.True(t, called, "handler must be reached for a {} body")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, optionalInputModel{}, got)
}

func TestParseOptionalInput_ValidEnvelopeDecodes(t *testing.T) {
	body := `{"data":{"type":"optional-inputs","attributes":{"name":"foo"}}}`
	w, called, got := runParseOptionalInput(t, body)
	require.True(t, called, "handler must be reached for a valid envelope")
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "foo", got.Name)
}

func TestParseOptionalInput_MalformedJSONIs400(t *testing.T) {
	w, called, _ := runParseOptionalInput(t, "{not json")
	require.False(t, called, "handler must not be reached for malformed JSON")
	assertJSONAPIBadRequest(t, w)
}

func TestParseOptionalInput_ValidJSONWithoutEnvelopeIs400(t *testing.T) {
	w, called, _ := runParseOptionalInput(t, `{"foo":1}`)
	require.False(t, called, "handler must not be reached for an envelope-less body")
	assertJSONAPIBadRequest(t, w)
}

func TestParseInput_MalformedJSONIs400JSONAPIShape(t *testing.T) {
	d, c := newOptionalInputDeps(t)
	h := server.ParseInput[optionalInputModel](d, c, func(_ *server.HandlerDependency, _ *server.HandlerContext, _ optionalInputModel) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("handler reached for malformed input")
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{not json"))
	w := httptest.NewRecorder()
	h(w, req)

	assertJSONAPIBadRequest(t, w)
}

func TestParseInput_EmptyBodyIsStill400JSONAPIShape(t *testing.T) {
	// ParseInput's own optionality is unchanged: an absent body remains an
	// error there. Only the error's shape changes (bare 400 -> vnd.api+json
	// errors-array document).
	d, c := newOptionalInputDeps(t)
	h := server.ParseInput[optionalInputModel](d, c, func(_ *server.HandlerDependency, _ *server.HandlerContext, _ optionalInputModel) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("handler reached for an empty body")
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	h(w, req)

	assertJSONAPIBadRequest(t, w)
}

func assertJSONAPIBadRequest(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "application/vnd.api+json", w.Header().Get("Content-Type"))
	require.NotEmpty(t, w.Body.Bytes(), "error response body must not be empty")

	var doc struct {
		Errors []jsonapi.Error `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))
	require.Len(t, doc.Errors, 1)
	require.Equal(t, "400", doc.Errors[0].Status)
}
