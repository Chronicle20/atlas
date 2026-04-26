package server_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeServer struct{}

func (fakeServer) GetBaseURL() string { return "" }
func (fakeServer) GetPrefix() string  { return "" }

type fakeRow struct {
	Id   string `json:"-"`
	Name string `json:"name"`
}

func (r fakeRow) GetName() string { return "fakes" }
func (r fakeRow) GetID() string   { return r.Id }
func (r *fakeRow) SetID(s string) error {
	r.Id = s
	return nil
}

func TestMarshalPaginatedResponse_EmitsMetaAndLinks(t *testing.T) {
	l, _ := test.NewNullLogger()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/fakes?page[number]=2&page[size]=10", nil)

	rows := []fakeRow{{Id: "1", Name: "A"}, {Id: "2", Name: "B"}}
	env := paginate.Envelope{Total: 25, PageNumber: 2, PageSize: 10}

	server.MarshalPaginatedResponse[[]fakeRow](l)(w)(fakeServer{})(map[string][]string{})(rows, env, req)

	require.Equal(t, 200, w.Code)
	var doc jsonapi.Document
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))

	require.NotNil(t, doc.Meta)
	assert.EqualValues(t, 25, doc.Meta["total"])
	page := doc.Meta["page"].(map[string]interface{})
	assert.EqualValues(t, 2, page["number"])
	assert.EqualValues(t, 10, page["size"])
	assert.EqualValues(t, 3, page["last"])

	require.NotNil(t, doc.Links)
	assert.Contains(t, doc.Links, "self")
	assert.Contains(t, doc.Links, "first")
	assert.Contains(t, doc.Links, "prev")
	assert.Contains(t, doc.Links, "next")
	assert.Contains(t, doc.Links, "last")
}
