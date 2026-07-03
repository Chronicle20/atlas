package quest_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-query-aggregator/quest"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// questDoc renders a JSON:API document for quests with QuestId [from, to]
// belonging to a character. meta describes the current page/total so
// requests.DrainProvider can decide whether to keep paging.
func questDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"quest-status","attributes":{"characterId":42,"questId":%d,"state":1,"startedAt":"2024-01-01T00:00:00Z","progress":[]}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetQuestsByCharacterDrainsBeyondOnePage proves GetQuestsByCharacter
// (via requests.DrainProvider) fetches every page of a character's quests
// rather than stopping after the first response. atlas-quest's GET
// /characters/{characterId}/quests is now paginated (task-117); the
// fixture server serves 300 quests across two pages of 250 -- only a
// genuine drain picks up questId 300, which lives on page 2 (needed by
// GetValidationContextProvider to build the complete questId->Model map
// used for validation checks).
func TestGetQuestsByCharacterDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(questDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(questDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := quest.NewProcessor(l, ctx).GetQuestsByCharacter(characterId)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 quests (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.QuestId() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("questId 300 (page 2) must be present; single-fetch impl would miss it")
	}
}

// TestGetStartedQuestsByCharacterDrainsBeyondOnePage proves
// GetStartedQuestsByCharacter (via requests.DrainProvider) fetches every
// page of a character's started quests rather than stopping after the
// first response. atlas-quest's GET
// /characters/{characterId}/quests/started is now paginated (task-117);
// this was a same-class gap to GetQuestsByCharacter's drain conversion,
// caught by task-117's task-29 acceptance sweep (grep 3).
func TestGetStartedQuestsByCharacterDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(questDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(questDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := quest.NewProcessor(l, ctx).GetStartedQuestsByCharacter(characterId)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 started quests (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.QuestId() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("questId 300 (page 2) must be present; single-fetch impl would miss it")
	}
}

// TestGetCompletedQuestsByCharacterDrainsBeyondOnePage proves
// GetCompletedQuestsByCharacter (via requests.DrainProvider) fetches every
// page of a character's completed quests rather than stopping after the
// first response. atlas-quest's GET
// /characters/{characterId}/quests/completed is now paginated (task-117);
// this was a same-class gap to GetQuestsByCharacter's drain conversion,
// caught by task-117's task-29 acceptance sweep (grep 3).
func TestGetCompletedQuestsByCharacterDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(questDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(questDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ms, err := quest.NewProcessor(l, ctx).GetCompletedQuestsByCharacter(characterId)()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 300 {
		t.Fatalf("expected 300 completed quests (full drain), got %d; a single-fetch implementation would return 250", len(ms))
	}

	foundLast := false
	for _, m := range ms {
		if m.QuestId() == 300 {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("questId 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
