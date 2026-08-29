package skill_test

import (
	"atlas-maker/skill"
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

// skillDoc renders a JSON:API document for skills [from, to]. meta describes
// the current page/total so requests.DrainProvider can decide whether to
// keep paging.
func skillDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(`{"id":"%d","type":"skills","attributes":{"level":5}}`, id))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetByCharacterIdDecodesLevel asserts the client parses a one-page
// response into the model.
func TestGetByCharacterIdDecodesLevel(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(skillDoc(1007, 1007, 1, 1, 250, 1)))
	}))
	defer func() { srv.Close() }()
	t.Setenv("SKILLS_SERVICE_URL", srv.URL+"/")

	ms, err := skill.NewProcessor(testLogger(), context.Background()).GetByCharacterId(1001)
	require.NoError(t, err)

	assert.Equal(t, "/characters/1001/skills", gotPath)
	require.Len(t, ms, 1)
	assert.EqualValues(t, 1007, ms[0].Id())
	assert.EqualValues(t, 5, ms[0].Level())
}

// TestGetByCharacterIdNotFound proves a 404 from atlas-skills surfaces as
// requests.ErrNotFound, distinguishable from a genuine read failure.
func TestGetByCharacterIdNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer func() { srv.Close() }()
	t.Setenv("SKILLS_SERVICE_URL", srv.URL+"/")

	_, err := skill.NewProcessor(testLogger(), context.Background()).GetByCharacterId(9999999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, requests.ErrNotFound))
}

// TestGetByCharacterIdDrainsBeyondOnePage proves GetByCharacterId (via
// requests.DrainProvider) fetches every page of a character's skill list
// rather than stopping after the first response. The fixture server serves
// skill ids [1, 300] across two pages of 250 -- only a genuine drain picks
// up skill id 1007 (the Maker skill, in this fixture placed on page 2).
// A single-fetch implementation would silently drop a character's Maker
// skill and reject every craft with skill_level_too_low.
func TestGetByCharacterIdDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(skillDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(skillDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer func() { srv.Close() }()
	t.Setenv("SKILLS_SERVICE_URL", srv.URL+"/")

	ms, err := skill.NewProcessor(testLogger(), context.Background()).GetByCharacterId(1001)
	require.NoError(t, err)
	require.Len(t, ms, 300)

	foundLast := false
	for _, m := range ms {
		if m.Id() == 300 {
			foundLast = true
		}
	}
	assert.True(t, foundLast, "skill id 300 (page 2) must be present; single-fetch impl would miss it")
}
