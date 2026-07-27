package state_test

import (
	"atlas-saga-orchestrator/quest/state"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// startedQuestDoc renders a JSON:API document for started quests with
// QuestId [from, to] belonging to a character. meta describes the current
// page/total so requests.DrainProvider can decide whether to keep paging.
func startedQuestDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"quest-status","attributes":{"characterId":42,"questId":%d,"state":1}}`,
			id, id,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestGetStartedQuestIdsDrainsBeyondOnePage proves GetStartedQuestIds (via
// requests.DrainProvider) fetches every page of a character's started
// quests rather than stopping after the first response. atlas-quest's GET
// /characters/{characterId}/quests/started is now paginated (task-117);
// the fixture server serves 300 started quests across two pages of 250 --
// only a genuine drain picks up questId 300, which lives on page 2 (needed
// so reactor-drop quest matching doesn't silently miss quests started
// while a character has more than 250 active quests).
func TestGetStartedQuestIdsDrainsBeyondOnePage(t *testing.T) {
	characterId := uint32(42)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(startedQuestDoc(251, 300, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(startedQuestDoc(1, 250, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("QUESTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ids, err := state.GetStartedQuestIds(l)(ctx)(characterId)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 300 {
		t.Fatalf("expected 300 started quest ids (full drain), got %d; a single-fetch implementation would return 250", len(ids))
	}
	if !ids[300] {
		t.Fatal("questId 300 (page 2) must be present; single-fetch impl would miss it")
	}
}
