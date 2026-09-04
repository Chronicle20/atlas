package quest_test

import (
	"atlas-maker/quest"
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

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// questDoc renders a JSON:API document for quest-status entries [from, to].
func questDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		_, _ = fmt.Fprintf(&b, `{"id":"%d","type":"quest-status","attributes":{"characterId":1001,"questId":%d,"state":2}}`, id, id)
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetByCharacterIdDecodesQuestIdAndState asserts the client parses a
// one-page response into the model.
func TestGetByCharacterIdDecodesQuestIdAndState(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(questDoc(21341, 21341, 1, 1, 250, 1)))
	}))
	defer func() { srv.Close() }()
	t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

	ms, err := quest.NewProcessor(testLogger(), context.Background()).GetByCharacterId(1001)
	require.NoError(t, err)

	assert.Equal(t, "/characters/1001/quests", gotPath)
	require.Len(t, ms, 1)
	assert.EqualValues(t, 1001, ms[0].CharacterId())
	assert.EqualValues(t, 21341, ms[0].QuestId())
	assert.EqualValues(t, 2, ms[0].State())
}

// TestGetByCharacterIdNotFound proves a 404 from atlas-quest surfaces as
// requests.ErrNotFound, distinguishable from a genuine read failure.
func TestGetByCharacterIdNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer func() { srv.Close() }()
	t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

	_, err := quest.NewProcessor(testLogger(), context.Background()).GetByCharacterId(9999999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, requests.ErrNotFound))
}

// TestGetByCharacterIdDrainsBeyondOnePage proves GetByCharacterId (via
// requests.DrainProvider) fetches every page of a character's quest list
// rather than stopping after the first response. A single-fetch
// implementation would silently miss a reqQuest ingredient whose quest lives
// past page one.
func TestGetByCharacterIdDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(questDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(questDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer func() { srv.Close() }()
	t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

	ms, err := quest.NewProcessor(testLogger(), context.Background()).GetByCharacterId(1001)
	require.NoError(t, err)
	require.Len(t, ms, 300)

	foundLast := false
	for _, m := range ms {
		if m.QuestId() == 300 {
			foundLast = true
		}
	}
	assert.True(t, foundLast, "quest id 300 (page 2) must be present; single-fetch impl would miss it")
}
