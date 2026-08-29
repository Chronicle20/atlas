package itemmake_test

import (
	"atlas-maker/data/itemmake"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

const itemMakeResponse = `{
  "data": {
    "type": "itemMakes",
    "id": "1382005",
    "attributes": {
      "group": 1,
      "reqLevel": 40,
      "reqSkillLevel": 1,
      "itemNum": 1,
      "tuc": 0,
      "meso": 5000,
      "catalyst": 4260000,
      "reqItem": 0,
      "reqEquip": 1382000,
      "recipe": [{"itemId": 4010000, "count": 5}],
      "randomReward": [{"itemId": 1382005, "itemNum": 1, "prob": 100}],
      "reqQuest": [{"questId": 21341, "state": 2}]
    }
  }
}`

// TestGetByIdDecodesRecipe asserts every field this service consumes
// decodes correctly, including the nested recipe/randomReward/reqQuest
// lists.
func TestGetByIdDecodesRecipe(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(itemMakeResponse))
	}))
	defer func() { srv.Close() }()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	m, err := itemmake.NewProcessor(testLogger(), context.Background()).GetById(item.Id(1382005))
	require.NoError(t, err)

	assert.Equal(t, "/data/item-makes/1382005", gotPath)
	assert.Equal(t, item.Id(1382005), m.Id())
	assert.EqualValues(t, 1, m.Group())
	assert.EqualValues(t, 40, m.ReqLevel())
	assert.EqualValues(t, 1, m.ReqSkillLevel())
	assert.EqualValues(t, 1, m.ItemNum())
	assert.EqualValues(t, 0, m.Tuc())
	assert.EqualValues(t, 5000, m.Meso())
	assert.EqualValues(t, 4260000, m.Catalyst())
	assert.EqualValues(t, 0, m.ReqItem())
	assert.EqualValues(t, 1382000, m.ReqEquip())

	require.Len(t, m.Recipe(), 1)
	assert.Equal(t, item.Id(4010000), m.Recipe()[0].ItemId())
	assert.EqualValues(t, 5, m.Recipe()[0].Count())

	require.Len(t, m.RandomReward(), 1)
	assert.Equal(t, item.Id(1382005), m.RandomReward()[0].ItemId())
	assert.EqualValues(t, 1, m.RandomReward()[0].ItemNum())
	assert.EqualValues(t, 100, m.RandomReward()[0].Prob())

	require.Len(t, m.ReqQuest(), 1)
	assert.EqualValues(t, 21341, m.ReqQuest()[0].QuestId())
	assert.EqualValues(t, 2, m.ReqQuest()[0].State())
}

// TestGetByIdNotFound proves a 404 from atlas-data surfaces as
// requests.ErrNotFound, distinguishable from a genuine read failure.
func TestGetByIdNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer func() { srv.Close() }()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	_, err := itemmake.NewProcessor(testLogger(), context.Background()).GetById(item.Id(9999999))
	require.Error(t, err)
	assert.True(t, errors.Is(err, requests.ErrNotFound))
}

// itemMakeDoc renders a JSON:API document for recipes [from, to], with a
// dummy reqEquip so the recipe attribute set decodes cleanly.
func itemMakeDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"itemMakes","attributes":{"group":1,"reqLevel":10,"reqSkillLevel":1,"itemNum":1,"tuc":0,"meso":100,"catalyst":0,"reqItem":0,"reqEquip":0,"recipe":[],"randomReward":[],"reqQuest":[]}}`,
			id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetAllDrainsBeyondOnePage proves GetAll (via requests.DrainProvider)
// fetches every page of the recipe catalog rather than stopping after the
// first response. atlas-data's GET /data/item-makes is paginated with a
// default page size of 50 (services/atlas-data/atlas.com/data/itemmake/resource.go);
// the fixture server serves 60 recipes across two pages of 50 -- only a
// genuine drain picks up recipe id 60, which lives on page 2.
func TestGetAllDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(itemMakeDoc(51, 60, 60, 2, 50, 2)))
			return
		}
		_, _ = w.Write([]byte(itemMakeDoc(1, 50, 60, 1, 50, 2)))
	}))
	defer func() { srv.Close() }()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	ms, err := itemmake.NewProcessor(testLogger(), context.Background()).GetAll()
	require.NoError(t, err)
	require.Len(t, ms, 60)

	foundLast := false
	for _, m := range ms {
		if m.Id() == 60 {
			foundLast = true
		}
	}
	assert.True(t, foundLast, "recipe id 60 (page 2) must be present; single-fetch impl would miss it")
}
